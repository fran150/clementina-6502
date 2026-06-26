package clementina

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Phase 5 editor state (clementina-rom kernel.inc KVARS tail) and the pen.
const (
	addrTextAttr    = 0x0302 // TEXT_ATTR: overlay attribute pen
	addrEditAttrBuf = 0x0380 // EDIT_ATTR_BUF: harvested per-cell attributes
	addrEditMode    = 0x03D0 // EDIT_MODE: glyph mode 0/1/2
	addrEditPaint   = 0x03D1 // EDIT_PAINT: nonzero while Paint mode active
	addrEditCmdPend = 0x03D2 // EDIT_CMD_PENDING
	keyEsc          = 0x1B

	ovNT   = 0x10080 // MIA overlay nametable base (40x25 tile codes)
	ovAttr = 0x10468 // MIA overlay attribute base (40x25 attr bytes)
)

type videoReader interface{ DebugReadVideo(uint32) uint8 }

// ntCell reads the overlay tile code at (row, col).
func ntCell(vr videoReader, row, col int) uint8 {
	return vr.DebugReadVideo(uint32(ovNT + row*40 + col))
}

func attrCell(vr videoReader, row, col int) uint8 {
	return vr.DebugReadVideo(uint32(ovAttr + row*40 + col))
}

// TestPhase5NoStrayLineFeedGlyph guards the regression where STRPRT_STYLED drew
// the LF ($0A) half of CR/LF as a raw glyph at column 0 of every message line
// instead of letting chrout ignore it (it rendered as a blank, shifting all
// message/READY text one column right). After boot no row may start with $0A,
// READY. must sit at column 0, and the prompt cursor must be at column 0.
func TestPhase5NoStrayLineFeedGlyph(t *testing.T) {
	computer, _ := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	require.Equal(t, uint8(0), peek(computer, addrCursorX), "prompt cursor should be at column 0")

	foundReady := false
	for r := 0; r < 25; r++ {
		require.NotEqualf(t, uint8(0x0A), ntCell(vr, r, 0), "row %d must not start with a stray LF glyph", r)
		if ntCell(vr, r, 0) == 'R' && ntCell(vr, r, 1) == 'E' && ntCell(vr, r, 5) == '.' {
			foundReady = true
		}
	}
	require.True(t, foundReady, "READY. should appear at column 0")
}

// TestPhase5GlyphModeCyclesAndOffsetsTile: ESC M cycles the glyph mode, and a
// printable typed in mode 1 is stored as tile (ascii + $60). 'A' ($41) -> $A1.
func TestPhase5GlyphModeCyclesAndOffsetsTile(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	require.Equal(t, uint8(0), peek(computer, addrEditMode), "mode starts at 0")

	injectKeys(computer, keyEsc, 'M') // cycle 0 -> 1
	editorTickN(computer, step, 200_000)
	require.Equal(t, uint8(1), peek(computer, addrEditMode), "ESC M should cycle to mode 1")

	injectKeys(computer, 'A', keyCR) // 'A' in mode 1 -> tile $A1
	editorTickN(computer, step, 400_000)

	require.Equal(t, uint8(1), peek(computer, addrEditLen), "one glyph harvested")
	require.Equal(t, uint8(0xA1), peek(computer, addrEditBuf+0), "'A' in mode 1 -> tile $A1")
}

// TestPhase5HighTileIsGlyphNotCursorMove is the key regression: '1' ($31) in
// mode 1 makes tile $91, which equals the cursor-up control code. The editor
// must DRAW it (cursor advances, row unchanged), not move the cursor up, and
// harvest it as $91.
func TestPhase5HighTileIsGlyphNotCursorMove(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	x0 := peek(computer, addrCursorX)
	y0 := peek(computer, addrCursorY)

	injectKeys(computer, keyEsc, 'M') // mode 1
	editorTickN(computer, step, 200_000)

	injectKeys(computer, '1') // -> tile $91 (== keyCursorUp)
	editorTickN(computer, step, 200_000)

	require.Equal(t, y0, peek(computer, addrCursorY), "tile $91 must not act as cursor-up")
	require.Equal(t, x0+1, peek(computer, addrCursorX), "tile $91 must advance the cursor like a glyph")

	injectKeys(computer, keyCR)
	editorTickN(computer, step, 200_000)
	require.Equal(t, uint8(0x91), peek(computer, addrEditBuf+0), "harvested as glyph $91")
}

