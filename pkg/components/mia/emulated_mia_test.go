package mia

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/stretchr/testify/assert"
)

type emulatedMiaTestCircuit struct {
	chip         *emulated_mia
	addressBus   buses.Bus[uint8]
	dataBus      buses.Bus[uint8]
	cs           *buses.StandaloneLine
	rw           *buses.StandaloneLine
	reset        *buses.StandaloneLine
	resetRequest *buses.StandaloneLine
	irq          *buses.StandaloneLine
	step         common.StepContext
}

// newEmulatedMiaTestCircuit wires an emulated MIA into standalone test buses.
func newEmulatedMiaTestCircuit() *emulatedMiaTestCircuit {
	chip := NewEmulatedMia().(*emulated_mia)
	circuit := &emulatedMiaTestCircuit{
		chip:         chip,
		addressBus:   buses.New8BitStandaloneBus(),
		dataBus:      buses.New8BitStandaloneBus(),
		cs:           buses.NewStandaloneLine(true),
		rw:           buses.NewStandaloneLine(true),
		reset:        buses.NewStandaloneLine(true),
		resetRequest: buses.NewStandaloneLine(true),
		irq:          buses.NewStandaloneLine(true),
		step:         common.NewStepContext(),
	}

	chip.AddressBus().Connect(circuit.addressBus)
	chip.DataBus().Connect(circuit.dataBus)
	chip.MiaCS().Connect(circuit.cs)
	chip.WriteEnable().Connect(circuit.rw)
	chip.Reset().Connect(circuit.reset)
	chip.ResetRequest().Connect(circuit.resetRequest)
	chip.Irq().Connect(circuit.irq)
	circuit.idle(miaCPUResetPulseCycles + 1)

	return circuit
}

// idle advances MIA service cycles without selecting the chip.
func (c *emulatedMiaTestCircuit) idle(cycles int) {
	c.cs.Set(false)
	for range cycles {
		c.chip.Tick(&c.step)
		c.step.NextCycle()
	}
	c.cs.Set(true)
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
		0x02:                 0x8D,
		0x03:                 0x00,
		miaRegIdxBPort:       0x40,
		miaRegIdxBSelector:   0x8D,
		miaRegCmdParam1:      0xF1,
		miaRegCmdParam2:      0xFF,
		miaRegCmdParam3:      0x80,
		miaRegCmdTrigger:     0xF6,
		miaRegStatusLSB:      0x4C,
		miaRegStatusMSB:      0xEA,
		miaRegErrorLSB:       0xFF,
		miaRegIRQStatusLSB:   0x00,
		miaRegIRQStatusMSB:   0x00,
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

// TestEmulatedMiaResetRestoresFastLoader verifies MIA reset request exits normal mode and re-seeds loader registers.
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

	circuit.resetRequest.Set(false)
	chip.Tick(&circuit.step)
	circuit.resetRequest.Set(true)

	assert.Equal(t, miaStateLoader, chip.state)
	assert.False(t, chip.canUpdateKernelPointer)
	assert.False(t, circuit.reset.Status())
	assert.Equal(t, uint8(0xA9), chip.readRegister(miaRegIdxAPort))
	assert.Equal(t, miaKernelData[0], chip.readRegister(miaRegIdxASelector))
	assert.Equal(t, uint16(0xFFE0), chip.readRegisterWord(miaRegResetVectorLSB))
}

// TestEmulatedMiaResetRequestPulsesCPUReset verifies MIA drives RESB for enough cycles.
func TestEmulatedMiaResetRequestPulsesCPUReset(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()

	circuit.resetRequest.Set(false)
	circuit.chip.Tick(&circuit.step)

	assert.False(t, circuit.reset.Status())

	circuit.resetRequest.Set(true)
	for range miaCPUResetPulseCycles - 1 {
		circuit.chip.Tick(&circuit.step)
		assert.False(t, circuit.reset.Status())
	}

	circuit.chip.Tick(&circuit.step)
	assert.True(t, circuit.reset.Status())
}

