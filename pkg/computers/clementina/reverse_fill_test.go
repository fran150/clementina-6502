package clementina

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// assertNoReversedSpaces fails if any space (tile $20) on the overlay carries the
// CHR_ALT reverse bit. A reversed space renders as a solid block, so every routine
// that blanks cells must clear that bit even when the pen is in reverse mode.
func assertNoReversedSpaces(t *testing.T, vr videoReader, ctx string) {
	t.Helper()
	for y := 0; y < 25; y++ {
		for x := 0; x < 40; x++ {
			if ntCell(vr, y, x) == 0x20 {
				require.Zerof(t, attrCell(vr, y, x)&0x80,
					"space at (%d,%d) must not carry the reverse bit %s", y, x, ctx)
			}
		}
	}
}

// TestReverseModeClsBlanksNotReversed: with the pen in reverse mode, CLS (clrscr)
// fills the whole overlay with spaces. Those spaces must use the non-reversed
// attribute. Regression for clrscr stamping cleared cells with the full TEXT_ATTR
// (including the CHR_ALT reverse bit), which turned the cleared screen into a wall
// of solid blocks.
func TestReverseModeClsBlanksNotReversed(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	// Reverse the editor pen (ESC R sets TEXT_ATTR's CHR_ALT bit).
	injectKeys(computer, keyEsc, 'R')
	editorTickN(computer, step, 300_000)
	require.NotZero(t, peek(computer, addrTextAttr)&0x80, "ESC R should set the reverse bit")

	// Run CLS; clrscr blanks every cell to a space.
	injectKeys(computer, 'C', 'L', 'S', keyCR)
	editorTickN(computer, step, 1_500_000)

	assertNoReversedSpaces(t, vr, "after CLS")
}

// TestReverseModeScrollBlankNotReversed: with the editor pen in reverse mode,
// pressing RETURN advances the cursor and, at the bottom row, scrolls - blanking
// the freed bottom row. The blank row must use the non-reversed attribute.
// Regression for scroll_up stamping the new bottom row with the full TEXT_ATTR
// (including the CHR_ALT reverse bit), which is exactly what the user reported:
// pressing enter on a blank line with reverse on filled the row with solid blocks.
func TestReverseModeScrollBlankNotReversed(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	// Reverse the editor pen (ESC R sets TEXT_ATTR's CHR_ALT bit); edit_line
	// reasserts this pen on every input line, so each RETURN's newline runs with
	// the reverse bit set.
	injectKeys(computer, keyEsc, 'R')
	editorTickN(computer, step, 300_000)
	require.NotZero(t, peek(computer, addrTextAttr)&0x80, "ESC R should set the reverse bit")

	// Press RETURN on blank lines until the cursor reaches the bottom and the
	// editor's newline scrolls repeatedly.
	for i := 0; i < 40; i++ {
		injectKeys(computer, keyCR)
		editorTickN(computer, step, 300_000)
	}

	assertNoReversedSpaces(t, vr, "after reverse-pen scroll on RETURN")
}

// NOTE: chrout's backspace erase (@bs) got the same and-#$7F fix for consistency,
// but it is not covered here: the editor's own backspace routes to edit_delete_left
// (see reverse_delete_test.go), and PRINT CHR$(8) does not run @bs with the reverse
// bit set in TEXT_ATTR, so there is no BASIC-visible path that reproduces it. The
// fix is by inspection.
