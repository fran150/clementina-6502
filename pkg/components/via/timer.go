package via

// Configuration of the timer. This allows to use the same code to
// both timers T1 and T2
type viaTimerConfiguration struct {
	timerInterruptBit viaIRQFlags         // IFR Bit to set when the timer counted to zero
	timerRunModeMask  viaTimerControlMask // Mask used to get the running mode
	timerOutputMask   viaTimerControlMask // Mask used to get the output mode
	lowLatches        *uint8              // Reference to low level latches
	highLatches       *uint8              // Reference to high level latches
	counter           *uint16             // Reference to 16-bit counter
	port              *viaPort            // Reference to chip's port associated with the timer
}

// Interval Timer (T1 and T2) consists of one 8-bit latch and a 16-bit counter. The latches serve to store data
// which is to be loaded into the counter. Once the counter is loaded under program control, it decrements at
// clock rate. Upon reaching zero, corresponding bit in IFR is set.
type viaTimer struct {
	timerEnabled                 bool // True if timer is enabled
	line7OutputStatusWhenEnabled bool // Returns the status of port bit line 7 when chip is enabled
	hasCountedToZero             bool // True if timer has counted to zero in the previous cycle
	hasCountedToZeroLow          bool // True if timer LSB has counted to zero in the previous cycle

	configuration *viaTimerConfiguration // Timer configuration

	auxiliaryControlRegister *uint8  // Reference to chip's ACR
	interrupts               *viaIFR // Reference to chip's IFR
}

// Creates a timer and attaches it to the specified via chip
func newViaTimer(via *Via65C22S, configuration *viaTimerConfiguration) *viaTimer {
	return &viaTimer{
		timerEnabled:                 false,
		line7OutputStatusWhenEnabled: false,
		hasCountedToZero:             false,
		hasCountedToZeroLow:          false,

		configuration: configuration,

		auxiliaryControlRegister: &via.registers.auxiliaryControl,
		interrupts:               &via.registers.interrupts,
	}
}

// Executes one simulation step.
func (t *viaTimer) tick(pbLine6Status bool) {
	if t.getRunningMode() != acrT2RunModePulseCounting || !pbLine6Status {
		*t.configuration.counter -= 1
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

		if t.getRunningMode() == acrTxRunModeOneShot {
			t.timerEnabled = false
		}

		if t.getRunningMode() == acrT1RunModeFree {
			*t.configuration.counter = uint16(*t.configuration.lowLatches)
			*t.configuration.counter |= (uint16(*t.configuration.highLatches) << 8)
		}
	} else {
		t.hasCountedToZero = false
	}
}

// Returns true if timer has output enabled
func (t *viaTimer) isTimerOutputEnabled() bool {
	return (*t.auxiliaryControlRegister & uint8(t.configuration.timerOutputMask)) > 0
}

// Returns the configured running mode of the chip
func (t *viaTimer) getRunningMode() viaTimerRunningMode {
	return viaTimerRunningMode(*t.auxiliaryControlRegister & uint8(t.configuration.timerRunModeMask))
}

// Sets the control lines if timer is in output mode
func (t *viaTimer) setControlLinesBasedOnTimerStatus() {
	// From the manual: With the output enabled (ACR7=1) a "write T1C-H operation will cause PB7 to go low.
	// I'm assuming that setting ACR7=1 with timer not running will cause PB7 to go high
	if t.isTimerOutputEnabled() {
		if !t.timerEnabled {
			t.configuration.port.connector.GetLine(7).Set(true)
		} else {
			if t.hasCountedToZero {
				if t.getRunningMode() == acrT1RunModeFree {
					t.line7OutputStatusWhenEnabled = !t.line7OutputStatusWhenEnabled
					t.configuration.port.connector.GetLine(7).Set(t.line7OutputStatusWhenEnabled)
				}
			} else {
				t.configuration.port.connector.GetLine(7).Set(t.line7OutputStatusWhenEnabled)
			}
		}
	}
}
