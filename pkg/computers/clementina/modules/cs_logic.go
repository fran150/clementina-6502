package modules

import (
	"github.com/fran150/clementina-6502/pkg/common"
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
	a1         [6]buses.LineConnector // A10 to A15 address lines
	picoHiRAME buses.LineConnector    // Allows Pico to enable / disable high RAM

	// Exposed lines
	ioOE   buses.Bus[uint8] // I/O output enable
	exRAME buses.Line       // External RAM enable (512 KB banked in 16 KB windows)
	hiRAME buses.Line       // High RAM enable (32 KB banked in 8 KB windows)

	// Internal components
	addressDecoder *decoders.Decoder74HC138 // Address decoder to slice RAM space in 16 KB windows
	ioDecoder      *decoders.Decoder74HC138 // I/O decoder to slice I/O space in 8 slots of 1KB each
	andGate        *gates.And74HC08         // Gates for logic operations
	orGate         *gates.Or74HC32          // Gates for logic operations

	addressDecoderOutput buses.Bus[uint8]
	vcc                  buses.Line // Lines connected to VCC (will always be high)
	ground               buses.Line // Lines connected to ground (will always be low)
}

// Creates a new Clementina CS Logic component
func NewClementinaCSLogic() *ClementinaCSLogic {
	csLogic := &ClementinaCSLogic{}

	// Create internal components
	csLogic.addressDecoder = decoders.NewDecoder74HC138()
	csLogic.ioDecoder = decoders.NewDecoder74HC138()
	csLogic.andGate = gates.New74HC08()
	csLogic.orGate = gates.New74HC32()

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

	// Expose connectors for Pico HiRAM enable on the B pin of the OR gate
	csLogic.picoHiRAME = csLogic.orGate.BPin(0)

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
	csLogic.orGate.APin(0).Connect(csLogic.addressDecoderOutput.GetBusLine(3))

	// Complete remaining connections on io decoder
	csLogic.ioDecoder.EPin(2).Connect(csLogic.vcc)

	csLogic.ioOE = buses.New8BitStandaloneBus()
	csLogic.exRAME = buses.NewStandaloneLine(true)
	csLogic.hiRAME = buses.NewStandaloneLine(true)

	// Connect output lines
	csLogic.ioDecoder.YPin().Connect(csLogic.ioOE)
	csLogic.andGate.YPin(0).Connect(csLogic.exRAME)
	csLogic.orGate.YPin(0).Connect(csLogic.hiRAME)

	return csLogic
}

// A1 returns the connector for the specified address line (A10 to A15)
func (circuit *ClementinaCSLogic) A1(index int) buses.LineConnector {
	if index >= 0 && index < len(circuit.a1) {
		return circuit.a1[index]
	}
	return nil
}

// PicoHiRAME returns the connector for the Pico HiRAM enable line
// This line is used by the Pico to enable or disable the high RAM
// If the line is low the pico is enabling the high RAM.
// When the clementina computer starts the Pico will respond to requests
// in high RAM space and keep this line high disabling access to HiRAM.
// This will be used to copy the kernel to the RAM
// avoiding the need to use ROMs. After the pico finishes the startup
// process it will enable the high RAM lowering this line and start
// the execution of the Clementina 6502 kernel.
func (circuit *ClementinaCSLogic) PicoHiRAME() buses.LineConnector {
	return circuit.picoHiRAME
}

// IOOE returns the connector for the I/O output enable bus
// Each line can be used to map a device in one of the 8 I/O slots
// Each device will have 1K of available address space
func (circuit *ClementinaCSLogic) IOOE() buses.Bus[uint8] {
	return circuit.ioOE
}

// ExRAME returns the connector for the external RAM enable line
// The extended RAM maps to a 512 KB space banked in 16 KB windows
func (circuit *ClementinaCSLogic) ExRAME() buses.Line {
	return circuit.exRAME
}

// HiRAME returns the connector for the high RAM enable line
// This line is used to enable or disable the high RAM.
// Hi RAM maps to a 32 KB space banked in 8 KB windows.
func (circuit *ClementinaCSLogic) HiRAME() buses.Line {
	return circuit.hiRAME
}

// Tick executes one emulation step
func (circuit *ClementinaCSLogic) Tick(stepContext *common.StepContext) {
	circuit.addressDecoder.Tick(stepContext)
	circuit.andGate.Tick(stepContext)
	circuit.ioDecoder.Tick(stepContext)
	circuit.orGate.Tick(stepContext)
}