// TestEmulatedMiaHeldResetRequestKeepsCPUResetAsserted verifies held input holds RESB low.
func TestEmulatedMiaHeldResetRequestKeepsCPUResetAsserted(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()

	circuit.resetRequest.Set(false)
	for range miaCPUResetPulseCycles + 2 {
		circuit.chip.Tick(&circuit.step)
		assert.False(t, circuit.reset.Status())
	}
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

	circuit.write(miaRegReserved15, 0xAB)

	assert.Equal(t, uint8(0xAB), circuit.read(miaRegReserved15))
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
	assert.Equal(t, uint16(0x4001), chip.readRegisterWord(0x03))

	circuit.write(miaRegIRQStatusMSB, 0x00)

	assert.Equal(t, miaKernelData[1], chip.readRegister(miaRegIdxASelector))
	assert.Equal(t, uint16(0x4001), chip.readRegisterWord(0x03))

	for chip.state == miaStateLoader {
		circuit.read(miaRegIdxASelector)
		circuit.write(miaRegIRQStatusMSB, 0x00)
	}

	assert.Equal(t, miaStateNormal, chip.state)
	assert.Equal(t, uint32(len(miaKernelData)), chip.kernelIndex)
	assert.Equal(t, uint8(0x00), chip.readRegister(miaRegCmdTrigger))
	assert.Equal(t, miaStatusMasterMode, chip.status())
	assert.Equal(t, uint16(miaKernelTargetAddress), chip.readRegisterWord(miaRegResetVectorLSB))
	assert.Equal(t, uint16(miaKernelTargetAddress), chip.readRegisterWord(miaRegNMIVectorLSB))
	assert.Equal(t, uint16(miaKernelTargetAddress), chip.readRegisterWord(miaRegIRQVectorLSB))
	assert.False(t, circuit.reset.Status())
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
	assert.Equal(t, uint16(miaKernelTargetAddress+len(miaKernelData)-1), chip.readRegisterWord(0x03))

	circuit.read(miaRegIdxASelector)
	circuit.write(miaRegIRQStatusMSB, 0x00)

	assert.Equal(t, miaStateNormal, chip.state)
	assert.Equal(t, uint8(0x00), chip.readRegister(miaRegCmdTrigger))
	assert.Equal(t, miaStatusMasterMode, chip.status())
	assert.Equal(t, uint16(miaKernelTargetAddress), chip.readRegisterWord(miaRegResetVectorLSB))
	assert.Equal(t, uint16(miaKernelTargetAddress), chip.readRegisterWord(miaRegNMIVectorLSB))
	assert.Equal(t, uint16(miaKernelTargetAddress), chip.readRegisterWord(miaRegIRQVectorLSB))
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
		flags:       1 << miaIndexFlagReadStep,
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
		flags:       1 << miaIndexFlagWriteStep,
	}
	chip.memory[1] = 0xBB

	circuit.write(miaRegIdxASelector, 4)
	circuit.write(miaRegIdxAPort, 0xAA)

	assert.Equal(t, uint8(0xAA), chip.memory[0])
	assert.Equal(t, uint32(1), chip.indexes[4].currentAddr)
	assert.Equal(t, uint8(0xBB), chip.readRegister(miaRegIdxAPort))
}

// TestEmulatedMiaCommandResetIndexesMatchesFirmware verifies reset commands use defaults.
func TestEmulatedMiaCommandResetIndexesMatchesFirmware(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip
	chip.state = miaStateNormal

	chip.indexes[1] = miaIndex{currentAddr: 0x20, defaultAddr: 0x10}
	chip.indexes[2] = miaIndex{currentAddr: 0x40, defaultAddr: 0x30}
	chip.indexes[3] = miaIndex{currentAddr: 0x60, defaultAddr: 0x50}
	chip.indexes[255] = miaIndex{currentAddr: 0x90, defaultAddr: 0x80}

	circuit.write(miaRegIdxASelector, 1)
	circuit.write(miaRegCmdTrigger, 0x00)
	assert.Equal(t, uint32(0x10), chip.indexes[1].currentAddr)

	circuit.write(miaRegIdxBSelector, 2)
	circuit.write(miaRegCmdTrigger, 0x01)
	assert.Equal(t, uint32(0x30), chip.indexes[2].currentAddr)

	circuit.write(miaRegCmdParam1, 3)
	circuit.write(miaRegCmdTrigger, 0x02)
	assert.Equal(t, uint32(0x50), chip.indexes[3].currentAddr)

	circuit.write(miaRegCmdTrigger, 0x05)
	assert.Equal(t, uint32(0x80), chip.indexes[255].currentAddr)
}

