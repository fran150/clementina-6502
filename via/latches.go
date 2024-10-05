package via

type viaLatchesConfifuration struct {
	outputConfigurationMask viaPCROutputMasks
	handshakeMode           viaPCROutputModes
	pulseMode               viaPCROutputModes
	fixedMode               viaPCROutputModes
	controlLines            *viaControlLines
}

type viaLatches struct {
	handshakeInProgress   bool
	handshakeCycleCounter uint8

	configuration *viaLatchesConfifuration

	peripheralControlRegister *uint8
}

func createViaLatches(via *Via65C22S, configuration *viaLatchesConfifuration) *viaLatches {
	return &viaLatches{
		handshakeInProgress:   false,
		handshakeCycleCounter: 0,

		configuration: configuration,

		peripheralControlRegister: &via.registers.peripheralControl,
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
