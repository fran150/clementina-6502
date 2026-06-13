package mia

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fran150/clementina-6502/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.bug.st/serial"
)

func TestEmulatedMiaConsoleStatusHelpAndSpeed(t *testing.T) {
	chip, mock := newMiaConsoleTest(t)
	var requested atomic.Uint32
	chip.SetPhi2HzChangedHandler(func(hz uint32) {
		requested.Store(hz)
	})

	waitForMiaConsoleOutput(t, mock, "MIA ready. Type 'help' for commands.\n> ")
	sendMiaConsoleInput(mock, "help\n")
	waitForMiaConsoleOutput(t, mock, "Commands:\n")
	waitForMiaConsoleOutput(t, mock, "monitor    Enter 65C02 machine language monitor\n")

	sendMiaConsoleInput(mock, "speed 42\n")
	waitForMiaConsoleOutput(t, mock, "PHI2 speed requested: 42 Hz")
	require.Eventually(t, func() bool {
		return requested.Load() == 42
	}, time.Second, 10*time.Millisecond)

	chip.mu.Lock()
	assert.Equal(t, uint32(42), chip.requestedPhi2Hz)
	assert.Equal(t, miaStatusSpeedChanging, chip.status()&miaStatusSpeedChanging)
	chip.speedService()
	chip.mu.Unlock()

	sendMiaConsoleInput(mock, "status\n")
	waitForMiaConsoleOutput(t, mock, "PHI2:   42 Hz\n")
	waitForMiaConsoleOutput(t, mock, "Wi-Fi: off\n")
}

func TestEmulatedMiaConsoleMonitorEditDumpAndDisassemble(t *testing.T) {
	_, mock := newMiaConsoleTest(t)

	waitForMiaConsoleOutput(t, mock, "> ")
	sendMiaConsoleInput(mock, "monitor\n")
	waitForMiaConsoleOutput(t, mock, "65C02 Monitor  [MIA RAM: 128KB, $00000-$1FFFF]\n")
	waitForMiaConsoleOutput(t, mock, "MON> ")

	sendMiaConsoleInput(mock, "e 4000 A9 01 80 FE 0F 10 02\n")
	waitForMiaConsoleOutput(t, mock, "e 4000 A9 01 80 FE 0F 10 02\nMON> ")

	sendMiaConsoleInput(mock, "m 4000 08\n")
	waitForMiaConsoleOutput(t, mock, "$04000: A9 01 80 FE 0F 10 02")

	sendMiaConsoleInput(mock, "u 4000 3\n")
	waitForMiaConsoleOutput(t, mock, "$04000: A9 01    LDA  #$01\n")
	waitForMiaConsoleOutput(t, mock, "$04002: 80 FE    BRA  $4002\n")
	waitForMiaConsoleOutput(t, mock, "$04004: 0F 10 02 BBR0 $10,$4009\n")

	sendMiaConsoleInput(mock, "quit\n")
	waitForMiaConsoleOutput(t, mock, "Exiting monitor.\n> ")
}

func TestEmulatedMiaMonitorDisassembleUsesCPUAddressModeData(t *testing.T) {
	chip := NewEmulatedMia().(*emulated_mia)

	chip.memory[0x4000] = 0x0A // ASL A
	chip.memory[0x4001] = 0xB5 // LDA $12,X
	chip.memory[0x4002] = 0x12
	chip.memory[0x4003] = 0x7C // JMP ($1234,X)
	chip.memory[0x4004] = 0x34
	chip.memory[0x4005] = 0x12

	out, next := chip.monitorDisassembleLocked(0x4000, 3)

	assert.Equal(t, uint32(0x4006), next)
	assert.Contains(t, out, "$04000: 0A       ASL  A\n")
	assert.Contains(t, out, "$04001: B5 12    LDA  $12,X\n")
	assert.Contains(t, out, "$04003: 7C 34 12 JMP  ($1234,X)\n")
}

func newMiaConsoleTest(t *testing.T) (*emulated_mia, *testutils.SerialPortMock) {
	t.Helper()

	chip := NewEmulatedMia().(*emulated_mia)
	mock := testutils.NewPortMock(&serial.Mode{})
	require.NoError(t, chip.ConnectToPort(mock))

	t.Cleanup(func() {
		chip.Close()
		_ = mock.Close()
	})

	return chip, mock
}

func sendMiaConsoleInput(mock *testutils.SerialPortMock, value string) {
	for _, b := range []byte(value) {
		mock.PortRxBuffer.Queue(b)
	}
}

func waitForMiaConsoleOutput(t *testing.T, mock *testutils.SerialPortMock, value string) {
	t.Helper()

	require.Eventually(t, func() bool {
		return strings.Contains(string(mock.PortTxBuffer.GetValues()), value)
	}, time.Second, 10*time.Millisecond)
}
