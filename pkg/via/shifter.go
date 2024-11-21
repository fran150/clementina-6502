package via

type viaShiftDirection uint8

const (
	viaShiftIn  viaShiftDirection = 0x00
	viaShiftOut viaShiftDirection = 0x10
)

type viaShiftRegisterModes uint8

const (
	viaShiftDisabled    viaShiftRegisterModes = 0x00
	viaShiftInT2        viaShiftRegisterModes = 0x04
	viaShiftInClock     viaShiftRegisterModes = 0x08
	viaShiftInExternal  viaShiftRegisterModes = 0xC
	viaShiftOutFree     viaShiftRegisterModes = 0x10
	viaShiftOutT2       viaShiftRegisterModes = 0x14
	viaShiftOutClock    viaShiftRegisterModes = 0x18
	viaShiftOutExternal viaShiftRegisterModes = 0x1C
)

type viaShifterConfiguration struct {
	timer        *ViaTimer
	controlLines *viaControlLines
}

type viaShifter struct {
	shifterEnabled bool
	bitCount       uint8
	bitShifted     bool
	shiftingPhase  bool
	outputBit      bool

	configuration *viaShifterConfiguration

	auxiliaryControlRegister *uint8
	shiftRegister            *uint8
	interrupts               *ViaIFR
}

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

func (s *viaShifter) getDirection() viaShiftDirection {
	return viaShiftDirection(*s.auxiliaryControlRegister & 0x10)
}

func (s *viaShifter) getMode() viaShiftRegisterModes {
	return viaShiftRegisterModes(*s.auxiliaryControlRegister & 0x1C)
}

func (s *viaShifter) isUnderTimerControl() bool {
	mode := s.getMode()
	return (mode == viaShiftInT2 || mode == viaShiftOutT2 || mode == viaShiftOutFree)
}

func (s *viaShifter) isUnderClockControl() bool {
	mode := s.getMode()
	return (mode == viaShiftInClock || mode == viaShiftOutClock)
}

func (s *viaShifter) isUnderExternalControl() bool {
	mode := s.getMode()
	return (mode == viaShiftInExternal || mode == viaShiftOutExternal)
}

func (s *viaShifter) writeShiftRegisterBitZero(bitEnabled bool) {
	if bitEnabled {
		*s.shiftRegister |= 0x01
	} else {
		*s.shiftRegister &= 0xFE
	}
}

func (s *viaShifter) rotateLeftShiftRegister() {
	*s.shiftRegister = *s.shiftRegister << 1
	s.bitShifted = true
	s.bitCount++
}

func (s *viaShifter) setShiftingPhase(isShiftingPhase bool) {
	s.bitShifted = false
	s.shiftingPhase = isShiftingPhase
}

func (s *viaShifter) checkBitCounter(driveCB1 bool) {
	if s.bitCount == 8 {
		s.shifterEnabled = false
		s.interrupts.setInterruptFlagBit(irqSR)

		if driveCB1 {
			s.configuration.controlLines.lines[0].SetEnable(true)
		}
	}
}

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

func (s *viaShifter) writeShifterOutput() {
	mode := s.getMode()

	// If free mode, disable the bit counter
	if mode == viaShiftOutFree {
		s.bitCount = 0
	}

	// If mode has shifter disabled, disable the flat
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
			if s.configuration.timer.hasCountedToZeroLow {
				s.resetTimer()

				s.setShiftingPhase(!s.shiftingPhase)
				s.checkBitCounter(true)
			}
		case s.isUnderClockControl():
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

func (s *viaShifter) initCounter() {
	// Documentation is not clear about how many cycles are needed before first shift is triggered.
	// See comments on Tick method of timer struct for clarification.
	// TODO: Might need to test in real hardware.
	*s.configuration.timer.configuration.counter = 0x0001
	s.shiftingPhase = false
}

func (s *viaShifter) resetTimer() {
	*s.configuration.timer.configuration.counter &= 0xFF00
	*s.configuration.timer.configuration.counter |= uint16(*s.configuration.timer.configuration.lowLatches)
}
