package beneater

import (
	"testing"

	"github.com/fran150/clementina6502/pkg/common"
	"github.com/fran150/clementina6502/pkg/terminal"
	"github.com/fran150/clementina6502/pkg/testutils"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"go.bug.st/serial"
)

const testDirectory string = "../../../assets/computer/beneater/"

func createComputer(t *testing.T) (*BenEaterComputer, *common.StepContext) {
	computer, err := NewBenEaterComputer(nil, false)
	if err != nil {
		t.Fatal("Failed to create computer:", err)
	}

	appConfig := &terminal.ApplicationConfig{}
	app := tview.NewApplication()
	computer.Init(app, appConfig)

	context := common.NewStepContext()

	return computer, &context
}

func TestNewBenEaterComputer(t *testing.T) {
	mock := testutils.NewPortMock(&serial.Mode{
		BaudRate: 19200,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	})

	t.Run("No port specified should work correctly", func(t *testing.T) {
		_, err := NewBenEaterComputer(nil, false)
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("Port specified should initialize and connect to the port correctly", func(t *testing.T) {
		_, err := NewBenEaterComputer(mock, false)
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("Port specified but fails on connection should return the error", func(t *testing.T) {
		mock.MakeCallsFailFrom = testutils.FailInSetMode
		_, err := NewBenEaterComputer(mock, false)
		assert.Error(t, err, "Expected error when connecting to a failing port")
	})
}

func TestLoadRom(t *testing.T) {
	t.Run("Successfully load ROM file", func(t *testing.T) {
		computer, err := NewBenEaterComputer(nil, false)
		if err != nil {
			t.Fatal("Failed to create computer:", err)
		}

		err = computer.LoadRom(testDirectory + "wozmon.bin")
		assert.NoError(t, err, "Expected no error when loading valid ROM file")
	})

	t.Run("Fail to load non-existent ROM file", func(t *testing.T) {
		computer, err := NewBenEaterComputer(nil, false)
		if err != nil {
			t.Fatal("Failed to create computer:", err)
		}

		err = computer.LoadRom(testDirectory + "nonexistent.bin")
		assert.Error(t, err, "Expected error when loading non-existent ROM file")
	})
}

func TestInit(t *testing.T) {
	t.Run("Successfully initialize computer", func(t *testing.T) {
		computer, err := NewBenEaterComputer(nil, false)
		if err != nil {
			t.Fatal("Failed to create computer:", err)
		}

		app := tview.NewApplication()
		appConfig := &terminal.ApplicationConfig{}

		computer.Init(app, appConfig)

		assert.NotNil(t, computer.appConfig, "appConfig should be initialized")
		assert.NotNil(t, computer.console, "console should be initialized")
	})

	t.Run("Initialize with nil application", func(t *testing.T) {
		computer, err := NewBenEaterComputer(nil, false)
		if err != nil {
			t.Fatal("Failed to create computer:", err)
		}

		assert.Panics(t, func() {
			computer.Init(nil, &terminal.ApplicationConfig{})
		}, "Init should panic when app is nil")
	})

	t.Run("Initialize with nil config", func(t *testing.T) {
		computer, err := NewBenEaterComputer(nil, false)
		if err != nil {
			t.Fatal("Failed to create computer:", err)
		}

		assert.Panics(t, func() {
			computer.Init(tview.NewApplication(), nil)
		}, "Init should panic when config is nil")
	})
}

func TestTick(t *testing.T) {
	t.Run("Normal tick operation", func(t *testing.T) {
		computer, context := createComputer(t)
		computer.Tick(context)

		// Verify that CPU is not paused by default
		assert.False(t, computer.pause, "Computer should not be paused by default")
	})

	t.Run("Paused computer should not tick", func(t *testing.T) {
		computer, context := createComputer(t)

		// Set computer to paused state
		computer.pause = true
		initialPC := computer.chips.cpu.GetProgramCounter()

		computer.Tick(context)

		// Verify PC hasn't changed since computer is paused
		assert.Equal(t, initialPC, computer.chips.cpu.GetProgramCounter(),
			"Program counter should not change when computer is paused")
	})

	t.Run("Single step operation", func(t *testing.T) {
		computer, context := createComputer(t)
		// Set computer to paused state but enable single step
		computer.pause = true
		computer.step = true
		initialPC := computer.chips.cpu.GetProgramCounter()

		computer.Tick(context)

		// Verify step flag was reset
		assert.False(t, computer.step, "Step flag should be reset after tick")

		// Verify PC changed due to single step
		assert.NotEqual(t, initialPC, computer.chips.cpu.GetProgramCounter(),
			"Program counter should change after single step")
	})
}
