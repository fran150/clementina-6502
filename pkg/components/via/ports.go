package via

import (
	"math"

	"github.com/fran150/clementina6502/pkg/components/buses"
)

// Contains values that allows to configure the port behavior. This allows to use the same
// struct to represent both port A and port B
type viaPortConfiguration struct {
	clearC2OnRWMask       viaPCRInterruptClearMasks // Masks used in PCR to read if IRQ should be cleared when reading or writing from IRA/ORA/IRB/ORB
	controlLinesIRQBits   [2]viaIRQFlags            // IFR flags for control line status
	inputRegister         *uint8                    // Reference to chip's input register IRA or IRB
	outputRegister        *uint8                    // Reference to chip's output register ORA or ORB
	dataDirectionRegister *uint8                    // Reference to chip's DDR (determines if a pin in the port is input or output)
	controlLines          *viaControlLines          // Reference to chip's control lines used in conjunction with this port
}

// Each port is an 8 line, bidirectional bus used for the transfer of data, control and status information between the
// W65C22 and a peripheral device. Each PA bus line may be individually programmed as either an input or
// output under control of DDR. Data flow direction may be selected on a line by line basis with intermixed
// input and output lines within the same port. When logic 0 is written to any bit position of DDR, the
// corresponding line will be programmed as an input. Likewise, when logic 1 is written into any bit position of
// the register, the corresponding data pin will serve as an output. The data read is determined by the output register when
// input data is latched into the input register under control of the control line 1. All modes are program controlled by way of
// the W65C22's internal control registers.
// With respect to PB, the output signal on line PB7 may be controlled by Timer 1 while Timer 2
// may be programmed to count pulses on the PB6 line.
type viaPort struct {
	connector *buses.BusConnector[uint8]

	configuration *viaPortConfiguration

	auxiliaryControlRegister  *uint8
	peripheralControlRegister *uint8
	interrupts                *viaIFR
}

// Creates a new via port and attach it to the specified chip
func createViaPort(via *Via65C22S, config *viaPortConfiguration) *viaPort {
	return &viaPort{
		connector: buses.CreateBusConnector[uint8](),

		configuration: config,

		auxiliaryControlRegister:  &via.registers.auxiliaryControl,
		peripheralControlRegister: &via.registers.auxiliaryControl,
		interrupts:                &via.registers.interrupts,
	}
}

// Returns the refernece to the bus connector used to represent the port
func (port *viaPort) getConnector() *buses.BusConnector[uint8] {
	return port.connector
}

// Return true if the specified bit is set
func isBitSet(value uint8, bitNumber uint8) bool {
	mask := uint8(math.Pow(2, float64(bitNumber)))

	return (value & mask) > 0
}

// Writes the value of the output register to the port bus
func (port *viaPort) writePortOutputRegister() {
	// TODO: Would it be easier to write the whole number instead of line by line?
	for i := range uint8(8) {
		if isBitSet(*port.configuration.dataDirectionRegister, i) {
			line := port.connector.GetLine(i)

			if line != nil {
				line.Set(isBitSet(*port.configuration.outputRegister, i))
			}
		}
	}
}

// Returns true if port is configured to clear the interrupt when reading input or output register values
func (port *viaPort) isSetToClearOnRW() bool {
	return (*port.peripheralControlRegister & uint8(port.configuration.clearC2OnRWMask)) == 0x00
}

// Clear the flags on IFR when reading input or output values if port is configured to do so
func (port *viaPort) clearControlLinesInterruptFlagOnRW() {
	port.interrupts.clearInterruptFlagBit(port.configuration.controlLinesIRQBits[0])

	if port.isSetToClearOnRW() {
		port.interrupts.clearInterruptFlagBit(port.configuration.controlLinesIRQBits[1])
	}
}

// Reads and stores in the input register the current value of the pins
func (port *viaPort) readPins() uint8 {
	// Read pin levels on port
	value := port.connector.Read()

	// Read pins are all the ones with 0 in the DDR
	readPins := ^*port.configuration.dataDirectionRegister

	return value & readPins
}
