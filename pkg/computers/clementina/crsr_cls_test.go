package clementina

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
)

func typeLine(computer *ClementinaComputer, step *common.StepContext, line string) {
	injectKeys(computer, []byte(line)...)
	injectKeys(computer, keyCR)
	editorTickN(computer, step, 400_000)
}

func require_eq(t *testing.T, want int, got uint8, msg string) {
	t.Helper()
	if int(got) != want {
		t.Fatalf("%s: want %d got %d", msg, want, got)
	}
}

// Run a program that positions the cursor then spins, so the cursor stays put
// (no READY prompt redraw) when we sample CURSOR_X/Y.
func TestCursorAtPositionsInProgram(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	typeLine(computer, step, "10 CRSR 5,7")
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_000_000)
	require_eq(t, 5, peek(computer, addrCursorX), "CURSOR_X while program holds at CRSR 5,7")
	require_eq(t, 7, peek(computer, addrCursorY), "CURSOR_Y while program holds at CRSR 5,7")
}

func TestCursorAtLowerRightInProgram(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	typeLine(computer, step, "10 CRSR 39,24")
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_000_000)
	require_eq(t, 39, peek(computer, addrCursorX), "CURSOR_X at CRSR 39,24")
	require_eq(t, 24, peek(computer, addrCursorY), "CURSOR_Y at CRSR 39,24")
}

func TestClsHomesCursorInProgram(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	typeLine(computer, step, "10 CRSR 10,10")
	typeLine(computer, step, "20 CLS")
	typeLine(computer, step, "30 GOTO 30")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_000_000)
	require_eq(t, 0, peek(computer, addrCursorX), "CURSOR_X after CLS")
	require_eq(t, 0, peek(computer, addrCursorY), "CURSOR_Y after CLS")
}

func TestCursorAtOutOfRangeRecovers(t *testing.T) {
	// column 40 is out of range (0..39) -> ILLEGAL QUANTITY, prompt must survive
	computer, step := bootClementinaToPrompt(t)
	typeLine(computer, step, "CRSR 40,0")
	x0 := peek(computer, addrCursorX)
	injectKeys(computer, 'Z')
	editorTickN(computer, step, 300_000)
	if peek(computer, addrCursorX) != x0+1 {
		t.Fatalf("out-of-range CRSR should error cleanly, not hang")
	}
}

func TestListDoesNotHang(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	typeLine(computer, step, "LIST")
	x0 := peek(computer, addrCursorX)
	injectKeys(computer, 'Z')
	editorTickN(computer, step, 300_000)
	if peek(computer, addrCursorX) != x0+1 {
		t.Fatalf("LIST hung")
	}
}
