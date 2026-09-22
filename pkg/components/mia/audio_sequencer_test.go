package mia

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seqNoteEvent(freq uint16, dur uint32) []byte {
	return []byte{
		miaSeqOpNote, byte(freq), byte(freq >> 8),
		byte(dur), byte(dur >> 8), byte(dur >> 16),
	}
}

func seqRestEvent(dur uint32) []byte {
	return []byte{miaSeqOpRest, byte(dur), byte(dur >> 8), byte(dur >> 16)}
}

// seqJumpEvent encodes a JUMP opcode with a signed 24-bit little-endian
// offset, relative to the byte right after this 4-byte record.
func seqJumpEvent(offset int32) []byte {
	return []byte{miaSeqOpJump, byte(offset), byte(offset >> 8), byte(offset >> 16)}
}

// TestMiaSeqDefaultRegionDoesNotOverlapSDFS is a regression guard for the bug
// that motivated relocating the sequencer's default addresses off $13000:
// the old defaults (voice 0-2) silently aliased live SD/FS state (control
// block, dir-entry buffer, transfer buffer), since both picked $13000 as a
// starting point independently. This asserts the two claimed ranges are
// disjoint using each subsystem's own named constants, so a future change to
// either one that reintroduces the overlap fails loudly here instead of
// silently corrupting a running track or an open file.
func TestMiaSeqDefaultRegionDoesNotOverlapSDFS(t *testing.T) {
	seqStart := uint32(miaSeqDefaultBaseOffset)
	seqEnd := seqStart + miaSeqDefaultRegionSize // exclusive

	sdStart := uint32(miaSDStateOffset)
	sdEnd := uint32(miaFSTransferOffset + miaFSTransferSize) // exclusive

	overlaps := seqStart < sdEnd && sdStart < seqEnd
	assert.False(t, overlaps,
		"sequencer default region $%05X-$%05X must not overlap SD/FS state $%05X-$%05X",
		seqStart, seqEnd-1, sdStart, sdEnd-1)
}

// writeSeqTrack writes a track's event stream at a voice's default base -
// no header, no length: just bytes, exactly as TRACK/SEQ_SET_BASE leave it.
func writeSeqTrack(chip *emulated_mia, voice int, track []byte) {
	base := miaSeqDefaultBase(voice)
	copy(chip.memory[base:base+uint32(len(track))], track)
}

// TestEmulatedMiaAudioSeqLoadResetsPosition verifies SEQ_LOAD snapshots the
// track out of MIA RAM (from trackBase onward) into the engine's own buffer
// and resets cursor/note index to the top, mirroring mia_audio_seq_load_track.
func TestEmulatedMiaAudioSeqLoadResetsPosition(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	track := append(seqNoteEvent(0x1234, 3), miaSeqOpEnd)
	writeSeqTrack(chip, 0, track)

	chip.audioSeqLoadTrack(0)

	chip.audio.mu.Lock()
	seq := chip.audio.seq[0]
	ram := chip.audio.seqRAM[0][:len(track)]
	chip.audio.mu.Unlock()

	assert.Zero(t, seq.cursor, "cursor starts at offset 0 from trackBase - there is no header anymore")
	assert.Zero(t, seq.noteIndex)
	assert.False(t, seq.running)
	assert.Equal(t, miaSeqDefaultBase(0), seq.trackBase, "unconfigured voices default to the legacy per-voice address")
	assert.Equal(t, track, ram, "engine copy should match what was written to MIA RAM")
}

// TestEmulatedMiaAudioSeqPlaysAndLoops steps the engine sample by sample and
// verifies notes trigger in order, the note index advances, and a trailing
// JUMP loops the track back to the top - mirroring audio_seq_advance's JUMP
// handling (there is no more header-based implicit loop).
func TestEmulatedMiaAudioSeqPlaysAndLoops(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	// note1(6B, offsets 0-5) + note2(6B, offsets 6-11) + JUMP(4B, offsets
	// 12-15, "end"=16) back to offset 0: delta = 0-16 = -16.
	track := append(append(
		seqNoteEvent(0x1111, 2),
		seqNoteEvent(0x2222, 1)...),
		seqJumpEvent(-16)...)
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
	// 6 finds countdown==0, decodes JUMP, and loops back to note 1.
	chip.audio.renderSampleLocked() // call 5
	chip.audio.renderSampleLocked() // call 6: advance -> JUMP -> trigger note 1 again
	assert.Equal(t, uint16(0x1111), chip.audio.voices[0].freqQ4)
	assert.Equal(t, uint16(3), chip.audio.seq[0].noteIndex, "JUMP does not reset note index like the old header-based loop did - it just keeps counting")
	assert.True(t, chip.audio.seq[0].running)
}

