package clementina

import (
	"os"
	"path/filepath"
	"testing"
)

// CD (change directory) and DIR (list directory) BASIC statements, handlers
// in clementina-rom src/basic/clementina_extra.s (BASIC_CD/BASIC_DIR). See
// docs/basic-file.md.
//
// CD relies entirely on FatFs's own current-directory tracking
// (FF_FS_RPATH, clementina-mia sd.c's sd_request_chdir): every other
// path-taking command already passed relative paths through unmodified, so
// once CD moves FatFs's current directory, OPEN/KILL/MIALOAD/etc. all
// resolve relative paths inside it for free - TestBasicCdThenRelativeOpen
// below is what actually proves that wiring, not just that CD's own FS_OK
// round-trips.

// TestBasicCdThenRelativeOpen verifies CD actually repositions FatFs's
// current directory: a relative OPEN issued after CD "SUB" reads from
// SUB/FILE.DAT, not the SD root.
func TestBasicCdThenRelativeOpen(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "SUB"), 0o755); err != nil {
		t.Fatalf("creating SUB: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SUB", "FILE.DAT"), []byte{88}, 0o644); err != nil {
		t.Fatalf("seeding SUB/FILE.DAT: %v", err)
	}

	computer, step := bootClementinaToPrompt(t)
	computer.SetMiaSDFolder(dir)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, `10 CD "SUB"`)
	typeLine(computer, step, `20 OPEN "FILE.DAT" FOR INPUT AS #1`)
	typeLine(computer, step, "30 BGET#1,B")
	typeLine(computer, step, "40 CLOSE #1")
	typeLine(computer, step, "50 PRINT B")
	typeLine(computer, step, "60 GOTO 60")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	// PRINT's positive-number formatting puts a leading space before the
	// digits (confirmed in TestBasicEof/TestBasicSeek); no other printed
	// text in this program contains "88".
	if _, ok := screenContainsRow(vr, " 88"); !ok {
		dumpScreen(t, vr)
		t.Fatalf("expected CD \"SUB\" to make a relative OPEN \"FILE.DAT\" read SUB/FILE.DAT (byte 88)")
	}
}

// TestBasicCdMissingDirectory verifies CD into a nonexistent directory
// raises a FILE I/O error rather than silently succeeding.
func TestBasicCdMissingDirectory(t *testing.T) {
	dir := t.TempDir()

	computer, step := bootClementinaToPrompt(t)
	computer.SetMiaSDFolder(dir)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, `10 CD "NOPE"`)
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_500_000)

	if _, ok := screenContainsRow(vr, "FILE I/O"); !ok {
		dumpScreen(t, vr)
		t.Fatalf("expected CD into a nonexistent directory to raise a FILE I/O error")
	}
}

// TestBasicDirListsCurrentDirectory verifies DIR with no argument lists the
// SD root by default, with a subdirectory marked by a trailing "/".
func TestBasicDirListsCurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "A.DAT"), []byte("x"), 0o644); err != nil {
		t.Fatalf("seeding A.DAT: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "SUBDIR"), 0o755); err != nil {
		t.Fatalf("creating SUBDIR: %v", err)
	}

	computer, step := bootClementinaToPrompt(t)
	computer.SetMiaSDFolder(dir)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, "10 DIR")
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_500_000)

	if _, ok := screenContainsRow(vr, "A.DAT"); !ok {
		dumpScreen(t, vr)
		t.Fatalf("expected DIR to list A.DAT")
	}
	if _, ok := screenContainsRow(vr, "SUBDIR/"); !ok {
		dumpScreen(t, vr)
		t.Fatalf("expected DIR to list SUBDIR with a trailing / marking it as a directory")
	}
}

// TestBasicDirWithPathArgument verifies DIR "path" lists an explicit
// directory without needing CD first.
func TestBasicDirWithPathArgument(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "SUBDIR"), 0o755); err != nil {
		t.Fatalf("creating SUBDIR: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SUBDIR", "B.DAT"), []byte("x"), 0o644); err != nil {
		t.Fatalf("seeding SUBDIR/B.DAT: %v", err)
	}

	computer, step := bootClementinaToPrompt(t)
	computer.SetMiaSDFolder(dir)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, `10 DIR "SUBDIR"`)
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_500_000)

	if _, ok := screenContainsRow(vr, "B.DAT"); !ok {
		dumpScreen(t, vr)
		t.Fatalf(`expected DIR "SUBDIR" to list B.DAT`)
	}
}

// TestBasicCdThenDirNoArgument chains CD and DIR together: after CD "SUB",
// a no-argument DIR must list SUB's contents, not the SD root's - the two
// commands' whole point is working together this way.
func TestBasicCdThenDirNoArgument(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ROOT.DAT"), []byte("x"), 0o644); err != nil {
		t.Fatalf("seeding ROOT.DAT: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "SUB"), 0o755); err != nil {
		t.Fatalf("creating SUB: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SUB", "C.DAT"), []byte("x"), 0o644); err != nil {
		t.Fatalf("seeding SUB/C.DAT: %v", err)
	}

	computer, step := bootClementinaToPrompt(t)
	computer.SetMiaSDFolder(dir)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, `10 CD "SUB"`)
	typeLine(computer, step, "20 DIR")
	typeLine(computer, step, "30 GOTO 30")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_500_000)

	if _, ok := screenContainsRow(vr, "C.DAT"); !ok {
		dumpScreen(t, vr)
		t.Fatalf(`expected DIR after CD "SUB" to list SUB's contents (C.DAT)`)
	}
	if _, ok := screenContainsRow(vr, "ROOT.DAT"); ok {
		dumpScreen(t, vr)
		t.Fatalf(`expected DIR after CD "SUB" NOT to list the SD root's contents (ROOT.DAT)`)
	}
}