// TestEmulatedMiaIndexWrapDirectionAndIRQGating verifies forward/backward wrap behavior.
func TestEmulatedMiaIndexWrapDirectionAndIRQGating(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip
	chip.state = miaStateNormal

	chip.indexes[7] = miaIndex{
		currentAddr: 2,
		defaultAddr: 0,
		limitAddr:   3,
		step:        1,
		flags:       (1 << miaIndexFlagReadStep) | (1 << miaIndexFlagWrap),
	}
	chip.memory[2] = 0x77
	circuit.write(miaRegIdxASelector, 7)

	assert.Equal(t, uint8(0x77), circuit.read(miaRegIdxAPort))
	assert.Equal(t, uint32(0), chip.indexes[7].currentAddr)
	assert.Zero(t, chip.irqStatus()&miaIRQIdxAWrap)

	chip.indexes[8] = miaIndex{
		currentAddr: 5,
		defaultAddr: 5,
		limitAddr:   8,
		step:        1,
		flags:       (1 << miaIndexFlagReadStep) | (1 << miaIndexFlagStepDir) | (1 << miaIndexFlagWrap) | (1 << miaIndexFlagWrapIRQ),
	}
	chip.memory[5] = 0x88
	circuit.write(miaRegIdxASelector, 8)

	assert.Equal(t, uint8(0x88), circuit.read(miaRegIdxAPort))
	assert.Equal(t, uint32(7), chip.indexes[8].currentAddr)
	assert.Equal(t, miaIRQIdxAWrap, chip.irqStatus()&miaIRQIdxAWrap)
}

// TestEmulatedMiaMemoryMirrorsAt256KiB verifies indexed RAM access mirrors like Pico RAM.
func TestEmulatedMiaMemoryMirrorsAt256KiB(t *testing.T) {
	chip := newEmulatedMiaTestCircuit().chip

	chip.indexes[9].currentAddr = miaRAMSize + 3
	chip.memory[3] = 0x66

	assert.Equal(t, uint8(0x66), chip.indexRead(9))
}

// TestEmulatedMiaCfgSelectorAndPortUseNaturalOrder verifies CFG bus semantics.
func TestEmulatedMiaCfgSelectorAndPortUseNaturalOrder(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip
	chip.state = miaStateNormal

	circuit.write(miaRegCfgSelector, 0x00)
	assert.Equal(t, uint8(0x00), circuit.read(miaRegCfgPort))

	circuit.write(miaRegCfgPort, 0x56)
	assert.Equal(t, uint8(0x56), circuit.read(miaRegCfgPort))
	assert.Equal(t, uint32(0x56), chip.indexes[0].currentAddr)

	circuit.write(miaRegCfgSelector, miaCfgSpeedL)
	assert.Equal(t, byteFrom24(miaDefaultPhi2Hz, 0), circuit.read(miaRegCfgPort))
	circuit.write(miaRegCfgPort, 0x34)
	assert.Equal(t, uint8(0x34), circuit.read(miaRegCfgPort))

	circuit.write(miaRegCfgSelector, miaCfgSpeedM)
	assert.Equal(t, byteFrom24(miaDefaultPhi2Hz, 1), circuit.read(miaRegCfgPort))
	circuit.write(miaRegCfgPort, 0x12)

	circuit.write(miaRegCfgSelector, miaCfgSpeedH)
	assert.Equal(t, byteFrom24(miaDefaultPhi2Hz, 2), circuit.read(miaRegCfgPort))
	circuit.write(miaRegCfgPort, 0x00)

	assert.Equal(t, miaStatusSpeedChanging, chip.status()&miaStatusSpeedChanging)
	assert.Equal(t, uint32(miaDefaultPhi2Hz), chip.appliedPhi2Hz)

	circuit.idle(1)

	assert.Zero(t, chip.status()&miaStatusSpeedChanging)
	assert.Equal(t, uint32(0x1234), chip.appliedPhi2Hz)
	assert.Equal(t, miaIRQSpeedChanged, chip.irqStatus()&miaIRQSpeedChanged)
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
	assert.Equal(t, miaErrorDMASizeZero, circuit.read(miaRegErrorLSB))
	assert.Zero(t, chip.status()&miaStatusErrors)
}

