package mia

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/stretchr/testify/assert"
)

type emulatedMiaTestCircuit struct {
	chip       *emulated_mia
	addressBus buses.Bus[uint8]
	dataBus    buses.Bus[uint8]
	cs         *buses.StandaloneLine
	rw         *buses.StandaloneLine
	reset      *buses.StandaloneLine
	irq        *buses.StandaloneLine
	step       common.StepContext
}

// newEmulatedMiaTestCircuit wires an emulated MIA into standalone test buses.
func newEmulatedMiaTestCircuit() *emulatedMiaTestCircuit {
	chip := NewEmulatedMia().(*emulated_mia)
	circuit := &emulatedMiaTestCircuit{
		chip:       chip,
		addressBus: buses.New8BitStandaloneBus(),
		dataBus:    buses.New8BitStandaloneBus(),
		cs:         buses.NewStandaloneLine(true),
		rw:         buses.NewStandaloneLine(true),
		reset:      buses.NewStandaloneLine(true),
		irq:        buses.NewStandaloneLine(true),
		step:       common.NewStepContext(),
	}

	chip.AddressBus().Connect(circuit.addressBus)
	chip.DataBus().Connect(circuit.dataBus)
	chip.MiaCS().Connect(circuit.cs)
	chip.WriteEnable().Connect(circuit.rw)
	chip.Reset().Connect(circuit.reset)
	chip.Irq().Connect(circuit.irq)

	return circuit
}

// read performs a selected CPU read cycle against the emulated MIA.
func (c *emulatedMiaTestCircuit) read(address uint8) uint8 {
	c.cs.Set(true)
	c.rw.Set(true)
	c.addressBus.Write(address)
	c.chip.Tick(&c.step)

	return c.dataBus.Read()
}

// write performs a selected CPU write cycle against the emulated MIA.
func (c *emulatedMiaTestCircuit) write(address uint8, value uint8) {
	c.cs.Set(true)
	c.rw.Set(false)
	c.addressBus.Write(address)
	c.dataBus.Write(value)
	c.chip.Tick(&c.step)
	c.rw.Set(true)
}

// TestEmulatedMiaFastLoaderInitialization verifies Pico fast-loader register seeding.
func TestEmulatedMiaFastLoaderInitialization(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip

	assert.Equal(t, miaStateLoader, chip.state)
	assert.Equal(t, uint32(1), chip.kernelIndex)

	expected := map[uint8]uint8{
		miaRegIdxAPort:       0xA9,
		miaRegIdxASelector:   miaKernelData[0],
		miaRegCfgPort:        0x8D,
		miaRegCfgSelector:    0x00,
		miaRegIdxBPort:       0x40,
		miaRegIdxBSelector:   0x8D,
		miaRegCmdParam1:      0xF1,
		miaRegCmdParam2:      0xFF,
		miaRegCmdParam3:      0x80,
		miaRegCmdTrigger:     0xF6,
		miaRegStatusLSB:      0x4C,
		miaRegStatusMSB:      0x00,
		miaRegErrorLSB:       0x40,
		miaRegIRQStatusLSB:   0x00,
		miaRegIRQStatusMSB:   0x80,
		miaRegResetVectorLSB: 0xE0,
		miaRegResetVectorMSB: 0xFF,
		miaRegIRQVectorLSB:   0x00,
		miaRegIRQVectorMSB:   0x00,
	}

	for address, value := range expected {
		assert.Equalf(t, value, chip.readRegister(address), "register %02X", address)
	}

	assert.Equal(t, uint16(0xFFE0), chip.readRegisterWord(miaRegResetVectorLSB))
	assert.True(t, circuit.irq.Status())
}

