package clementina

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// findRow scans the overlay nametable for the byte sequence `seq` and returns
// the (row, col) of its first occurrence, or ok=false if not present.
func findSeq(vr videoReader, seq []uint8) (row, col int, ok bool) {
	for r := 0; r < 25; r++ {
		for c := 0; c+len(seq) <= 40; c++ {
			match := true
			for i, b := range seq {
				if ntCell(vr, r, c+i) != b {
					match = false
					break
				}
			}
			if match {
				return r, c, true
			}
		}
	}
	return 0, 0, false
}

// TestPenIndependence: the editor pen (TEXT_ATTR set via ESC) and the BASIC pen
// (BASIC_DEFAULT_ATTR) are independent. After setting the editor pen and running
// `?"FRAN"`, the printed FRAN carries the editor pen, the READY. message carries
// the BASIC pen, and TEXT_ATTR is left at the editor pen so the cursor keeps its
// color (regression for the pen "snapping back" to the BASIC default).
func TestPenIndependence(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	basicPen := peek(computer, addrTextAttr)
	require.Equal(t, uint8(0x00), basicPen, "boot pen is ATTR_WHITE ($00)")

	// Set the editor pen to palette 2 via ESC 2.
	injectKeys(computer, keyEsc, '2')
	editorTickN(computer, step, 300_000)
	editorPen := peek(computer, addrTextAttr)
	require.Equal(t, uint8(0x02), editorPen, "ESC 2 sets the editor pen to palette 2")

	// Type  ?"FRAN"  and RETURN.
	injectKeys(computer, '?', '"', 'F', 'R', 'A', 'N', '"', keyCR)
	editorTickN(computer, step, 3_000_000)

	// The editor pen must survive the PRINT + READY. (cursor keeps its color).
	require.Equal(t, editorPen, peek(computer, addrTextAttr),
		"editor pen must be preserved after PRINT and READY.")

	// The printed FRAN must be in the editor pen.
	fr, fc, ok := findSeq(vr, []uint8{'F', 'R', 'A', 'N'})
	require.True(t, ok, "FRAN should appear on screen")
	for i := 0; i < 4; i++ {
		require.Equalf(t, editorPen, attrCell(vr, fr, fc+i),
			"FRAN cell %d should carry the editor pen", i)
	}

	// READY. must be in the BASIC pen.
	rr, rc, ok := findSeq(vr, []uint8{'R', 'E', 'A', 'D', 'Y', '.'})
	require.True(t, ok, "READY. should appear on screen")
	for i := 0; i < 6; i++ {
		require.Equalf(t, basicPen, attrCell(vr, rr, rc+i),
			"READY. cell %d should carry the BASIC pen", i)
	}
}
