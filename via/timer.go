package via

type ViaTimer struct {
	side *ViaSide

	timerEnabled            bool
	outputStatusWhenEnabled bool

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

func (t *ViaTimer) tick() {
	if t.getRunningMode() != t2RunModePulseCounting {
		t.side.registers.counter -= 1
	}

	if t.side.registers.counter == 0xFFFE {
		if t.getRunningMode() == txRunModeOneShot {
			t.timerEnabled = false
		}

		if t.getRunningMode() == t1RunModeFree {
			t.side.registers.counter = uint16(t.side.registers.lowLatches)
			t.side.registers.counter |= (uint16(t.side.registers.highLatches) << 8)
		}
	}
}

func (t *ViaTimer) isTimerOutputEnabled() bool {
	return (*t.auxiliaryControlRegister & uint8(t.side.configuration.timerOutputMask)) > 0
}

func (t *ViaTimer) getRunningMode() viaTimerRunningMode {
	return viaTimerRunningMode(*t.auxiliaryControlRegister & uint8(t.side.configuration.timerRunModeMask))
}

func (t *ViaTimer) timerInterruptHandling() {
	if t.side.registers.counter == 0xFFFF && t.timerEnabled {
		t.interrupts.setInterruptFlagBit(t.side.configuration.timerInterruptBit)
	}
}
