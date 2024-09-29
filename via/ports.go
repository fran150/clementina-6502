package via

import (
	"math"

	"github.com/fran150/clementina6502/buses"
)

type ViaPort struct {
	side *ViaSide

	auxiliaryControlRegister  *uint8
	peripheralControlRegister *uint8

	interrupts *ViaIFR

	connector *buses.BusConnector[uint8]
}

func (port *ViaPort) getConnector() *buses.BusConnector[uint8] {
	return port.connector
}

func (port *ViaPort) isLatchingEnabled() bool {
	return *port.auxiliaryControlRegister&uint8(port.side.configuration.latchingEnabledMasks) > 0x00
}

func (port *ViaPort) countDownPulseIfEnabled() {
	if port.side.timer.timerEnabled && port.side.timer.getRunningMode() == t2RunModePulseCounting && !port.connector.GetLine(6).Status() {
		port.side.registers.counter -= 1
	}
}

func (port *ViaPort) latchPort() {
	// Read pin levels on port
	value := port.connector.Read()

	// Read pins are all the ones with 0 in the DDR
	readPins := ^port.side.registers.dataDirectionRegister

	if port.isLatchingEnabled() {
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

// TODO: Would it be easier to write the whole number instead of line by line?
func (port *ViaPort) writePortOutputRegister() {
	for i := range uint8(8) {
		if isByteSet(port.side.registers.dataDirectionRegister, i) {
			port.connector.GetLine(i).Set(isByteSet(port.side.registers.outputRegister, i))
		}
	}
}

func (port *ViaPort) writeTimerOutput() {
	// From the manual: With the output enabled (ACR7=1) a "write T1C-H operation will cause PB7 to go low.
	// I'm assuming that setting ACR7=1 with timer not running will cause PB7 to go high
	if port.side.timer.isTimerOutputEnabled() {
		if !port.side.timer.timerEnabled {
			port.side.peripheralPort.connector.GetLine(7).Set(true)
		} else {
			if port.side.registers.counter == 0xFFFF {
				switch port.side.timer.getRunningMode() {
				case txRunModeOneShot:
					port.connector.GetLine(7).Set(true)
				case t1RunModeFree:
					port.side.timer.outputStatusWhenEnabled = !port.side.timer.outputStatusWhenEnabled
					port.side.peripheralPort.connector.GetLine(7).Set(port.side.timer.outputStatusWhenEnabled)
				}
			} else {
				port.side.peripheralPort.connector.GetLine(7).Set(port.side.timer.outputStatusWhenEnabled)
			}
		}
	}

}

func (port *ViaPort) isSetToClearOnRW() bool {
	return (*port.peripheralControlRegister & uint8(port.side.configuration.clearC2OnRWMask)) == 0x00
}

func (port *ViaPort) clearControlLinesInterruptFlagOnRW() {
	port.interrupts.clearInterruptFlagBit(port.side.configuration.controlLinesIRQBits[0])

	if port.isSetToClearOnRW() {
		port.interrupts.clearInterruptFlagBit(port.side.configuration.controlLinesIRQBits[1])
	}
}