// TestEmulatedMiaAudioSeqJumpLoopsToArbitraryOffset verifies a JUMP can loop
// back to any offset within the track, not just the top - the "intro once,
// then loop the body" composition the removed header-based LOOP field could
// not express (it could only ever resume from a single cached offset, but
// nothing stopped that offset being non-zero; the real gap was that the
// *cursor at load* was always the top, and only END consulted the loop
// field - JUMP lets any point in the stream redirect anywhere, live).
func TestEmulatedMiaAudioSeqJumpLoopsToArbitraryOffset(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	intro := seqNoteEvent(0x1111, 1) // 6 bytes: offsets 0-5, plays once
	body := seqNoteEvent(0x2222, 1)  // 6 bytes: offsets 6-11, loops forever
	// JUMP at offsets 12-15, "end"=16, target=6 (top of body): delta=6-16=-10.
	track := append(append(intro, body...), seqJumpEvent(-10)...)
	writeSeqTrack(chip, 0, track)

	chip.audioSeqLoadTrack(0)
	chip.audioSeqStart(1 << 0)
	chip.audioEnable()

	chip.audio.renderSampleLocked() // call 1: trigger intro, countdown=1
	assert.Equal(t, uint16(0x1111), chip.audio.voices[0].freqQ4)

	chip.audio.renderSampleLocked() // call 2: countdown 1->0
	chip.audio.renderSampleLocked() // call 3: advance -> trigger body, countdown=1
	assert.Equal(t, uint16(0x2222), chip.audio.voices[0].freqQ4)
	assert.Equal(t, uint16(2), chip.audio.seq[0].noteIndex)

	chip.audio.renderSampleLocked() // call 4: countdown 1->0
	chip.audio.renderSampleLocked() // call 5: advance -> JUMP -> re-trigger body, not intro
	assert.Equal(t, uint16(0x2222), chip.audio.voices[0].freqQ4, "loop must return to the body, not replay the intro")
	assert.Equal(t, uint16(3), chip.audio.seq[0].noteIndex)
}

// TestEmulatedMiaAudioSeqSetBaseRelocatesTrack verifies a track can be
// loaded and played from any MIA RAM address, not just the legacy per-voice
// default - the whole point of making trackBase configurable and dropping
// the fixed 1024-byte-per-voice ceiling.
func TestEmulatedMiaAudioSeqSetBaseRelocatesTrack(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	const base = uint32(0x20000) // far from any legacy default track region
	track := append(seqNoteEvent(0x4321, 1), miaSeqOpEnd)
	copy(chip.memory[base:base+uint32(len(track))], track)

	chip.audioSeqSetBase(0, base)
	chip.audioSeqLoadTrack(0)
	chip.audioSeqStart(1 << 0)
	chip.audioEnable()

	chip.audio.mu.Lock()
	trackBase := chip.audio.seq[0].trackBase
	chip.audio.mu.Unlock()
	assert.Equal(t, base, trackBase)

	chip.audio.renderSampleLocked()
	assert.Equal(t, uint16(0x4321), chip.audio.voices[0].freqQ4)
}

// TestEmulatedMiaAudioSeqDecodeBudgetStopsRunawayJumpCycle verifies a track
// whose JUMPs cycle with no NOTE/REST/END in between is fail-safe stopped by
// miaSeqDecodeBudget rather than hanging seqAdvance (and the shared audio
// render path) forever.
func TestEmulatedMiaAudioSeqDecodeBudgetStopsRunawayJumpCycle(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	// A JUMP that targets its own tag byte: offset -4 relative to the byte
	// right after its own 4-byte record lands back on offset 0.
	track := seqJumpEvent(-4)
	writeSeqTrack(chip, 0, track)

	chip.audioSeqLoadTrack(0)
	chip.audioSeqStart(1 << 0)
	chip.audioEnable()

	done := make(chan struct{})
	go func() {
		chip.audio.renderSampleLocked()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("renderSampleLocked did not return - a JUMP cycle must be budget-limited, not infinite")
	}

	chip.audio.mu.Lock()
	running := chip.audio.seq[0].running
	chip.audio.mu.Unlock()
	assert.False(t, running, "budget exhaustion should stop the voice, matching an unrecognized opcode")
}

