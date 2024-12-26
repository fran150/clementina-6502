package via

// Masks used in the ACR to determine if the shifter is shifting as input or output
type viaShiftDirection uint8

const (
	viaShiftIn  viaShiftDirection = 0x00 // Shifter is configured as input
	viaShiftOut viaShiftDirection = 0x10 // Shifter is configured as output
)

// Different values of the ACR used to represent shifter modes
type viaShiftRegisterModes uint8

const (
	viaShiftDisabled    viaShiftRegisterModes = 0x00 // Shifting is disabled
	viaShiftInT2        viaShiftRegisterModes = 0x04 // Shift in using timer 2 as clock
	viaShiftInClock     viaShiftRegisterModes = 0x08 // Shift in using internal clock
	viaShiftInExternal  viaShiftRegisterModes = 0xC  // Shift in using external clock (signals are received through control lines)
	viaShiftOutFree     viaShiftRegisterModes = 0x10 // Similar to shift using t2 but keeps shifting continuosly
	viaShiftOutT2       viaShiftRegisterModes = 0x14 // Shift out using timer 2 as clock
	viaShiftOutClock    viaShiftRegisterModes = 0x18 // Shift out using internal clock
	viaShiftOutExternal viaShiftRegisterModes = 0x1C // Shift out using external clock (signals are receive through control lines)
)

// Shifter configuration. It stores reference to chip's supporting circuits
type viaShifterConfiguration struct {
	timer        *viaTimer        // Reference to the associated via timer (T2)
	controlLines *viaControlLines // Reference to the associated control lines (CB)
}

// The Shift Register (SR) performs bidirectional serial data transfers on line CB2. These transfers are
// controlled by an internal modulo-8 counter. Shift pulses can be applied to the CB1 line from an external
// source, or (with proper mode selection) shift pulses may be generated internally which will appear on the
// CB1 line for controlling external devices. Each SR operating mode is controlled by control bits within the
// ACR.
type viaShifter struct {
	shifterEnabled bool  // True if shifting is enabled
	bitCount       uint8 // Number of bits shifted
	bitShifted     bool  // True if a bit was shifted
	shiftingPhase  bool  // True if we are in shifting phase
	outputBit      bool  //

	configuration *viaShifterConfiguration // Configuration of the shift register

	auxiliaryControlRegister *uint8  // Reference to chip's ACR
	shiftRegister            *uint8  // Referemce to the chip's SR
	interrupts               *viaIFR // Refernce to the chip's IFR
}

// Creates a shifter and attaches it to the specified chip
func createViaShifter(via *Via65C22S, configuration *viaShifterConfiguration) *viaShifter {
	return &viaShifter{
		shifterEnabled: false,
		bitCount:       0,
		bitShifted:     false,
		shiftingPhase:  false,

		configuration: configuration,

		auxiliaryControlRegister: &via.registers.auxiliaryControl,
		shiftRegister:            &via.registers.shiftRegister,
		interrupts:               &via.registers.interrupts,
	}
}

// Returns the direction of shifting based on the chip's configuration on ACR
func (s *viaShifter) getDirection() viaShiftDirection {
	return viaShiftDirection(*s.auxiliaryControlRegister & 0x10)
}

// Gets the shifting mode based on the shipt's configuration on ACR
func (s *viaShifter) getMode() viaShiftRegisterModes {
	return viaShiftRegisterModes(*s.auxiliaryControlRegister & 0x1C)
}

// Returns true is shifting is under T2 control
func (s *viaShifter) isUnderTimerControl() bool {
	mode := s.getMode()
	return (mode == viaShiftInT2 || mode == viaShiftOutT2 || mode == viaShiftOutFree)
}

// Returns true if shifting is under timer control (this shift's at the circuit's frequency)
func (s *viaShifter) isUnderClockControl() bool {
	mode := s.getMode()
	return (mode == viaShiftInClock || mode == viaShiftOutClock)
}

// Returns true if shift is under external control (using control lines)
func (s *viaShifter) isUnderExternalControl() bool {
	mode := s.getMode()
	return (mode == viaShiftInExternal || mode == viaShiftOutExternal)
}

// Enables or disables bit 0 of the shift register (this is used in shifting)
func (s *viaShifter) writeShiftRegisterBitZero(bitEnabled bool) {
	if bitEnabled {
		*s.shiftRegister |= 0x01
	} else {
		*s.shiftRegister &= 0xFE
	}
}

