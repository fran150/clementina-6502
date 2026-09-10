package clementina

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// PLAY is the blocking MML music statement (handler in clementina-rom
// src/basic/clementina_extra.s). It writes FREQ + the gate on the audio block
// at $12000 and times notes off the kernel jiffy counter KJIFFY ($00F7). When a
// PLAY string finishes it silences every voice.

const addrKJiffy = 0x00F7

func jiffies(computer *ClementinaComputer) int {
	return int(peek(computer, addrKJiffy)) | int(peek(computer, addrKJiffy+1))<<8
}

// TestKernelJiffyClockAdvances checks the VIA Timer-1 IRQ increments KJIFFY.
func TestKernelJiffyClockAdvances(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	t0 := jiffies(computer)
	editorTickN(computer, step, 600_000) // ~96 ticks at 6250 cycles/tick
	dt := jiffies(computer) - t0
	if dt < 80 || dt > 112 {
		t.Fatalf("KJIFFY advanced by %d over 600k cycles, expected ~96", dt)
	}
}

// TestBasicPlayMidNoteGated: a few thousand cycles into a long PLAY note, the
// target voice has its FREQ set and its gate on.
func TestBasicPlayMidNoteGated(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, "10 PLAY \"T60 L1 O4 C\"")
	typeLine(computer, step, "20 GOTO 20")
	injectKeys(computer, []byte("RUN")...)
	injectKeys(computer, keyCR)
	editorTickN(computer, step, 250_000) // parse + into the note, nowhere near its end

	// C4 = octave-0 262 << 4 = 4192 = $1060.
	require_eq(t, 0x60, peekAudio(computer, voiceField(0, 0)), "v0 FREQ_L during PLAY note")
	require_eq(t, 0x10, peekAudio(computer, voiceField(0, 1)), "v0 FREQ_H during PLAY note")
	require_eq(t, 0x03, peekAudio(computer, voiceField(0, 7)), "v0 CONTROL gated+retrig during PLAY note")
}

// TestBasicPlayFinishesSilenced: after a short PLAY the note's frequency is
// still there but every voice's gate has been cleared.
func TestBasicPlayFinishesSilenced(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, "10 PLAY \"T3 L8 V2 O5 G\"")
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_500_000) // T3 eighth note = ~1.5 ticks; long done

	// G5: octave 5 base 60 + G(7) = 67. note_freq(67): octave 5, semitone 7 ->
	// table[7]=392, <<5 = 12544 = $3100.
	require_eq(t, 0x00, peekAudio(computer, voiceField(2, 0)), "v2 FREQ_L (G5 = 12544)")
	require_eq(t, 0x31, peekAudio(computer, voiceField(2, 1)), "v2 FREQ_H")
	require_eq(t, 0x00, peekAudio(computer, voiceField(2, 7)), "v2 CONTROL cleared after PLAY")
	require_eq(t, 0x00, peekAudio(computer, voiceField(0, 7)), "v0 CONTROL cleared after PLAY")
}

// TestBasicPlaySharpFlatAndVoice exercises accidentals and voice routing.
func TestBasicPlaySharpFlatAndVoice(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	// V0 C#4 held long; V1 set later in the string is only reached after V0's
	// note, so sample mid-first-note.
	typeLine(computer, step, "10 PLAY \"T60 L1 O4 C#\"")
	typeLine(computer, step, "20 GOTO 20")
	injectKeys(computer, []byte("RUN")...)
	injectKeys(computer, keyCR)
	editorTickN(computer, step, 250_000)

	// C#4 = semitone 49 -> octave 4 (48) + 1. note_freq(49): octave 4, semitone 1
	// -> table[1] = 277, <<4 = 4432 = $1150.
	require_eq(t, 0x50, peekAudio(computer, voiceField(0, 0)), "v0 FREQ_L (C#4 = 4432)")
	require_eq(t, 0x11, peekAudio(computer, voiceField(0, 1)), "v0 FREQ_H")
}

// TestBasicPlayMelodySequence runs a multi-note string with octave shifts and a
// rest, and checks the final note lands and the string ran to completion.
func TestBasicPlayMelodySequence(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	// O3 C, '>' -> O4 C, rest, '<' '<' -> O2, C.
	typeLine(computer, step, "10 PLAY \"T3 L8 O3 C > C R < < C\"")
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 3_000_000)

	// C2 = semitone 24 -> note_freq: octave 2, table[0]=262, <<2 = 1048 = $0418.
	require_eq(t, 0x18, peekAudio(computer, voiceField(0, 0)), "last note FREQ_L (C2 = 1048)")
	require_eq(t, 0x04, peekAudio(computer, voiceField(0, 1)), "last note FREQ_H")
	require_eq(t, 0x00, peekAudio(computer, voiceField(0, 7)), "all voices silenced at end")
}

