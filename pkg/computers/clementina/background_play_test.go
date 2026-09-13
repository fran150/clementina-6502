package clementina

import (
	"strings"
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
)

// rowText returns the trimmed text of one screen row (printable ASCII as
// literal characters, everything else as a space), for exact comparisons -
// screenContainsRow's substring search is too easily fooled by the boot
// banner (e.g. "1.0" and "1977" both contain "1" and "0").
func rowText(vr videoReader, row int) string {
	b := make([]byte, 40)
	for c := 0; c < 40; c++ {
		v := ntCell(vr, row, c)
		if v >= 0x20 && v < 0x7f {
			b[c] = v
		} else {
			b[c] = ' '
		}
	}
	return strings.TrimSpace(string(b))
}

// screenContainsRowFrom is screenContainsRow scoped to rows >= from, so a
// repeated command (typed more than once in one test) finds the later echo
// instead of always the first.
func screenContainsRowFrom(vr videoReader, want string, from int) (row int, ok bool) {
	for r := from; r < 25; r++ {
		for c := 0; c+len(want) <= 40; c++ {
			match := true
			for i := 0; i < len(want); i++ {
				if ntCell(vr, r, c+i) != want[i] {
					match = false
					break
				}
			}
			if match {
				return r, true
			}
		}
	}
	return 0, false
}

// assertNoSyntaxError fails the test if "SYNTAX ERROR" appears anywhere on
// screen - the strong, unambiguous check that the new TOKEN_EXTFN dispatch
// (PLAYING(0)) actually ran instead of erroring.
func assertNoSyntaxError(t *testing.T, vr videoReader) {
	t.Helper()
	if _, ok := screenContainsRow(vr, "SYNTAX ERROR"); ok {
		dumpScreen(t, vr)
		t.Fatal("got a SYNTAX ERROR")
	}
}

// TestPlayingFunctionSkeleton is the first checkpoint for background PLAY:
// PLAYING(0) is a real BASIC function (dispatched through the new
// TOKEN_EXTFN mechanism, since the primary function table has no room left),
// so it must parse, evaluate, and print - and PLAY has not been wired to set
// its flag yet, so it must read 0.
func TestPlayingFunctionSkeleton(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, `PRINT PLAYING(0)`)
	editorTickN(computer, step, 300_000)

	assertNoSyntaxError(t, vr)
	cmdRow, ok := screenContainsRow(vr, "PLAYING(0)")
	if !ok {
		dumpScreen(t, vr)
		t.Fatal("command line not found on screen")
	}
	if got := rowText(vr, cmdRow+1); got != "0" {
		dumpScreen(t, vr)
		t.Fatalf("PRINT PLAYING(0) printed %q, want \"0\"", got)
	}
}

// TestPlayingFunctionLists verifies the LIST detokenizer expands the new
// TOKEN_EXTFN prefix back into "PLAYING" text (not garbage).
func TestPlayingFunctionLists(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, `10 PRINT PLAYING(0)`)
	typeLine(computer, step, "LIST")
	editorTickN(computer, step, 300_000)

	if _, ok := screenContainsRow(vr, "PLAYING(0)"); !ok {
		dumpScreen(t, vr)
		t.Fatal("LIST did not show PLAYING(0)")
	}
}

// addrBGPFlags is the background-PLAY control block's flags byte ($5C00,
// BGP_FLAGS in clementina_extra.s) - fixed RAM below RAMSTART2. Raised from
// $4400, then $4C00, alongside BGP_BASE/RAMSTART2/MAX_KERNEL_BYTES; see
// docs/basic-file.md in the clementina-rom repo.
const addrBGPFlags = 0x5C00

