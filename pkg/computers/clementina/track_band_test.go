package clementina

import "testing"

// TRACK/BAND/VTAKE/VGIVE/PLAYING/CUE (handlers in clementina-rom
// src/basic/clementina_extra.s) are the background sequencer's BASIC
// surface, replacing the retired PLAY. TRACK compiles a voice's MML string
// straight into MIA's sequencer bytecode at $13000+voice*$400 (see
// clementina-mia's docs/audio-sequencer.md); these tests boot the real ROM,
// type a program, run it, and read the encoded bytes and command effects
// back - the same style sound_test.go uses for the direct register
// statements.
//
// Like sound_test.go, BAND is only exercised for its synchronous, RAM-only
// effects (SEQ_LOAD/SEQ_START/SEQ_STOP/VOICE_TAKE/VOICE_RELEASE all mutate
// sequencer state directly in the command handler): whether a track's note
// index actually advances over time depends on the audio engine's real-time
// render loop, already covered in the mia package's own tests
// (TestEmulatedMiaAudioSeq*), not exercised here.

const seqTrackBase = 0x13000
const seqTrackSize = 0x400

func peekSeq(computer *ClementinaComputer, voice uint32, offset uint32) uint8 {
	return computer.chips.mia.(audioMemReader).DebugReadVideo(seqTrackBase + voice*seqTrackSize + offset)
}

// TestBasicTrackEncodesNote verifies TRACK's encoder against a hand-computed
// expected byte sequence: T60 L8 O5 C is a single NOTE event (C5, an eighth
// note at tempo 60 -> 30 kernel ticks -> 30*150 = 4500 audio samples) then
// END, with no loop.
func TestBasicTrackEncodesNote(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, `10 TRACK 0,"T60 L8 O5 C"`)
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	// Header: no loop.
	require_eq(t, 0xFF, peekSeq(computer, 0, 0), "LOOP_L")
	require_eq(t, 0xFF, peekSeq(computer, 0, 1), "LOOP_H")

	// Event stream starts at offset 4.
	require_eq(t, 0x01, peekSeq(computer, 0, 4), "opcode NOTE")
	// C5: note_freq(60) = octave-0 262 << 5 = 8384 = $20C0.
	require_eq(t, 0xC0, peekSeq(computer, 0, 5), "freq_l")
	require_eq(t, 0x20, peekSeq(computer, 0, 6), "freq_h")
	// T60, eighth note (k=8: T>>1) = 30 ticks; 30*150 = 4500 = $001194.
	require_eq(t, 0x94, peekSeq(computer, 0, 7), "dur_l")
	require_eq(t, 0x11, peekSeq(computer, 0, 8), "dur_m")
	require_eq(t, 0x00, peekSeq(computer, 0, 9), "dur_h")
	require_eq(t, 0x00, peekSeq(computer, 0, 10), "opcode END")
}

// TestBasicTrackLoopMarker verifies "|" records the byte offset of whatever
// follows it, not a fixed/guessed value - the intro (first note, 6 bytes:
// NOTE takes op+freq_l+freq_h+dur(3)) plays once, only the second note loops.
func TestBasicTrackLoopMarker(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, `10 TRACK 1,"C|D"`)
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	require_eq(t, 0x06, peekSeq(computer, 1, 0), "LOOP_L points past the first NOTE")
	require_eq(t, 0x00, peekSeq(computer, 1, 1), "LOOP_H")
	require_eq(t, 0x01, peekSeq(computer, 1, 4), "first opcode NOTE (C)")
	require_eq(t, 0x01, peekSeq(computer, 1, 10), "second opcode NOTE (D), right at the loop point")
}

// TestBasicTrackWaveToken verifies the W n inline token emits SET_WAVE.
func TestBasicTrackWaveToken(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, `10 TRACK 2,"W2 C"`)
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	require_eq(t, 0x03, peekSeq(computer, 2, 4), "opcode SET_WAVE")
	require_eq(t, 0x02, peekSeq(computer, 2, 5), "waveform value (saw)")
	require_eq(t, 0x01, peekSeq(computer, 2, 6), "opcode NOTE follows immediately (zero duration)")
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

