package clementina

import "testing"

// TRACK/BAND/VTAKE/VGIVE/PLAYING/CUE (handlers in clementina-rom
// src/basic/clementina_extra.s) are the background sequencer's BASIC
// surface, replacing the retired PLAY. TRACK compiles a voice's MML string
// straight into MIA's sequencer bytecode, by default at $13000+voice*$400
// (a third, optional argument relocates it anywhere in MIA RAM - see
// clementina-mia's docs/audio-sequencer.md); these tests boot the real ROM,
// type a program, run it, and read the encoded bytes and command effects
// back - the same style sound_test.go uses for the direct register
// statements.
//
// Most of BAND is exercised only for its synchronous, RAM-only effects
// (SEQ_LOAD/SEQ_START/SEQ_STOP/VOICE_TAKE/VOICE_RELEASE all mutate sequencer
// state directly in the command handler) - like sound_test.go, SNDON itself
// is avoided here since it opens a real host audio device. The one test that
// needs playback to actually advance over rendered time
// (TestBasicTrackBandAdvancesLiveRegisters) uses the mia package's
// DebugSetAudioActive/DebugAudioReadPCM/DebugAudioVoiceState hooks instead -
// force the engine active without a real device, then render frames directly
// and read back what the sequencer did, the same shape the old play_test.go
// used for blocking PLAY. The sequencer's own timing/catch-up logic is
// already covered in the mia package's own tests (TestEmulatedMiaAudioSeq*);
// this is the seam connecting real BASIC-driven commands to that engine.

const seqTrackBase = 0x14000
const seqTrackSize = 0x1000

func peekSeq(computer *ClementinaComputer, voice uint32, offset uint32) uint8 {
	return computer.chips.mia.(audioMemReader).DebugReadVideo(seqTrackBase + voice*seqTrackSize + offset)
}

type audioAdvancer interface {
	DebugSetAudioActive(bool)
	DebugAudioReadPCM(int) []int16
	DebugAudioVoiceState(int) (freqQ4 uint16, control uint8, adsr uint8, vol uint32)
}

// TestBasicTrackBandAdvancesLiveRegisters is the end-to-end check: a real
// BASIC program compiles a two-note TRACK and starts it with BAND, then the
// audio engine is forced forward by rendered samples (bypassing wall-clock
// pacing and a real audio device, not the sequencer logic itself) and the
// live voice registers are read back - confirming a BAND-started track
// actually reaches the render loop and advances note to note, not just that
// the commands that start it dispatch correctly.
func TestBasicTrackBandAdvancesLiveRegisters(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, `10 TRACK 0,"T60 L8 O5 C D"`)
	typeLine(computer, step, "20 BAND 1")
	typeLine(computer, step, "30 GOTO 30")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	audio := computer.chips.mia.(audioAdvancer)
	audio.DebugSetAudioActive(true)

	// T60, eighth note (k=8: T>>1) = 30 ticks; 30*150 = 4500 samples, plus one
	// trigger tick per event (see audio_sequencer.go's seqAdvance): each note
	// spans 4501 rendered samples. 2000 lands well inside C5; the cumulative
	// 8000 lands well inside D5 (started at 4501, ends at 9002).
	audio.DebugAudioReadPCM(2000)
	freq, control, _, _ := audio.DebugAudioVoiceState(0)
	require_eq_u16(t, 0x20C0, freq, "C5 (note_freq(60) = 262<<5) mid-first-note")
	require_eq(t, 0x03, control, "gate+reset-phase mid-first-note (fresh trigger)")

	audio.DebugAudioReadPCM(6000) // cumulative 8000
	freq, control, _, _ = audio.DebugAudioVoiceState(0)
	require_eq_u16(t, 0x24C0, freq, "D5 (note_freq(62) = 294<<5) mid-second-note")
	require_eq(t, 0x03, control, "gate+reset-phase mid-second-note (fresh trigger)")
}