// TestBackgroundPlaySetsAndClearsFlag: statement plumbing only (no sequencer
// wired up yet) - "PLAY s$,1" must set BGP_FLAGS, bare "PLAY" must clear it,
// and PLAYING(0) must agree. Uses a very long note (T255 L1 ~= 6.4M cycles)
// so it is still playing across several typeLine calls (each of which ticks
// 400,000 cycles internally) until the explicit bare "PLAY" stops it.
func TestBackgroundPlaySetsAndClearsFlag(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, `PLAY "T255 L1 C",1`)
	assertNoSyntaxError(t, vr)
	if peek(computer, addrBGPFlags) == 0 {
		t.Fatal(`PLAY "T255 L1 C",1 should set the background-playing flag`)
	}

	typeLine(computer, step, `PRINT PLAYING(0)`)
	editorTickN(computer, step, 300_000)
	assertNoSyntaxError(t, vr)
	cmdRow, ok := screenContainsRow(vr, "PLAYING(0)")
	if !ok {
		dumpScreen(t, vr)
		t.Fatal("command line not found on screen")
	}
	if got := rowText(vr, cmdRow+1); got != "1" {
		dumpScreen(t, vr)
		t.Fatalf("PLAYING(0) printed %q, want \"1\" while the background flag is set", got)
	}

	typeLine(computer, step, `PLAY`)
	editorTickN(computer, step, 300_000)
	assertNoSyntaxError(t, vr)
	if peek(computer, addrBGPFlags) != 0 {
		t.Fatal("bare PLAY should clear the background-playing flag")
	}

	typeLine(computer, step, `PRINT PLAYING(0)`)
	editorTickN(computer, step, 300_000)
	assertNoSyntaxError(t, vr)
	cmdRow, ok = screenContainsRowFrom(vr, "PLAYING(0)", cmdRow+2)
	if !ok {
		dumpScreen(t, vr)
		t.Fatal("command line not found on screen")
	}
	if got := rowText(vr, cmdRow+1); got != "0" {
		dumpScreen(t, vr)
		t.Fatalf("PLAYING(0) printed %q, want \"0\" after bare PLAY", got)
	}
}

// TestBackgroundPlayZeroFlagStillBlocks: "PLAY s$,0" must behave exactly like
// "PLAY s$" - block, never touch BGP_FLAGS.
func TestBackgroundPlayZeroFlagStillBlocks(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, `10 PLAY "T60 L1 O4 C",0`)
	typeLine(computer, step, "20 GOTO 20")
	injectKeys(computer, []byte("RUN")...)
	injectKeys(computer, keyCR)
	editorTickN(computer, step, 250_000) // parse + into the note, nowhere near its end

	require_eq(t, 0x60, peekAudio(computer, voiceField(0, 0)), "v0 FREQ_L during blocking PLAY note")
	require_eq(t, 0x03, peekAudio(computer, voiceField(0, 7)), "v0 CONTROL gated+retrig during blocking PLAY note")
	if peek(computer, addrBGPFlags) != 0 {
		t.Fatal(`PLAY s$,0 must not set the background-playing flag - it blocks like plain PLAY`)
	}
}

// TestBackgroundPlaySingleNoteSoundsAndFinishes is the core checkpoint for the
// IRQ-driven sequencer: the note must actually sound (FREQ + gate written by
// bg_play_tick, not the foreground), and it must release and stop on its own
// once its duration elapses - all without anything else driving it forward
// besides the Timer-1 IRQ. Uses injectKeys/editorTickN directly (not
// typeLine, which ticks 400,000 cycles internally) for precise control over
// how far into the note's ~1.5 s lifetime each check lands.
func TestBackgroundPlaySingleNoteSoundsAndFinishes(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, "SNDON:WAVE 0,1:ADSR 0,0,4,10,4")

	injectKeys(computer, []byte(`PLAY "T60 L1 O4 C",1`)...)
	injectKeys(computer, keyCR)
	editorTickN(computer, step, 100_000) // parse + a little way into the note

	require_eq(t, 0x60, peekAudio(computer, voiceField(0, 0)), "v0 FREQ_L (C4) while the note plays")
	require_eq(t, 0x10, peekAudio(computer, voiceField(0, 1)), "v0 FREQ_H (C4) while the note plays")
	require_eq(t, 0x03, peekAudio(computer, voiceField(0, 7)), "v0 CONTROL gated while the note plays")
	if peek(computer, addrBGPFlags) == 0 {
		t.Fatal("background player should still be active mid-note")
	}

	editorTickN(computer, step, 1_700_000) // past T60 L1's ~240 ticks (~1.5s)

	require_eq(t, 0x00, peekAudio(computer, voiceField(0, 7)), "v0 CONTROL released once the note ends")
	if peek(computer, addrBGPFlags) != 0 {
		t.Fatal("background player should have stopped once the note finished")
	}
}