// TestEmulatedMiaResetRestoresFastLoader verifies reset exits normal mode and re-seeds loader registers.
func TestEmulatedMiaResetRestoresFastLoader(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip

	chip.state = miaStateNormal
	chip.canUpdateKernelPointer = true
	chip.indexes[0].currentAddr = 0x1234
	chip.writeRegister(miaRegIdxAPort, 0x00)
	chip.writeRegister(miaRegIdxASelector, 0x00)
	chip.writeRegisterWord(miaRegResetVectorLSB, 0x0000)

	assert.Equal(t, uint8(0x00), circuit.read(miaRegIdxAPort))

	circuit.reset.Set(false)
	chip.Tick(&circuit.step)
	circuit.reset.Set(true)

	assert.Equal(t, miaStateLoader, chip.state)
	assert.False(t, chip.canUpdateKernelPointer)
	assert.Equal(t, uint8(0xA9), circuit.read(miaRegIdxAPort))
	assert.Equal(t, miaKernelData[0], circuit.read(miaRegIdxASelector))
	assert.Equal(t, uint16(0xFFE0), chip.readRegisterWord(miaRegResetVectorLSB))
}

// TestEmulatedMiaDoesNotDriveDataBusWhenChipIsInactive verifies inactive CS behavior.
func TestEmulatedMiaDoesNotDriveDataBusWhenChipIsInactive(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()

	circuit.cs.Set(false)
	circuit.rw.Set(true)
	circuit.addressBus.Write(miaRegIdxAPort)
	circuit.dataBus.Write(0x55)
	circuit.chip.Tick(&circuit.step)

	assert.Equal(t, uint8(0x55), circuit.dataBus.Read())
}

// TestEmulatedMiaReadsAndWritesRegisters verifies basic register window access.
func TestEmulatedMiaReadsAndWritesRegisters(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()

	circuit.write(miaRegReserved12, 0xAB)

	assert.Equal(t, uint8(0xAB), circuit.read(miaRegReserved12))
}

// TestEmulatedMiaPeekReadsRegistersWithoutSideEffects verifies debugger register access.
func TestEmulatedMiaPeekReadsRegistersWithoutSideEffects(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip

	chip.writeRegister(miaRegIdxASelector, 0x42)
	chip.canUpdateKernelPointer = false

	assert.Equal(t, uint8(0x42), chip.Peek(0xFFE1))
	assert.False(t, chip.canUpdateKernelPointer)
}

// TestEmulatedMiaLoaderAdvancesKernelAfterReadThenWrite verifies boot-loader advancement.
func TestEmulatedMiaLoaderAdvancesKernelAfterReadThenWrite(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip

	assert.Equal(t, miaKernelData[0], circuit.read(miaRegIdxASelector))
	circuit.write(miaRegIRQStatusMSB, 0x00)

	assert.Equal(t, miaKernelData[1], chip.readRegister(miaRegIdxASelector))
	assert.Equal(t, uint16(0x4001), chip.readRegisterWord(miaRegCfgSelector))

	circuit.write(miaRegIRQStatusMSB, 0x00)

	assert.Equal(t, miaKernelData[1], chip.readRegister(miaRegIdxASelector))
	assert.Equal(t, uint16(0x4001), chip.readRegisterWord(miaRegCfgSelector))

	for chip.state == miaStateLoader {
		circuit.read(miaRegIdxASelector)
		circuit.write(miaRegIRQStatusMSB, 0x00)
	}

	assert.Equal(t, miaStateNormal, chip.state)
	assert.Equal(t, uint32(len(miaKernelData)), chip.kernelIndex)
	assert.Equal(t, uint8(0x00), chip.readRegister(miaRegCmdTrigger))
	assert.Equal(t, uint8(0x4C), chip.readRegister(miaRegStatusLSB))
}

