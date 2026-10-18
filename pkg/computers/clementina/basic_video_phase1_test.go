package clementina

import "testing"

// Phase 1 BASIC video statements (handlers in clementina-rom
// src/basic/clementina_extra.s, see docs/basic-video.md): background/sprite
// layer control, CHR bank/plane config, and palette writes. These boot the
// real ROM, type a program, run it, and read MIA render-control RAM back.

const renderCtlBase = 0x00020

func TestBasicVideoLayerToggles(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, "10 BGON")
	typeLine(computer, step, "20 SPRON")
	typeLine(computer, step, "30 GOTO 30")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_500_000)

	require_eq(t, 0x07, vr.DebugReadVideo(renderCtlBase+0x01), "LAYER_ENABLE after BGON+SPRON (overlay already on at boot | bg | sprite)")
}

func TestBasicVideoLayerToggleOff(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, "10 BGON")
	typeLine(computer, step, "20 BGOFF")
	typeLine(computer, step, "30 GOTO 30")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_500_000)

	require_eq(t, 0x02, vr.DebugReadVideo(renderCtlBase+0x01), "LAYER_ENABLE back to overlay-only after BGON;BGOFF")
}

func TestBasicVideoModeAndScroll(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, "10 BGMODE 5")
	typeLine(computer, step, "20 BGSET 1")
	typeLine(computer, step, "30 SCROLL 4660,22136")
	typeLine(computer, step, "40 GOTO 40")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	require_eq(t, 5, vr.DebugReadVideo(renderCtlBase+0x02), "BG_VIEWPORT_MODE")
	require_eq(t, 1, vr.DebugReadVideo(renderCtlBase+0x03), "BG_ACTIVE_SET")
	require_eq(t, 0x34, vr.DebugReadVideo(renderCtlBase+0x04), "SCROLL_X low (4660 = $1234)")
	require_eq(t, 0x12, vr.DebugReadVideo(renderCtlBase+0x05), "SCROLL_X high")
	require_eq(t, 0x78, vr.DebugReadVideo(renderCtlBase+0x06), "SCROLL_Y low (22136 = $5678)")
	require_eq(t, 0x56, vr.DebugReadVideo(renderCtlBase+0x07), "SCROLL_Y high")
}

func TestBasicVideoBanksAndCounts(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, "10 BGBANK 3")
	typeLine(computer, step, "20 BGALT 4")
	typeLine(computer, step, "30 SPRBANK 5")
	typeLine(computer, step, "40 SPRCOUNT 42")
	typeLine(computer, step, "50 GOTO 50")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	require_eq(t, 3, vr.DebugReadVideo(renderCtlBase+0x08), "BG_CHR_BANK")
	require_eq(t, 4, vr.DebugReadVideo(renderCtlBase+0x09), "BG_ALT_CHR_BANK")
	require_eq(t, 5, vr.DebugReadVideo(renderCtlBase+0x0C), "SPRITE_CHR_BANK")
	require_eq(t, 42, vr.DebugReadVideo(renderCtlBase+0x10), "OAM_LAST_INDEX")
}

func TestBasicVideoChrModeAndPlane(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	// Bank 0/1 already 1bpp from boot (video_init). Add bank 3, then clear bank 0.
	typeLine(computer, step, "10 CHRMODE 3,1")
	typeLine(computer, step, "20 CHRMODE 0,0")
	typeLine(computer, step, "30 CHRPLANE 1,2,1")
	typeLine(computer, step, "40 GOTO 40")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	require_eq(t, 0x0A, vr.DebugReadVideo(renderCtlBase+0x0D), "CHR_1BPP_MASK: bank1 already set, bank3 added, bank0 cleared -> %00001010")
	require_eq(t, 0x19, vr.DebugReadVideo(renderCtlBase+0x0E), "CHR_1BPP_PLANES: bg=1 | spr=2<<2 | ovl=1<<4 = %00011001")
}

// A 1bpp bank has planes 0-2 only, so CHRPLANE rejects 3 in any argument and
// leaves CHR_1BPP_PLANES as it was, although its 2-bit fields could hold 3.
func TestBasicVideoChrPlaneRejectsPlane3(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, "10 CHRPLANE 1,2,1")
	typeLine(computer, step, "20 CHRPLANE 0,0,3")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	if _, ok := screenContainsRow(vr, "ILLEGAL QUANTITY"); !ok {
		t.Fatalf("expected an ILLEGAL QUANTITY error for overlay plane 3, screen:\n%s", dumpScreenString(vr))
	}
	require_eq(t, 0x19, vr.DebugReadVideo(renderCtlBase+0x0E), "CHR_1BPP_PLANES kept after overlay plane 3")

	typeLine(computer, step, "CHRPLANE 3,0,0")
	typeLine(computer, step, "CHRPLANE 0,3,0")
	editorTickN(computer, step, 1_000_000)
	require_eq(t, 0x19, vr.DebugReadVideo(renderCtlBase+0x0E), "CHR_1BPP_PLANES kept after background and sprite plane 3")
}

func TestBasicVideoPalette(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	// bank 5, index 3, r=31,g=63,b=17 -> hi=(31<<3)|(63>>3)=0xF8|0x07=0xFF, lo=((63&7)<<5)|17=0xE0|0x11=0xF1
	typeLine(computer, step, "10 PALETTE 5,3,31,63,17")
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_500_000)

	addr := uint32(0x00100 + 5*16 + 3*2)
	require_eq(t, 0xF1, vr.DebugReadVideo(addr), "palette bank5 color3 low byte")
	require_eq(t, 0xFF, vr.DebugReadVideo(addr+1), "palette bank5 color3 high byte")
}

func TestBasicVideoOnOff(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	typeLine(computer, step, "10 VIDOFF")
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_500_000)

	require_eq(t, 0, vr.DebugReadVideo(renderCtlBase+0x00), "VIDEO_MODE after VIDOFF")
}