// TestBackgroundPlayMultiNoteSequence exercises the full bg_* token set
// (letters, octave shifts, a long run of repeated notes) end to end, mirrors
// the blocking-PLAY regression for the same string (play_test.go).
func TestBackgroundPlayMultiNoteSequence(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, "SNDON")
	editorTickN(computer, step, 200_000)
	typeLine(computer, step, `PLAY "T4 L16 O4 C D E F G A B C D E F G A B A A A A G",1`)
	editorTickN(computer, step, 6_000_000)

	// 19 notes, all octave 4; final note G4 (semitone 7): table[7]=392, <<4 = 6272 = $1880.
	require_eq(t, 0x80, peekAudio(computer, voiceField(0, 0)), "final note FREQ_L (G4)")
	require_eq(t, 0x18, peekAudio(computer, voiceField(0, 1)), "final note FREQ_H")
	require_eq(t, 0x00, peekAudio(computer, voiceField(0, 7)), "silenced at end of background string")
	if peek(computer, addrBGPFlags) != 0 {
		t.Fatal("background player should have stopped once the string finished")
	}
}

// TestBackgroundPlayReplacesPrevious: starting a new background PLAY while
// one is already sounding must not leave the old note orphaned (stuck
// gated forever) - it silences and replaces it. Uses injectKeys/editorTickN
// directly (not typeLine) so the replacement note's own ~0.5s lifetime isn't
// eaten by typeLine's internal 400,000-cycle tick before the check runs.
func TestBackgroundPlayReplacesPrevious(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, "SNDON:WAVE 0,1:ADSR 0,0,4,10,4")

	injectKeys(computer, []byte(`PLAY "T255 L1 O4 C",1`)...) // a very long note - still gated when replaced
	injectKeys(computer, keyCR)
	editorTickN(computer, step, 100_000)
	require_eq(t, 0x03, peekAudio(computer, voiceField(0, 7)), "first background note gated")

	injectKeys(computer, []byte(`PLAY "T80 L4 O5 E",1`)...)
	injectKeys(computer, keyCR)
	editorTickN(computer, step, 100_000)

	// The new note must be sounding - not the old one, orphaned and held forever.
	require_eq(t, 0x03, peekAudio(computer, voiceField(0, 7)), "replacement note gated")
	// E5: semitone 60+4. note_freq(64): octave 5, table[4]=330, <<5 = 10560 = $2940.
	require_eq(t, 0x40, peekAudio(computer, voiceField(0, 0)), "replacement note FREQ_L (E5)")
	require_eq(t, 0x29, peekAudio(computer, voiceField(0, 1)), "replacement note FREQ_H")
}

