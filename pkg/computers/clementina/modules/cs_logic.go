package modules

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/fran150/clementina-6502/pkg/components/other/decoders"
	"github.com/fran150/clementina-6502/pkg/components/other/gates"
)

// This component from the Clementina 6502 computer implements the logic for the CS lines
// It creates the following map:
//
// $E000 - $FFFF High RAM (32 KB banked in 8 KB windows if enabled by pico)
// $C000 - $DFFF IO (8 slots of 1 KB each)
// $8000 - $BFFF Extended RAM (512 KB banked in 16 KB windows)
// $0000 - $7FFF Base RAM (32 KB)
//
// It follows the "CS Logic" sheet of the schematic:
//
//   - U8 (74AC138) decodes A13-A15 into 8 KB blocks. ~Y0 and ~Y1 feed U7A, ~Y3 feeds U9C; ~Y2 is unused.
//   - U6 (74AC138) decodes A10-A12 into the eight 1 KB I/O slots. It is enabled straight from the
//     address: ~E0 = A13, ~E1 = ~A15 (from U9D), E2 = A14.
//   - U7A (74AC08) ANDs ~Y0 and ~Y1 into the extended RAM chip select.
//   - U9C and U9D are two gates of the 74AC00 shared with the OE / RW / PHI2 sync sheet, each wired
//     as an inverter: MIACS = NAND(~Y3, ~Y3) and ~A15 = NAND(A15, A15).
type ClementinaCSLogic struct {
	// Exponsed line connectors
	a1 [6]buses.LineConnector // A10 to A15 address lines

	// Exposed lines
	ioCS    buses.Bus[uint8] // I/O output enable
	exRAMCS buses.Line       // External RAM enable (512 KB banked in 16 KB windows)
	miaCS   buses.Line       // High RAM enable (32 KB banked in 8 KB windows)

	// Internal components
	addressDecoder       components.Decoder74HC138 // U8: slices the upper 32 KB in 8 KB blocks
	ioDecoder            components.Decoder74HC138 // U6: slices I/O space in 8 slots of 1KB each
	andGate              components.LogicGateArray // U7 (74AC08), gate A
	nand                 components.LogicGateArray // U9 (74AC00), gates C and D; shared with the sync sheet
	addressDecoderOutput buses.Bus[uint8]
	notA15               buses.Line // ~A15, U9D output to U6 ~E1
	ground               buses.Line // Lines connected to ground (will always be low)
}

// NewClementinaCSLogic creates a new Clementina CS Logic component.
//
// Parameters:
//   - nand: U9, the 74AC00 whose gates C and D this sheet uses. Gates A and B belong to the
//     OE / RW / PHI2 sync sheet, which receives the same chip.
func NewClementinaCSLogic(nand components.LogicGateArray) *ClementinaCSLogic {
	csLogic := &ClementinaCSLogic{}

	// Create internal components
	csLogic.addressDecoder = decoders.NewDecoder74AC138()
	csLogic.ioDecoder = decoders.NewDecoder74AC138()
	csLogic.andGate = gates.NewAnd74AC08()
	csLogic.nand = nand

	csLogic.addressDecoderOutput = buses.New8BitStandaloneBus() // Output bus for the address decoder
	csLogic.notA15 = buses.NewStandaloneLine(true)
	csLogic.ground = buses.NewStandaloneLine(false) // Ground line, always low

	// Expose connectors for address lines A10 to A15. A13 to A15 each reach two chips.
	csLogic.a1[0] = csLogic.ioDecoder.APin(0)
	csLogic.a1[1] = csLogic.ioDecoder.APin(1)
	csLogic.a1[2] = csLogic.ioDecoder.APin(2)
	csLogic.a1[3] = newNet(csLogic.addressDecoder.APin(0), csLogic.ioDecoder.EPin(0))
	csLogic.a1[4] = newNet(csLogic.addressDecoder.APin(1), csLogic.ioDecoder.EPin(2))
	csLogic.a1[5] = newNet(csLogic.addressDecoder.EPin(2), csLogic.nand.APin(3), csLogic.nand.BPin(3))

	// Address decoder connections
	csLogic.addressDecoder.APin(2).Connect(csLogic.ground)
	csLogic.addressDecoder.EPin(0).Connect(csLogic.ground)
	csLogic.addressDecoder.EPin(1).Connect(csLogic.ground)
	csLogic.addressDecoder.YPin().Connect(csLogic.addressDecoderOutput)

	// Map gates to address decoder output
	csLogic.andGate.APin(0).Connect(csLogic.addressDecoderOutput.GetBusLine(0))
	csLogic.andGate.BPin(0).Connect(csLogic.addressDecoderOutput.GetBusLine(1))
	csLogic.nand.APin(2).Connect(csLogic.addressDecoderOutput.GetBusLine(3))
	csLogic.nand.BPin(2).Connect(csLogic.addressDecoderOutput.GetBusLine(3))

	// ~A15 enables the io decoder
	csLogic.nand.YPin(3).Connect(csLogic.notA15)
	csLogic.ioDecoder.EPin(1).Connect(csLogic.notA15)

	csLogic.ioCS = buses.New8BitStandaloneBus()
	csLogic.exRAMCS = buses.NewStandaloneLine(true)
	csLogic.miaCS = buses.NewStandaloneLine(false)

	// Connect output lines
	csLogic.ioDecoder.YPin().Connect(csLogic.ioCS)
	csLogic.andGate.YPin(0).Connect(csLogic.exRAMCS)
	csLogic.nand.YPin(2).Connect(csLogic.miaCS)

	return csLogic
}

// A1 returns the connector for the specified address line (A10 to A15).
//
// Parameters:
//   - index: The address line index (0-5 corresponding to A10-A15)
//
// Returns:
//   - The line connector for the specified address line, or nil if index is out of range
func (circuit *ClementinaCSLogic) A1(index int) buses.LineConnector {
	if index >= 0 && index < len(circuit.a1) {
		return circuit.a1[index]
	}
	return nil
}

// IOCS returns the connector for the I/O chip select bus.
// Each line can be used to map a device in one of the 8 I/O slots.
// Each device will have 1K of available address space.
func (circuit *ClementinaCSLogic) IOCS() buses.Bus[uint8] {
	return circuit.ioCS
}

// ExRAMCS returns the connector for the external RAM chip select line.
// The extended RAM maps to a 512 KB space banked in 16 KB windows.
func (circuit *ClementinaCSLogic) ExRAMCS() buses.Line {
	return circuit.exRAMCS
}

// MiaCS returns the connector for the MIA CS line
// This line is used to enable or disable the MIA chip (PICO).
func (circuit *ClementinaCSLogic) MiaCS() buses.Line {
	return circuit.miaCS
}

// Tick executes one emulation step.
// U9 goes after U8, whose ~Y3 it inverts, and before U6, which it enables through ~A15.
//
// Parameters:
//   - stepContext: The current step context for the emulation cycle
func (circuit *ClementinaCSLogic) Tick(stepContext *common.StepContext) {
	circuit.addressDecoder.Tick(stepContext)
	circuit.nand.Tick(stepContext)
	circuit.ioDecoder.Tick(stepContext)
	circuit.andGate.Tick(stepContext)
}
