package clementina

import (
	"testing"
)

// pcReader / spReader expose CPU state so the test can tell an idle READY wait
// from a runaway.
type pcReader interface{ GetProgramCounter() uint16 }
type spReader interface{ GetStackPointer() uint8 }

// screenContainsRow reports the first row that contains an ASCII substring
// (uppercase/digits/punctuation are stored unchanged as overlay tile codes).
func screenContainsRow(vr videoReader, want string) (row int, ok bool) {
	for r := 0; r < 25; r++ {
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

func dumpScreen(t *testing.T, vr videoReader) {
	for r := 0; r < 25; r++ {
		row := make([]byte, 40)
		for c := 0; c < 40; c++ {
			b := ntCell(vr, r, c)
			switch {
			case b >= 0x20 && b < 0x7f:
				row[c] = b
			case b >= 1 && b <= 26:
				row[c] = 'a' + (b - 1) // lowercase screen codes
			default:
				row[c] = ' '
			}
		}
		t.Logf("row %2d |%s|", r, string(row))
	}
}

// TestGuessingGameReturnsToReady is a regression test for the Z14/TEMPPT
// zero-page aliasing bug: a PRINT+INPUT loop (a number-guessing game) that
// looped several times used to corrupt the temporary-string descriptor stack
// pointer and crash the interpreter at end-of-program instead of returning to
// the READY prompt. See clementina-rom src/basic/defines_clementina.s (Z14).
func TestGuessingGameReturnsToReady(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	prog := []string{
		"10 a=int(rnd(1)*100):n=0",
		"20 ?\"input a number\"",
		"30 input a:n=n+1",
		"40 if a<n then ?\"mi number is bigger\":goto 20",
		"50 if a>n then ?\"mi number is smaller\":goto 20",
		"60 ?\"congrats that is my number\"",
		"70 ?\"you made it in \";n;\"attempts\"",
	}
	for _, l := range prog {
		typeLine(computer, step, l)
	}
	typeLine(computer, step, "run")
	editorTickN(computer, step, 1_000_000)

	// Four wrong guesses (a=9 > n on attempts 1..4 -> "smaller" + GOTO 20), then
	// win on attempt 5 where n=5 and a=5 satisfies neither branch.
	for _, g := range []byte{'9', '9', '9', '9', '5'} {
		injectKeys(computer, g, keyCR)
		editorTickN(computer, step, 1_500_000)
	}

	// The interpreter should have printed line 70 and returned to READY.
	if _, ok := screenContainsRow(vr, "attempts"); !ok {
		dumpScreen(t, vr)
		t.Fatal("program never printed line 70 (\"attempts\") - it did not run as expected")
	}

	attemptsRow, _ := screenContainsRow(vr, "attempts")
	readyRow, readyOK := screenContainsRow(vr, "READY.")
	if !readyOK || readyRow <= attemptsRow {
		dumpScreen(t, vr)
		t.Fatalf("no fresh READY. prompt after the program output (attempts row=%d, READY row=%d, found=%v): "+
			"the interpreter hung at end-of-program", attemptsRow, readyRow, readyOK)
	}

	// The CPU must be idle in valid code (editor input wait), never running away
	// into I/O space ($C000+). 2026-09 RAM/ROM reorg: the loaded image is now
	// top-anchored so it always ends exactly at $BFFF, but its *start* varies
	// build to build (computed by the two-pass link, clementina-rom Makefile) -
	// unlike the old fixed RAMSTART2 ceiling, there's no single stable low bound
	// to mirror here anymore. This deliberately widens to the whole non-I/O flat
	// range ($0400-$BFFF, which now covers working RAM/heap as well as code) -
	// less precise about excluding the heap than the old per-build-tuned bound,
	// but it still catches the failure modes that matter (PC in I/O/MIA space,
	// or in the unused zero-page/stack/vector gaps below $0400).
	pc := computer.chips.cpu.(pcReader)
	inValidCode := func(p uint16) bool {
		return p <= 0x0005 || (p >= 0x00C2 && p <= 0x00DE) || (p >= 0x0400 && p <= 0xBFFF)
	}
	for i := 0; i < 300_000; i++ {
		tickComputer(computer, step)
		if p := pc.GetProgramCounter(); !inValidCode(p) {
			t.Fatalf("CPU ran away to $%04X after the program ended", p)
		}
		step.NextCycle()
	}
}
