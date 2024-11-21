package cpu

import "math"

// As instructions are executed a set of processor flags are set or clear to record the results of the operation. T
// his flags and some additional control flags are held in a special status register. Each flag has a single bit within the register.
// See https://www.6502.org/users/obelisk/6502/registers.html
type StatusRegister uint8

// Name of the bits in the status register
type StatusBit uint8

const (
	CarryFlagBit        StatusBit = 0
	ZeroFlagBit         StatusBit = 1
	IrqDisableFlagBit   StatusBit = 2
	DecimalModeFlagBit  StatusBit = 3
	BreakCommandFlagBit StatusBit = 4
	UnusedFlagBit       StatusBit = 5
	OverflowFlagBit     StatusBit = 6
	NegativeFlagBit     StatusBit = 7
)

// Creates the status register with it's default value. The BRK (B) and unused (U) flag are always set to 1.
func CreateStatusRegister(value uint8) StatusRegister {
	return StatusRegister(value | 0x30)
}

// Returns whether the specified bit of the status register is set
func (status StatusRegister) Flag(bit StatusBit) bool {
	mask := uint8(math.Pow(2, float64(bit)))

	return (uint8(status) & mask) > 0
}

// Allows to set or unset an specific bit of the status register
func (status *StatusRegister) SetFlag(bit StatusBit, set bool) {
	mask := uint8(math.Pow(2, float64(bit)))

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