// TestBasicPlayTempoTiming: a slower tempo makes the same note take longer, so
// the RUN reaches its GOTO loop later (more jiffies elapse before it settles).
func TestBasicPlayTempoTiming(t *testing.T) {
	measure := func(prog string) int {
		computer, step := bootClementinaToPrompt(t)
		typeLine(computer, step, prog)
		typeLine(computer, step, "20 GOTO 20")
		t0 := jiffies(computer)
		injectKeys(computer, []byte("RUN")...)
		injectKeys(computer, keyCR)
		// The gate is $03 during the (single) note and $00 once PLAY returns;
		// count kernel ticks elapsed from RUN to the gate clearing.
		for i := 0; i < 400; i++ {
			editorTickN(computer, step, 50_000)
			if peekAudio(computer, voiceField(0, 7)) == 0x00 && jiffies(computer)-t0 > 20 {
				return jiffies(computer) - t0
			}
		}
		return jiffies(computer) - t0
	}

	fast := measure(`10 PLAY "T30 L1 C"`)  // whole note = 30<<2 = 120 ticks
	slow := measure(`10 PLAY "T120 L1 C"`) // whole note = 120<<2 = 480 ticks
	if slow < fast*2 {
		t.Fatalf("4x tempo should block far longer: fast=%d slow=%d ticks", fast, slow)
	}
}

// TestBasicPlayLongStringAndRepeats: a 40-char string with a repeated note
// plays every note through to the end (regression: an earlier cut only ran the
// first ~7 because the gate never dropped between notes).
func TestBasicPlayLongStringAndRepeats(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, "10 PLAY \"T4 L16 O4 C D E F G A B C D E F G A B A A A A G\"")
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 6_000_000)

	// 19 notes, all octave 4; final note G4 (semitone 7): table[7]=392, <<4 = 6272 = $1880.
	require_eq(t, 0x80, peekAudio(computer, voiceField(0, 0)), "final note FREQ_L (G4)")
	require_eq(t, 0x18, peekAudio(computer, voiceField(0, 1)), "final note FREQ_H")
	require_eq(t, 0x00, peekAudio(computer, voiceField(0, 7)), "silenced at end of long string")
}

// pcmReader renders PCM the exact way the oto callback does. A bursty read
// pattern here stands in for a starved oto callback.
type pcmReader interface {
	DebugAudioReadPCM(frames int) []int16
}

// windowRMS returns the RMS of the left channel of pcm[from:to] (interleaved).
func windowRMS(pcm []int16, from, to int) float64 {
	var sum int64
	for f := from; f < to; f++ {
		s := int64(pcm[f*2])
		sum += s * s
	}
	return math.Sqrt(float64(sum) / float64(to-from))
}

// TestBasicPlayShortNotesAllSound is the regression for "L8/L16 only plays 2
// notes": every note in a run of short notes must produce audible output, even
// when the PCM consumer reads in large, bursty chunks (a starved oto callback).
func TestBasicPlayShortNotesAllSound(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	pr := computer.chips.mia.(pcmReader)

	typeLine(computer, step, `SNDON:WAVE 0,1:ADSR 0,0,4,10,4`)
	pr.DebugAudioReadPCM(1 << 20) // discard silence buffered so far

	injectKeys(computer, []byte(`PLAY "T80 L16 C D E F G"`)...)
	injectKeys(computer, keyCR)

	// Run the program, pulling PCM only every ~40k cycles (a deliberately bursty,
	// starved reader). Collect everything into one stream.
	var pcm []int16
	for i := 0; i < 400; i++ {
		editorTickN(computer, step, 40_000)
		pcm = append(pcm, pr.DebugAudioReadPCM(50_000)...)
		if peekAudio(computer, voiceField(0, 7)) == 0x00 && i > 4 {
			editorTickN(computer, step, 40_000)
			pcm = append(pcm, pr.DebugAudioReadPCM(50_000)...)
			break
		}
	}

	// Trim leading silence (parse time before the first note gates on).
	frames := len(pcm) / 2
	start := 0
	for start < frames && abs16(pcm[start*2]) < 50 {
		start++
	}
	pcm = pcm[start*2:]
	frames = len(pcm) / 2
	require.Greater(t, frames, 5000, "expected a few tenths of a second of audio after the first note")

	// Split the note-bearing stream into 5 equal windows and require each to
	// carry real energy - no dropped notes.
	win := frames / 5
	for n := 0; n < 5; n++ {
		rms := windowRMS(pcm, n*win, (n+1)*win)
		require.Greater(t, rms, 200.0, "note %d/5 is silent - dropped (rms %.0f)", n+1, rms)
	}
}

