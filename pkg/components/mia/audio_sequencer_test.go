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

// seqTieEvent encodes a TIE: a NOTE that only changes the frequency.
func seqTieEvent(freq uint16, dur uint32) []byte {
	event := seqNoteEvent(freq, dur)
	event[0] = miaSeqOpTie
	return event
}

// seqSlowAttack is a SET_ADSR with a 100 ms attack, so a few ticks into a
// note the envelope is still climbing and a restart is easy to see.
var seqSlowAttack = []byte{miaSeqOpSetADSR, 0x80, 0xF0}

// seqJumpEvent encodes a JUMP opcode with a signed 24-bit little-endian
// offset, relative to the byte right after this 4-byte record.
func seqJumpEvent(offset int32) []byte {
	return []byte{miaSeqOpJump, byte(offset), byte(offset >> 8), byte(offset >> 16)}
}

// renderTick renders one sequencer tick: miaAudioSamplesPerTick output
// samples, the first of which steps the sequencer.
func renderTick(a *miaAudioState) {
	for i := 0; i < miaAudioSamplesPerTick; i++ {
		a.renderSampleLocked()
	}
}

// testSeqBase is where these tests put each voice's track. Any free MIA RAM
// works: the sequencer has no default address of its own.
func testSeqBase(voice int) uint32 {
	return 0x20000 + uint32(voice)*0x1000
}

// writeSeqTrack writes a track's event stream and points the voice at it -
// no header, no length: just bytes and a SET_BASE, exactly as TRACK leaves it.
func writeSeqTrack(chip *emulated_mia, voice int, track []byte) {
	base := testSeqBase(voice)
	copy(chip.memory[base:base+uint32(len(track))], track)
	chip.audioSeqSetBase(voice, base)
}

func TestDebugAudioSnapshotIncludesLiveSequencerStatus(t *testing.T) {
	chip := silentAudioCircuit(t).chip
	require.Len(t, chip.DebugAudioSnapshot(), miaAudioStateSize)
	writeSeqTrack(chip, 0, append(seqNoteEvent(0x1234, 100), miaSeqOpEnd))
	chip.audioSeqLoadTrack(0)
	chip.audioSeqStart(1)
	chip.audioEnable()
	renderTick(&chip.audio)

	base := miaAudioHeaderSize
	snapshot := chip.DebugAudioSnapshot()
	assert.Equal(t, uint8(1), snapshot[base+miaAudioVoiceSeqNoteIndexL])
	assert.Equal(t, miaAudioSeqStatusRunning, snapshot[base+miaAudioVoiceSeqStatus])
	assert.Zero(t, chip.memory[miaAudioVoiceOffset(0)+miaAudioVoiceSeqNoteIndexL], "live note index is absent from emulator RAM")
	chip.audioVoiceTake(1)
	snapshot = chip.DebugAudioSnapshot()
	assert.Equal(t, miaAudioSeqStatusRunning|miaAudioSeqStatusTaken, snapshot[base+miaAudioVoiceSeqStatus])
	snapshot[0] = 0
	assert.Equal(t, uint8(miaAudioVersion), chip.DebugAudioSnapshot()[0], "snapshot owns its bytes")
}

// TestEmulatedMiaAudioSeqLoadResetsPosition verifies SEQ_LOAD keeps the
// track's MIA RAM view and resets cursor/note index to the top.
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
	assert.Equal(t, testSeqBase(0), seq.trackBase, "the voice keeps the base its track was written at")
	assert.Equal(t, track, ram, "engine view should show the bytes in MIA RAM")
}

