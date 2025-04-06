package via

import "github.com/fran150/clementina-6502/pkg/components/buses"

// Mask to use on the PCR (peripheral control register) to configure control lines CA and CB.
type viaControlLineConfiguration struct {
	transitionConfigurationMasks [2]viaPCRTransitionMasks // Controls if the line acts on the upper or lower edge of the pulse
	controlLinesIRQBits          [2]viaIRQFlags           // Which bits of the IFR (Interrupt Flag Register) are controlled by this lines
}

// The 65C22 has 2 A and B control lines. CA1 and CA2 serve as interrupt inputs or handshake outputs for port A
// CA1 also controls the latching of Input Data on PA.
// CB lines also serve as a serial data port under control of the SR.
// These lines control an internal Interrupt Flag with a corresponding Interrupt Enable bit.
type viaControlLines struct {
	lines          [2]*buses.ConnectorEnabledHigh // Connectors for the lines
	previousStatus [2]bool                        // Previous status of the lines, this is used to detect positive or negative edge transitions

	configuration viaControlLineConfiguration // Control line configuration on the PCR

	peripheralControlRegister *uint8  // Reference to the chip's PCR
	interrupts                *viaIFR // Reference to chip's IFR
}

// Create and attach control lines to the specified chip. Configuration allows to control how
// how the lines behave to emulate CA or CB
func newViaControlLines(via *Via65C22S, config *viaControlLineConfiguration) *viaControlLines {
	return &viaControlLines{
		lines: [2]*buses.ConnectorEnabledHigh{
			buses.NewConnectorEnabledHigh(),
			buses.NewConnectorEnabledHigh(),
		},
		previousStatus: [2]bool{false, false},

		configuration: *config,

		peripheralControlRegister: &via.registers.peripheralControl,
		interrupts:                &via.registers.interrupts,
	}
}

// Gets the reference to the specified line
func (cl *viaControlLines) getLine(num int) *buses.ConnectorEnabledHigh {
	if num < len(cl.lines) {
		return cl.lines[num]
	} else {
		return nil
	}
}

// Sets the configuration so the chip acts on the positive edge
func (cl *viaControlLines) configForTransitionOnPositiveEdge(num int) bool {
	mask := cl.configuration.transitionConfigurationMasks[num]

	return (*cl.peripheralControlRegister & uint8(mask)) > 0x00
}

// Returns true if the control line has transitioned, this means that it changed from high to low
// or visceversa depending on the configuration.
func (cl *viaControlLines) checkControlLineTransitioned(num int) bool {
	onPositive := cl.configForTransitionOnPositiveEdge(num)

	currentCrl := cl.lines[num].Enabled()
	previousCtrl := cl.previousStatus[num]

	return (onPositive && !previousCtrl && currentCrl) || (!onPositive && previousCtrl && !currentCrl)
}

// Sets the corresponding flags on the IFR when the lines transitioned
func (cl *viaControlLines) setInterruptFlagOnControlLinesTransition() {

	if cl.checkControlLineTransitioned(0) {
		cl.interrupts.setInterruptFlagBit(cl.configuration.controlLinesIRQBits[0])
	}

	if cl.checkControlLineTransitioned(1) {
		cl.interrupts.setInterruptFlagBit(cl.configuration.controlLinesIRQBits[1])
	}
}

// Stores the previous values of the lines to detect transitions on the next cycle
func (cl *viaControlLines) storePreviousControlLinesValues() {
	cl.previousStatus[0] = cl.lines[0].Enabled()
	cl.previousStatus[1] = cl.lines[1].Enabled()
}
