package via

import "github.com/fran150/clementina6502/buses"

type viaControlLines struct {
	side           *ViaSide
	lines          [2]*buses.ConnectorEnabledHigh
	previousStatus [2]bool

	peripheralControlRegister *uint8
	interrupts                *ViaIFR

	handshakeInProgress   bool
	handshakeCycleCounter uint8
}

func (cl *viaControlLines) getLine(num uint8) *buses.ConnectorEnabledHigh {
	return cl.lines[num]
}

func (cl *viaControlLines) configForTransitionOnPositiveEdge(num int) bool {
	mask := cl.side.configuration.transitionConfigurationMasks[num]

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
	mask := cl.side.configuration.outputConfigurationMask
	return viaPCROutputModes(*cl.peripheralControlRegister & uint8(mask))
}

func (cl *viaControlLines) setOutputMode() {
	switch cl.getOutputMode() {
	case cl.side.configuration.handshakeMode:
		if cl.handshakeInProgress && cl.checkControlLineTransitioned(0) {
			cl.handshakeInProgress = false
		}

		cl.lines[1].SetEnable(!cl.handshakeInProgress)

	case cl.side.configuration.pulseMode:
		if cl.handshakeInProgress {
			cl.handshakeCycleCounter += 1
		}

		if cl.handshakeCycleCounter > 2 && !cl.lines[1].Enabled() {
			cl.handshakeInProgress = false
		}

		cl.lines[1].SetEnable(!cl.handshakeInProgress)

	case cl.side.configuration.fixedMode:
		cl.lines[1].SetEnable(cl.configForTransitionOnPositiveEdge(1))
	}
}

func (cl *viaControlLines) setInterruptFlagOnControlLinesTransition() {

	if cl.checkControlLineTransitioned(0) {
		cl.interrupts.setInterruptFlagBit(cl.side.configuration.controlLinesIRQBits[0])
	}

	if cl.side.controlLines.checkControlLineTransitioned(1) {
		cl.interrupts.setInterruptFlagBit(cl.side.configuration.controlLinesIRQBits[1])
	}
}
