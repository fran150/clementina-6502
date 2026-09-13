package clementina

import (
	"os"
	"path/filepath"
	"testing"
)

// File I/O BASIC statements (handlers in clementina-rom src/basic/clementina_extra.s,
// see docs/basic-file.md): OPEN/CLOSE/BGET#/BPUT# against MIA's SD/FAT layer.
// These boot the real ROM, attach a host folder as the emulated SD card, type a
// program, run it, and check the resulting file (or read-back values) against
// the host folder.

func TestBasicFileOpenPutClose(t *testing.T) {
	dir := t.TempDir()
	computer, step := bootClementinaToPrompt(t)
	computer.SetMiaSDFolder(dir)

	typeLine(computer, step, `10 OPEN "TEST.DAT" FOR OUTPUT AS #1`)
	typeLine(computer, step, "20 BPUT#1,65")
	typeLine(computer, step, "30 BPUT#1,66")
	typeLine(computer, step, "40 BPUT#1,67")
	typeLine(computer, step, "50 CLOSE #1")
	typeLine(computer, step, "60 GOTO 60")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	data, err := os.ReadFile(filepath.Join(dir, "TEST.DAT"))
	if err != nil {
		vr := computer.chips.mia.(videoReader)
		dumpScreen(t, vr)
		t.Fatalf("reading TEST.DAT: %v", err)
	}
	if string(data) != "ABC" {
		t.Fatalf("TEST.DAT content = %q, want %q", data, "ABC")
	}
}

// TestBasicFileOpenGetClose exercises the read path: BGET# into numeric
// variables, verified by PRINTing them to the screen (the overlay text
// after RUN completes).
func TestBasicFileOpenGetClose(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SRC.DAT"), []byte{88, 89}, 0o644); err != nil {
		t.Fatalf("seeding SRC.DAT: %v", err)
	}

	computer, step := bootClementinaToPrompt(t)
	computer.SetMiaSDFolder(dir)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, `10 OPEN "SRC.DAT" FOR INPUT AS #2`)
	typeLine(computer, step, "20 BGET#2,A")
	typeLine(computer, step, "30 BGET#2,B")
	typeLine(computer, step, "40 CLOSE #2")
	typeLine(computer, step, "50 PRINT A;B")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	found := false
	for r := 0; r < 25; r++ {
		for c := 0; c < 38; c++ {
			if ntCell(vr, r, c) == '8' && ntCell(vr, r, c+1) == '8' {
				found = true
			}
		}
	}
	if !found {
		dumpScreen(t, vr)
		t.Fatalf("did not find printed \"88\" (A=88) on screen")
	}
}
