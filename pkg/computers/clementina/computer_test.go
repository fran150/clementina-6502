package clementina

import (
	"strings"
	"testing"
	"time"

	"github.com/fran150/clementina-6502/assets"
	"github.com/fran150/clementina-6502/internal/testutils"
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.bug.st/serial"
)

// TestClementinaPotentialOperatorsReadsMiaRange verifies debugger operand lookup in MIA memory.
func TestClementinaPotentialOperatorsReadsMiaRange(t *testing.T) {
	computer, err := NewClementinaComputer()
	require.NoError(t, err)

	require.NotEmpty(t, assets.MiaKernel)
	// 2026-09 RAM/ROM reorg: descending loader - the first byte sent is the
	// kernel image's *last* byte (meant for the highest address, $BFFF), not
	// its first.
	lastByte := assets.MiaKernel[len(assets.MiaKernel)-1]
	assert.Equal(t, [2]uint8{0xA9, lastByte}, computer.getPotentialOperators(0xFFE0))
	assert.Equal(t, [2]uint8{lastByte, 0x8D}, computer.getPotentialOperators(0xFFE1))
	assert.Equal(t, [2]uint8{0xFF, 0xBF}, computer.getPotentialOperators(0xFFE3)) // STA $BFFF operand (load top, lo/hi)
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

	require.NotEmpty(t, assets.MiaKernel)
	// 2026-09 RAM/ROM reorg: descending loader - see TestClementinaPotentialOperatorsReadsMiaRange.
	lastByte := assets.MiaKernel[len(assets.MiaKernel)-1]
	assert.Equal(t, [2]uint8{0x00, lastByte}, computer.getPotentialOperators(0xFFE0))

	computer.Reset(true)
	tickComputer(computer, &step)
	computer.Reset(false)

	assert.Equal(t, [2]uint8{0xA9, lastByte}, computer.getPotentialOperators(0xFFE0))
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

// TestClementinaKernelLoads verifies the MIA loader writes the embedded kernel.
func TestClementinaKernelLoads(t *testing.T) {
	computer, err := NewClementinaComputer()
	require.NoError(t, err)
	require.NotEmpty(t, assets.MiaKernel)

	step := common.NewStepContext()

	computer.Reset(true)
	for range 3 {
		tickComputer(computer, &step)
		step.NextCycle()
	}
	computer.Reset(false)

	// 2026-09 RAM/ROM reorg: the image is top-anchored, ending exactly at
	// $BFFF, so its start varies with its size instead of being the fixed
	// $0400 load base. The loader also now writes DESCENDING - array index 0
	// (kernelStart) is written *last*, not index len-1 - so the "ready"
	// probe scans forward from index 0 for a non-zero byte, instead of
	// backward from the end.
	kernelStart := uint32(0xC000) - uint32(len(assets.MiaKernel))
	kernelReadyOffset := 0
	for kernelReadyOffset < len(assets.MiaKernel)-1 && assets.MiaKernel[kernelReadyOffset] == 0 {
		kernelReadyOffset++
	}
	kernelReadyAddress := kernelStart + uint32(kernelReadyOffset)

	// The image now spans both baseram ($0000-$7FFF) and exram's currently
	// selected bank ($8000-$BFFF) - peekMappedMemory routes to whichever one
	// a given address actually falls in, same as getPotentialOperators does.
	for range len(assets.MiaKernel) * 120 {
		tickComputer(computer, &step)

		readyByte, _ := computer.peekMappedMemory(uint16(kernelReadyAddress))
		if readyByte == assets.MiaKernel[kernelReadyOffset] {
			for i, expected := range assets.MiaKernel {
				actual, _ := computer.peekMappedMemory(uint16(kernelStart + uint32(i)))
				assert.Equalf(t, expected, actual, "kernel byte %d", i)
			}
			return
		}

		step.NextCycle()
	}

	t.Fatal("MIA loader did not write the embedded kernel")
}

func TestClementinaMiaConsoleSpeedUpdatesEmulatorTarget(t *testing.T) {
	computer, err := NewClementinaComputer()
	require.NoError(t, err)

	emulator, err := NewClemetinaEmulator(computer, 1.2, 15)
	require.NoError(t, err)
	clementinaEmulator := emulator.(*clementinaEmulator)

	mock := testutils.NewPortMock(&serial.Mode{})
	require.NoError(t, computer.ConnectMiaConsole(mock))
	t.Cleanup(func() {
		computer.Close()
		_ = mock.Close()
	})

	waitForClementinaMiaConsoleOutput(t, mock, "> ")
	sendClementinaMiaConsoleInput(mock, "speed 2500000\n")

	require.Eventually(t, func() bool {
		return clementinaEmulator.speedController.GetTargetSpeed() == 2.5
	}, time.Second, 10*time.Millisecond)
}

func TestClementinaEmulatorSpeedUpdatesMiaPhi2(t *testing.T) {
	computer, err := NewClementinaComputer()
	require.NoError(t, err)
	t.Cleanup(computer.Close)

	emulator, err := NewClemetinaEmulator(computer, 1.2, 15)
	require.NoError(t, err)
	clementinaEmulator := emulator.(*clementinaEmulator)

	clementinaEmulator.speedController.SetTargetSpeed(2.5)

	step := common.NewStepContext()
	tickComputer(computer, &step)

	phi2Reader := computer.chips.mia.(interface {
		AppliedPhi2Hz() uint32
	})
	assert.Equal(t, uint32(2_500_000), phi2Reader.AppliedPhi2Hz())
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

func sendClementinaMiaConsoleInput(mock *testutils.SerialPortMock, value string) {
	for _, b := range []byte(value) {
		mock.PortRxBuffer.Queue(b)
	}
}

func waitForClementinaMiaConsoleOutput(t *testing.T, mock *testutils.SerialPortMock, value string) {
	t.Helper()

	require.Eventually(t, func() bool {
		return strings.Contains(string(mock.PortTxBuffer.GetValues()), value)
	}, time.Second, 10*time.Millisecond)
}
