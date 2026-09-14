package mia

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seqTestTrack builds a track buffer: LOOP header + the given events.
func seqTestTrack(loop uint16, events ...byte) []byte {
	track := []byte{byte(loop), byte(loop >> 8), 0, 0}
	return append(track, events...)
}

func seqNoteEvent(freq uint16, dur uint32) []byte {
	return []byte{
		miaSeqOpNote, byte(freq), byte(freq >> 8),
		byte(dur), byte(dur >> 8), byte(dur >> 16),
	}
}

func seqRestEvent(dur uint32) []byte {
	return []byte{miaSeqOpRest, byte(dur), byte(dur >> 8), byte(dur >> 16)}
}

func writeSeqTrack(chip *emulated_mia, voice int, track []byte) {
	base := miaSeqTrackOffset(voice)
	copy(chip.memory[base:base+uint32(len(track))], track)
}

// TestEmulatedMiaAudioSeqLoadResetsPosition verifies SEQ_LOAD copies the
// track into the engine's own buffer and resets cursor/note index, mirroring
// mia_audio_seq_load_track.
func TestEmulatedMiaAudioSeqLoadResetsPosition(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	track := seqTestTrack(0, append(seqNoteEvent(0x1234, 3), miaSeqOpEnd)...)
	writeSeqTrack(chip, 0, track)

	chip.audioSeqLoadTrack(0)

	chip.audio.mu.Lock()
	seq := chip.audio.seq[0]
	ram := chip.audio.seqRAM[0][:len(track)]
	chip.audio.mu.Unlock()

	assert.Equal(t, uint32(miaSeqTrackEventsOffset), seq.cursor)
	assert.Zero(t, seq.noteIndex)
	assert.False(t, seq.running)
	assert.True(t, seq.hasLoop, "LOOP=0 should mean loop the whole body")
	assert.Equal(t, uint32(miaSeqTrackEventsOffset), seq.loopCursor)
	assert.Equal(t, track, ram, "engine copy should match what was written to MIA RAM")
}

// TestEmulatedMiaAudioSeqPlaysAndLoops steps the engine sample by sample and
// verifies notes trigger in order, the note index advances, and the track
// loops back to note 1 - mirroring audio_seq_advance's loop handling.
func TestEmulatedMiaAudioSeqPlaysAndLoops(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	track := seqTestTrack(0,
		append(append(seqNoteEvent(0x1111, 2), seqNoteEvent(0x2222, 1)...), miaSeqOpEnd)...)
	writeSeqTrack(chip, 0, track)

	chip.audioSeqLoadTrack(0)
	chip.audioSeqStart(1 << 0)
	chip.audioEnable()

	// Note 1 triggers on the very first render (countdown starts at 0).
	chip.audio.renderSampleLocked() // call 1: trigger note 1, countdown=2
	assert.Equal(t, uint16(0x1111), chip.audio.voices[0].freqQ4)
	assert.Equal(t, uint16(1), chip.audio.seq[0].noteIndex)

	// countdown=2 holds note 1 for two more renders (calls 2,3: 2->1->0)
	// before call 4 finds countdown==0 and advances to note 2.
	chip.audio.renderSampleLocked() // call 2
	chip.audio.renderSampleLocked() // call 3
	chip.audio.renderSampleLocked() // call 4: advance -> trigger note 2, countdown=1
	assert.Equal(t, uint16(0x2222), chip.audio.voices[0].freqQ4)
	assert.Equal(t, uint16(2), chip.audio.seq[0].noteIndex)

	// countdown=1 holds note 2 for one more render (call 5: 1->0), then call
	// 6 finds countdown==0, hits END, and loops back to note 1.
	chip.audio.renderSampleLocked() // call 5
	chip.audio.renderSampleLocked() // call 6: advance -> END loops -> trigger note 1 again
	assert.Equal(t, uint16(0x1111), chip.audio.voices[0].freqQ4)
	assert.Equal(t, uint16(1), chip.audio.seq[0].noteIndex, "loop should restore note index to 1, not keep climbing")
	assert.True(t, chip.audio.seq[0].running)
}