// TestEmulatedMiaAudioSeqReadsCPUWritesAfterLoad verifies that the 6502 can
// extend a running stream through an indexed MIA RAM window. The END byte is
// replaced after SEQ_LOAD, before the decoder reaches it.
func TestEmulatedMiaAudioSeqReadsCPUWritesAfterLoad(t *testing.T) {
	circuit := silentAudioCircuit(t)
	chip := circuit.chip
	base := testSeqBase(0)
	writeSeqTrack(chip, 0, append(seqNoteEvent(0x1111, 1), miaSeqOpEnd))
	chip.audioSeqLoadTrack(0)
	chip.audioSeqStart(1)
	chip.audioEnable()
	reader := &miaAudioReader{chip: chip}
	frame := make([]byte, 4)

	n, err := reader.Read(frame)
	require.NoError(t, err)
	require.Equal(t, 4, n)
	assert.Equal(t, uint16(0x1111), chip.audio.voices[0].freqQ4)

	const index = uint8(0x40)
	chip.indexes[index] = miaIndex{currentAddr: base + 6, defaultAddr: base + 6,
		limitAddr: base + 13, step: 1, flags: 1 << miaIndexFlagWriteStep}
	circuit.write(miaRegIdxASelector, index)
	for _, b := range append(seqNoteEvent(0x2222, 0), miaSeqOpEnd) {
		circuit.write(miaRegIdxAPort, b)
	}

	for range miaAudioSamplesPerTick { // finish the first note's single tick, then decode the newly written note
		n, err = reader.Read(frame)
		require.NoError(t, err)
		require.Equal(t, 4, n)
	}
	assert.Equal(t, uint16(0x2222), chip.audio.voices[0].freqQ4)
	assert.Equal(t, uint16(2), chip.audio.seq[0].noteIndex)
	assert.True(t, chip.audio.seq[0].running)
}

