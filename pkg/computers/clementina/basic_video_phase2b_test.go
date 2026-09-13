package clementina

import "testing"

// Phase 2b BASIC video statements - the commands cut for ROM size in the
// first Phase 2 pass, restored after growing RAMSTART2/BGP_BASE/
// MAX_KERNEL_BYTES (clementina-rom src/basic/defines_clementina.s,
// clementina_extra.s, Makefile): BGCHAR, and the single-field sprite setters
// (SPRTILE/SPRX/SPRY/SPRCOLOR/SPRFLIP/SPRPRI). See docs/basic-video.md.
//
// PALLOAD/CHRLOAD/BGLOAD's DATA-sourced behavior (tested below as PALREAD/
// CHRREAD/NTREAD+ATRREAD) was renamed in the file-I/O work: the bare names
// now mean "load from an SD file" instead, matching LOAD's real meaning; see
// docs/basic-file.md.

const (
	chrBase    = 0x00200
	chrBankLen = 6144
	bgNTBase   = 0x0C200
	bgAttrBase = 0x0E140
	bgTableLen = 1000
	palBase    = 0x00100
)

func peekCHR(computer *ClementinaComputer, bank, offset int) uint8 {
	vr := computer.chips.mia.(videoReader)
	return vr.DebugReadVideo(uint32(chrBase + bank*chrBankLen + offset))
}

func peekBGNT(computer *ClementinaComputer, table, cell int) uint8 {
	vr := computer.chips.mia.(videoReader)
	return vr.DebugReadVideo(uint32(bgNTBase + table*bgTableLen + cell))
}

func peekBGAttr(computer *ClementinaComputer, table, cell int) uint8 {
	vr := computer.chips.mia.(videoReader)
	return vr.DebugReadVideo(uint32(bgAttrBase + table*bgTableLen + cell))
}

func peekPalette(computer *ClementinaComputer, bank, offset int) uint8 {
	vr := computer.chips.mia.(videoReader)
	return vr.DebugReadVideo(uint32(palBase + bank*16 + offset))
}

func TestBasicPalload(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, "10 DATA 10,20,30,40")
	typeLine(computer, step, "20 PALREAD 2,4,4")
	typeLine(computer, step, "30 GOTO 30")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_500_000)

	require_eq(t, 10, peekPalette(computer, 2, 4), "palette bank2 offset4")
	require_eq(t, 20, peekPalette(computer, 2, 5), "palette bank2 offset5")
	require_eq(t, 30, peekPalette(computer, 2, 6), "palette bank2 offset6")
	require_eq(t, 40, peekPalette(computer, 2, 7), "palette bank2 offset7")
}

func TestBasicChrload(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, "10 DATA 5,6,7")
	typeLine(computer, step, "20 CHRREAD 1,100,3")
	typeLine(computer, step, "30 GOTO 30")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_500_000)

	require_eq(t, 5, peekCHR(computer, 1, 100), "CHR bank1 offset100")
	require_eq(t, 6, peekCHR(computer, 1, 101), "CHR bank1 offset101")
	require_eq(t, 7, peekCHR(computer, 1, 102), "CHR bank1 offset102")
}

// BGLOAD's old interleaved (tile,attr,tile,attr,...) single-call convenience
// is retired - see docs/basic-file.md. NTREAD and ATRREAD each pull from the
// shared DATA stream independently, so the tile values are listed first (for
// NTREAD to consume) followed by the attribute values (for ATRREAD, picking
// up the stream where NTREAD left off).
func TestBasicBgload(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, "10 DATA 1,3")
	typeLine(computer, step, "20 NTREAD 3,50,2")
	typeLine(computer, step, "30 DATA 2,4")
	typeLine(computer, step, "40 ATRREAD 3,50,2")
	typeLine(computer, step, "50 GOTO 50")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_500_000)

	require_eq(t, 1, peekBGNT(computer, 3, 50), "BG table3 nametable cell50 (tile)")
	require_eq(t, 2, peekBGAttr(computer, 3, 50), "BG table3 attribute cell50")
	require_eq(t, 3, peekBGNT(computer, 3, 51), "BG table3 nametable cell51 (tile)")
	require_eq(t, 4, peekBGAttr(computer, 3, 51), "BG table3 attribute cell51")
}

