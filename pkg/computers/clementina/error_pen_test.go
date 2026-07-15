package clementina

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestErrorMessageUsesBasicPen: the whole BASIC error line (?<name> ERROR) must
// print in the BASIC pen, even when the editor pen differs. Regression for the
// "?<name>" prefix (printed via OUTQUES/OUTDO at the live pen) coming out in the
// editor pen while only " ERROR" (via STROUT) used the BASIC pen.
func TestErrorMessageUsesBasicPen(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	basicPen := peek(computer, addrTextAttr) // ATTR_WHITE = 0 at boot
	require.Equal(t, uint8(0x00), basicPen)

	// Set a distinct editor pen (palette 2).
	injectKeys(computer, keyEsc, '2')
	editorTickN(computer, step, 300_000)
	require.Equal(t, uint8(0x02), peek(computer, addrTextAttr), "editor pen = palette 2")

	// Trigger a SYNTAX ERROR ('@' is not a valid statement).
	injectKeys(computer, '@', keyCR)
	editorTickN(computer, step, 2_500_000)

	// Find the row containing "ERROR" and check the whole printed line, including
	// the '?' prefix and the error name, is the BASIC pen.
	er, _, ok := findSeq(vr, []uint8{'E', 'R', 'R', 'O', 'R'})
	require.True(t, ok, "ERROR should appear on screen")

	require.Equal(t, uint8('?'), ntCell(vr, er, 0), "error line should start with '?'")
	for c := 0; c < 40; c++ {
		tile := ntCell(vr, er, c)
		if tile == 0x20 || tile >= 0xF8 { // skip spaces and any cursor glyph
			continue
		}
		require.Equalf(t, basicPen, attrCell(vr, er, c),
			"error-line cell (%d,%d) tile $%02X should be the BASIC pen", er, c, tile)
	}
}
