package clementina

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const addrCursorVisible = 0x0309 // CURSOR_VISIBLE: nonzero while the cursor glyph is drawn

// cursorCell scans the overlay for a cursor indicator tile ($F8-$FF) and returns
// its (row, col), or ok=false if the cursor is not currently drawn.
func cursorCell(vr videoReader) (row, col int, ok bool) {
	for r := 0; r < 25; r++ {
		for c := 0; c < 40; c++ {
			if ntCell(vr, r, c) >= 0xF8 {
				return r, c, true
			}
		}
	}
	return 0, 0, false
}

// TestCursorKeepsEditorPenAfterOutput: after program output (LIST -> READY.), the
// prompt cursor must be drawn in the editor pen, never the BASIC pen. Regression
// for the cursor being painted white by output (chrout used to re-show it in the
// output's pen and leave it stranded there until a blink).
func TestCursorKeepsEditorPenAfterOutput(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	injectKeys(computer, keyEsc, '2') // editor pen = palette 2
	editorTickN(computer, step, 300_000)
	editorPen := peek(computer, addrTextAttr)
	require.Equal(t, uint8(0x02), editorPen, "ESC 2 sets the editor pen to palette 2")

	// LIST with no program prints READY. and returns to the prompt.
	injectKeys(computer, 'L', 'I', 'S', 'T', keyCR)
	editorTickN(computer, step, 2_000_000)

	// Sample across several blink periods. Every time the cursor glyph is on
	// screen, its cell must carry the editor pen (masking CHR_ALT, which the
	// preview clears) — never the BASIC pen ($00).
	sawCursor := false
	for i := 0; i < 80; i++ {
		editorTickN(computer, step, 50_000)
		if peek(computer, addrCursorVisible) == 0 {
			continue
		}
		r, c, ok := cursorCell(vr)
		if !ok {
			continue
		}
		sawCursor = true
		require.Equalf(t, editorPen&0x7F, attrCell(vr, r, c)&0x7F,
			"visible cursor at (%d,%d) must use the editor pen, got $%02X",
			r, c, attrCell(vr, r, c))
	}
	require.True(t, sawCursor, "cursor should become visible at the prompt")
}

// TestCursorEditorPenAfterListingProgram is the bug-2 case: LISTing an actual
// (multi-line) program must not leave the cursor in the BASIC pen.
func TestCursorEditorPenAfterListingProgram(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	// Enter a program line, then set the editor pen.
	injectKeys(computer, '1', '0', ' ', '?', '"', 'H', 'I', '"', keyCR)
	editorTickN(computer, step, 1_500_000)
	injectKeys(computer, keyEsc, '2')
	editorTickN(computer, step, 300_000)
	editorPen := peek(computer, addrTextAttr)
	require.Equal(t, uint8(0x02), editorPen, "editor pen = palette 2")

	// LIST the program (line 10 + READY.).
	injectKeys(computer, 'L', 'I', 'S', 'T', keyCR)
	editorTickN(computer, step, 2_500_000)

	sawCursor := false
	for i := 0; i < 80; i++ {
		editorTickN(computer, step, 50_000)
		if peek(computer, addrCursorVisible) == 0 {
			continue
		}
		r, c, ok := cursorCell(vr)
		if !ok {
			continue
		}
		sawCursor = true
		require.Equalf(t, editorPen&0x7F, attrCell(vr, r, c)&0x7F,
			"cursor at (%d,%d) should be editor pen after LIST, got $%02X",
			r, c, attrCell(vr, r, c))
	}
	require.True(t, sawCursor, "cursor should become visible at the prompt after LIST")
}