// TestPhase5EscColorSetsPalette: ESC 5 sets palette 5; ESC F sets palette 15;
// the flip/reverse bits are preserved.
func TestPhase5EscColorSetsPalette(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	injectKeys(computer, keyEsc, '5')
	editorTickN(computer, step, 200_000)
	require.Equal(t, uint8(0x05), peek(computer, addrTextAttr), "ESC 5 -> palette 5")

	injectKeys(computer, keyEsc, 'F')
	editorTickN(computer, step, 200_000)
	require.Equal(t, uint8(0x0F), peek(computer, addrTextAttr), "ESC F -> palette 15")
}

// TestPhase5EscReverseFlipToggles: R/H/V toggle the reverse/flip-H/flip-V bits;
// SPACE resets the pen and glyph mode.
func TestPhase5EscReverseFlipToggles(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	injectKeys(computer, keyEsc, 'R', keyEsc, 'H') // reverse + flip-H
	editorTickN(computer, step, 200_000)
	require.Equal(t, uint8(0x90), peek(computer, addrTextAttr), "ESC R + ESC H -> bits 7 and 4")

	injectKeys(computer, keyEsc, 'M', keyEsc, ' ') // mode 1 then reset
	editorTickN(computer, step, 200_000)
	require.Equal(t, uint8(0x00), peek(computer, addrTextAttr), "ESC SPACE resets the pen")
	require.Equal(t, uint8(0x00), peek(computer, addrEditMode), "ESC SPACE resets the glyph mode")
}

// TestPhase5PaintModeEnterExit: ESC P enters Paint mode; ESC leaves it.
func TestPhase5PaintModeEnterExit(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	injectKeys(computer, keyEsc, 'P')
	editorTickN(computer, step, 200_000)
	require.NotEqual(t, uint8(0), peek(computer, addrEditPaint), "ESC P enters Paint mode")

	injectKeys(computer, keyEsc)
	editorTickN(computer, step, 200_000)
	require.Equal(t, uint8(0), peek(computer, addrEditPaint), "ESC exits Paint mode")
}

// TestPhase5PaintModeStampsAttribute: type AB (default attr), pick palette 3,
// enter Paint, move back over A, SPACE-stamp A and B, exit, RETURN. The glyphs
// are unchanged but their attributes are repainted to palette 3, and the harvest
// captures it.
func TestPhase5PaintModeStampsAttribute(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	injectKeys(computer, 'A', 'B') // two cells with the default pen (attr 0)
	editorTickN(computer, step, 200_000)

	injectKeys(computer, keyEsc, '3') // pen -> palette 3
	editorTickN(computer, step, 200_000)

	injectKeys(computer, keyEsc, 'P') // enter Paint mode
	editorTickN(computer, step, 200_000)

	injectKeys(computer, keyCursorLt, keyCursorLt) // back over 'A'
	editorTickN(computer, step, 200_000)

	injectKeys(computer, ' ', ' ') // stamp A and B, each advancing right
	editorTickN(computer, step, 200_000)

	injectKeys(computer, keyEsc, keyCR) // exit Paint, harvest the line
	editorTickN(computer, step, 400_000)

	require.Equal(t, uint8('A'), peek(computer, addrEditBuf+0), "glyph A unchanged by paint")
	require.Equal(t, uint8('B'), peek(computer, addrEditBuf+1), "glyph B unchanged by paint")
	require.Equal(t, uint8(0x03), peek(computer, addrEditAttrBuf+0), "A repainted to palette 3")
	require.Equal(t, uint8(0x03), peek(computer, addrEditAttrBuf+1), "B repainted to palette 3")
}

// TestPhase5StyledHarvestCapturesAttribute: typing in palette 5 captures the
// attribute alongside the glyph (EDIT_ATTR_BUF parallels EDIT_BUF).
func TestPhase5StyledHarvestCapturesAttribute(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	injectKeys(computer, keyEsc, '5') // palette 5
	editorTickN(computer, step, 200_000)

	injectKeys(computer, 'Z', keyCR)
	editorTickN(computer, step, 400_000)

	require.Equal(t, uint8('Z'), peek(computer, addrEditBuf+0), "glyph harvested")
	require.Equal(t, uint8(0x05), peek(computer, addrEditAttrBuf+0), "attribute (palette 5) harvested")
}