// TestBackgroundPlayRunsWhileInputBlocks is the whole point of this feature:
// the sequencer is IRQ-driven, so it must keep advancing even while the
// foreground program is stuck in a blocking INPUT wait for a keystroke that
// never comes - proving it is not merely "non-blocking at the PLAY
// statement" but genuinely concurrent with arbitrary BASIC execution.
func TestBackgroundPlayRunsWhileInputBlocks(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, "SNDON:WAVE 0,1:ADSR 0,0,4,10,4")
	typeLine(computer, step, `10 PLAY "T80 L4 C D E F",1`)
	typeLine(computer, step, `20 INPUT A$`)
	injectKeys(computer, []byte("RUN")...)
	injectKeys(computer, keyCR)

	// Reach the INPUT prompt and into the first note (T80 L4 = 80 ticks ~0.5s).
	editorTickN(computer, step, 300_000)
	freq1L := peekAudio(computer, voiceField(0, 0))
	freq1H := peekAudio(computer, voiceField(0, 1))

	// Still sitting in INPUT - no CR sent. Tick well past the first note's end
	// (80 ticks * 6250 cycles = 500,000) so a truly independent sequencer must
	// have moved on to a later note.
	editorTickN(computer, step, 600_000)
	freq2L := peekAudio(computer, voiceField(0, 0))
	freq2H := peekAudio(computer, voiceField(0, 1))

	if freq1L == freq2L && freq1H == freq2H {
		t.Fatalf("background PLAY did not advance while INPUT blocked (frequency stuck at %02X%02X)", freq2H, freq2L)
	}

	// The program must still be responsive at the INPUT prompt, not crashed.
	x0 := peek(computer, addrCursorX)
	injectKeys(computer, 'X')
	editorTickN(computer, step, 100_000)
	if peek(computer, addrCursorX) != x0+1 {
		t.Fatal("program should still be responsive at the INPUT prompt")
	}
}

// startLongBackgroundNote gets a long-lived (T255 L1, ~6.4M cycle) background
// note gated and playing, so tests can verify a teardown path stops it well
// before it would have finished on its own.
func startLongBackgroundNote(t *testing.T, computer *ClementinaComputer, step *common.StepContext) {
	t.Helper()
	typeLine(computer, step, "SNDON:WAVE 0,1:ADSR 0,0,4,10,4")
	injectKeys(computer, []byte(`PLAY "T255 L1 O4 C",1`)...)
	injectKeys(computer, keyCR)
	editorTickN(computer, step, 100_000)
	require_eq(t, 0x03, peekAudio(computer, voiceField(0, 7)), "background note gated before the teardown check")
}

// TestBackgroundPlayStopsOnCtrlC: Ctrl-C must stop the background player and
// release its voices, not just interrupt the foreground program (ISCNTC -
// which is only polled between statements of a *running* program, hence the
// GOTO loop here - falls straight into STOP, which now also calls
// bg_play_stop).
func TestBackgroundPlayStopsOnCtrlC(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	startLongBackgroundNote(t, computer, step)

	typeLine(computer, step, "20 GOTO 20")
	injectKeys(computer, []byte("RUN")...)
	injectKeys(computer, keyCR)
	editorTickN(computer, step, 100_000)

	injectKeys(computer, 0x03) // Ctrl-C
	editorTickN(computer, step, 200_000)

	require_eq(t, 0x00, peekAudio(computer, voiceField(0, 7)), "voice released after Ctrl-C")
	if peek(computer, addrBGPFlags) != 0 {
		t.Fatal("Ctrl-C should stop the background player")
	}
}

// TestBackgroundPlayStopsOnStopStatement: a STOP statement in the running
// program must also silence the background player.
func TestBackgroundPlayStopsOnStopStatement(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	startLongBackgroundNote(t, computer, step)

	typeLine(computer, step, "10 STOP")
	injectKeys(computer, []byte("RUN")...)
	injectKeys(computer, keyCR)
	editorTickN(computer, step, 200_000)

	require_eq(t, 0x00, peekAudio(computer, voiceField(0, 7)), "voice released after STOP")
	if peek(computer, addrBGPFlags) != 0 {
		t.Fatal("STOP should stop the background player")
	}
}

// TestBackgroundPlayStopsOnEndStatement: same, for END.
func TestBackgroundPlayStopsOnEndStatement(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	startLongBackgroundNote(t, computer, step)

	typeLine(computer, step, "10 END")
	injectKeys(computer, []byte("RUN")...)
	injectKeys(computer, keyCR)
	editorTickN(computer, step, 200_000)

	require_eq(t, 0x00, peekAudio(computer, voiceField(0, 7)), "voice released after END")
	if peek(computer, addrBGPFlags) != 0 {
		t.Fatal("END should stop the background player")
	}
}

