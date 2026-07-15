package clementina

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEnterAdvancesOneRow: pressing RETURN on a blank line at the prompt must move
// the cursor down exactly one row. Regression for edit_line emitting its own CR on
// top of BASIC's INLIN/CRDO newline, which advanced the cursor two rows per RETURN.
func TestEnterAdvancesOneRow(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	prev := int(peek(computer, addrCursorY))
	for i := 1; i <= 5; i++ {
		injectKeys(computer, keyCR)
		editorTickN(computer, step, 1_500_000)
		got := int(peek(computer, addrCursorY))
		require.Equalf(t, prev+1, got,
			"RETURN #%d should advance CURSOR_Y by 1 (was %d, got %d)", i, prev, got)
		prev = got
	}
}