func require_eq_u16(t *testing.T, want uint16, got uint16, msg string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: want %04X got %04X", msg, want, got)
	}
}

// TestBasicTrackEncodesNote verifies TRACK's encoder against a hand-computed
// expected byte sequence: T60 L8 O5 C is a single NOTE event (C5, an eighth
// note at tempo 60 -> 30 kernel ticks -> 30*150 = 4500 audio samples) then
// END, with no loop. There is no header - the event stream starts at offset 0.
func TestBasicTrackEncodesNote(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, `10 TRACK 0,"T60 L8 O5 C"`)
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	require_eq(t, 0x01, peekSeq(computer, 0, 0), "opcode NOTE")
	// C5: note_freq(60) = octave-0 262 << 5 = 8384 = $20C0.
	require_eq(t, 0xC0, peekSeq(computer, 0, 1), "freq_l")
	require_eq(t, 0x20, peekSeq(computer, 0, 2), "freq_h")
	// T60, eighth note (k=8: T>>1) = 30 ticks; 30*150 = 4500 = $001194.
	require_eq(t, 0x94, peekSeq(computer, 0, 3), "dur_l")
	require_eq(t, 0x11, peekSeq(computer, 0, 4), "dur_m")
	require_eq(t, 0x00, peekSeq(computer, 0, 5), "dur_h")
	require_eq(t, 0x00, peekSeq(computer, 0, 6), "opcode END")
}

// TestBasicTrackLoopMarker verifies "|" records the absolute address of
// whatever follows it and, at the end of the string, emits a JUMP back to
// it instead of an END - the intro (first note, 6 bytes: NOTE takes
// op+freq_l+freq_h+dur(3)) plays once, only the second note loops.
func TestBasicTrackLoopMarker(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, `10 TRACK 1,"C|D"`)
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	require_eq(t, 0x01, peekSeq(computer, 1, 0), "first opcode NOTE (C)")
	require_eq(t, 0x01, peekSeq(computer, 1, 6), "second opcode NOTE (D), right at the loop point")
	// JUMP's record is offsets 12-15 ("end" = 16); target is offset 6 (D):
	// delta = 6-16 = -10, 24-bit two's complement = $FFFFF6.
	require_eq(t, 0x08, peekSeq(computer, 1, 12), "opcode JUMP back to the loop point")
	require_eq(t, 0xF6, peekSeq(computer, 1, 13), "offset_l")
	require_eq(t, 0xFF, peekSeq(computer, 1, 14), "offset_m")
	require_eq(t, 0xFF, peekSeq(computer, 1, 15), "offset_h")
}

// TestBasicTrackWaveToken verifies the W n inline token emits SET_WAVE.
func TestBasicTrackWaveToken(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, `10 TRACK 2,"W2 C"`)
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	require_eq(t, 0x03, peekSeq(computer, 2, 0), "opcode SET_WAVE")
	require_eq(t, 0x02, peekSeq(computer, 2, 1), "waveform value (saw)")
	require_eq(t, 0x01, peekSeq(computer, 2, 2), "opcode NOTE follows immediately (zero duration)")
}

// TestBasicTrackExplicitAddress verifies TRACK's optional third argument
// relocates a voice's track to an arbitrary MIA RAM address instead of the
// legacy per-voice default, and that BAND/PLAYING still work from there -
// the point of dropping the fixed 1024-byte-per-voice slot.
func TestBasicTrackExplicitAddress(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	const relocated = 0x20000 // far from any legacy default track slot

	typeLine(computer, step, `10 TRACK 0,"C",131072`) // 131072 = $20000
	typeLine(computer, step, "20 BAND 0,1")
	typeLine(computer, step, "30 POKE 752,PLAYING(0)")
	typeLine(computer, step, "40 GOTO 40")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	reader := computer.chips.mia.(audioMemReader)
	require_eq(t, 0x01, reader.DebugReadVideo(relocated), "opcode NOTE compiled at the explicit address, not the legacy default")
	require_eq(t, 1, peek(computer, addrScratch), "PLAYING(0) true - the relocated track actually started")
	if got := peekSeq(computer, 0, 0); got == 0x01 {
		t.Fatalf("legacy default slot for voice 0 must not also have been written: got opcode NOTE there too")
	}
}