// TestBackgroundPlayStopsOnError: any runtime error must also silence the
// background player (ERROR funnels every error, anywhere, through one point).
func TestBackgroundPlayStopsOnError(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	startLongBackgroundNote(t, computer, step)

	injectKeys(computer, []byte("PRINT 1/0")...) // division by zero
	injectKeys(computer, keyCR)
	editorTickN(computer, step, 200_000)

	require_eq(t, 0x00, peekAudio(computer, voiceField(0, 7)), "voice released after a runtime error")
	if peek(computer, addrBGPFlags) != 0 {
		t.Fatal("a runtime error should stop the background player")
	}
}

// TestBackgroundPlayStopsOnNew: NEW must also silence the background player.
func TestBackgroundPlayStopsOnNew(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	startLongBackgroundNote(t, computer, step)

	injectKeys(computer, []byte("NEW")...)
	injectKeys(computer, keyCR)
	editorTickN(computer, step, 200_000)

	require_eq(t, 0x00, peekAudio(computer, voiceField(0, 7)), "voice released after NEW")
	if peek(computer, addrBGPFlags) != 0 {
		t.Fatal("NEW should stop the background player")
	}
}

// TestBackgroundPlayMalformedStringStopsGracefully: a bad token or a bad
// number in a background string must not raise SYNTAX ERROR/ILLEGAL
// QUANTITY the way blocking PLAY does - the sequencer runs inside an IRQ,
// which can never safely reach BASIC's error handler (it resets the 6502
// stack). It just stops silently, and the prompt stays usable.
func TestBackgroundPlayMalformedStringStopsGracefully(t *testing.T) {
	cases := []string{
		`PLAY "Z",1`,    // Z is not a token
		`PLAY "L5 C",1`, // 5 is not a valid note length
	}
	for _, line := range cases {
		t.Run(line, func(t *testing.T) {
			computer, step := bootClementinaToPrompt(t)
			vr := computer.chips.mia.(videoReader)
			typeLine(computer, step, "SNDON:WAVE 0,1:ADSR 0,0,4,10,4")

			injectKeys(computer, []byte(line)...)
			injectKeys(computer, keyCR)
			editorTickN(computer, step, 300_000) // several IRQ ticks - plenty to hit the bad token

			if _, ok := screenContainsRow(vr, "ERROR"); ok {
				dumpScreen(t, vr)
				t.Fatalf("%s: a malformed background string must not raise a BASIC error", line)
			}
			if peek(computer, addrBGPFlags) != 0 {
				t.Fatalf("%s: background player should have stopped on the bad token/number", line)
			}
			require_eq(t, 0x00, peekAudio(computer, voiceField(0, 7)), "voice silenced, not left orphaned")

			x0 := peek(computer, addrCursorX)
			injectKeys(computer, 'Z')
			editorTickN(computer, step, 200_000)
			if peek(computer, addrCursorX) != x0+1 {
				t.Fatalf("%s: prompt should stay usable after a malformed background string", line)
			}
		})
	}
}

// TestPlayingFunctionComposesInExpressions: PLAYING(0) must work as an
// ordinary numeric factor - inside IF, arithmetic, wherever - not just as a
// bare PRINT argument.
func TestPlayingFunctionComposesInExpressions(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, `PLAY "T255 L1 C",1`)
	assertNoSyntaxError(t, vr)

	typeLine(computer, step, `IF PLAYING(0) THEN PRINT "YES"`)
	assertNoSyntaxError(t, vr)
	if _, ok := screenContainsRow(vr, "YES"); !ok {
		dumpScreen(t, vr)
		t.Fatal(`IF PLAYING(0) THEN did not take the true branch while playing`)
	}

	typeLine(computer, step, `PRINT PLAYING(0)+41`)
	assertNoSyntaxError(t, vr)
	cmdRow, ok := screenContainsRow(vr, "PLAYING(0)+41")
	if !ok {
		dumpScreen(t, vr)
		t.Fatal("command line not found on screen")
	}
	if got := rowText(vr, cmdRow+1); got != "42" {
		dumpScreen(t, vr)
		t.Fatalf("PLAYING(0)+41 printed %q, want \"42\"", got)
	}
}
