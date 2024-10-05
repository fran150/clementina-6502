package via

import "github.com/fran150/clementina6502/buses"

type viaControlLineConfiguration struct {
	transitionConfigurationMasks [2]viaPCRTranstitionMasks
	outputConfigurationMask      viaPCROutputMasks
	handshakeMode                viaPCROutputModes
	pulseMode                    viaPCROutputModes
	fixedMode                    viaPCROutputModes
	controlLinesIRQBits          [2]viaIRQFlags
}

type viaControlLines struct {
	lines                 [2]*buses.ConnectorEnabledHigh
	previousStatus        [2]bool
	handshakeInProgress   bool
	handshakeCycleCounter uint8

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
		previousStatus:        [2]bool{false, false},
		handshakeInProgress:   false,
		handshakeCycleCounter: 0,

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

func (cl *viaControlLines) initHandshake() {
	cl.handshakeCycleCounter = 0
	cl.handshakeInProgress = true
}

func (cl *viaControlLines) storePreviousControlLinesValues() {
	cl.previousStatus[0] = cl.lines[0].Enabled()
	cl.previousStatus[1] = cl.lines[1].Enabled()
}

func (cl *viaControlLines) getOutputMode() viaPCROutputModes {
	mask := cl.configuration.outputConfigurationMask
	return viaPCROutputModes(*cl.peripheralControlRegister & uint8(mask))
}

func (cl *viaControlLines) setOutput() {
	switch cl.getOutputMode() {
	case cl.configuration.handshakeMode:
		if cl.handshakeInProgress && cl.checkControlLineTransitioned(0) {
			cl.handshakeInProgress = false
		}

		cl.lines[1].SetEnable(!cl.handshakeInProgress)

	case cl.configuration.pulseMode:
		if cl.handshakeInProgress {
			cl.handshakeCycleCounter += 1
		}

		if cl.handshakeCycleCounter > 2 && !cl.lines[1].Enabled() {
			cl.handshakeInProgress = false
		}

		cl.lines[1].SetEnable(!cl.handshakeInProgress)

	case cl.configuration.fixedMode:
		cl.lines[1].SetEnable(cl.configForTransitionOnPositiveEdge(1))
	}
}

func (cl *viaControlLines) setInterruptFlagOnControlLinesTransition() {

	if cl.checkControlLineTransitioned(0) {
		cl.interrupts.setInterruptFlagBit(cl.configuration.controlLinesIRQBits[0])
	}

	if cl.checkControlLineTransitioned(1) {
		cl.interrupts.setInterruptFlagBit(cl.configuration.controlLinesIRQBits[1])
	}
}
