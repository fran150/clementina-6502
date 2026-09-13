package clementina

import (
	"os"
	"path/filepath"
	"testing"
)

// Filesystem management BASIC statements (handlers in clementina-rom
// src/basic/clementina_extra.s, see docs/basic-file.md): KILL/MKDIR/RMDIR/
// NAME/SEEK#/EOF against MIA's SD/FAT layer.

func TestBasicKill(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "DOOMED.DAT"), []byte("x"), 0o644); err != nil {
		t.Fatalf("seeding DOOMED.DAT: %v", err)
	}

	computer, step := bootClementinaToPrompt(t)
	computer.SetMiaSDFolder(dir)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, `10 KILL "DOOMED.DAT"`)
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	if _, err := os.Stat(filepath.Join(dir, "DOOMED.DAT")); !os.IsNotExist(err) {
		dumpScreen(t, vr)
		t.Fatalf("DOOMED.DAT should have been deleted, stat err = %v", err)
	}
}

func TestBasicMkdirRmdir(t *testing.T) {
	dir := t.TempDir()
	computer, step := bootClementinaToPrompt(t)
	computer.SetMiaSDFolder(dir)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, `10 MKDIR "SUBDIR"`)
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	info, err := os.Stat(filepath.Join(dir, "SUBDIR"))
	if err != nil || !info.IsDir() {
		dumpScreen(t, vr)
		t.Fatalf("SUBDIR should exist as a directory after MKDIR, err=%v isDir=%v", err, info != nil && info.IsDir())
	}

	// Direct-mode command at the READY prompt (line 10's program already
	// finished, rather than re-running MKDIR on a directory that now exists).
	typeLine(computer, step, `RMDIR "SUBDIR"`)
	editorTickN(computer, step, 1_000_000)

	if _, err := os.Stat(filepath.Join(dir, "SUBDIR")); !os.IsNotExist(err) {
		dumpScreen(t, vr)
		t.Fatalf("SUBDIR should have been removed by RMDIR, stat err = %v", err)
	}
}

func TestBasicName(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "OLD.DAT"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("seeding OLD.DAT: %v", err)
	}

	computer, step := bootClementinaToPrompt(t)
	computer.SetMiaSDFolder(dir)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, `10 NAME "OLD.DAT" AS "NEW.DAT"`)
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	if _, err := os.Stat(filepath.Join(dir, "OLD.DAT")); !os.IsNotExist(err) {
		dumpScreen(t, vr)
		t.Fatalf("OLD.DAT should no longer exist, stat err = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "NEW.DAT"))
	if err != nil || string(data) != "hello" {
		dumpScreen(t, vr)
		t.Fatalf("NEW.DAT should contain %q, got %q (err %v)", "hello", data, err)
	}
}

// TestBasicSeek seeks directly to offset 2 of a 4-byte file and reads from
// there, verifying SEEK# actually repositions the file rather than being a
// no-op accepted without effect.
func TestBasicSeek(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "R.DAT"), []byte{10, 20, 30, 40}, 0o644); err != nil {
		t.Fatalf("seeding R.DAT: %v", err)
	}

	computer, step := bootClementinaToPrompt(t)
	computer.SetMiaSDFolder(dir)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, `10 OPEN "R.DAT" FOR INPUT AS #3`)
	typeLine(computer, step, "20 SEEK#3,2")
	typeLine(computer, step, "30 BGET#3,B")
	typeLine(computer, step, "40 CLOSE #3")
	typeLine(computer, step, "50 PRINT B")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	// Line 30 of the listed program ("30 BGET#3,B") itself contains "30" with
	// no leading space, so match PRINT's positive-number formatting (a
	// leading space before the digits, confirmed in TestBasicEof) instead of
	// a bare "30" to avoid a false positive against the listing.
	if _, ok := screenContainsRow(vr, " 30"); !ok {
		dumpScreen(t, vr)
		t.Fatalf("expected SEEK#3,2 then BGET#3,B to read byte 30 (offset 2 of {10,20,30,40})")
	}
}

// TestBasicEof verifies EOF(n) reads false before the last byte and true
// once the file is exhausted.
func TestBasicEof(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "E.DAT"), []byte{1, 2}, 0o644); err != nil {
		t.Fatalf("seeding E.DAT: %v", err)
	}

	computer, step := bootClementinaToPrompt(t)
	computer.SetMiaSDFolder(dir)
	vr := computer.chips.mia.(videoReader)

	// This BASIC dialect has no IF...THEN...ELSE (confirmed the hard way: it
	// silently skips a false THEN's whole line, but chokes with a syntax
	// error on ELSE once a true THEN clause has run) - B/C are pre-zeroed and
	// only conditionally set to 1 instead.
	typeLine(computer, step, `10 OPEN "E.DAT" FOR INPUT AS #1`)
	typeLine(computer, step, "20 BGET#1,A")
	typeLine(computer, step, "30 B=0 : IF EOF(1) THEN B=1")
	typeLine(computer, step, "40 BGET#1,A")
	typeLine(computer, step, "50 C=0 : IF EOF(1) THEN C=1")
	typeLine(computer, step, "60 CLOSE #1")
	typeLine(computer, step, "70 PRINT B;C")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	// PRINT B;C with B=0, C=1 formats as " 0 " + " 1" (each number gets a
	// leading sign-or-space and, for a positive number, a trailing space) -
	// confirmed directly against a screen dump rather than assumed.
	if _, ok := screenContainsRow(vr, " 0  1"); !ok {
		dumpScreen(t, vr)
		t.Fatalf("expected B=0 (not at EOF after 1st byte of 2) and C=1 (at EOF after 2nd)")
	}
}
