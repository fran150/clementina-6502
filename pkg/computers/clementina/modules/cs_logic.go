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
type ClementinaCSLogic struct {
	// Exponsed line connectors
	a1 [6]buses.LineConnector // A10 to A15 address lines

	// Exposed lines
	ioCS    buses.Bus[uint8] // I/O output enable
	exRAMCS buses.Line       // External RAM enable (512 KB banked in 16 KB windows)
	miaCS   buses.Line       // High RAM enable (32 KB banked in 8 KB windows)

	// Internal components
	addressDecoder       components.Decoder74HC138 // Address decoder to slice RAM space in 16 KB windows
	ioDecoder            components.Decoder74HC138 // I/O decoder to slice I/O space in 8 slots of 1KB each
	andGate              components.LogicGateArray
	inverter             components.InverterArray
	addressDecoderOutput buses.Bus[uint8]
	vcc                  buses.Line // Lines connected to VCC (will always be high)
	ground               buses.Line // Lines connected to ground (will always be low)
}

// NewClementinaCSLogic creates a new Clementina CS Logic component.
func NewClementinaCSLogic() *ClementinaCSLogic {
	csLogic := &ClementinaCSLogic{}

	// Create internal components
	csLogic.addressDecoder = decoders.NewDecoder74HC138()
	csLogic.ioDecoder = decoders.NewDecoder74HC138()
	csLogic.andGate = gates.NewAnd74HC08()
	csLogic.inverter = gates.NewNot74HC04()

	csLogic.addressDecoderOutput = buses.New8BitStandaloneBus() // Output bus for the address decoder
	csLogic.vcc = buses.NewStandaloneLine(true)                 // VCC line, always high
	csLogic.ground = buses.NewStandaloneLine(false)             // Ground line, always low

	// Expose connectors for address lines A10 to A15
	csLogic.a1[0] = csLogic.ioDecoder.APin(0)
	csLogic.a1[1] = csLogic.ioDecoder.APin(1)
	csLogic.a1[2] = csLogic.ioDecoder.APin(2)
	csLogic.a1[3] = csLogic.addressDecoder.APin(0)
	csLogic.a1[4] = csLogic.addressDecoder.APin(1)
	csLogic.a1[5] = csLogic.addressDecoder.EPin(2)

	// Address decoder connections
	csLogic.addressDecoder.APin(2).Connect(csLogic.ground)
	csLogic.addressDecoder.EPin(0).Connect(csLogic.ground)
	csLogic.addressDecoder.EPin(1).Connect(csLogic.ground)
	csLogic.addressDecoder.YPin().Connect(csLogic.addressDecoderOutput)

	// Map gates to address decoder output
	csLogic.andGate.APin(0).Connect(csLogic.addressDecoderOutput.GetBusLine(0))
	csLogic.andGate.BPin(0).Connect(csLogic.addressDecoderOutput.GetBusLine(1))
	csLogic.ioDecoder.EPin(0).Connect(csLogic.addressDecoderOutput.GetBusLine(2))
	csLogic.ioDecoder.EPin(1).Connect(csLogic.addressDecoderOutput.GetBusLine(2))
	csLogic.inverter.APin(0).Connect(csLogic.addressDecoderOutput.GetBusLine(3))

	// Complete remaining connections on io decoder
	csLogic.ioDecoder.EPin(2).Connect(csLogic.vcc)

	csLogic.ioCS = buses.New8BitStandaloneBus()
	csLogic.exRAMCS = buses.NewStandaloneLine(true)
	csLogic.miaCS = buses.NewStandaloneLine(false)

	// Connect output lines
	csLogic.ioDecoder.YPin().Connect(csLogic.ioCS)
	csLogic.andGate.YPin(0).Connect(csLogic.exRAMCS)
	csLogic.inverter.YPin(0).Connect(csLogic.miaCS)

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
//
// Parameters:
//   - stepContext: The current step context for the emulation cycle
func (circuit *ClementinaCSLogic) Tick(stepContext *common.StepContext) {
	circuit.addressDecoder.Tick(stepContext)
	circuit.ioDecoder.Tick(stepContext)
	circuit.andGate.Tick(stepContext)
	circuit.inverter.Tick(stepContext)
}