// TestEmulatedMiaAudioSeqEndWithoutLoopStopsAndDefersIRQ verifies a
// non-looping track stops and gates off, and that the render goroutine
// defers IRQ_AUDIO_SEQ_DONE for audioSeqService (called from Tick) to raise -
// it cannot call irqSetFlag itself. See the audio_sequencer.go file comment.
func TestEmulatedMiaAudioSeqEndWithoutLoopStopsAndDefersIRQ(t *testing.T) {
	circuit := silentAudioCircuit(t)
	chip := circuit.chip
	chip.writeRegisterWord(miaRegIRQMaskLSB, miaIRQAudioSeqDone)

	track := seqTestTrack(miaSeqNoLoop, append(seqNoteEvent(0x1234, 0), miaSeqOpEnd)...)
	writeSeqTrack(chip, 0, track)

	chip.audioSeqLoadTrack(0)
	chip.audioSeqStart(1 << 0)
	chip.audioEnable()

	chip.audio.renderSampleLocked() // triggers the note, countdown=0
	chip.audio.renderSampleLocked() // countdown already 0: advances straight to END

	chip.audio.mu.Lock()
	running := chip.audio.seq[0].running
	pending := chip.audio.seqDonePending
	chip.audio.mu.Unlock()
	assert.False(t, running)
	assert.True(t, pending)
	assert.Zero(t, chip.irqStatus()&miaIRQAudioSeqDone, "not raised until audioSeqService drains it")

	chip.audioSeqService()
	assert.NotZero(t, chip.irqStatus()&miaIRQAudioSeqDone)

	chip.audio.mu.Lock()
	pending = chip.audio.seqDonePending
	chip.audio.mu.Unlock()
	assert.False(t, pending, "draining should clear the pending flag")
}

// TestEmulatedMiaAudioSeqTakeThenQuickGiveResumesSameEvent verifies that
// releasing a voice before its current event would naturally have ended
// resumes that same event without retriggering it - the "already_spent"
// term in the reconciliation, see docs/audio-sequencer.md.
func TestEmulatedMiaAudioSeqTakeThenQuickGiveResumesSameEvent(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	track := seqTestTrack(0, append(seqNoteEvent(0x1234, 100), miaSeqOpEnd)...)
	writeSeqTrack(chip, 0, track)

	chip.audioSeqLoadTrack(0)
	chip.audioSeqStart(1 << 0)
	chip.audioEnable()

	chip.audio.renderSampleLocked() // note triggers: countdown=100, eventDuration=100
	chip.audio.renderSampleLocked() // countdown -> 99
	chip.audio.renderSampleLocked() // countdown -> 98

	chip.audioVoiceTake(1 << 0)
	// Simulate an SFX lasting 10 samples while taken.
	for i := 0; i < 10; i++ {
		chip.audio.renderSampleLocked()
	}
	chip.audio.mu.Lock()
	countdownWhileTaken := chip.audio.seq[0].countdown
	chip.audio.mu.Unlock()
	require.Equal(t, uint32(98), countdownWhileTaken, "sequencer must not advance while taken")

	chip.audioVoiceRelease(1 << 0)

	chip.audio.mu.Lock()
	seq := chip.audio.seq[0]
	chip.audio.mu.Unlock()
	assert.False(t, seq.taken)
	assert.False(t, seq.catchingUp, "10 elapsed samples fit well within the note's remaining 98")
	// already_spent (2) + elapsed (10) = 12 spent of 100 -> 88 remaining.
	assert.Equal(t, uint32(88), seq.countdown)
	assert.Equal(t, uint16(1), seq.noteIndex, "must not have retriggered/advanced to a new note")
}

// TestEmulatedMiaAudioSeqTakeThenLongGiveCatchesUpPastEvents verifies that a
// long TAKE causes VOICE_RELEASE to skip whole events that fully elapsed
// while taken, landing live on the first one that didn't - not simply
// resuming from the frozen cursor.
func TestEmulatedMiaAudioSeqTakeThenLongGiveCatchesUpPastEvents(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	// Note A: 10 samples. Note B: 10 samples. Note C: 10 samples. Loop.
	track := seqTestTrack(0,
		append(append(append(
			seqNoteEvent(0xA000, 10),
			seqNoteEvent(0xB000, 10)...),
			seqNoteEvent(0xC000, 10)...),
			miaSeqOpEnd)...)
	writeSeqTrack(chip, 0, track)

	chip.audioSeqLoadTrack(0)
	chip.audioSeqStart(1 << 0)
	chip.audioEnable()

	chip.audio.renderSampleLocked() // note A triggers: eventDuration=10, countdown=10
	for i := 0; i < 4; i++ {
		chip.audio.renderSampleLocked() // 4 decrements: countdown 10 -> 6
	}
	// already_spent = eventDuration - countdown = 10 - 6 = 4.

	chip.audioVoiceTake(1 << 0)
	// Elapse 22 samples while taken. total_elapsed = already_spent(4) + 22 = 26.
	// A(10) is already behind the cursor; B(10) fits entirely within the
	// remaining 26-10=16 budget (fully in the past); C(10) does not fit in
	// what's left (16-10=6 < 10), so C is where playback should resume, live.
	for i := 0; i < 22; i++ {
		chip.audio.renderSampleLocked()
	}

	chip.audioVoiceRelease(1 << 0)
	// Catch-up resolves within the budget in a single subsequent sample.
	chip.audio.renderSampleLocked()

	chip.audio.mu.Lock()
	seq := chip.audio.seq[0]
	freq := chip.audio.voices[0].freqQ4
	chip.audio.mu.Unlock()

	assert.False(t, seq.catchingUp)
	assert.Equal(t, uint16(0xC000), freq, "must land on C, not resume A or stop at B")
	assert.Equal(t, uint16(3), seq.noteIndex)
}

