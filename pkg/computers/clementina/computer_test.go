package clementina

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClementinaPotentialOperatorsReadsMiaRange verifies debugger operand lookup in MIA memory.
func TestClementinaPotentialOperatorsReadsMiaRange(t *testing.T) {
	computer, err := NewClementinaComputer()
	require.NoError(t, err)

	assert.Equal(t, [2]uint8{0xA9, 0xA9}, computer.getPotentialOperators(0xFFE0))
	assert.Equal(t, [2]uint8{0xA9, 0x8D}, computer.getPotentialOperators(0xFFE1))
	assert.Equal(t, [2]uint8{0x00, 0x40}, computer.getPotentialOperators(0xFFE3))
}

// TestClementinaPotentialOperatorsReadsAcrossMappedRegions verifies per-byte memory mapping.
func TestClementinaPotentialOperatorsReadsAcrossMappedRegions(t *testing.T) {
	computer, err := NewClementinaComputer()
	require.NoError(t, err)

	computer.BaseRamPoke(0x0000, 0xCA)

	assert.Equal(t, [2]uint8{0x00, 0xCA}, computer.getPotentialOperators(0xFFFF))
}

// TestClementinaResetRestoresMiaLoaderWindow verifies reset re-seeds MIA through computer wiring.
func TestClementinaResetRestoresMiaLoaderWindow(t *testing.T) {
	computer, err := NewClementinaComputer()
	require.NoError(t, err)

	step := common.NewStepContext()

	writeMiaByte(computer, &step, 0xFFE0, 0x00)

	assert.Equal(t, [2]uint8{0x00, 0xA9}, computer.getPotentialOperators(0xFFE0))

	computer.Reset(true)
	computer.Tick(&step)
	computer.Reset(false)

	assert.Equal(t, [2]uint8{0xA9, 0xA9}, computer.getPotentialOperators(0xFFE0))
}

// TestClementinaResetFetchesMiaLoaderOpcode verifies the CPU fetches the loader after reset.
func TestClementinaResetFetchesMiaLoaderOpcode(t *testing.T) {
	computer, err := NewClementinaComputer()
	require.NoError(t, err)

	step := common.NewStepContext()
	writeMiaByte(computer, &step, 0xFFE0, 0x00)

	computer.Reset(true)
	for range 3 {
		computer.Tick(&step)
		step.NextCycle()
	}
	computer.Reset(false)

	for range 20 {
		computer.Tick(&step)

		instruction := computer.chips.cpu.GetCurrentInstruction()
		if computer.circuit.addressBus.Read() == 0xFFE0 && computer.chips.cpu.IsReadingOpcode() {
			require.NotNil(t, instruction)
			assert.Equal(t, components.OpCode(0xA9), instruction.OpCode())
			return
		}

		step.NextCycle()
	}

	t.Fatal("CPU did not fetch an opcode from the MIA reset vector")
}

// TestClementinaKernelJumpOperands verifies the loader writes the full JMP $4002 operand.
func TestClementinaKernelJumpOperands(t *testing.T) {
	computer, err := NewClementinaComputer()
	require.NoError(t, err)

	step := common.NewStepContext()

	computer.Reset(true)
	for range 3 {
		computer.Tick(&step)
		step.NextCycle()
	}
	computer.Reset(false)

	for range 500 {
		computer.Tick(&step)

		if computer.circuit.addressBus.Read() == 0x400C && computer.chips.cpu.IsReadingOpcode() {
			assert.Equal(t, uint8(0x4C), computer.chips.baseram.Peek(0x400C))
			assert.Equal(t, uint8(0x02), computer.chips.baseram.Peek(0x400D))
			assert.Equal(t, uint8(0x40), computer.chips.baseram.Peek(0x400E))
			assert.Equal(t, [2]uint8{0x02, 0x40}, computer.getPotentialOperators(0x400D))
			return
		}

		step.NextCycle()
	}

	t.Fatal("CPU did not reach kernel JMP at $400C")
}

func writeMiaByte(computer *ClementinaComputer, step *common.StepContext, address uint16, value uint8) {
	computer.circuit.addressBus.Write(address)
	computer.circuit.dataBus.Write(value)
	computer.circuit.cpuRW.Set(false)
	computer.chips.csLogic.Tick(step)
	computer.chips.oeRWSync.Tick(step)
	computer.chips.mia.Tick(step)
	computer.circuit.cpuRW.Set(true)
}
