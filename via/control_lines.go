package via

import "github.com/fran150/clementina6502/buses"

type ViaControlLines struct {
	lines                        [2]*buses.ConnectorEnabledHigh
	transitionConfigurationMasks [2]viaPCRTranstitionMasks
	outputConfigurationMask      viaPCROutputMasks
	enabledOutputModes           [3]viaPCROutputModes
	previousStatus               [2]bool

	handshakeInProgress   bool
	handshakeCycleCounter uint8
}

func createControlLines(transitionConfigurationMasks [2]viaPCRTranstitionMasks, outputConfigurationMask viaPCROutputMasks, enabledOutputModes [3]viaPCROutputModes) *ViaControlLines {
	return &ViaControlLines{
		lines: [2]*buses.ConnectorEnabledHigh{
			buses.CreateConnectorEnabledHigh(),
			buses.CreateConnectorEnabledHigh(),
		},
		transitionConfigurationMasks: transitionConfigurationMasks,
		enabledOutputModes:           enabledOutputModes,
		outputConfigurationMask:      outputConfigurationMask,

		previousStatus: [2]bool{false, false},

		handshakeInProgress:   false,
		handshakeCycleCounter: 0,
	}
}

func (cb *ViaControlLines) GetLine(num uint8) *buses.ConnectorEnabledHigh {
	return cb.lines[num]
}

func (cb *ViaControlLines) checkControlLineTransitioned(pcr *ViaPeripheralControlRegister, num int) bool {
	caCtrlPositive := pcr.isTransitionPositive(cb.transitionConfigurationMasks[num])

	currentCrl := cb.lines[num].Enabled()
	previousCtrl := cb.previousStatus[num]

	return (caCtrlPositive && !previousCtrl && currentCrl) || (!caCtrlPositive && previousCtrl && !currentCrl)
}

func (cb *ViaControlLines) setOutputHandshakeMode(pcr *ViaPeripheralControlRegister) {
	if cb.handshakeInProgress && cb.checkControlLineTransitioned(pcr, 0) {
		cb.handshakeInProgress = false
	}

	cb.lines[1].SetEnable(!cb.handshakeInProgress)
}

func (cb *ViaControlLines) setOutputPulseMode() {
	if cb.handshakeInProgress {
		cb.handshakeCycleCounter += 1
	}

	if cb.handshakeCycleCounter > 2 && !cb.lines[1].Enabled() {
		cb.handshakeInProgress = false
	}

	cb.lines[1].SetEnable(!cb.handshakeInProgress)
}

func (cb *ViaControlLines) setFixedMode(pcr *ViaPeripheralControlRegister) {
	cb.lines[1].SetEnable(pcr.isTransitionPositive(cb.transitionConfigurationMasks[1]))
}

func (cb *ViaControlLines) initHandshake() {
	cb.handshakeCycleCounter = 0
	cb.handshakeInProgress = true
}

func (cb *ViaControlLines) storePreviousControlLinesValues() {
	cb.previousStatus[0] = cb.lines[0].Enabled()
	cb.previousStatus[1] = cb.lines[1].Enabled()
}

func (cb *ViaControlLines) setOutputMode(pcr *ViaPeripheralControlRegister) {
	switch pcr.getOutputMode(cb.outputConfigurationMask) {
	case cb.enabledOutputModes[0]:
		cb.setOutputHandshakeMode(pcr)
	case cb.enabledOutputModes[1]:
		cb.setOutputPulseMode()
	case cb.enabledOutputModes[2]:
		cb.setFixedMode(pcr)
	}
}
