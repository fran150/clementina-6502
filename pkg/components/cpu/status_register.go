package cpu

// StatusRegister represents the processor status register in the 6502 CPU.
// It contains flags that record the results of operations and control CPU behavior.
// See https://www.6502.org/users/obelisk/6502/registers.html
type StatusRegister uint8

// StatusBit defines the bit positions for each flag in the status register.
// These constants are used to access and modify individual flags.
type StatusBit uint8

const (
	CarryFlagBit        StatusBit = 0 // Carry flag (C)
	ZeroFlagBit         StatusBit = 1 // Zero flag (Z)
	IrqDisableFlagBit   StatusBit = 2 // Interrupt disable flag (I)
	DecimalModeFlagBit  StatusBit = 3 // Decimal mode flag (D)
	BreakCommandFlagBit StatusBit = 4 // Break command flag (B)
	UnusedFlagBit       StatusBit = 5 // Unused flag (always set to 1)
	OverflowFlagBit     StatusBit = 6 // Overflow flag (V)
	NegativeFlagBit     StatusBit = 7 // Negative flag (N)
)

// NewStatusRegister creates a new status register with the specified initial value.
// The BRK (B) and unused (U) flags are always set to 1, regardless of the input value.
func NewStatusRegister(value uint8) StatusRegister {
	return StatusRegister(value | 0x30)
}

// Returns whether the specified bit of the status register is set
func (status StatusRegister) Flag(bit StatusBit) bool {
	mask := uint8(1 << bit)

	return (uint8(status) & mask) > 0
}

// Allows to set or unset an specific bit of the status register
func (status *StatusRegister) SetFlag(bit StatusBit, set bool) {
	mask := uint8(1 << bit)

	if set {
		*status = StatusRegister(uint8(*status) | (0x00 + mask))
	} else {
		*status = StatusRegister(uint8(*status) & (0xFF - mask))
	}
}

// Sets an explicit value for the status register. The value of BRK (B) and Unused flags will
// always be set to 1 as they cannot be changed
func (status *StatusRegister) SetValue(value uint8) {
	*status = StatusRegister(value) | 0x30
}

// Returns the byte value of the status register.
func (status *StatusRegister) ReadValue() uint8 {
	return uint8(*status) | 0x30
}