// TestBasicBgchar covers every BGMODE branch (0-5) plus BGSET's +4 table
// offset, matching bgTableAndLocal (clementina-video-client
// internal/render/renderer.go) - each case's expected table/cell was computed
// independently in Python from that same formula, not by re-deriving the 6502
// code's own logic.
func TestBasicBgchar(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, "10 BGMODE 0 : BGCHAR 5,10,7,1")
	typeLine(computer, step, "20 BGMODE 1 : BGCHAR 45,3,9,2")
	typeLine(computer, step, "30 BGMODE 2 : BGCHAR 10,30,3,0")
	typeLine(computer, step, "40 BGMODE 3 : BGCHAR 125,2,4,0")
	typeLine(computer, step, "50 BGMODE 4 : BGCHAR 8,77,6,0")
	typeLine(computer, step, "60 BGMODE 5 : BGCHAR 45,30,8,0")
	typeLine(computer, step, "70 BGMODE 0 : BGSET 1 : BGCHAR 0,0,42,0")
	typeLine(computer, step, "80 GOTO 80")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_500_000)

	require_eq(t, 7, peekBGNT(computer, 0, 405), "mode0 col5,row10 -> table0 cell405")
	require_eq(t, 1, peekBGAttr(computer, 0, 405), "mode0 attr")

	require_eq(t, 9, peekBGNT(computer, 1, 125), "mode1 col45,row3 -> table1 cell125")
	require_eq(t, 2, peekBGAttr(computer, 1, 125), "mode1 attr")

	require_eq(t, 3, peekBGNT(computer, 2, 210), "mode2 col10,row30 -> table2 cell210")

	require_eq(t, 4, peekBGNT(computer, 3, 85), "mode3 col125,row2 -> table3 cell85")

	require_eq(t, 6, peekBGNT(computer, 3, 88), "mode4 col8,row77 -> table3 cell88")

	require_eq(t, 8, peekBGNT(computer, 3, 205), "mode5 col45,row30 -> table3 cell205")

	require_eq(t, 42, peekBGNT(computer, 4, 0), "mode0+BGSET1 col0,row0 -> table4 (0+4) cell0")
}

func TestBasicBgcharOutOfRange(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	// mode 0 is 40x25: col=40 is out of range (valid is 0-39).
	typeLine(computer, step, "10 BGMODE 0")
	typeLine(computer, step, "20 BGCHAR 40,0,1,0")
	typeLine(computer, step, "30 GOTO 30")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_500_000)

	if _, ok := screenContainsRow(vr, "ILLEGAL"); !ok {
		t.Fatalf("expected an ILLEGAL QUANTITY error for col=40 in 40-wide mode 0, screen:\n%s", dumpScreenString(vr))
	}
}

func TestBasicSpriteSingleFieldSetters(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, "10 SPRITE 2,1,10,20,3,0")
	typeLine(computer, step, "20 SPRTILE 2,99")
	typeLine(computer, step, "30 SPRX 2,-5")
	typeLine(computer, step, "40 SPRY 2,-100")
	typeLine(computer, step, "50 SPRCOLOR 2,7")
	typeLine(computer, step, "60 SPRFLIP 2,1,1")
	typeLine(computer, step, "70 SPRPRI 2,1")
	typeLine(computer, step, "80 GOTO 80")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	require_eq(t, 99, peekOAM(computer, 2, 0), "tile after SPRTILE")
	require_eq(t, 0xFB, peekOAM(computer, 2, 1), "xlo after SPRX -5 ($FB)")
	require_eq(t, 0x9C, peekOAM(computer, 2, 2), "ylo after SPRY -100 ($9C)")
	// attr: pal=7 (SPRCOLOR) | flipX<<5 | flipY<<6 (SPRFLIP) | priority<<4 (SPRPRI)
	// = 7 | 0x20 | 0x40 | 0x10 = 0x77
	require_eq(t, 0x77, peekOAM(computer, 2, 3), "attr after SPRCOLOR/SPRFLIP/SPRPRI")
	// ext: xhi = (-5 as 16-bit = 0xFFFB, high byte 0xFF, &3 = 3)
	//      yhi = (-100 as 16-bit = 0xFF9C, high byte 0xFF, &1 = 1, <<2 = 4)
	require_eq(t, 0x07, peekOAM(computer, 2, 4), "ext after SPRX/SPRY (xhi=3, yhi bit=4 -> 3|4=7)")
}

// SPRTILE/SPRX/SPRY/SPRCOLOR/SPRFLIP/SPRPRI must only touch their own
// field(s), never clobbering an unrelated field written by SPRITE first.
func TestBasicSpriteSingleFieldSettersPreserveOtherFields(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, "10 SPRITE 0,55,0,0,9,0")
	typeLine(computer, step, "20 SPRTILE 0,77")
	typeLine(computer, step, "30 GOTO 30")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_500_000)

	require_eq(t, 77, peekOAM(computer, 0, 0), "tile updated")
	require_eq(t, 9, peekOAM(computer, 0, 3)&0x0F, "palette (attr low nibble) untouched by SPRTILE")
}
