package via

import "github.com/fran150/clementina6502/buses"

type ViaControlLines struct {
	lines          [2]*buses.ConnectorEnabledHigh
	masks          [2]viaPCRTranstitionMasks
	previousStatus [2]bool

	handshakeInProgress   bool
	handshakeCycleCounter uint8
}

func createControlLines(masks [2]viaPCRTranstitionMasks) *ViaControlLines {
	return &ViaControlLines{
		lines: [2]*buses.ConnectorEnabledHigh{
			buses.CreateConnectorEnabledHigh(),
			buses.CreateConnectorEnabledHigh(),
		},
		masks:          masks,
		previousStatus: [2]bool{false, false},

		handshakeInProgress:   false,
		handshakeCycleCounter: 0,
	}
}

func (cb *ViaControlLines) GetLine(num uint8) *buses.ConnectorEnabledHigh {
	return cb.lines[num]
}

func (cb *ViaControlLines) checkControlLineTransitioned(pcr *ViaPeripheralControlRegister, num int) bool {
	caCtrlPositive := pcr.isTransitionPositive(cb.masks[num])

	currentCrl := cb.lines[num].Enabled()
	previousCtrl := cb.previousStatus[num]

	return (caCtrlPositive && !previousCtrl && currentCrl) || (!caCtrlPositive && previousCtrl && !currentCrl)
}

func (cb *ViaControlLines) setOutputHandshakeMode(pcr *ViaPeripheralControlRegister) {
	if cb.handshakeInProgress && cb.checkControlLineTransitioned(pcr, 0) {
		cb.handshakeInProgress = false
	}

	if cb.handshakeInProgress {
		cb.lines[1].SetEnable(false)
	} else {
		cb.lines[1].SetEnable(true)
	}
}

func (cb *ViaControlLines) setOutputPulseMode() {
	if cb.handshakeInProgress {
		cb.handshakeCycleCounter += 1
	}

	if cb.handshakeCycleCounter > 2 && !cb.lines[1].Enabled() {
		cb.handshakeInProgress = false
	}

	if cb.handshakeInProgress {
		cb.lines[1].SetEnable(false)
	} else {
		cb.lines[1].SetEnable(true)
	}
}

func (cb *ViaControlLines) setFixedMode(pcr *ViaPeripheralControlRegister) {
	if pcr.isTransitionPositive(cb.masks[1]) {
		cb.lines[1].SetEnable(true)
	} else {
		cb.lines[1].SetEnable(false)
	}
}

func (cb *ViaControlLines) initHandshake() {
	cb.handshakeCycleCounter = 0
	cb.handshakeInProgress = true
}

func (cb *ViaControlLines) storePreviousControlLinesValues() {
	cb.previousStatus[0] = cb.lines[0].Enabled()
	cb.previousStatus[1] = cb.lines[1].Enabled()
}
