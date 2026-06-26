package clementina

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/stretchr/testify/require"
)

// Kernel variable / BASIC buffer addresses (see clementina-rom kernel.inc and
// the BASIC zeropage layout). These 6502-visible base-RAM locations are what the
// screen editor and BASIC line input touch, so the harness asserts against them
// after driving the ROM.
const (
	addrCursorX = 0x0300
	addrCursorY = 0x0301
	addrEditBuf = 0x0325 // EDIT_BUF: harvested logical line
	addrEditLen = 0x0375 // EDIT_LEN: harvested length (trailing spaces trimmed)
)

// PETSCII control codes the kernel decode table emits / CHROUT acts on.
const (
	keyCR        = 0x0D
	keyCursorUp  = 0x91
	keyCursorDn  = 0x11
	keyCursorRt  = 0x1D
	keyCursorLt  = 0x9D
	keyHome      = 0x13
	keyBackspace = 0x08
	keyDelete    = 0x7F // forward delete
	keyInsert    = 0x94
)

// harvestedLine reads EDIT_BUF[0:EDIT_LEN] as a string after a RETURN.
func harvestedLine(computer *ClementinaComputer) string {
	n := int(peek(computer, addrEditLen))
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = peek(computer, uint32(addrEditBuf+i))
	}
	return string(b)
}

type miaInjector interface{ DebugQueueInput(uint8) }

// bootClementinaToPrompt resets the computer, lets the MIA load the kernel image,
// and runs long enough for BASIC to reach its input wait (in the screen editor).
func bootClementinaToPrompt(t *testing.T) (*ClementinaComputer, *common.StepContext) {
	t.Helper()

	computer, err := NewClementinaComputer()
	require.NoError(t, err)
	t.Cleanup(computer.Close)

	step := common.NewStepContext()
	computer.Reset(true)
	for range 3 {
		tickComputer(computer, &step)
		step.NextCycle()
	}
	computer.Reset(false)

	editorTickN(computer, &step, 4_000_000)
	return computer, &step
}

func editorTickN(computer *ClementinaComputer, step *common.StepContext, n int) {
	for range n {
		tickComputer(computer, step)
		step.NextCycle()
	}
}

func injectKeys(computer *ClementinaComputer, bytes ...uint8) {
	inj := computer.chips.mia.(miaInjector)
	for _, b := range bytes {
		inj.DebugQueueInput(b)
	}
}

func peek(computer *ClementinaComputer, addr uint32) uint8 {
	return computer.chips.baseram.Peek(addr)
}

// TestClementinaEditorEchoesTypedChar: a typed character is drawn by the kernel
// editor and advances the cursor (BASIC no longer echoes).
func TestClementinaEditorEchoesTypedChar(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	x0 := peek(computer, addrCursorX)
	injectKeys(computer, 'A')
	editorTickN(computer, step, 200_000)

	require.Equalf(t, x0+1, peek(computer, addrCursorX),
		"CURSOR_X should advance after typing 'A' (was %d)", x0)
}

// TestClementinaEditorCursorLeftMoves: a cursor-left code now moves the cursor
// during line editing instead of being dropped by BASIC.
func TestClementinaEditorCursorLeftMoves(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	x0 := peek(computer, addrCursorX)
	injectKeys(computer, 'A', 'B', 'C')
	editorTickN(computer, step, 200_000)
	require.Equal(t, x0+3, peek(computer, addrCursorX), "three chars should advance the cursor by 3")

	injectKeys(computer, keyCursorLt)
	editorTickN(computer, step, 200_000)
	require.Equal(t, x0+2, peek(computer, addrCursorX), "cursor-left should step the cursor back one")
}

// TestClementinaEditorOvertypeAndHarvest: type AB, cursor-left over B, type C to
// overwrite it, RETURN. The kernel harvests the on-screen logical line, so
// EDIT_BUF holds the edited result "AC".
func TestClementinaEditorOvertypeAndHarvest(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	injectKeys(computer, 'A', 'B', keyCursorLt, 'C', keyCR)
	editorTickN(computer, step, 400_000)

	require.Equal(t, uint8(2), peek(computer, addrEditLen), "harvested length should be 2")
	require.Equal(t, uint8('A'), peek(computer, addrEditBuf+0), "EDIT_BUF[0]")
	require.Equal(t, uint8('C'), peek(computer, addrEditBuf+1), "EDIT_BUF[1] (B overtyped with C)")
}

// TestClementinaEditorCursorUpMoves: cursor-up moves the cursor to a previous
// row (the basis for editing a line above the current input line).
func TestClementinaEditorCursorUpMoves(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	y0 := peek(computer, addrCursorY)
	require.Greater(t, y0, uint8(0), "input line should sit below the banner")

	injectKeys(computer, keyCursorUp)
	editorTickN(computer, step, 200_000)
	require.Equal(t, y0-1, peek(computer, addrCursorY), "cursor-up should move up one row")
}

// TestClementinaEditorExecutesTypedCommand drives the whole pipeline: type a
// POKE command, press RETURN, and confirm BASIC tokenized and executed the
// harvested line by checking the byte it wrote. POKE 752,42 -> $02F0 = 42.
func TestClementinaEditorExecutesTypedCommand(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	require.Equal(t, uint8(0), peek(computer, 0x02F0), "target byte should start clear")

	injectKeys(computer, []byte("POKE 752,42")...)
	injectKeys(computer, keyCR)
	editorTickN(computer, step, 600_000)

	require.Equal(t, uint8(42), peek(computer, 0x02F0),
		"BASIC should have executed the harvested POKE command")
}

// TestClementinaEditorBackspaceClosesGap: backspace deletes the char to the left
// of the cursor and pulls the rest of the line in (gap-closing, not just erase).
// Type AXBC, move left to sit over B, backspace removes X -> ABC.
func TestClementinaEditorBackspaceClosesGap(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	injectKeys(computer, 'A', 'X', 'B', 'C', keyCursorLt, keyCursorLt, keyBackspace, keyCR)
	editorTickN(computer, step, 400_000)

	require.Equal(t, "ABC", harvestedLine(computer), "backspace should delete X and close the gap")
}

// TestClementinaEditorForwardDelete: Delete removes the char at the cursor and
// closes the gap. Type AXBC, move left over X, Delete removes X -> ABC.
func TestClementinaEditorForwardDelete(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	injectKeys(computer, 'A', 'X', 'B', 'C', keyCursorLt, keyCursorLt, keyCursorLt, keyDelete, keyCR)
	editorTickN(computer, step, 400_000)

	require.Equal(t, "ABC", harvestedLine(computer), "Delete should remove X at the cursor and close the gap")
}

// TestClementinaEditorInsertOpensGap: Insert opens a blank at the cursor (rest of
// line shifts right); typing into it inserts. Type ABC, move left over B, Insert
// then X -> AXBC.
func TestClementinaEditorInsertOpensGap(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	injectKeys(computer, 'A', 'B', 'C', keyCursorLt, keyCursorLt, keyInsert, 'X', keyCR)
	editorTickN(computer, step, 400_000)

	require.Equal(t, "AXBC", harvestedLine(computer), "Insert should open a gap that X fills")
}
