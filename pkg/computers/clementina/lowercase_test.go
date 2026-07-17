package clementina

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLowercaseKeywords: BASIC keywords may be typed in lower case. The
// tokenizer folds them, so `crsr` is the CRSR token and the program runs.
func TestLowercaseKeywords(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, "10 crsr 5,7")
	typeLine(computer, step, "20 goto 20")
	typeLine(computer, step, "run")
	editorTickN(computer, step, 1_000_000)

	require_eq(t, 5, peek(computer, addrCursorX), "CURSOR_X after lower-case `crsr 5,7`")
	require_eq(t, 7, peek(computer, addrCursorY), "CURSOR_Y after lower-case `crsr 5,7`")
}

// TestLowercaseVariablesFoldToUpper: a variable typed in lower case is the same
// variable as its upper-case spelling (program text is stored folded).
func TestLowercaseVariablesFoldToUpper(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, "10 a=5")
	typeLine(computer, step, "20 CRSR A,7")
	typeLine(computer, step, "30 goto 30")
	typeLine(computer, step, "run")
	editorTickN(computer, step, 1_000_000)

	require_eq(t, 5, peek(computer, addrCursorX), "`a=5` then `CRSR A,7`: a and A must be one variable")
}

// TestLowercaseMixedCaseKeyword: folding is per character, not per word.
func TestLowercaseMixedCaseKeyword(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, "10 CrSr 5,7")
	typeLine(computer, step, "20 GoTo 20")
	typeLine(computer, step, "run")
	editorTickN(computer, step, 1_000_000)

	require_eq(t, 5, peek(computer, addrCursorX), "CURSOR_X after mixed-case `CrSr 5,7`")
	require_eq(t, 7, peek(computer, addrCursorY), "CURSOR_Y after mixed-case `CrSr 5,7`")
}

// TestStringLiteralsKeepTypedCase: the fold runs only on the tokenizer's
// non-literal path, so text inside quotes prints exactly as it was typed. The
// program clears the screen first, so the only text on the overlay is what
// PRINT emitted (not the typed source line).
func TestStringLiteralsKeepTypedCase(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, "10 cls")
	typeLine(computer, step, `20 print "Hello World"`)
	typeLine(computer, step, "30 goto 30")
	typeLine(computer, step, "run")
	editorTickN(computer, step, 2_000_000)

	_, _, ok := findSeq(vr, []uint8("Hello World"))
	require.True(t, ok, `PRINT "Hello World" must print with the case it was typed in`)
}

// TestRemKeepsTypedCase: REM text is copied verbatim like a string literal, so
// LIST shows the comment as it was typed.
func TestRemKeepsTypedCase(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, "10 rem Keep this comment")
	typeLine(computer, step, "cls")
	typeLine(computer, step, "list")
	editorTickN(computer, step, 2_000_000)

	_, _, ok := findSeq(vr, []uint8("Keep this comment"))
	require.True(t, ok, "LIST must show REM text with the case it was typed in")
}
