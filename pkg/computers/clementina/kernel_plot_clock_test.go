package clementina

import "testing"

// KERN_PLOT, KERN_RDTIM and KERN_SETTIM are kernel jump table entries
// (clementina-rom docs/memory-map.md §6). KERN_RDTIM copies MIA's clock
// snapshot to the fixed CPU address CLOCK_SNAPSHOT ($02F8-$02FF).
const (
	addrKernPlot      = 0x042D
	addrKernRdtim     = 0x0430
	addrKernSettim    = 0x0433
	addrClockSnapshot = 0x02F8
	addrTestCode      = 0x02C0 // 704: machine code poked by these tests
)

// pokeCode writes a machine-code routine at addrTestCode.
func pokeCode(computer *ClementinaComputer, code []uint8) {
	for i, b := range code {
		computer.chips.baseram.Poke(uint16(addrTestCode+i), b)
	}
}

// TestKernelPlotReadsAndMovesTheCursor moves the cursor with KERN_PLOT, reads
// it back, and checks that an out-of-range row is refused with C=1.
func TestKernelPlotReadsAndMovesTheCursor(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	lo, hi := uint8(addrKernPlot&0xFF), uint8(addrKernPlot>>8)
	pokeCode(computer, []uint8{
		0x18,    // CLC: move
		0xA2, 5, // LDX #5 (row)
		0xA0, 10, // LDY #10 (column)
		0x20, lo, hi, // JSR KERN_PLOT
		0x38,         // SEC: read
		0x20, lo, hi, // JSR KERN_PLOT
		0x8E, 0x80, 0x02, // STX $0280
		0x8C, 0x81, 0x02, // STY $0281
		0xA2, 25, // LDX #25: one row past the screen
		0xA0, 0, // LDY #0
		0x18,         // CLC: move
		0x20, lo, hi, // JSR KERN_PLOT
		0xA9, 0, // LDA #0
		0x2A,             // ROL A: A = carry
		0x8D, 0x82, 0x02, // STA $0282
		0x60, // RTS
	})

	// The program spins after SYS so no READY prompt moves the cursor.
	typeLine(computer, step, "10 SYS 704")
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_000_000)

	require_eq(t, 5, peek(computer, 0x0280), "KERN_PLOT read returns the row in X")
	require_eq(t, 10, peek(computer, 0x0281), "KERN_PLOT read returns the column in Y")
	require_eq(t, 1, peek(computer, 0x0282), "out-of-range KERN_PLOT returns C=1")
	require_eq(t, 10, peek(computer, addrCursorX), "CURSOR_X after KERN_PLOT")
	require_eq(t, 5, peek(computer, addrCursorY), "CURSOR_Y after KERN_PLOT (unchanged by the refused move)")
}

// TestKernelRdtimFillsTheClockSnapshot calls KERN_RDTIM from BASIC with SYS
// and checks the snapshot's version byte and that its milliseconds advance.
func TestKernelRdtimFillsTheClockSnapshot(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	for _, s := range []string{
		"10 SYS 1072:GOSUB 100:A=M",
		"20 DELAY 200",
		"30 SYS 1072:GOSUB 100:B=M",
		"40 POKE 640,PEEK(767)",
		"50 IF B-A>=190 THEN POKE 641,1",
		"60 GOTO 60",
		"100 M=PEEK(760)+256*PEEK(761)",
		"110 M=M+65536*PEEK(762):RETURN",
	} {
		typeLine(computer, step, s)
	}
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 3_000_000)

	require_eq(t, 1, peek(computer, 0x0280), "CLOCK_VERSION after KERN_RDTIM")
	require_eq(t, 1, peek(computer, 0x0281), "CLOCK_MS advanced across DELAY 200")
	if peek(computer, addrClockSnapshot+7) != 1 {
		t.Fatalf("snapshot not at $%04X", addrClockSnapshot)
	}
}

// TestKernelSettimSetsTheSixtyHertzClock sets MIA's 60 Hz clock to 10000
// through KERN_SETTIM and reads it back through BASIC's TI.
func TestKernelSettimSetsTheSixtyHertzClock(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	pokeCode(computer, []uint8{
		0xA9, 0x10, // LDA #$10 (10000 = $002710)
		0xA2, 0x27, // LDX #$27
		0xA0, 0x00, // LDY #$00
		0x4C, uint8(addrKernSettim & 0xFF), uint8(addrKernSettim >> 8), // JMP KERN_SETTIM
	})
	for _, s := range []string{
		"10 SYS 704",
		"20 T=TI",
		"30 IF T<10000 OR T>10060 THEN POKE 640,2",
		"40 POKE 641,123",
		"50 GOTO 50",
	} {
		typeLine(computer, step, s)
	}
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_000_000)

	require_eq(t, 0, peek(computer, 0x0280), "TI right after KERN_SETTIM 10000")
	require_eq(t, 123, peek(computer, 0x0281), "program completed")
}
