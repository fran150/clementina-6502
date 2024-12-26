package via

// This structs stores the configuration of the latches allowing the same
// struct to be used to reprenset both latches on port A and B
type viaLatchesConfiguration struct {
	latchingEnabledMasks    viaACRLatchingMasks // Value of mask used to control latching
	outputConfigurationMask viaPCROutputMasks   // Mask used in PCR to check output configuration of control lines
	handshakeMode           viaPCROutputModes   // Bit value of PCR when set to handshake mode
	pulseMode               viaPCROutputModes   // Bit value of PCR when set to pulse mode
	fixedModeLow            viaPCROutputModes   // Bit vavlue of PCR when set to fixed mode low
	fixedModeHigh           viaPCROutputModes   // Bit vavlue of PCR when set to fixed mode high

	inputRegister *uint8           // Reference to chip's input register (IRA or IRB)
	port          *viaPort         // Reference to chip's port (Port A or B)
	controlLines  *viaControlLines // Reference to chip's control lines (CA1/2 or CB1/2)
}

// Latching circuits on 65C22 allow the port A or B to latch the value for input or output.
// The chip will hold those values in IRA / IRB if it's configured as input (regardless of changes in the port values)
// or will keep the port lines in the correct status if configured as output
type viaLatches struct {
	handshakeInProgress   bool  // Handshake is in process (see handshake operation section in the manual)
	handshakeCycleCounter uint8 // Count how many cycles the chip has been in handshake mode

	configuration *viaLatchesConfiguration // Stores the latch configuration. This allow to use the same struct for Port A or B latches.

	peripheralControlRegister *uint8 // Pointer to the chip's PCR
	auxiliaryControlRegister  *uint8 // Pointer to the chip's ACR
}

// Creates the latches and attaches them to the specified via chip
func createViaLatches(via *Via65C22S, configuration *viaLatchesConfiguration) *viaLatches {
	return &viaLatches{
		handshakeInProgress:   false,
		handshakeCycleCounter: 0,

		configuration: configuration,

		peripheralControlRegister: &via.registers.peripheralControl,
		auxiliaryControlRegister:  &via.registers.auxiliaryControl,
	}
}

// Returns true if latching is enabled on ACR
func (l *viaLatches) isLatchingEnabled() bool {
	return *l.auxiliaryControlRegister&uint8(l.configuration.latchingEnabledMasks) > 0x00
}

// Attempts to latch the port values for input
func (l *viaLatches) latchPort() {
	if l.isLatchingEnabled() {
		// If latching is enabled value is the one at the time of transition
		if l.configuration.controlLines.checkControlLineTransitioned(0) {
			*l.configuration.inputRegister = l.configuration.port.readPins()
		}
	}
}

// Starts a new handshake process by setting the flag and reseting the counter
func (l *viaLatches) initHandshake() {
	l.handshakeCycleCounter = 0
	l.handshakeInProgress = true
}

// Returns the currently configured output mode of control lines. This is important for the latch
// circuit as it uses the control lines to signal the handshake process
func (l *viaLatches) getOutputMode() viaPCROutputModes {
	mask := l.configuration.outputConfigurationMask
	return viaPCROutputModes(*l.peripheralControlRegister & uint8(mask))
}

// Depending on the chip configuration set the control line status.
// See manual for behavior of handshake and pulse mode.
// Fixed mode will keep line low or high as indicated.
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

	case l.configuration.fixedModeLow:
		l.configuration.controlLines.lines[1].SetEnable(false)
	case l.configuration.fixedModeHigh:
		l.configuration.controlLines.lines[1].SetEnable(true)

	}
}
