package via

import (
	"math"

	"github.com/fran150/clementina6502/buses"
)

type ViaPort struct {
	side *ViaSide

	auxiliaryControlRegister  *viaAuxiliaryControlRegister
	peripheralControlRegister *ViaPeripheralControlRegister

	connector *buses.BusConnector[uint8]
}

func (port *ViaPort) getConnector() *buses.BusConnector[uint8] {
	return port.connector
}

func (port *ViaPort) latchPort() {
	// Read pin levels on port
	value := port.connector.Read()

	// Read pins are all the ones with 0 in the DDR
	readPins := ^port.side.registers.dataDirectionRegister

	if port.auxiliaryControlRegister.isLatchingEnabledForSide(port.side) {
		// If latching is enabled value is the one at the time of transition
		if port.side.controlLines.checkControlLineTransitioned(0) {
			port.side.registers.inputRegister = value & readPins
		}
	}
}

func isByteSet(value uint8, bitNumber uint8) bool {
	mask := uint8(math.Pow(2, float64(bitNumber)))

	return (value & mask) > 0
}

func (port *ViaPort) writePort() {
	for i := range uint8(8) {
		if isByteSet(port.side.registers.dataDirectionRegister, i) {
			port.connector.GetLine(i).Set(isByteSet(port.side.registers.outputRegister, i))
		}
	}
}