// TestEmulatedMiaAudioSeqStatusIndexesReadLiveState verifies PLAYING(v)/CUE(v)
// read live sequencer state through the dedicated $D6-$D9 indexes.
func TestEmulatedMiaAudioSeqStatusIndexesReadLiveState(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	track := seqTestTrack(0, append(seqNoteEvent(0x1234, 5), miaSeqOpEnd)...)
	writeSeqTrack(chip, 0, track)

	chip.audioSeqLoadTrack(0)
	chip.audioSeqStart(1 << 0)
	chip.audioEnable()
	chip.audio.renderSampleLocked()

	assert.Equal(t, uint8(1), chip.indexRead(miaAudioIndexSeqVoice0), "note index low byte")
	assert.Equal(t, uint8(0), chip.indexStepAndRead(miaAudioIndexSeqVoice0, miaIndexWindowA), "note index high byte")
	status := chip.indexStepAndRead(miaAudioIndexSeqVoice0, miaIndexWindowA)
	assert.NotZero(t, status&miaAudioSeqStatusRunning)
	assert.Zero(t, status&miaAudioSeqStatusTaken)

	chip.audioVoiceTake(1 << 0)
	// current_addr sits at the status byte ($0B); three more steps wrap
	// $0B -> $09 -> $0A -> $0B and land back on it.
	chip.indexStepAndRead(miaAudioIndexSeqVoice0, miaIndexWindowA)
	chip.indexStepAndRead(miaAudioIndexSeqVoice0, miaIndexWindowA)
	status = chip.indexStepAndRead(miaAudioIndexSeqVoice0, miaIndexWindowA)
	assert.NotZero(t, status&miaAudioSeqStatusTaken)
}

// TestEmulatedMiaAudioSeqCommandsViaBus drives SEQ_LOAD/SEQ_START/SEQ_STOP/
// VOICE_TAKE/VOICE_RELEASE through the actual register bus, mirroring
// TestEmulatedMiaAudioCommandsToggleState's style for the original commands.
func TestEmulatedMiaAudioSeqCommandsViaBus(t *testing.T) {
	circuit := silentAudioCircuit(t)
	chip := circuit.chip

	track := seqTestTrack(0, append(seqNoteEvent(0x1234, 5), miaSeqOpEnd)...)
	writeSeqTrack(chip, 0, track)

	trigger := func(cmd uint8, mask uint8) {
		circuit.write(miaRegCmdParam1, mask)
		circuit.write(miaRegCmdParam2, 0)
		circuit.write(miaRegCmdParam3, 0)
		circuit.write(miaRegCmdTrigger, cmd)
	}

	trigger(miaCmdAudioSeqLoad, 1<<0)
	chip.audio.mu.Lock()
	assert.Equal(t, uint32(miaSeqTrackEventsOffset), chip.audio.seq[0].cursor)
	chip.audio.mu.Unlock()

	trigger(miaCmdAudioSeqStart, 1<<0)
	chip.audio.mu.Lock()
	assert.True(t, chip.audio.seq[0].running)
	chip.audio.mu.Unlock()

	trigger(miaCmdAudioVoiceTake, 1<<0)
	chip.audio.mu.Lock()
	assert.True(t, chip.audio.seq[0].taken)
	chip.audio.mu.Unlock()

	trigger(miaCmdAudioVoiceRelease, 1<<0)
	chip.audio.mu.Lock()
	assert.False(t, chip.audio.seq[0].taken)
	chip.audio.mu.Unlock()

	trigger(miaCmdAudioSeqStop, 1<<0)
	chip.audio.mu.Lock()
	assert.False(t, chip.audio.seq[0].running)
	chip.audio.mu.Unlock()
}

// TestEmulatedMiaAudioSeqResetClearsTracksAndState verifies AUDIO_RESET (0x62)
// clears track buffers back to "no loop" and resets sequencer runtime state,
// mirroring mia_audio_reset_runtime_state's sequencer extension.
func TestEmulatedMiaAudioSeqResetClearsTracksAndState(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	track := seqTestTrack(0, append(seqNoteEvent(0x1234, 5), miaSeqOpEnd)...)
	writeSeqTrack(chip, 0, track)
	chip.audioSeqLoadTrack(0)
	chip.audioSeqStart(1 << 0)

	chip.audioReset()

	base := miaSeqTrackOffset(0)
	assert.Equal(t, uint8(0xFF), chip.memory[base+miaSeqTrackLoopL])
	assert.Equal(t, uint8(0xFF), chip.memory[base+miaSeqTrackLoopH])

	chip.audio.mu.Lock()
	seq := chip.audio.seq[0]
	chip.audio.mu.Unlock()
	assert.False(t, seq.running)
	assert.False(t, seq.hasLoop)
	assert.Zero(t, seq.noteIndex)
}