// TestBasicPlayLongNoteContinuous is the regression for "L2 plays each note 3
// times with cuts in the middle": one long note must render as one continuous
// tone even when the PCM consumer reads in bursts. No mid-note silence, no
// repeats.
func TestBasicPlayLongNoteContinuous(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	pr := computer.chips.mia.(pcmReader)

	typeLine(computer, step, `SNDON:WAVE 0,1:ADSR 0,0,4,12,4`)
	pr.DebugAudioReadPCM(1 << 20)

	injectKeys(computer, []byte(`PLAY "T80 L1 C"`)...) // whole note: 80*4 = 320 ticks ~= 1.7 s
	injectKeys(computer, keyCR)

	var pcm []int16
	for i := 0; i < 400; i++ {
		editorTickN(computer, step, 60_000) // bursty: pull far less often than 24 kHz
		pcm = append(pcm, pr.DebugAudioReadPCM(80_000)...)
		if peekAudio(computer, voiceField(0, 7)) == 0x00 && i > 4 {
			break
		}
	}

	frames := len(pcm) / 2
	start := 0
	for start < frames && abs16(pcm[start*2]) < 50 {
		start++
	}
	// End at the note's release (trailing silence once the gate drops).
	end := frames
	for end > start && abs16(pcm[(end-1)*2]) < 50 {
		end--
	}
	note := pcm[start*2 : end*2]
	noteFrames := len(note) / 2
	require.Greater(t, noteFrames, 24000, "expected ~1 s+ of a single note")

	// Every 1/20th of the note must carry energy - a "cut in the middle" would
	// show up as one or more silent windows.
	win := noteFrames / 20
	for n := 0; n < 20; n++ {
		rms := windowRMS(note, n*win, (n+1)*win)
		require.Greater(t, rms, 150.0,
			"window %d/20 of one long note is silent - the note was cut (rms %.0f)", n+1, rms)
	}
}

func abs16(v int16) int16 {
	if v < 0 {
		return -v
	}
	return v
}

// TestBasicPlayCtrlCAborts: Ctrl-C during a blocking PLAY note silences every
// voice and returns to a usable prompt (no hang, no ringing note).
func TestBasicPlayCtrlCAborts(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, "10 PLAY \"T120 L1 O4 C\"") // ~480 ticks, effectively forever here
	typeLine(computer, step, "20 GOTO 20")
	injectKeys(computer, []byte("RUN")...)
	injectKeys(computer, keyCR)
	editorTickN(computer, step, 300_000)
	require_eq(t, 0x03, peekAudio(computer, voiceField(0, 7)), "gated while PLAY blocks")

	injectKeys(computer, 0x03) // Ctrl-C
	editorTickN(computer, step, 400_000)

	require_eq(t, 0x00, peekAudio(computer, voiceField(0, 7)), "Ctrl-C silences the voice")

	x0 := peek(computer, addrCursorX)
	injectKeys(computer, 'Z')
	editorTickN(computer, step, 200_000)
	if peek(computer, addrCursorX) != x0+1 {
		t.Fatalf("prompt should be usable after Ctrl-C out of PLAY")
	}
}

// TestBasicPlayErrorsRecover: a bad token / bad number errors cleanly and the
// prompt stays usable.
func TestBasicPlayErrorsRecover(t *testing.T) {
	cases := []string{
		`PLAY "Q"`,  // Q is not a token -> SYNTAX ERROR (before any blocking note)
		`PLAY "O9"`, // octave 9 > 7 -> ILLEGAL QUANTITY
		`PLAY "V4"`, // voice 4 > 3
		`PLAY "L5"`, // 5 is not a valid note length
		`PLAY "T"`,  // T with no number -> SYNTAX ERROR
		`PLAY 5`,    // not a string -> TYPE MISMATCH
	}
	for _, line := range cases {
		line := line
		t.Run(line, func(t *testing.T) {
			computer, step := bootClementinaToPrompt(t)
			typeLine(computer, step, line)
			x0 := peek(computer, addrCursorX)
			injectKeys(computer, 'Z')
			editorTickN(computer, step, 400_000)
			if peek(computer, addrCursorX) != x0+1 {
				t.Fatalf("%q should error and leave the prompt usable", line)
			}
			// voices must be silenced after the aborted PLAY
			require_eq(t, 0x00, peekAudio(computer, voiceField(0, 7)), "v0 gate cleared after PLAY error")
		})
	}
}