// Rotates SR to the left (all bits are move one position to the left)
// It also increases the internal counters and set the flag that a new bit was shifted
func (s *viaShifter) rotateLeftShiftRegister() {
	*s.shiftRegister = *s.shiftRegister << 1
	s.bitShifted = true
	s.bitCount++
}

// Sets if we are in the shifting phase
func (s *viaShifter) setShiftingPhase(isShiftingPhase bool) {
	s.bitShifted = false
	s.shiftingPhase = isShiftingPhase
}

// Checks if a full byte was shifted. If the parameter is true,
// it will set the CB1 line to high when shifting is completed
// Also, when shifting is completed it updates the interrupt flag in the
// IFR
func (s *viaShifter) checkBitCounter(driveCB1 bool) {
	if s.bitCount == 8 {
		s.shifterEnabled = false
		s.interrupts.setInterruptFlagBit(irqSR)

		if driveCB1 {
			s.configuration.controlLines.lines[0].SetEnable(true)
		}
	}
}

// Executes one emulation step
func (s *viaShifter) tick() {
	if s.shiftingPhase {
		if s.getDirection() == viaShiftIn {
			// Rotate the shift register once
			if !s.bitShifted {
				s.rotateLeftShiftRegister()
			}

			// While on shifting phase keep updating the bit zero from CB2
			s.writeShiftRegisterBitZero(s.configuration.controlLines.lines[1].Enabled())
		} else {
			if !s.bitShifted {
				// Read bit 8, rotate shift register and set bit 8 on bit 0
				s.outputBit = (*s.shiftRegister & 0x80) == 0x80
				s.rotateLeftShiftRegister()
				s.writeShiftRegisterBitZero(s.outputBit)
			}
		}
	}
}

// Sets the control lines based on the status of the shifter.
func (s *viaShifter) setControlLinesBasedOnShifterStatus() {
	mode := s.getMode()

	// If free mode, disable the bit counter
	if mode == viaShiftOutFree {
		s.bitCount = 0
	}

	// If mode has shifter disabled, disable the flag
	if mode == viaShiftDisabled {
		s.shifterEnabled = false
	}

	if s.shifterEnabled {
		// When not under external control CB1 is set low on the shifting phase
		if !s.isUnderExternalControl() {
			s.configuration.controlLines.lines[0].SetEnable(!s.shiftingPhase)
		}

		// When shifting out, the output in CB2 is driven to the opposite of ouptut bit or
		// high if outside the shifting phase
		if s.getDirection() == viaShiftOut {
			s.configuration.controlLines.lines[1].SetEnable(!s.outputBit || !s.shiftingPhase)
		}

		switch {
		case s.isUnderTimerControl():
			// If is under internal timer control and has counted to zero
			if s.configuration.timer.hasCountedToZeroLow {
				s.resetTimer()

				// Toggle shifting phase and check if bit counter has counter
				// to zero. Sets the control line accordingly
				s.setShiftingPhase(!s.shiftingPhase)
				s.checkBitCounter(true)
			}
		case s.isUnderClockControl():
			// Toggle shifting phase and check if bit counter has counter
			// to zero. Sets the control line accordingly
			s.setShiftingPhase(!s.shiftingPhase)
			s.checkBitCounter(true)

		case s.isUnderExternalControl():
			// If we were not shifting and the line drops it means that
			// it transitioned to shifting
			if !s.shiftingPhase && !s.configuration.controlLines.lines[0].Enabled() {
				s.setShiftingPhase(true)
			}

			// If we were shifting and control line is not enable we should stop
			if s.shiftingPhase && s.configuration.controlLines.lines[0].Enabled() {
				s.setShiftingPhase(false)
				s.checkBitCounter(false)
			}
		}
	}
}

// Initialize the internal counter
func (s *viaShifter) initCounter() {
	// Documentation is not clear about how many cycles are needed before first shift is triggered.
	// See comments on Tick method of timer struct for clarification.
	// TODO: Might need to test in real hardware.
	*s.configuration.timer.configuration.counter = 0x0001
	s.shiftingPhase = false
}

// Resets the timer
func (s *viaShifter) resetTimer() {
	*s.configuration.timer.configuration.counter &= 0xFF00
	*s.configuration.timer.configuration.counter |= uint16(*s.configuration.timer.configuration.lowLatches)
}