// TestEmulatedMiaErrorQueueOverflowReportsOverflowCode verifies that overrunning
// the 15-entry queue drops the oldest entry and surfaces ERROR_QUEUE_OVERFLOW.
func TestEmulatedMiaErrorQueueOverflowReportsOverflowCode(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip
	chip.state = miaStateNormal

	for range 15 {
		chip.errors.Push(chip, miaErrorDMASizeZero)
	}

	assert.Equal(t, miaStatusErrors, chip.status()&miaStatusErrors)
	assert.Equal(t, miaErrorDMASizeZero, chip.readRegister(miaRegErrorLSB))

	// One push past capacity drops the oldest entry; the new tail entry becomes
	// the overflow marker while the visible head error is refreshed.
	chip.errors.Push(chip, miaErrorCmdUnknown)
	assert.Equal(t, miaErrorDMASizeZero, chip.readRegister(miaRegErrorLSB))

	var last uint8
	for chip.status()&miaStatusErrors != 0 {
		last = circuit.read(miaRegErrorLSB)
	}

	assert.Equal(t, miaErrorQueueOverflow, last)
	assert.Zero(t, chip.readRegister(miaRegErrorLSB))
}

// TestEmulatedMiaUnknownCommandReportsError verifies unassigned command ids queue
// ERROR_CMD_UNKNOWN.
func TestEmulatedMiaUnknownCommandReportsError(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip
	chip.state = miaStateNormal

	circuit.write(miaRegCmdTrigger, 0x99)

	assert.Equal(t, miaStatusErrors, chip.status()&miaStatusErrors)
	assert.Equal(t, miaErrorCmdUnknown, chip.readRegister(miaRegErrorLSB))
}

// TestEmulatedMiaInputCommandsReportErrors verifies the input set-mode and
// set-probe commands queue errors when the request cannot be satisfied.
func TestEmulatedMiaInputCommandsReportErrors(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip
	chip.state = miaStateNormal

	circuit.write(miaRegCmdParam1, uint8(miaInputModeUSBHost))
	circuit.write(miaRegCmdTrigger, 0x50)
	assert.Equal(t, miaErrorInputModeUnavailable, chip.readRegister(miaRegErrorLSB))

	chip.errors.reset(chip)

	circuit.write(miaRegCmdParam1, 16)
	circuit.write(miaRegCmdParam2, 0)
	circuit.write(miaRegCmdTrigger, 0x51)
	assert.Equal(t, miaErrorInputProbeInvalid, chip.readRegister(miaRegErrorLSB))
}

func TestEmulatedMiaIRQStatusReadClearsAllSources(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip
	chip.state = miaStateNormal

	circuit.write(miaRegIRQMaskLSB, uint8(miaIRQError|miaIRQCommand))
	chip.irqSetFlag(miaIRQError | miaIRQCommand)
	circuit.idle(1)

	assert.Equal(t, miaIRQTriggered, chip.irqStatus()&miaIRQTriggered)
	assert.False(t, circuit.irq.Status())

	statusLSB := circuit.read(miaRegIRQStatusLSB)

	assert.Equal(t, uint8(miaIRQError|miaIRQCommand), statusLSB&(uint8(miaIRQError)|uint8(miaIRQCommand)))
	assert.Zero(t, chip.irqStatus())
	assert.True(t, circuit.irq.Status())
}

func TestEmulatedMiaIRQStatusWritesAreIgnoredInNormalMode(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip
	chip.state = miaStateNormal

	chip.irqSetFlag(miaIRQCommand)
	circuit.write(miaRegIRQStatusLSB, 0x00)

	assert.Equal(t, miaIRQCommand, chip.irqStatus()&miaIRQCommand)
}

func TestEmulatedMiaSpeedRequestIsClampedWhenApplied(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip
	chip.state = miaStateNormal

	chip.stagedPhi2Hz = 0
	chip.commitPhi2Hz()
	circuit.idle(1)
	assert.Equal(t, uint32(miaMinPhi2Hz), chip.appliedPhi2Hz)

	chip.stagedPhi2Hz = miaMaxPhi2Hz + 1
	chip.commitPhi2Hz()
	circuit.idle(1)
	assert.Equal(t, uint32(miaMaxPhi2Hz), chip.appliedPhi2Hz)
}
