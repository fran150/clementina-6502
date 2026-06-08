package clementina

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
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
	tickComputer(computer, &step)
	computer.Reset(false)

	assert.Equal(t, [2]uint8{0xA9, 0xA9}, computer.getPotentialOperators(0xFFE0))
}

// TestClementinaResetIsDrivenByMia verifies the computer reset input goes through MIA.
func TestClementinaResetIsDrivenByMia(t *testing.T) {
	computer, err := NewClementinaComputer()
	require.NoError(t, err)

	step := common.NewStepContext()

	computer.Reset(true)
	assert.False(t, computer.circuit.miaResetRequest.Status())
	assert.True(t, computer.circuit.cpuReset.Status())

	tickComputer(computer, &step)
	assert.False(t, computer.circuit.cpuReset.Status())

	computer.Reset(false)
	for range 4 {
		tickComputer(computer, &step)
		step.NextCycle()
	}

	assert.True(t, computer.circuit.miaResetRequest.Status())
	assert.True(t, computer.circuit.cpuReset.Status())
}

// TestClementinaResetFetchesMiaLoaderOpcode verifies the CPU fetches the loader after reset.
func TestClementinaResetFetchesMiaLoaderOpcode(t *testing.T) {
	computer, err := NewClementinaComputer()
	require.NoError(t, err)

	step := common.NewStepContext()
	writeMiaByte(computer, &step, 0xFFE0, 0x00)

	computer.Reset(true)
	for range 3 {
		tickComputer(computer, &step)
		step.NextCycle()
	}
	computer.Reset(false)

	for range 20 {
		computer.Tick(&step)

		if computer.circuit.addressBus.Read() == 0xFFE0 && computer.chips.cpu.IsReadingOpcode() {
			assert.Equal(t, uint8(0xA9), computer.circuit.dataBus.Read())
			return
		}

		computer.PostTick(&step)
		step.NextCycle()
	}

	t.Fatal("CPU did not fetch an opcode from the MIA reset vector")
}

// TestClementinaKernelVideoLoopLoads verifies the loader writes the video bootstrap.
func TestClementinaKernelVideoLoopLoads(t *testing.T) {
	computer, err := NewClementinaComputer()
	require.NoError(t, err)

	step := common.NewStepContext()

	computer.Reset(true)
	for range 3 {
		tickComputer(computer, &step)
		step.NextCycle()
	}
	computer.Reset(false)

	for range 6000 {
		tickComputer(computer, &step)

		if computer.chips.baseram.Peek(0x408C) == 0x4C &&
			computer.chips.baseram.Peek(0x408D) == 0x72 &&
			computer.chips.baseram.Peek(0x408E) == 0x40 {
			assert.Equal(t, uint8(0xA9), computer.chips.baseram.Peek(0x4000))
			assert.Equal(t, uint8(0x40), computer.chips.baseram.Peek(0x4001))
			assert.Equal(t, uint8(0x8D), computer.chips.baseram.Peek(0x4002))
			assert.Equal(t, uint8(0xF0), computer.chips.baseram.Peek(0x4073))
			assert.Equal(t, [2]uint8{0x72, 0x40}, computer.getPotentialOperators(0x408D))
			return
		}

		step.NextCycle()
	}

	t.Fatal("MIA loader did not write the video bootstrap loop")
}

func tickComputer(computer *ClementinaComputer, step *common.StepContext) {
	computer.Tick(step)
	computer.PostTick(step)
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
