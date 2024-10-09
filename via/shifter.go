package via

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
	shiftingIn     bool

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
		shiftingIn:     false,

		configuration: configuration,

		auxiliaryControlRegister: &via.registers.auxiliaryControl,
		shiftRegister:            &via.registers.shiftRegister,
		interrupts:               &via.registers.interrupts,
	}
}

/*
func (s *viaShifter) isInputMode() bool {
	return (*s.auxiliaryControlRegister & 0x10) == 0x10
}*/

func (s *viaShifter) getMode() viaShiftRegisterModes {
	return viaShiftRegisterModes(*s.auxiliaryControlRegister & 0x1C)
}

func (s *viaShifter) isUnderTimerControl() bool {
	mode := s.getMode()
	return (mode == viaShiftInT2 || mode == viaShiftOutT2)
}

func (s *viaShifter) isUnderClockControl() bool {
	mode := s.getMode()
	return (mode == viaShiftInClock || mode == viaShiftOutClock)
}

func (s *viaShifter) tick() {
	if s.shiftingIn {
		if !s.bitShifted {
			*s.shiftRegister = *s.shiftRegister << 1
			s.bitShifted = true
			s.bitCount++
		}

		if s.configuration.controlLines.lines[1].Enabled() {
			*s.shiftRegister |= 0x01
		} else {
			*s.shiftRegister &= 0xFE
		}
	}
}

func (s *viaShifter) writeShifterOutput() {
	mode := s.getMode()

	if s.shifterEnabled {
		if s.shiftingIn {
			s.configuration.controlLines.lines[0].SetEnable(false)
		} else {
			s.configuration.controlLines.lines[0].SetEnable(true)
		}
	}

	if s.shifterEnabled {
		switch {
		case mode == viaShiftDisabled:
			s.shifterEnabled = false
		case s.isUnderTimerControl():
			if s.configuration.timer.hasCountedToZeroLow {
				s.resetTimer()
				s.bitShifted = false
				s.shiftingIn = !s.shiftingIn

				if s.bitCount == 8 {
					s.shifterEnabled = false
					s.interrupts.setInterruptFlagBit(irqSR)
					s.configuration.controlLines.lines[0].SetEnable(true)
				}
			}
		case s.isUnderClockControl():
			s.shiftingIn = !s.shiftingIn
		}
	}
}

func (s *viaShifter) initCounter() {
	// Documentation is not clear about how many cycles are needed before first shift is triggered.
	// See comments on Tick method of timer struct for clarification.
	// TODO: Might need to test in real hardware.
	*s.configuration.timer.configuration.counter = 0x0001
	s.shiftingIn = false
}

func (s *viaShifter) resetTimer() {
	if s.isUnderTimerControl() {
		*s.configuration.timer.configuration.counter &= 0xFF00
		*s.configuration.timer.configuration.counter |= uint16(*s.configuration.timer.configuration.lowLatches)
	}
}
