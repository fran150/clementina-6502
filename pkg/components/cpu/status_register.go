package cpu

import "github.com/fran150/clementina-6502/pkg/components"

// statusRegister represents the processor status register in the 6502 CPU.
// It contains flags that record the results of operations and control CPU behavior.
// See https://www.6502.org/users/obelisk/6502/registers.html
type statusRegister uint8

const (
	CarryFlagBit        components.StatusBit = 0 // Carry flag (C)
	ZeroFlagBit         components.StatusBit = 1 // Zero flag (Z)
	IrqDisableFlagBit   components.StatusBit = 2 // Interrupt disable flag (I)
	DecimalModeFlagBit  components.StatusBit = 3 // Decimal mode flag (D)
	BreakCommandFlagBit components.StatusBit = 4 // Break command flag (B)
	UnusedFlagBit       components.StatusBit = 5 // Unused flag (always set to 1)
	OverflowFlagBit     components.StatusBit = 6 // Overflow flag (V)
	NegativeFlagBit     components.StatusBit = 7 // Negative flag (N)
)

// NewStatusRegister creates a new status register with the specified initial value.
// The BRK (B) and unused (U) flags are always set to 1, regardless of the input value.
func NewStatusRegister(value uint8) components.StatusRegister {
	return newStatusRegister(value)
}

func newStatusRegister(value uint8) statusRegister {
	return statusRegister(value | 0x30)
}

// Returns whether the specified bit of the status register is set
func (status statusRegister) Flag(bit components.StatusBit) bool {
	mask := uint8(1 << bit)

	return (uint8(status) & mask) > 0
}

// Allows to set or unset an specific bit of the status register
func (status *statusRegister) SetFlag(bit components.StatusBit, set bool) {
	mask := uint8(1 << bit)

	if set {
		*status = statusRegister(uint8(*status) | (0x00 + mask))
	} else {
		*status = statusRegister(uint8(*status) & (0xFF - mask))
	}
}

// Sets an explicit value for the status register. The value of BRK (B) and Unused flags will
// always be set to 1 as they cannot be changed
func (status *statusRegister) SetValue(value uint8) {
	*status = statusRegister(value) | 0x30
}

// Returns the byte value of the status register.
func (status *statusRegister) ReadValue() uint8 {
	return uint8(*status) | 0x30
}
