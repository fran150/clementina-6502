package clementina

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// LOAD/SAVE - a BASIC program's own tokenized text to/from an SD file, not
// to be confused with MIALOAD/MIASAVE (raw MIA RAM) or the video/audio asset
// family's *LOAD/*SAVE. Handlers in clementina-rom src/basic/
// clementina_extra.s (BASIC_LOAD/BASIC_SAVE). See docs/basic-file.md.

// TestBasicSaveThenLoad writes a small program, SAVEs it, wipes it with NEW,
// LOADs it back, and RUNs the reloaded copy - proving the round trip
// actually restores a runnable program, not just matching bytes on disk.
func TestBasicSaveThenLoad(t *testing.T) {
	dir := t.TempDir()
	computer, step := bootClementinaToPrompt(t)
	computer.SetMiaSDFolder(dir)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, `10 PRINT "HELLO FROM DISK"`)
	typeLine(computer, step, "20 END")
	typeLine(computer, step, `SAVE "PROG.BAS"`)
	editorTickN(computer, step, 1_000_000)

	if _, err := os.Stat(filepath.Join(dir, "PROG.BAS")); err != nil {
		dumpScreen(t, vr)
		t.Fatalf("expected SAVE to create PROG.BAS: %v", err)
	}

	typeLine(computer, step, "NEW")
	typeLine(computer, step, `LOAD "PROG.BAS"`)
	editorTickN(computer, step, 1_000_000)
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_000_000)

	if _, ok := screenContainsRow(vr, "HELLO FROM DISK"); !ok {
		dumpScreen(t, vr)
		t.Fatalf("expected the reloaded program to RUN and print its message")
	}
}

// TestBasicLoadMissingFile verifies LOADing a nonexistent file raises a
// FILE I/O error (not the pre-existing bare stub's ?SYNTAX ERROR) and
// leaves the current program untouched.
func TestBasicLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	computer, step := bootClementinaToPrompt(t)
	computer.SetMiaSDFolder(dir)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, `10 PRINT "STILL HERE"`)
	typeLine(computer, step, `LOAD "NOPE.BAS"`)
	editorTickN(computer, step, 1_000_000)

	if _, ok := screenContainsRow(vr, "FILE I/O"); !ok {
		dumpScreen(t, vr)
		t.Fatalf("expected LOAD of a missing file to raise a FILE I/O error")
	}

	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_000_000)

	if _, ok := screenContainsRow(vr, "STILL HERE"); !ok {
		dumpScreen(t, vr)
		t.Fatalf("expected the original program to survive a failed LOAD")
	}
}

// TestBasicSaveLoadMultiLine round-trips a program with several lines and
// variables, confirming FIX_LINKS correctly relinks more than a single line.
func TestBasicSaveLoadMultiLine(t *testing.T) {
	dir := t.TempDir()
	computer, step := bootClementinaToPrompt(t)
	computer.SetMiaSDFolder(dir)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, "10 A=1")
	typeLine(computer, step, "20 B=2")
	typeLine(computer, step, "30 PRINT A+B")
	typeLine(computer, step, `SAVE "SUM.BAS"`)
	editorTickN(computer, step, 1_000_000)

	typeLine(computer, step, "NEW")
	typeLine(computer, step, `LOAD "SUM.BAS"`)
	editorTickN(computer, step, 1_000_000)
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_000_000)

	// PRINT's positive-number formatting: leading and trailing space (see
	// TestBasicEof/TestBasicSeek), so " 3" (1+2=3) with a leading space.
	if _, ok := screenContainsRow(vr, " 3"); !ok {
		dumpScreen(t, vr)
		t.Fatalf("expected the reloaded 3-line program to RUN and print A+B=3")
	}
}

// TestBasicSaveLoadMultiChunk round-trips a program comfortably over
// LOADSAVE_CHUNK (128 bytes of tokenized text - confirmed at 777 bytes here),
// exercising the multi-chunk continuation path in both BASIC_SAVE and
// BASIC_LOAD, not just the single-chunk case every other test above fits in.
func TestBasicSaveLoadMultiChunk(t *testing.T) {
	dir := t.TempDir()
	computer, step := bootClementinaToPrompt(t)
	computer.SetMiaSDFolder(dir)
	vr := computer.chips.mia.(videoReader)

	for i := 0; i < 20; i++ {
		typeLine(computer, step, fmt.Sprintf("%d A=A+1 : REM PADDING LINE NUMBER %d", 10+i, i))
	}
	typeLine(computer, step, `300 PRINT "BIG PROGRAM A=";A`)
	typeLine(computer, step, `SAVE "BIG.BAS"`)
	editorTickN(computer, step, 1_500_000)

	info, err := os.Stat(filepath.Join(dir, "BIG.BAS"))
	if err != nil || info.Size() <= 128 {
		dumpScreen(t, vr)
		t.Fatalf("test program should exceed LOADSAVE_CHUNK (128 bytes): size=%v err=%v", info, err)
	}

	typeLine(computer, step, "NEW")
	typeLine(computer, step, `LOAD "BIG.BAS"`)
	editorTickN(computer, step, 1_500_000)
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_500_000)

	if _, ok := screenContainsRow(vr, "BIG PROGRAM A= 20"); !ok {
		dumpScreen(t, vr)
		t.Fatalf("expected the reloaded multi-chunk program to RUN correctly (A should reach 20)")
	}
}
