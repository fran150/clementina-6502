package via

import (
	"math"

	"github.com/fran150/clementina6502/buses"
)

type ViaPort struct {
	connector    *buses.BusConnector[uint8]
	acr          *ViaAuxiliaryControlRegister
	pcr          *ViaPeripheralControlRegister
	controlLines *ViaControlLines
}

func createViaPort(acr *ViaAuxiliaryControlRegister, pcr *ViaPeripheralControlRegister, controlLines *ViaControlLines) *ViaPort {
	return &ViaPort{
		connector:    buses.CreateBusConnector[uint8](),
		acr:          acr,
		pcr:          pcr,
		controlLines: controlLines,
	}
}

func (port *ViaPort) getConnector() *buses.BusConnector[uint8] {
	return port.connector
}

func (port *ViaPort) latchPort(ddr uint8, register *uint8, mask viaACRLatchingMasks) {
	// Read pin levels on port
	value := port.connector.Read()

	// Read pins are all the ones with 0 in the DDR
	readPins := ^ddr

	if port.acr.isLatchingEnabled(mask) {
		// If latching is enabled value is the one at the time of transition
		if port.controlLines.checkControlLineTransitioned(port.pcr, 0) {
			*register = value & readPins
		}
	}
}

func isByteSet(value uint8, bitNumber uint8) bool {
	mask := uint8(math.Pow(2, float64(bitNumber)))

	return (value & mask) > 0
}

func (port *ViaPort) writePort(ddr uint8, register uint8) {
	for i := range uint8(8) {
		if isByteSet(ddr, i) {
			port.connector.GetLine(i).Set(isByteSet(register, i))
		}
	}
}