const addrScratch = 0x02F0 // 752, the low-RAM POKE/PEEK scratch cell other tests already use.

// TestBasicBandStartsAndStopsGlobally verifies BAND 1 starts every voice with
// a loaded track and BAND 0 stops all of them, observed through PLAYING(v).
// Each half boots its own computer: a program stuck in an infinite GOTO never
// returns to accept new typed lines, so "start, then stop" cannot be driven
// as two RUNs of the same still-looping program.
func TestBasicBandStartsAndStopsGlobally(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	typeLine(computer, step, `10 TRACK 0,"C"`)
	typeLine(computer, step, `20 TRACK 1,"C"`)
	typeLine(computer, step, "30 BAND 1")
	typeLine(computer, step, "40 POKE 752,PLAYING(0)*10+PLAYING(1)")
	typeLine(computer, step, "50 GOTO 50")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)
	require_eq(t, 11, peek(computer, addrScratch), "both voices running after BAND 1")

	computer2, step2 := bootClementinaToPrompt(t)
	typeLine(computer2, step2, `10 TRACK 0,"C"`)
	typeLine(computer2, step2, `20 TRACK 1,"C"`)
	typeLine(computer2, step2, "30 BAND 1")
	typeLine(computer2, step2, "40 BAND 0")
	typeLine(computer2, step2, "50 POKE 752,PLAYING(0)*10+PLAYING(1)")
	typeLine(computer2, step2, "60 GOTO 60")
	typeLine(computer2, step2, "RUN")
	editorTickN(computer2, step2, 2_000_000)
	require_eq(t, 0, peek(computer2, addrScratch), "both voices stopped after BAND 0")
}

// TestBasicBandPerVoice verifies BAND v,n only affects the named voice.
func TestBasicBandPerVoice(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, `10 TRACK 0,"C"`)
	typeLine(computer, step, `20 TRACK 1,"C"`)
	typeLine(computer, step, "30 BAND 1")
	typeLine(computer, step, "40 BAND 0,0")
	typeLine(computer, step, "50 POKE 752,PLAYING(0)*10+PLAYING(1)")
	typeLine(computer, step, "60 GOTO 60")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	require_eq(t, 1, peek(computer, addrScratch), "voice 0 stopped, voice 1 still running")
}

// TestBasicVtakeVgiveRoundTrip verifies VTAKE/VGIVE dispatch without
// disturbing PLAYING(v)'s "has an active track" reading (VTAKE freezes
// advancement, it does not stop the track - see docs/audio-sequencer.md).
func TestBasicVtakeVgiveRoundTrip(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, `10 TRACK 0,"C"`)
	typeLine(computer, step, "20 BAND 0,1")
	typeLine(computer, step, "30 VTAKE 0")
	typeLine(computer, step, "40 POKE 752,PLAYING(0)")
	typeLine(computer, step, "50 VGIVE 0")
	typeLine(computer, step, "60 POKE 753,PLAYING(0)")
	typeLine(computer, step, "70 GOTO 70")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	require_eq(t, 1, peek(computer, addrScratch), "still PLAYING while taken")
	require_eq(t, 1, peek(computer, 0x02F1), "still PLAYING after being given back")
}

// TestBasicCueBeforeStart verifies CUE(v) reads 0 for a freshly loaded track
// that has not started advancing yet (0 = never started, 1-based once it has).
func TestBasicCueBeforeStart(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, `10 TRACK 3,"C"`)
	typeLine(computer, step, "20 POKE 752,CUE(3)")
	typeLine(computer, step, "30 GOTO 30")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	require_eq(t, 0, peek(computer, addrScratch), "CUE(3) before BAND starts it")
}

