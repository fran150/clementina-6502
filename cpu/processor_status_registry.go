package cpu

import "math"

type ProcessorStatusRegistry uint8
type ProcessorStatusBit uint8

const (
	CarryFlagBit        = 0
	ZeroFlagBit         = 1
	IrqDisableFlagBit   = 2
	DecimalModeFlagBit  = 3
	BreakCommandFlagBit = 4
	OverflowFlagBit     = 6
	NegativeFlagBit     = 7
)

func (status ProcessorStatusRegistry) Flag(bit ProcessorStatusBit) bool {
	mask := uint8(math.Pow(2, float64(bit)))
	return (uint8(status) & mask) > 0
}

func (status *ProcessorStatusRegistry) SetFlag(bit ProcessorStatusBit, set bool) {
	mask := uint8(math.Pow(2, float64(bit)))

	if set {
		*status = ProcessorStatusRegistry(uint8(*status) | (0x00 + mask))
	} else {
		*status = ProcessorStatusRegistry(uint8(*status) & (0xFF - mask))
	}
}
