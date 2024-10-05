package via

type viaTimerConfiguration struct {
	timerInterruptBit viaIRQFlags
	timerRunModeMask  viaTimerControlMask
	timerOutputMask   viaTimerControlMask
	lowLatches        *uint8
	highLatches       *uint8
	counter           *uint16
}

type ViaTimer struct {
	timerEnabled            bool
	outputStatusWhenEnabled bool
	hasCountedToZero        bool

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