// TestPhase5BackspaceShiftsAttributesWithGlyphs guards the regression where the
// editor closed the nametable gap but left the attribute plane in place. Deleting
// the last red char before orange text must pull both orange glyphs and orange
// attrs left together.
func TestPhase5BackspaceShiftsAttributesWithGlyphs(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	injectKeys(computer,
		keyEsc, '1', 'A', 'B', 'C',
		keyEsc, '2', 'D', 'E', 'F',
		keyCursorLt, keyCursorLt, keyCursorLt,
		keyBackspace,
		keyCR,
	)
	editorTickN(computer, step, 1_000_000)

	require.Equal(t, "ABDEF", harvestedLine(computer), "backspace should remove C and close the glyph gap")
	wantAttrs := []uint8{0x01, 0x01, 0x02, 0x02, 0x02}
	for i, want := range wantAttrs {
		require.Equalf(t, want, peek(computer, addrEditAttrBuf+uint32(i)), "EDIT_ATTR_BUF[%d]", i)
	}
}

// TestPhase6ProgramLiteralSidecarPrintsStyledAttrs stores a numbered BASIC line
// whose string literal has per-letter editor attributes, then RUNs it. The typed
// source line remains on screen, so success is seeing a second styled FRAN: one
// occurrence from source entry and one from BASIC PRINT rehydrating the sidecar.
func TestPhase6ProgramLiteralSidecarPrintsStyledAttrs(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	injectKeys(computer, []byte(`10 PRINT "`)...)
	injectKeys(computer,
		keyEsc, '1', 'F',
		keyEsc, '2', 'R',
		keyEsc, '3', 'A',
		keyEsc, '4', 'N',
	)
	injectKeys(computer, '"', keyCR)
	editorTickN(computer, step, 900_000)

	injectKeys(computer, []byte("RUN")...)
	injectKeys(computer, keyCR)
	editorTickN(computer, step, 2_000_000)

	wantText := []uint8{'F', 'R', 'A', 'N'}
	wantAttrs := []uint8{0x01, 0x02, 0x03, 0x04}
	matches := 0
	for row := 0; row < 25; row++ {
		for col := 0; col <= 40-len(wantText); col++ {
			ok := true
			for i := range wantText {
				if ntCell(vr, row, col+i) != wantText[i] || attrCell(vr, row, col+i) != wantAttrs[i] {
					ok = false
					break
				}
			}
			if ok {
				matches++
			}
		}
	}
	require.GreaterOrEqual(t, matches, 2, "source entry and RUN output should both show styled FRAN")
}

// TestPhase6ListSkipsSidecarAndShowsStyledLiteral stores a styled literal plus a
// following line, then LISTs the program. LIST must not walk into the sidecar as
// if it were another line, and the listed literal should retain its attrs.
func TestPhase6ListSkipsSidecarAndShowsStyledLiteral(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	typeLine := func(bytes ...uint8) {
		injectKeys(computer, bytes...)
		injectKeys(computer, keyCR)
		editorTickN(computer, step, 900_000)
	}

	injectKeys(computer, []byte(`10 PRINT "`)...)
	injectKeys(computer,
		keyEsc, '1', 'F',
		keyEsc, '2', 'R',
		keyEsc, '3', 'A',
		keyEsc, '4', 'N',
	)
	typeLine('"')
	typeLine([]byte("20 GOTO 10")...)
	typeLine([]byte("LIST")...)
	editorTickN(computer, step, 2_000_000)

	wantText := []uint8{'F', 'R', 'A', 'N'}
	wantAttrs := []uint8{0x01, 0x02, 0x03, 0x04}
	styledMatches := 0
	readyRows := 0
	line20Rows := 0
	for row := 0; row < 25; row++ {
		for col := 0; col <= 40-len(wantText); col++ {
			styled := true
			for i := range wantText {
				if ntCell(vr, row, col+i) != wantText[i] || attrCell(vr, row, col+i) != wantAttrs[i] {
					styled = false
					break
				}
			}
			if styled {
				styledMatches++
			}
		}
		for col := 0; col <= 40-len("READY."); col++ {
			if ntCell(vr, row, col+0) == 'R' && ntCell(vr, row, col+1) == 'E' && ntCell(vr, row, col+5) == '.' {
				readyRows++
			}
		}
		for col := 0; col <= 40-len("20 GOTO"); col++ {
			if ntCell(vr, row, col+0) == '2' && ntCell(vr, row, col+1) == '0' && ntCell(vr, row, col+3) == 'G' && ntCell(vr, row, col+6) == 'O' {
				line20Rows++
			}
		}
	}

	require.GreaterOrEqual(t, styledMatches, 2, "source entry and LIST output should both show styled FRAN")
	require.GreaterOrEqual(t, readyRows, 1, "LIST should finish and return to READY")
	require.LessOrEqual(t, line20Rows, 2, "LIST should not repeat line 20 indefinitely")
}

