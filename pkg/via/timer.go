package via

type viaTimerConfiguration struct {
	timerInterruptBit viaIRQFlags
	timerRunModeMask  viaTimerControlMask
	timerOutputMask   viaTimerControlMask
	lowLatches        *uint8
	highLatches       *uint8
	counter           *uint16
	port              *ViaPort
}

type ViaTimer struct {
	timerEnabled            bool
	outputStatusWhenEnabled bool
	hasCountedToZero        bool
	hasCountedToZeroLow     bool

	configuration *viaTimerConfiguration

	auxiliaryControlRegister *uint8
	interrupts               *ViaIFR
}

type viaTimerControlMask uint8

const (
	t1ControlRunModeMask viaTimerControlMask = 0x40
	t1ControlOutputMask  viaTimerControlMask = 0x80
	t2ControlRunModeMask viaTimerControlMask = 0x20
)

type viaTimerRunningMode uint8

const (
	txRunModeOneShot       viaTimerRunningMode = 0x00
	t1RunModeFree          viaTimerRunningMode = 0x40
	t2RunModePulseCounting viaTimerRunningMode = 0x20
)

func createViaTimer(via *Via65C22S, configuration *viaTimerConfiguration) *ViaTimer {
	return &ViaTimer{
		timerEnabled:            false,
		outputStatusWhenEnabled: false,
		hasCountedToZero:        false,
		hasCountedToZeroLow:     false,

		configuration: configuration,

		auxiliaryControlRegister: &via.registers.auxiliaryControl,
		interrupts:               &via.registers.interrupts,
	}
}

func (t *ViaTimer) tick(pbLine6Status bool) {
	if t.getRunningMode() != t2RunModePulseCounting {
		*t.configuration.counter -= 1
	} else {
		if !pbLine6Status {
			*t.configuration.counter -= 1
		}
	}

	// Counting on the low part of the counter is used mainly by the shift register.
	// https://web.archive.org/web/20160108173129if_/http://archive.6502.org/datasheets/mos_6522_preliminary_nov_1977.pdf is not clear
	// on expected timing of CB1 pulses.
	// https://swh.princeton.edu/~mae412/TEXT/NTRAK2002/20.pdf points to N+2 which seems aligned with behaviour for timers
	// although is not clear on how much time passes between SR R/W and the initial drop of CB1 (first shift)
	// According to WDC W65C22S datasheet (page 22) it seems to drop 1.5 cycles after the SR set.
	// I will use an assumption of 1.5 cycles for preparation and N+2 pulse size.
	// TODO: Might need to validate in real hardware
	if (*t.configuration.counter & 0x00FF) == 0xFE {
		t.hasCountedToZeroLow = true
	} else {
		t.hasCountedToZeroLow = false
	}

	if *t.configuration.counter == 0xFFFE {
		if t.timerEnabled {
			t.hasCountedToZero = true
			t.interrupts.setInterruptFlagBit(t.configuration.timerInterruptBit)
		}

		if t.getRunningMode() == txRunModeOneShot {
			t.timerEnabled = false
		}

		if t.getRunningMode() == t1RunModeFree {
			*t.configuration.counter = uint16(*t.configuration.lowLatches)
			*t.configuration.counter |= (uint16(*t.configuration.highLatches) << 8)
		}
	} else {
		t.hasCountedToZero = false
	}
}

func (t *ViaTimer) isTimerOutputEnabled() bool {
	return (*t.auxiliaryControlRegister & uint8(t.configuration.timerOutputMask)) > 0
}

func (t *ViaTimer) getRunningMode() viaTimerRunningMode {
	return viaTimerRunningMode(*t.auxiliaryControlRegister & uint8(t.configuration.timerRunModeMask))
}

func (t *ViaTimer) writeTimerOutput() {
	// From the manual: With the output enabled (ACR7=1) a "write T1C-H operation will cause PB7 to go low.
	// I'm assuming that setting ACR7=1 with timer not running will cause PB7 to go high
	if t.isTimerOutputEnabled() {
		if !t.timerEnabled {
			t.configuration.port.connector.GetLine(7).Set(true)
		} else {
			if t.hasCountedToZero {
				switch t.getRunningMode() {
				case txRunModeOneShot:
					t.configuration.port.connector.GetLine(7).Set(true)
				case t1RunModeFree:
					t.outputStatusWhenEnabled = !t.outputStatusWhenEnabled
					t.configuration.port.connector.GetLine(7).Set(t.outputStatusWhenEnabled)
				}
			} else {
				t.configuration.port.connector.GetLine(7).Set(t.outputStatusWhenEnabled)
			}
		}
	}
}
