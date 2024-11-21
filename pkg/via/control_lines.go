package via

import "github.com/fran150/clementina6502/pkg/buses"

type viaControlLineConfiguration struct {
	transitionConfigurationMasks [2]viaPCRTranstitionMasks
	controlLinesIRQBits          [2]viaIRQFlags
}

type viaControlLines struct {
	lines          [2]*buses.ConnectorEnabledHigh
	previousStatus [2]bool

	configuration viaControlLineConfiguration

	peripheralControlRegister *uint8
	interrupts                *ViaIFR
}

func createViaControlLines(via *Via65C22S, config *viaControlLineConfiguration) *viaControlLines {
	return &viaControlLines{
		lines: [2]*buses.ConnectorEnabledHigh{
			buses.CreateConnectorEnabledHigh(),
			buses.CreateConnectorEnabledHigh(),
		},
		previousStatus: [2]bool{false, false},

		configuration: *config,

		peripheralControlRegister: &via.registers.peripheralControl,
		interrupts:                &via.registers.interrupts,
	}
}

func (cl *viaControlLines) getLine(num uint8) *buses.ConnectorEnabledHigh {
	return cl.lines[num]
}

func (cl *viaControlLines) configForTransitionOnPositiveEdge(num int) bool {
	mask := cl.configuration.transitionConfigurationMasks[num]

	return (*cl.peripheralControlRegister & uint8(mask)) > 0x00
}

func (cl *viaControlLines) checkControlLineTransitioned(num int) bool {
	onPositive := cl.configForTransitionOnPositiveEdge(num)

	currentCrl := cl.lines[num].Enabled()
	previousCtrl := cl.previousStatus[num]

	return (onPositive && !previousCtrl && currentCrl) || (!onPositive && previousCtrl && !currentCrl)
}

func (cl *viaControlLines) setInterruptFlagOnControlLinesTransition() {

	if cl.checkControlLineTransitioned(0) {
		cl.interrupts.setInterruptFlagBit(cl.configuration.controlLinesIRQBits[0])
	}

	if cl.checkControlLineTransitioned(1) {
		cl.interrupts.setInterruptFlagBit(cl.configuration.controlLinesIRQBits[1])
	}
}

func (cl *viaControlLines) storePreviousControlLinesValues() {
	cl.previousStatus[0] = cl.lines[0].Enabled()
	cl.previousStatus[1] = cl.lines[1].Enabled()
}
