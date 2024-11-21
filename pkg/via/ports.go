package via

import (
	"math"

	"github.com/fran150/clementina6502/pkg/buses"
)

type viaPortConfiguration struct {
	clearC2OnRWMask       viaPCRInterruptClearMasks
	controlLinesIRQBits   [2]viaIRQFlags
	inputRegister         *uint8
	outputRegister        *uint8
	dataDirectionRegister *uint8
	controlLines          *viaControlLines
}

type ViaPort struct {
	connector *buses.BusConnector[uint8]

	configuration *viaPortConfiguration

	auxiliaryControlRegister  *uint8
	peripheralControlRegister *uint8
	interrupts                *ViaIFR
}

func createViaPort(via *Via65C22S, config *viaPortConfiguration) *ViaPort {
	return &ViaPort{
		connector: buses.CreateBusConnector[uint8](),

		configuration: config,

		auxiliaryControlRegister:  &via.registers.auxiliaryControl,
		peripheralControlRegister: &via.registers.auxiliaryControl,
		interrupts:                &via.registers.interrupts,
	}
}

func (port *ViaPort) getConnector() *buses.BusConnector[uint8] {
	return port.connector
}

func isByteSet(value uint8, bitNumber uint8) bool {
	mask := uint8(math.Pow(2, float64(bitNumber)))

	return (value & mask) > 0
}

// TODO: Would it be easier to write the whole number instead of line by line?
func (port *ViaPort) writePortOutputRegister() {
	for i := range uint8(8) {
		if isByteSet(*port.configuration.dataDirectionRegister, i) {
			port.connector.GetLine(i).Set(isByteSet(*port.configuration.outputRegister, i))
		}
	}
}

func (port *ViaPort) isSetToClearOnRW() bool {
	return (*port.peripheralControlRegister & uint8(port.configuration.clearC2OnRWMask)) == 0x00
}

func (port *ViaPort) clearControlLinesInterruptFlagOnRW() {
	port.interrupts.clearInterruptFlagBit(port.configuration.controlLinesIRQBits[0])

	if port.isSetToClearOnRW() {
		port.interrupts.clearInterruptFlagBit(port.configuration.controlLinesIRQBits[1])
	}
}

func (port *ViaPort) readPins() uint8 {
	// Read pin levels on port
	value := port.connector.Read()

	// Read pins are all the ones with 0 in the DDR
	readPins := ^*port.configuration.dataDirectionRegister

	return value & readPins
}
