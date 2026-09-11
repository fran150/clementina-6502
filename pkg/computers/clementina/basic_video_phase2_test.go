package clementina

import "testing"

// Phase 2 BASIC video statements: OAMLOAD (bulk sprite loading from DATA) and
// SPRITE (full per-sprite setup), handlers in clementina-rom
// src/basic/clementina_extra.s. See docs/basic-video.md.

const oamBase = 0x10850

func peekOAM(computer *ClementinaComputer, n int, field int) uint8 {
	vr := computer.chips.mia.(videoReader)
	return vr.DebugReadVideo(uint32(oamBase + n*5 + field))
}

func TestBasicSprite(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	// n=3, tile=65, x=-2, y=200 (Y's valid range is -256..255), pal=5,
	// flags: priority(bit1)+flipY(bit3) = 10
	typeLine(computer, step, "10 SPRITE 3,65,-2,200,5,10")
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_500_000)

	require_eq(t, 65, peekOAM(computer, 3, 0), "tile")
	require_eq(t, 0xFE, peekOAM(computer, 3, 1), "xlo (-2 = $FE)")
	require_eq(t, 0xC8, peekOAM(computer, 3, 2), "ylo (200 = $C8)")
	// attr: pal=5 | priority<<4 | flipY<<6 = 5 | 0x10 | 0x40 = 0x55
	require_eq(t, 0x55, peekOAM(computer, 3, 3), "attr (pal 5, priority, flipY)")
	// ext: xhi (0xFFFE>>8 & 3 = 3) | yhi(200>>8=0)<<2 | disable(0)<<3 = 3
	require_eq(t, 0x03, peekOAM(computer, 3, 4), "ext (xhi=3, yhi=0)")
}

func TestBasicSpriteDisableFlag(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, "10 SPRITE 0,1,0,0,0,1")
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_500_000)

	require_eq(t, 0x08, peekOAM(computer, 0, 4), "ext bit3 (disable) set")
	require_eq(t, 0x00, peekOAM(computer, 0, 3), "attr all zero (no priority/flip)")
}

func TestBasicSpriteOutOfRange(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, "10 SPRITE 0,1,512,0,0,0")
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_500_000)

	if _, ok := screenContainsRow(vr, "ILLEGAL QUANTITY"); !ok {
		if _, ok2 := screenContainsRow(vr, "ILLEGAL"); !ok2 {
			t.Fatalf("expected an ILLEGAL QUANTITY error for x=512 (out of -512..511), screen:\n%s", dumpScreenString(vr))
		}
	}
}

func dumpScreenString(vr videoReader) string {
	s := ""
	for row := 0; row < 25; row++ {
		for col := 0; col < 40; col++ {
			s += string(rune(vr.DebugReadVideo(uint32(ovNT + row*40 + col))))
		}
		s += "\n"
	}
	return s
}

func TestBasicOamload(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, "10 DATA 10,1,2,3,4")
	typeLine(computer, step, "20 DATA 20,6,7,8,9")
	typeLine(computer, step, "30 OAMLOAD 5,2")
	typeLine(computer, step, "40 GOTO 40")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_500_000)

	require_eq(t, 10, peekOAM(computer, 5, 0), "sprite5 tile")
	require_eq(t, 1, peekOAM(computer, 5, 1), "sprite5 xlo")
	require_eq(t, 2, peekOAM(computer, 5, 2), "sprite5 ylo")
	require_eq(t, 3, peekOAM(computer, 5, 3), "sprite5 attr")
	require_eq(t, 4, peekOAM(computer, 5, 4), "sprite5 ext")
	require_eq(t, 20, peekOAM(computer, 6, 0), "sprite6 tile")
	require_eq(t, 8, peekOAM(computer, 6, 3), "sprite6 attr")
	require_eq(t, 9, peekOAM(computer, 6, 4), "sprite6 ext")
}

// OAMLOAD must leave the real DATA pointer correctly advanced for a normal
// READ afterward, and must not leak its internal Z9 scratch variable's value.
func TestBasicOamloadThenRead(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, "10 DATA 1,2,3,4,5,99")
	typeLine(computer, step, "20 OAMLOAD 0,1")
	typeLine(computer, step, "30 READ A")
	typeLine(computer, step, "40 PRINT A")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_500_000)

	vr := computer.chips.mia.(videoReader)
	if _, ok := screenContainsRow(vr, "99"); !ok {
		t.Fatalf("expected READ A after OAMLOAD to pick up 99 (the DATA item right after OAMLOAD's 5), screen:\n%s", dumpScreenString(vr))
	}
}
