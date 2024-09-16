package via

import "github.com/fran150/clementina6502/buses"

type viaControlLines struct {
	side           *ViaSide
	lines          [2]*buses.ConnectorEnabledHigh
	previousStatus [2]bool

	peripheralControlRegister *uint8

	handshakeInProgress   bool
	handshakeCycleCounter uint8
}

func (ctrlLine *viaControlLines) getLine(num uint8) *buses.ConnectorEnabledHigh {
	return ctrlLine.lines[num]
}

func (ctrlLine *viaControlLines) configForTransitionOnPositiveEdge(num int) bool {
	mask := ctrlLine.side.configuration.transitionConfigurationMasks[num]

	return (*ctrlLine.peripheralControlRegister & uint8(mask)) > 0x00
}

func (ctrlLine *viaControlLines) checkControlLineTransitioned(num int) bool {
	onPositive := ctrlLine.configForTransitionOnPositiveEdge(num)

	currentCrl := ctrlLine.lines[num].Enabled()
	previousCtrl := ctrlLine.previousStatus[num]

	return (onPositive && !previousCtrl && currentCrl) || (!onPositive && previousCtrl && !currentCrl)
}

func (crtlLine *viaControlLines) setOutputHandshakeMode() {
	if crtlLine.handshakeInProgress && crtlLine.checkControlLineTransitioned(0) {
		crtlLine.handshakeInProgress = false
	}

	crtlLine.lines[1].SetEnable(!crtlLine.handshakeInProgress)
}

func (crtlLine *viaControlLines) setOutputPulseMode() {
	if crtlLine.handshakeInProgress {
		crtlLine.handshakeCycleCounter += 1
	}

	if crtlLine.handshakeCycleCounter > 2 && !crtlLine.lines[1].Enabled() {
		crtlLine.handshakeInProgress = false
	}

	crtlLine.lines[1].SetEnable(!crtlLine.handshakeInProgress)
}

func (crtlLine *viaControlLines) setFixedMode() {
	crtlLine.lines[1].SetEnable(crtlLine.configForTransitionOnPositiveEdge(1))
}

func (ctrlLine *viaControlLines) initHandshake() {
	ctrlLine.handshakeCycleCounter = 0
	ctrlLine.handshakeInProgress = true
}

func (ctrlLine *viaControlLines) storePreviousControlLinesValues() {
	ctrlLine.previousStatus[0] = ctrlLine.lines[0].Enabled()
	ctrlLine.previousStatus[1] = ctrlLine.lines[1].Enabled()
}

func (ctrlLine *viaControlLines) getOutputMode() viaPCROutputModes {
	mask := ctrlLine.side.configuration.outputConfigurationMask
	return viaPCROutputModes(*ctrlLine.peripheralControlRegister & uint8(mask))
}

func (ctrlLine *viaControlLines) setOutputMode() {
	switch ctrlLine.getOutputMode() {
	case ctrlLine.side.configuration.handshakeMode:
		ctrlLine.setOutputHandshakeMode()
	case ctrlLine.side.configuration.pulseMode:
		ctrlLine.setOutputPulseMode()
	case ctrlLine.side.configuration.fixedMode:
		ctrlLine.setFixedMode()
	}
}
