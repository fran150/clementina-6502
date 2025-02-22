package cpu

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstructionsGetters(t *testing.T) {
	instructionSet := NewInstructionSet()
	loadAccumulatorInstruction := instructionSet.GetByOpCode(0xA9)

	assert.Equal(t, OpCode(0xA9), loadAccumulatorInstruction.OpCode())
	assert.Equal(t, LDA, loadAccumulatorInstruction.Mnemonic())
	assert.Equal(t, AddressModeImmediate, loadAccumulatorInstruction.AddressMode())
}