// TestEmulatedMiaAudioSeqEndWithoutLoopStopsAndDefersIRQ verifies a
// non-looping track (no trailing JUMP) stops and gates off, and that the
// render goroutine defers IRQ_AUDIO_SEQ_DONE for audioSeqService (called
// from Tick) to raise - it cannot call irqSetFlag itself. See the
// audio_sequencer.go file comment.
func TestEmulatedMiaAudioSeqEndWithoutLoopStopsAndDefersIRQ(t *testing.T) {
	circuit := silentAudioCircuit(t)
	chip := circuit.chip
	chip.writeRegisterWord(miaRegIRQMaskLSB, miaIRQAudioSeqDone)

	track := append(seqNoteEvent(0x1234, 0), miaSeqOpEnd)
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

	track := append(seqNoteEvent(0x1234, 100), miaSeqOpEnd)
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

	// Note A: 10 samples. Note B: 10 samples. Note C: 10 samples. Then END.
	track := append(append(append(
		seqNoteEvent(0xA000, 10),
		seqNoteEvent(0xB000, 10)...),
		seqNoteEvent(0xC000, 10)...),
		miaSeqOpEnd)
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
// read live sequencer state through the dedicated $EC-$EF indexes.
func TestEmulatedMiaAudioSeqStatusIndexesReadLiveState(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	track := append(seqNoteEvent(0x1234, 5), miaSeqOpEnd)
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

	track := append(seqNoteEvent(0x1234, 5), miaSeqOpEnd)
	writeSeqTrack(chip, 0, track)

	trigger := func(cmd uint8, mask uint8) {
		circuit.write(miaRegCmdParam1, mask)
		circuit.write(miaRegCmdParam2, 0)
		circuit.write(miaRegCmdParam3, 0)
		circuit.write(miaRegCmdTrigger, cmd)
	}

	trigger(miaCmdAudioSeqLoad, 1<<0)
	chip.audio.mu.Lock()
	assert.Zero(t, chip.audio.seq[0].cursor)
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

// TestEmulatedMiaAudioSeqSetBaseCommandViaBus drives the per-voice
// AUDIO_SEQ_SET_BASE<voice> commands through the actual register bus and
// verifies each lands on the right voice with a 24-bit little-endian address.
func TestEmulatedMiaAudioSeqSetBaseCommandViaBus(t *testing.T) {
	circuit := silentAudioCircuit(t)
	chip := circuit.chip

	trigger := func(cmd uint8, addr uint32) {
		circuit.write(miaRegCmdParam1, uint8(addr))
		circuit.write(miaRegCmdParam2, uint8(addr>>8))
		circuit.write(miaRegCmdParam3, uint8(addr>>16))
		circuit.write(miaRegCmdTrigger, cmd)
	}

	trigger(miaCmdAudioSeqSetBase0, 0x20000)
	trigger(miaCmdAudioSeqSetBase1, 0x21000)
	trigger(miaCmdAudioSeqSetBase2, 0x22000)
	trigger(miaCmdAudioSeqSetBase3, 0x23000)

	chip.audio.mu.Lock()
	defer chip.audio.mu.Unlock()
	assert.Equal(t, uint32(0x20000), chip.audio.seq[0].trackBase)
	assert.Equal(t, uint32(0x21000), chip.audio.seq[1].trackBase)
	assert.Equal(t, uint32(0x22000), chip.audio.seq[2].trackBase)
	assert.Equal(t, uint32(0x23000), chip.audio.seq[3].trackBase)
}

// TestEmulatedMiaAudioSeqResetClearsTracksAndState verifies AUDIO_RESET (0x62)
// clears the legacy default track region and resets sequencer runtime state
// (including reseeding trackBase back to the legacy default), mirroring
// mia_audio_reset_runtime_state's sequencer extension.
func TestEmulatedMiaAudioSeqResetClearsTracksAndState(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	track := append(seqNoteEvent(0x1234, 5), miaSeqOpEnd)
	writeSeqTrack(chip, 0, track)
	chip.audioSeqLoadTrack(0)
	chip.audioSeqStart(1 << 0)

	chip.audioReset()

	base := miaSeqDefaultBase(0)
	assert.Zero(t, chip.memory[base], "reset clears the legacy default track region")

	chip.audio.mu.Lock()
	seq := chip.audio.seq[0]
	chip.audio.mu.Unlock()
	assert.False(t, seq.running)
	assert.Equal(t, base, seq.trackBase, "reset reseeds the legacy default base")
	assert.Zero(t, seq.noteIndex)
}