// TestEmulatedMiaAudioSeqConcurrentRAMWrites exercises the audio reader while
// the CPU edits an upcoming note through its indexed window. Both paths must
// finish without a lock inversion, and the race detector must see no RAM race.
func TestEmulatedMiaAudioSeqConcurrentRAMWrites(t *testing.T) {
	circuit := silentAudioCircuit(t)
	chip := circuit.chip
	base := testSeqBase(0)
	writeSeqTrack(chip, 0, append(seqNoteEvent(0x1111, 0), seqJumpEvent(-10)...))
	chip.audioSeqLoadTrack(0)
	chip.audioSeqStart(1)
	chip.audioEnable()

	const index = uint8(0x40)
	chip.indexes[index] = miaIndex{currentAddr: base + 1, defaultAddr: base + 1,
		limitAddr: base + 2, step: 0}
	circuit.write(miaRegIdxASelector, index)

	reader := &miaAudioReader{chip: chip}
	done := make(chan struct{})
	go func() {
		defer close(done)
		frame := make([]byte, 4)
		for range 1000 {
			if _, err := reader.Read(frame); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	for i := range 1000 {
		circuit.write(miaRegIdxAPort, uint8(i))
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("audio reader stalled while the CPU wrote MIA RAM")
	}
}

// TestEmulatedMiaAudioSeqPlaysAndLoops steps the engine tick by tick and
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
	renderTick(&chip.audio) // call 1: trigger note 1, countdown=1
	assert.Equal(t, uint16(0x1111), chip.audio.voices[0].freqQ4)
	assert.Equal(t, uint16(1), chip.audio.seq[0].noteIndex)

	// Note 1 lasts exactly 2 ticks: call 2 counts 1->0, then call 3 finds
	// countdown==0 and advances to note 2.
	renderTick(&chip.audio) // call 2
	renderTick(&chip.audio) // call 3: advance -> trigger note 2, countdown=0
	assert.Equal(t, uint16(0x2222), chip.audio.voices[0].freqQ4)
	assert.Equal(t, uint16(2), chip.audio.seq[0].noteIndex)

	// Note 2 lasts exactly 1 tick, so call 4 decodes JUMP and loops back to
	// note 1.
	renderTick(&chip.audio) // call 4: advance -> JUMP -> trigger note 1 again
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

	renderTick(&chip.audio) // call 1: trigger intro, countdown=0
	assert.Equal(t, uint16(0x1111), chip.audio.voices[0].freqQ4)

	renderTick(&chip.audio) // call 2: advance -> trigger body, countdown=0
	assert.Equal(t, uint16(0x2222), chip.audio.voices[0].freqQ4)
	assert.Equal(t, uint16(2), chip.audio.seq[0].noteIndex)

	renderTick(&chip.audio) // call 3: advance -> JUMP -> re-trigger body, not intro
	assert.Equal(t, uint16(0x2222), chip.audio.voices[0].freqQ4, "loop must return to the body, not replay the intro")
	assert.Equal(t, uint16(3), chip.audio.seq[0].noteIndex)
}

// TestEmulatedMiaAudioSeqSetBaseRelocatesTrack verifies a track can be
// loaded and played from any MIA RAM address - the whole point of making
// trackBase configurable and dropping the fixed 1024-byte-per-voice ceiling.
func TestEmulatedMiaAudioSeqSetBaseRelocatesTrack(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	const base = uint32(0x3A000) // anywhere free works; not where testSeqBase puts tracks
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

	renderTick(&chip.audio)
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
		renderTick(&chip.audio)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("renderTick did not return - a JUMP cycle must be budget-limited, not infinite")
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

	renderTick(&chip.audio) // triggers the note, countdown=0
	renderTick(&chip.audio) // countdown already 0: advances straight to END

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

	renderTick(&chip.audio) // note triggers: eventDuration=100, countdown=99
	renderTick(&chip.audio) // countdown -> 98
	renderTick(&chip.audio) // countdown -> 97

	chip.audioVoiceTake(1 << 0)
	// Simulate an SFX lasting 10 ticks while taken.
	for i := 0; i < 10; i++ {
		renderTick(&chip.audio)
	}
	chip.audio.mu.Lock()
	countdownWhileTaken := chip.audio.seq[0].countdown
	chip.audio.mu.Unlock()
	require.Equal(t, uint32(97), countdownWhileTaken, "sequencer must not advance while taken")

	chip.audioVoiceRelease(1 << 0)

	chip.audio.mu.Lock()
	seq := chip.audio.seq[0]
	chip.audio.mu.Unlock()
	assert.False(t, seq.taken)
	assert.False(t, seq.catchingUp, "10 elapsed ticks fit well within the note's remaining 97")
	// already_spent (3) + elapsed (10) = 13 spent of 100 -> 87 remaining.
	assert.Equal(t, uint32(87), seq.countdown)
	assert.Equal(t, uint16(1), seq.noteIndex, "must not have retriggered/advanced to a new note")
}

// TestEmulatedMiaAudioSeqTakeThenLongGiveCatchesUpPastEvents verifies that a
// long TAKE causes VOICE_RELEASE to skip whole events that fully elapsed
// while taken, landing live on the first one that didn't - not simply
// resuming from the frozen cursor.
func TestEmulatedMiaAudioSeqTakeThenLongGiveCatchesUpPastEvents(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	// Note A: 10 ticks. Note B: 10 ticks. Note C: 10 ticks. Then END.
	track := append(append(append(
		seqNoteEvent(0xA000, 10),
		seqNoteEvent(0xB000, 10)...),
		seqNoteEvent(0xC000, 10)...),
		miaSeqOpEnd)
	writeSeqTrack(chip, 0, track)

	chip.audioSeqLoadTrack(0)
	chip.audioSeqStart(1 << 0)
	chip.audioEnable()

	renderTick(&chip.audio) // note A triggers: eventDuration=10, countdown=9
	for i := 0; i < 4; i++ {
		renderTick(&chip.audio) // 4 decrements: countdown 9 -> 5
	}
	// already_spent = eventDuration - countdown = 10 - 5 = 5 ticks played.

	chip.audioVoiceTake(1 << 0)
	// Elapse 22 ticks while taken. total_elapsed = already_spent(5) + 22 = 27.
	// A(10) is already behind the cursor; B(10) fits entirely within the
	// remaining 27-10=17 budget (fully in the past); C(10) does not fit in
	// what's left (17-10=7 < 10), so C is where playback should resume, live.
	for i := 0; i < 22; i++ {
		renderTick(&chip.audio)
	}

	chip.audioVoiceRelease(1 << 0)
	// Catch-up resolves within the budget in a single subsequent tick.
	renderTick(&chip.audio)

	chip.audio.mu.Lock()
	seq := chip.audio.seq[0]
	freq := chip.audio.voices[0].freqQ4
	chip.audio.mu.Unlock()

	assert.False(t, seq.catchingUp)
	assert.Equal(t, uint16(0xC000), freq, "must land on C, not resume A or stop at B")
	assert.Equal(t, uint16(3), seq.noteIndex)
}

// TestEmulatedMiaAudioSeqEventLastsExactlyDurTicks pins the event length the
// sequencer guide documents: a NOTE/TIE/REST lasts exactly dur ticks, counting
// the tick that decodes it, and a dur of 0 still takes that one tick.
func TestEmulatedMiaAudioSeqEventLastsExactlyDurTicks(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	track := append(append(append(
		seqNoteEvent(0xA000, 3),
		miaSeqOpRest, 0, 0, 0),
		seqNoteEvent(0xB000, 2)...),
		miaSeqOpEnd)
	writeSeqTrack(chip, 0, track)

	chip.audioSeqLoadTrack(0)
	chip.audioSeqStart(1 << 0)
	chip.audioEnable()

	// Note index after each tick: A holds ticks 1-3, the zero-length REST
	// tick 4, B ticks 5-6, and tick 7 reaches END.
	for tick, want := range []uint16{1, 1, 1, 2, 3, 3} {
		renderTick(&chip.audio)
		assert.Equal(t, want, chip.audio.seq[0].noteIndex, "tick %d", tick+1)
	}
	assert.True(t, chip.audio.seq[0].running)

	renderTick(&chip.audio)
	assert.False(t, chip.audio.seq[0].running, "END decodes on the tick after B's two ticks")
}

// TestEmulatedMiaAudioSeqNoteAfterNoteRestartsEnvelope verifies a NOTE
// straight after a NOTE drops and raises the gate, so the second note gets
// its own attack from silence instead of joining the first.
func TestEmulatedMiaAudioSeqNoteAfterNoteRestartsEnvelope(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	track := append(append(append(append([]byte{},
		seqSlowAttack...),
		seqNoteEvent(0xA000, 3)...),
		seqNoteEvent(0xB000, 3)...),
		miaSeqOpEnd)
	writeSeqTrack(chip, 0, track)

	chip.audioSeqLoadTrack(0)
	chip.audioSeqStart(1 << 0)
	chip.audioEnable()

	renderTick(&chip.audio) // tick 1: A starts, one attack step
	firstStep := chip.audio.voices[0].vol
	renderTick(&chip.audio)
	renderTick(&chip.audio) // ticks 2-3: A keeps climbing
	require.Greater(t, chip.audio.voices[0].vol, firstStep)

	renderTick(&chip.audio) // tick 4: B starts
	voice := chip.audio.voices[0]
	assert.Equal(t, uint16(0xB000), voice.freqQ4)
	assert.Equal(t, miaADSRAttack, voice.adsr)
	assert.Equal(t, firstStep, voice.vol, "B's envelope restarts from silence, like A's did")
	assert.Equal(t, uint16(2), chip.audio.seq[0].noteIndex)
}

// TestEmulatedMiaAudioSeqTieContinuesSoundingNote verifies a TIE changes only
// the frequency: the gate stays up and the envelope keeps going, and it still
// counts as an event for CUE.
func TestEmulatedMiaAudioSeqTieContinuesSoundingNote(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	track := append(append(append(append([]byte{},
		seqSlowAttack...),
		seqNoteEvent(0xA000, 3)...),
		seqTieEvent(0xB000, 3)...),
		miaSeqOpEnd)
	writeSeqTrack(chip, 0, track)

	chip.audioSeqLoadTrack(0)
	chip.audioSeqStart(1 << 0)
	chip.audioEnable()

	for i := 0; i < 3; i++ {
		renderTick(&chip.audio) // ticks 1-3: A
	}
	before := chip.audio.voices[0].vol

	renderTick(&chip.audio) // tick 4: the TIE moves to B
	voice := chip.audio.voices[0]
	assert.Equal(t, uint16(0xB000), voice.freqQ4)
	assert.Equal(t, miaADSRAttack, voice.adsr)
	assert.Greater(t, voice.vol, before, "the envelope carries on from where A left it")
	assert.NotZero(t, voice.control&miaAudioControlGate)
	assert.Equal(t, uint16(2), chip.audio.seq[0].noteIndex)
}

// TestEmulatedMiaAudioSeqTieAfterRestStartsNote verifies a TIE has nothing
// to continue after a REST, so it starts a new note exactly like NOTE.
func TestEmulatedMiaAudioSeqTieAfterRestStartsNote(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	track := append(append(append(append([]byte{},
		seqSlowAttack...),
		seqNoteEvent(0xA000, 3)...),
		miaSeqOpRest, 1, 0, 0),
		seqTieEvent(0xB000, 3)...)
	track = append(track, miaSeqOpEnd)
	writeSeqTrack(chip, 0, track)

	chip.audioSeqLoadTrack(0)
	chip.audioSeqStart(1 << 0)
	chip.audioEnable()

	renderTick(&chip.audio) // tick 1: A starts
	firstStep := chip.audio.voices[0].vol
	renderTick(&chip.audio)
	renderTick(&chip.audio) // ticks 2-3: A
	renderTick(&chip.audio) // tick 4: REST releases the voice
	renderTick(&chip.audio) // tick 5: TIE
	voice := chip.audio.voices[0]
	assert.Equal(t, uint16(0xB000), voice.freqQ4)
	assert.NotZero(t, voice.control&miaAudioControlGate)
	assert.Equal(t, miaADSRAttack, voice.adsr)
	assert.Equal(t, firstStep, voice.vol, "the TIE starts its own attack from silence")
	assert.Equal(t, uint16(3), chip.audio.seq[0].noteIndex)
}

// TestEmulatedMiaAudioSeqCatchUpLandingOnTieStartsNote verifies VGIVE
// landing on a TIE starts a new note even though the gate is still up: the
// note it continued was skipped, so the envelope restarts.
func TestEmulatedMiaAudioSeqCatchUpLandingOnTieStartsNote(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	track := append(append(append(append([]byte{},
		seqSlowAttack...),
		seqNoteEvent(0xA000, 10)...),
		seqTieEvent(0xB000, 10)...),
		miaSeqOpEnd)
	writeSeqTrack(chip, 0, track)

	chip.audioSeqLoadTrack(0)
	chip.audioSeqStart(1 << 0)
	chip.audioEnable()

	renderTick(&chip.audio) // tick 1: A starts
	firstStep := chip.audio.voices[0].vol
	chip.audioVoiceTake(1 << 0)
	for i := 0; i < 11; i++ {
		renderTick(&chip.audio) // ticks 2-12: A keeps sounding; the TIE is 2 ticks in
	}
	chip.audioVoiceRelease(1 << 0)
	renderTick(&chip.audio)

	chip.audio.mu.Lock()
	seq := chip.audio.seq[0]
	voice := chip.audio.voices[0]
	chip.audio.mu.Unlock()

	assert.False(t, seq.catchingUp)
	assert.Equal(t, uint16(2), seq.noteIndex)
	assert.Equal(t, uint16(0xB000), voice.freqQ4)
	assert.Equal(t, miaADSRAttack, voice.adsr)
	assert.Equal(t, firstStep, voice.vol, "the TIE restarts the envelope instead of continuing A's")
}

// TestEmulatedMiaAudioSeqCatchUpLandsWhereLivePlaybackIs verifies VGIVE's
// catch-up counts events the same length live playback does: a voice taken
// and released lands on the same event, with the same countdown, as an
// identical voice that was never taken.
func TestEmulatedMiaAudioSeqCatchUpLandsWhereLivePlaybackIs(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	track := append(append(append(
		seqNoteEvent(0xA000, 10),
		seqNoteEvent(0xB000, 10)...),
		seqNoteEvent(0xC000, 10)...),
		miaSeqOpEnd)
	writeSeqTrack(chip, 0, track)
	writeSeqTrack(chip, 1, track)

	chip.audioSeqLoadTrack(0)
	chip.audioSeqLoadTrack(1)
	chip.audioSeqStart(1<<0 | 1<<1)
	chip.audioEnable()

	renderTick(&chip.audio) // tick 1: both voices start A
	chip.audioVoiceTake(1 << 0)
	for i := 0; i < 19; i++ {
		renderTick(&chip.audio) // ticks 2-20: A ends on tick 10, B on tick 20
	}
	chip.audioVoiceRelease(1 << 0)
	renderTick(&chip.audio) // tick 21: the live voice starts C, the released one catches up to it

	chip.audio.mu.Lock()
	taken, live := chip.audio.seq[0], chip.audio.seq[1]
	chip.audio.mu.Unlock()

	assert.Equal(t, uint16(3), live.noteIndex, "live voice is on C")
	assert.Equal(t, live.noteIndex, taken.noteIndex)
	assert.Equal(t, live.countdown, taken.countdown)
	assert.False(t, taken.catchingUp)
}

// TestEmulatedMiaAudioSeqStatusIndexesReadLiveState verifies PLAYING(v)/CUE(v)
// read live state through dedicated and configured general indexes alike.
func TestEmulatedMiaAudioSeqStatusIndexesReadLiveState(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	track := append(seqNoteEvent(0x1234, 5), miaSeqOpEnd)
	writeSeqTrack(chip, 0, track)

	chip.audioSeqLoadTrack(0)
	chip.audioSeqStart(1 << 0)
	chip.audioEnable()
	renderTick(&chip.audio)

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

	// SongPosition configures a general index for these same three addresses.
	const general = uint8(0x40)
	base := miaAudioVoiceOffset(0) + miaAudioVoiceSeqNoteIndexL
	chip.indexes[general] = miaIndex{currentAddr: base, defaultAddr: base, limitAddr: base + 3, step: 1,
		flags: (1 << miaIndexFlagReadStep) | (1 << miaIndexFlagWrap)}
	assert.Equal(t, uint8(1), chip.indexRead(general))
	assert.Equal(t, uint8(0), chip.indexStepAndRead(general, miaIndexWindowA))
	assert.Equal(t, status, chip.indexStepAndRead(general, miaIndexWindowA))
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

// TestEmulatedMiaAudioSeqResetForgetsTracks verifies AUDIO_RESET (0x62) forgets
// every voice's track and resets sequencer runtime state, but leaves the track
// bytes in MIA RAM alone, mirroring mia_audio_reset_runtime_state.
func TestEmulatedMiaAudioSeqResetForgetsTracks(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	track := append(seqNoteEvent(0x1234, 5), miaSeqOpEnd)
	writeSeqTrack(chip, 0, track)
	chip.audioSeqLoadTrack(0)
	chip.audioSeqStart(1 << 0)

	chip.audioReset()

	assert.Equal(t, track[0], chip.memory[testSeqBase(0)], "reset leaves the track bytes where the program put them")

	chip.audio.mu.Lock()
	seq := chip.audio.seq[0]
	chip.audio.mu.Unlock()
	assert.False(t, seq.running)
	assert.Equal(t, miaSeqNoBase, seq.trackBase, "reset forgets the voice's track")
	assert.Zero(t, seq.noteIndex)
}

// TestEmulatedMiaAudioSeqStartWithoutTrackLeavesVoiceStopped verifies
// SEQ_START skips a voice no program has given a track: it doesn't run, gate
// off or raise IRQ_AUDIO_SEQ_DONE, while a voice with a track in the same
// mask starts normally - mirroring audio_seq_start_one.
func TestEmulatedMiaAudioSeqStartWithoutTrackLeavesVoiceStopped(t *testing.T) {
	chip := silentAudioCircuit(t).chip

	writeSeqTrack(chip, 0, append(seqNoteEvent(0x1234, 5), miaSeqOpEnd))
	chip.audioSeqLoadTrack(0)
	chip.audioSeqLoadTrack(1) // voice 1 has no base, so there is nothing to load
	chip.audioSeqStart(0x03)
	chip.audioEnable()

	for i := 0; i < 4; i++ {
		renderTick(&chip.audio)
	}

	chip.audio.mu.Lock()
	voice0, voice1 := chip.audio.seq[0], chip.audio.seq[1]
	pending := chip.audio.seqDonePending
	chip.audio.mu.Unlock()
	assert.True(t, voice0.running, "voice 0 has a track and starts")
	assert.False(t, voice1.running, "voice 1 has no track and stays stopped")
	assert.Equal(t, miaSeqNoBase, voice1.trackBase)
	assert.Zero(t, voice1.noteIndex)
	assert.False(t, pending, "a voice that never started raises no sequence-done")
}
