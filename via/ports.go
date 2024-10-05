package via

import (
	"math"

	"github.com/fran150/clementina6502/buses"
)

type viaPortConfiguration struct {
	latchingEnabledMasks  viaACRLatchingMasks
	clearC2OnRWMask       viaPCRInterruptClearMasks
	controlLinesIRQBits   [2]viaIRQFlags
	inputRegister         *uint8
	outputRegister        *uint8
	dataDirectionRegister *uint8
	controlLines          *viaControlLines
	timer                 *ViaTimer
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

func (port *ViaPort) isLatchingEnabled() bool {
	return *port.auxiliaryControlRegister&uint8(port.configuration.latchingEnabledMasks) > 0x00
}

func (port *ViaPort) latchPort() {
	// Read pin levels on port
	value := port.connector.Read()

	// Read pins are all the ones with 0 in the DDR
	readPins := ^*port.configuration.dataDirectionRegister

	if port.isLatchingEnabled() {
		// If latching is enabled value is the one at the time of transition
		if port.configuration.controlLines.checkControlLineTransitioned(0) {
			*port.configuration.inputRegister = value & readPins
		}
	}
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

func (port *ViaPort) writeTimerOutput() {
	// From the manual: With the output enabled (ACR7=1) a "write T1C-H operation will cause PB7 to go low.
	// I'm assuming that setting ACR7=1 with timer not running will cause PB7 to go high
	if port.configuration.timer.isTimerOutputEnabled() {
		if !port.configuration.timer.timerEnabled {
			port.connector.GetLine(7).Set(true)
		} else {
			if port.configuration.timer.hasCountedToZero {
				switch port.configuration.timer.getRunningMode() {
				case txRunModeOneShot:
					port.connector.GetLine(7).Set(true)
				case t1RunModeFree:
					port.configuration.timer.outputStatusWhenEnabled = !port.configuration.timer.outputStatusWhenEnabled
					port.connector.GetLine(7).Set(port.configuration.timer.outputStatusWhenEnabled)
				}
			} else {
				port.connector.GetLine(7).Set(port.configuration.timer.outputStatusWhenEnabled)
			}
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