// TestEmulatedMiaLoaderStoresFinalKernelByteBeforeExiting verifies the final byte is still looped through.
func TestEmulatedMiaLoaderStoresFinalKernelByteBeforeExiting(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip

	for range len(miaKernelData) - 1 {
		circuit.read(miaRegIdxASelector)
		circuit.write(miaRegIRQStatusMSB, 0x00)
	}

	assert.Equal(t, miaStateLoader, chip.state)
	assert.Equal(t, uint32(len(miaKernelData)), chip.kernelIndex)
	assert.Equal(t, miaKernelData[len(miaKernelData)-1], chip.readRegister(miaRegIdxASelector))
	assert.Equal(t, uint8(0xF6), chip.readRegister(miaRegCmdTrigger))
	assert.Equal(t, uint16(miaKernelTargetAddress+len(miaKernelData)-1), chip.readRegisterWord(miaRegCfgSelector))

	circuit.read(miaRegIdxASelector)
	circuit.write(miaRegIRQStatusMSB, 0x00)

	assert.Equal(t, miaStateNormal, chip.state)
	assert.Equal(t, uint8(0x00), chip.readRegister(miaRegCmdTrigger))
}

// TestEmulatedMiaIndexedPortReadStepsAfterCpuReceivesCurrentValue verifies read step timing.
func TestEmulatedMiaIndexedPortReadStepsAfterCpuReceivesCurrentValue(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip
	chip.state = miaStateNormal

	chip.indexes[3] = miaIndex{
		currentAddr: 0,
		limitAddr:   3,
		step:        1,
		flags:       (1 << miaIndexFlagReadStep) | (1 << miaIndexFlagStepDir),
	}
	chip.memory[0] = 0x11
	chip.memory[1] = 0x22

	circuit.write(miaRegIdxASelector, 3)

	assert.Equal(t, uint8(0x11), circuit.read(miaRegIdxAPort))
	assert.Equal(t, uint32(1), chip.indexes[3].currentAddr)
	assert.Equal(t, uint8(0x22), chip.readRegister(miaRegIdxAPort))
	assert.Equal(t, uint8(0x22), circuit.read(miaRegIdxAPort))
}

// TestEmulatedMiaIndexedPortWriteStoresThenSteps verifies write-through indexed port behavior.
func TestEmulatedMiaIndexedPortWriteStoresThenSteps(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip
	chip.state = miaStateNormal

	chip.indexes[4] = miaIndex{
		currentAddr: 0,
		limitAddr:   3,
		step:        1,
		flags:       (1 << miaIndexFlagWriteStep) | (1 << miaIndexFlagStepDir),
	}
	chip.memory[1] = 0xBB

	circuit.write(miaRegIdxASelector, 4)
	circuit.write(miaRegIdxAPort, 0xAA)

	assert.Equal(t, uint8(0xAA), chip.memory[0])
	assert.Equal(t, uint32(1), chip.indexes[4].currentAddr)
	assert.Equal(t, uint8(0xBB), chip.readRegister(miaRegIdxAPort))
}

// TestEmulatedMiaDMACommandCopiesMemoryAndQueuesErrors verifies DMA command side effects.
func TestEmulatedMiaDMACommandCopiesMemoryAndQueuesErrors(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip
	chip.state = miaStateNormal

	chip.indexes[1].currentAddr = 10
	chip.indexes[2].currentAddr = 20
	chip.memory[10] = 0xDE
	chip.memory[11] = 0xAD

	circuit.write(miaRegCmdParam1, 1)
	circuit.write(miaRegCmdParam2, 2)
	circuit.write(miaRegCmdParam3, 2)
	circuit.write(miaRegCmdTrigger, 0x10)

	assert.Equal(t, []uint8{0xDE, 0xAD}, chip.memory[20:22])
	assert.Zero(t, chip.status()&miaStatusDMARunning)

	circuit.write(miaRegCmdParam3, 0)
	circuit.write(miaRegCmdTrigger, 0x10)

	assert.Equal(t, miaStatusErrors, chip.status()&miaStatusErrors)
	assert.Equal(t, miaErrorDMASizeZero, chip.errors.Pull(chip))
}
