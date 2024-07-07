package cpu

import "math"

type StatusRegister uint8

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

func (status StatusRegister) Flag(bit StatusBit) bool {
	mask := uint8(math.Pow(2, float64(bit)))

	return (uint8(status) & mask) > 0
}

func (status *StatusRegister) SetFlag(bit StatusBit, set bool) {
	mask := uint8(math.Pow(2, float64(bit)))

	if set {
		*status = StatusRegister(uint8(*status) | (0x00 + mask))
	} else {
		*status = StatusRegister(uint8(*status) & (0xFF - mask))
	}
}