// TestPhase6ProgramLiteralStoresGlyphModeTiles guards the BASIC line-input path:
// mode 1/2 glyphs are raw tile bytes, not terminal text. The stored program
// literal must keep those tile bytes so LIST and RUN reproduce the same glyphs.
func TestPhase6ProgramLiteralStoresGlyphModeTiles(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	typeLine := func(bytes ...uint8) {
		injectKeys(computer, bytes...)
		injectKeys(computer, keyCR)
		editorTickN(computer, step, 900_000)
	}
	countTileRun := func(want []uint8) int {
		matches := 0
		for row := 0; row < 25; row++ {
			for col := 0; col <= 40-len(want); col++ {
				ok := true
				for i := range want {
					if ntCell(vr, row, col+i) != want[i] {
						ok = false
						break
					}
				}
				if ok {
					matches++
				}
			}
		}
		return matches
	}

	injectKeys(computer, []byte(`10 PRINT "`)...)
	injectKeys(computer,
		keyEsc, 'M', 'A', // mode 1: 'A' -> $A1
		keyEsc, 'M', 'B', // mode 2: 'B' -> $02
		'M',         // mode 2: 'M' -> $0D
		keyEsc, ' ', // reset mode before the closing quote
	)
	typeLine('"')
	typeLine([]byte("LIST")...)
	typeLine([]byte("RUN")...)
	editorTickN(computer, step, 2_000_000)

	require.GreaterOrEqual(t, countTileRun([]uint8{0xA1, 0x02, 0x0D}), 3,
		"source entry, LIST output, and RUN output should all show mode glyph tiles")
}

func TestPhase6Chr13StillPrintsNewline(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	vr := computer.chips.mia.(videoReader)

	typeLine := func(s string) {
		injectKeys(computer, []byte(s)...)
		injectKeys(computer, keyCR)
		editorTickN(computer, step, 900_000)
	}

	typeLine(`10 PRINT "A"+CHR$(13)+"B"`)
	typeLine("RUN")
	editorTickN(computer, step, 2_000_000)

	found := false
	for row := 0; row < 24; row++ {
		if ntCell(vr, row, 0) == 'A' && ntCell(vr, row+1, 0) == 'B' {
			found = true
			break
		}
	}
	require.True(t, found, "CHR$(13) should still move the next character to column 0 on the next row")
}

func TestPhase6ReadSkipsStyledSidecars(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine := func(bytes ...uint8) {
		injectKeys(computer, bytes...)
		injectKeys(computer, keyCR)
		editorTickN(computer, step, 900_000)
	}

	injectKeys(computer, []byte(`5 PRINT "`)...)
	injectKeys(computer, keyEsc, '3', 'X')
	typeLine('"')
	injectKeys(computer, []byte(`10 DATA "`)...)
	injectKeys(computer,
		keyEsc, '1', 'F',
		keyEsc, '2', 'R',
		keyEsc, '3', 'A',
		keyEsc, '4', 'N',
	)
	typeLine('"')
	typeLine([]byte("20 DATA 42")...)
	typeLine([]byte("30 READ A$,N")...)
	typeLine([]byte("40 POKE 752,N")...)
	typeLine([]byte("RUN")...)
	editorTickN(computer, step, 3_000_000)

	require.Equal(t, uint8(42), peek(computer, 0x02F0),
		"READ should skip styled sidecars while scanning DATA lines")
}

// TestPhase5StyledPrintHighTileKeepsFlow drives the BASIC PRINT path for a styled
// string that carries a high tile ($91, via CHR$(145)). STRPRT_STYLED must draw
// it through the raw-glyph path (MONCOUT_GLYPH) without hanging or corrupting the
// loop registers, so the program runs to completion. Observable via base RAM:
// LEN is correct and the POKE after PRINT executes.
func TestPhase5StyledPrintHighTileKeepsFlow(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine := func(s string) {
		injectKeys(computer, []byte(s)...)
		injectKeys(computer, keyCR)
		editorTickN(computer, step, 600_000)
	}
	typeLine(`10 A$=CHR$(145)+"OK"`)
	typeLine(`20 POKE 752,LEN(A$)`)
	typeLine(`30 PRINT A$`)
	typeLine(`40 POKE 753,7`)
	typeLine(`RUN`)
	editorTickN(computer, step, 3_000_000)

	require.Equal(t, uint8(3), peek(computer, 0x02F0), `LEN(CHR$(145)+"OK") should be 3`)
	require.Equal(t, uint8(7), peek(computer, 0x02F1),
		"flow must continue past PRINT of a high-tile styled string")
}
