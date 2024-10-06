package via

type viaLatchesConfifuration struct {
	latchingEnabledMasks    viaACRLatchingMasks
	outputConfigurationMask viaPCROutputMasks
	handshakeMode           viaPCROutputModes
	pulseMode               viaPCROutputModes
	fixedMode               viaPCROutputModes

	inputRegister *uint8
	port          *ViaPort
	controlLines  *viaControlLines
}

type viaLatches struct {
	handshakeInProgress   bool
	handshakeCycleCounter uint8

	configuration *viaLatchesConfifuration

	peripheralControlRegister *uint8
	auxiliaryControlRegister  *uint8
}

func createViaLatches(via *Via65C22S, configuration *viaLatchesConfifuration) *viaLatches {
	return &viaLatches{
		handshakeInProgress:   false,
		handshakeCycleCounter: 0,

		configuration: configuration,

		peripheralControlRegister: &via.registers.peripheralControl,
		auxiliaryControlRegister:  &via.registers.auxiliaryControl,
	}
}

func (l *viaLatches) isLatchingEnabled() bool {
	return *l.auxiliaryControlRegister&uint8(l.configuration.latchingEnabledMasks) > 0x00
}

func (l *viaLatches) latchPort() {
	if l.isLatchingEnabled() {
		// If latching is enabled value is the one at the time of transition
		if l.configuration.controlLines.checkControlLineTransitioned(0) {
			*l.configuration.inputRegister = l.configuration.port.readPins()
		}
	}
}

func (l *viaLatches) initHandshake() {
	l.handshakeCycleCounter = 0
	l.handshakeInProgress = true
}

func (l *viaLatches) getOutputMode() viaPCROutputModes {
	mask := l.configuration.outputConfigurationMask
	return viaPCROutputModes(*l.peripheralControlRegister & uint8(mask))
}

func (l *viaLatches) setOutput() {
	switch l.getOutputMode() {
	case l.configuration.handshakeMode:
		if l.handshakeInProgress && l.configuration.controlLines.checkControlLineTransitioned(0) {
			l.handshakeInProgress = false
		}

		l.configuration.controlLines.lines[1].SetEnable(!l.handshakeInProgress)

	case l.configuration.pulseMode:
		if l.handshakeInProgress {
			l.handshakeCycleCounter += 1
		}

		if l.handshakeCycleCounter > 2 && !l.configuration.controlLines.lines[1].Enabled() {
			l.handshakeInProgress = false
		}

		l.configuration.controlLines.lines[1].SetEnable(!l.handshakeInProgress)

	case l.configuration.fixedMode:
		l.configuration.controlLines.lines[1].SetEnable(l.configuration.controlLines.configForTransitionOnPositiveEdge(1))
	}
}
