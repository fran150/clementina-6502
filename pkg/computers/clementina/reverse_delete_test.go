package clementina

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestReverseModeDeleteBlanksNotReversed: with reverse mode enabled, deleting
// characters shifts the line left and fills freed cells with spaces. Those
// spaces must use the regular (non-reversed) attribute — a reversed space renders
// as a solid block. Regression for the editor stamping freed cells with the full
// TEXT_ATTR (including the CHR_ALT reverse bit).
func TestReverseModeDeleteBlanksNotReversed(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	// Enable reverse mode (ESC R sets the CHR_ALT bit).
	injectKeys(computer, keyEsc, 'R')
	editorTickN(computer, step, 300_000)
	require.NotZero(t, peek(computer, addrTextAttr)&0x80, "ESC R should set the reverse bit")

	// Type reversed text.
	injectKeys(computer, 'A', 'B', 'C', 'D', 'E')
	editorTickN(computer, step, 900_000)

	// Delete several times; each closes the gap and blanks a freed cell.
	for i := 0; i < 3; i++ {
		injectKeys(computer, keyBackspace)
		editorTickN(computer, step, 400_000)
	}

	// No blank (space tile) on the edited row may be reversed.
	y := int(peek(computer, addrCursorY))
	for c := 0; c < 40; c++ {
		if ntCell(vr, y, c) == 0x20 { // space
			require.Zerof(t, attrCell(vr, y, c)&0x80,
				"space at (%d,%d) must not carry the reverse bit", y, c)
		}
	}

	// Forward-delete from the middle exercises the same blank-fill path.
	injectKeys(computer, keyHome)
	editorTickN(computer, step, 200_000)
	injectKeys(computer, keyCursorRt, keyDelete) // move in one, delete forward
	editorTickN(computer, step, 400_000)
	y = int(peek(computer, addrCursorY))
	for c := 0; c < 40; c++ {
		if ntCell(vr, y, c) == 0x20 {
			require.Zerof(t, attrCell(vr, y, c)&0x80,
				"space at (%d,%d) must not carry the reverse bit after fwd-delete", y, c)
		}
	}
}
