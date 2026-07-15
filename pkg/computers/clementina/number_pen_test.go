package clementina

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestUnstyledNumberUsesBasicPen reproduces  10 A=7:?"FR";A  where "FR" is typed
// in one color and the program then runs under a different editor pen. The styled
// literal FR must keep its written color, but the unstyled number A must print in
// the BASIC pen (not the live editor pen). Regression for @lit printing unstyled
// output at the editor pen.
func TestUnstyledNumberUsesBasicPen(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	basicPen := peek(computer, addrTextAttr) // ATTR_WHITE = 0
	require.Equal(t, uint8(0x00), basicPen)

	// Enter the program line with the editor pen = palette 5, so "FR" stores color 5.
	injectKeys(computer, keyEsc, '5')
	editorTickN(computer, step, 300_000)
	injectKeys(computer, '1', '0', ' ', 'A', '=', '7', ':', '?', '"', 'F', 'R', '"', ';', 'A', keyCR)
	editorTickN(computer, step, 1_800_000)

	// Change the editor pen to palette 2, then RUN.
	injectKeys(computer, keyEsc, '2')
	editorTickN(computer, step, 300_000)
	injectKeys(computer, 'R', 'U', 'N', keyCR)
	editorTickN(computer, step, 3_000_000)

	// Find the program OUTPUT line: "FR" followed shortly by the digit '7'
	// (the typed line ...?"FR";A has a quote after FR and no nearby digit).
	outRow, frCol := -1, -1
	for r := 0; r < 25 && outRow < 0; r++ {
		for c := 0; c+3 < 40; c++ {
			if ntCell(vr, r, c) == 'F' && ntCell(vr, r, c+1) == 'R' &&
				(ntCell(vr, r, c+2) == '7' || ntCell(vr, r, c+3) == '7') {
				outRow, frCol = r, c
				break
			}
		}
	}
	require.GreaterOrEqual(t, outRow, 0, "program output 'FR..7' should appear")

	// The styled literal keeps its written color (palette 5).
	require.Equal(t, uint8(0x05), attrCell(vr, outRow, frCol)&0x7F, "F should keep stored color 5")
	require.Equal(t, uint8(0x05), attrCell(vr, outRow, frCol+1)&0x7F, "R should keep stored color 5")

	// The number must use the BASIC pen (not the editor pen, palette 2).
	found7 := false
	for c := frCol + 2; c < 40; c++ {
		if ntCell(vr, outRow, c) == '7' {
			found7 = true
			require.Equalf(t, basicPen, attrCell(vr, outRow, c),
				"number 7 (col %d) should use the BASIC pen, got $%02X", c, attrCell(vr, outRow, c))
			break
		}
	}
	require.True(t, found7, "number 7 should appear in the output")
}
