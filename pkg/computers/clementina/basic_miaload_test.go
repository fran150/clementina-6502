package clementina

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// MIALOAD/MIASAVE (generic MIA RAM loader/saver) and the file-sourced form of
// the video/audio asset family (CHRLOAD etc.), handlers in clementina-rom
// src/basic/clementina_extra.s. See docs/basic-file.md.

func TestBasicMialoadMiasave(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SRC.BIN"), []byte{10, 20, 30, 40}, 0o644); err != nil {
		t.Fatalf("seeding SRC.BIN: %v", err)
	}

	computer, step := bootClementinaToPrompt(t)
	computer.SetMiaSDFolder(dir)
	vr := computer.chips.mia.(videoReader)

	// 78000 is a scratch MIA RAM address well clear of any register block or
	// syncable video region.
	typeLine(computer, step, `10 MIALOAD "SRC.BIN",78000`)
	typeLine(computer, step, "20 MIASAVE \"DST.BIN\",78000,4")
	typeLine(computer, step, "30 GOTO 30")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	for i, want := range []uint8{10, 20, 30, 40} {
		got := vr.DebugReadVideo(uint32(78000 + i))
		if got != want {
			dumpScreen(t, vr)
			t.Fatalf("MIA RAM at %d: want %d got %d", 78000+i, want, got)
		}
	}

	data, err := os.ReadFile(filepath.Join(dir, "DST.BIN"))
	if err != nil {
		dumpScreen(t, vr)
		t.Fatalf("reading DST.BIN: %v", err)
	}
	want := []byte{10, 20, 30, 40}
	if len(data) != len(want) {
		t.Fatalf("DST.BIN length = %d, want %d (content %v)", len(data), len(want), data)
	}
	for i := range want {
		if data[i] != want[i] {
			t.Fatalf("DST.BIN[%d] = %d, want %d", i, data[i], want[i])
		}
	}
}

// TestBasicChrloadFile exercises CHRLOAD's new file-sourced meaning (was
// DATA-sourced; that behavior is now CHRREAD - see basic_video_phase2b_test.go).
func TestBasicChrloadFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "FONT.BIN"), []byte{11, 22, 33}, 0o644); err != nil {
		t.Fatalf("seeding FONT.BIN: %v", err)
	}

	computer, step := bootClementinaToPrompt(t)
	computer.SetMiaSDFolder(dir)

	typeLine(computer, step, `10 CHRLOAD 1,100,3,"FONT.BIN"`)
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	require_eq(t, 11, peekCHR(computer, 1, 100), "CHR bank1 offset100 (from file)")
	require_eq(t, 22, peekCHR(computer, 1, 101), "CHR bank1 offset101 (from file)")
	require_eq(t, 33, peekCHR(computer, 1, 102), "CHR bank1 offset102 (from file)")
}

// TestBasicChrsaveFile exercises CHRSAVE (CHR bank -> file), the reverse of
// TestBasicChrloadFile, chained onto CHRREAD (DATA -> CHR bank) so the whole
// round trip is covered without a separate seeding mechanism.
func TestBasicChrsaveFile(t *testing.T) {
	dir := t.TempDir()
	computer, step := bootClementinaToPrompt(t)
	computer.SetMiaSDFolder(dir)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, "10 DATA 44,55,66")
	typeLine(computer, step, "20 CHRREAD 2,200,3")
	typeLine(computer, step, `30 CHRSAVE 2,200,3,"DUMP.BIN"`)
	typeLine(computer, step, "40 GOTO 40")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	data, err := os.ReadFile(filepath.Join(dir, "DUMP.BIN"))
	if err != nil {
		dumpScreen(t, vr)
		t.Fatalf("reading DUMP.BIN: %v", err)
	}
	want := []byte{44, 55, 66}
	if len(data) != len(want) {
		t.Fatalf("DUMP.BIN length = %d, want %d (content %v)", len(data), len(want), data)
	}
	for i := range want {
		if data[i] != want[i] {
			t.Fatalf("DUMP.BIN[%d] = %d, want %d", i, data[i], want[i])
		}
	}
}

// TestBasicOamloadFile exercises OAMLOAD's file-sourced meaning. Its `count`
// means sprites (5 bytes each, tile/xlo/ylo/attr/ext), matching OAMREAD's
// units - not a raw byte length. Regression test for a 2026-09-16 fix: count
// used to go straight into the byte-length register untouched, so
// `OAMLOAD 0,2,"S.OAM"` loaded 2 bytes (spilling into just sprite 0's
// tile/xlo) instead of 2 whole sprites (10 bytes).
func TestBasicOamloadFile(t *testing.T) {
	dir := t.TempDir()
	data := []byte{
		1, 2, 3, 4, 5, // sprite 0: tile,xlo,ylo,attr,ext
		6, 7, 8, 9, 10, // sprite 1
	}
	if err := os.WriteFile(filepath.Join(dir, "S.OAM"), data, 0o644); err != nil {
		t.Fatalf("seeding S.OAM: %v", err)
	}

	computer, step := bootClementinaToPrompt(t)
	computer.SetMiaSDFolder(dir)

	typeLine(computer, step, `10 OAMLOAD 0,2,"S.OAM"`)
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	for n := 0; n < 2; n++ {
		for f := 0; f < 5; f++ {
			want := int(data[n*5+f])
			require_eq(t, want, peekOAM(computer, n, f), fmt.Sprintf("sprite%d field%d", n, f))
		}
	}
}

// TestBasicOamsaveFile exercises OAMSAVE (OAM -> file), the reverse of
// TestBasicOamloadFile, chained onto OAMREAD (DATA -> OAM) so the round trip
// is covered without a separate seeding mechanism. Same count-unit fix as
// TestBasicOamloadFile above.
func TestBasicOamsaveFile(t *testing.T) {
	dir := t.TempDir()
	computer, step := bootClementinaToPrompt(t)
	computer.SetMiaSDFolder(dir)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, "10 DATA 1,2,3,4,5,6,7,8,9,10")
	typeLine(computer, step, "20 OAMREAD 0,2")
	typeLine(computer, step, `30 OAMSAVE 0,2,"DUMP.OAM"`)
	typeLine(computer, step, "40 GOTO 40")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	data, err := os.ReadFile(filepath.Join(dir, "DUMP.OAM"))
	if err != nil {
		dumpScreen(t, vr)
		t.Fatalf("reading DUMP.OAM: %v", err)
	}
	want := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	if len(data) != len(want) {
		t.Fatalf("DUMP.OAM length = %d, want %d (content %v)", len(data), len(want), data)
	}
	for i := range want {
		if data[i] != want[i] {
			t.Fatalf("DUMP.OAM[%d] = %d, want %d", i, data[i], want[i])
		}
	}
}
