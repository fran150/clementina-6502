package beneater

import (
	"fmt"
	"testing"

	"github.com/fran150/clementina6502/pkg/common"
	"github.com/fran150/clementina6502/pkg/terminal"
	"github.com/fran150/clementina6502/pkg/terminal/ui"
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

	t.Run("Breakpoint handling", func(t *testing.T) {
		computer, context := createComputer(t)

		if breakpointForm := GetWindow[ui.BreakPointForm](computer.console, "breakpoint"); breakpointForm != nil {
			pc := computer.chips.cpu.GetProgramCounter()
			text := fmt.Sprintf("%04X", pc)
			breakpointForm.AddBreakpointAddress(text)

			computer.Tick(context)

			if computer.pause {
				assert.True(t, breakpointForm.CheckBreakpoint(pc),
					"Computer should be paused when breakpoint is hit")
			} else {
				t.Fatal("Computer should be paused when breakpoint is hit")
			}
		} else {
			t.Fatal("Breakpoint form should be initialized")
		}
	})
}

func TestPause(t *testing.T) {
	t.Run("Pause computer", func(t *testing.T) {
		computer, context := createComputer(t)
		computer.Pause(context)
		assert.True(t, computer.pause, "Computer should be paused")
	})
}

func TestResume(t *testing.T) {
	t.Run("Resume computer", func(t *testing.T) {
		computer, context := createComputer(t)

		// First pause the computer
		computer.Pause(context)
		assert.True(t, computer.pause, "Computer should be paused initially")

		// Then resume it
		computer.Resume(context)
		assert.False(t, computer.pause, "Computer should not be paused after resume")
		assert.True(t, computer.step, "Step flag should be set after resume")
	})
}

func TestStep(t *testing.T) {
	t.Run("Step computer", func(t *testing.T) {
		computer, context := createComputer(t)
		computer.Step(context)
		assert.True(t, computer.step, "Step flag should be set")
	})
}

func TestDraw(t *testing.T) {
	t.Run("Draw computer console", func(t *testing.T) {
		computer, context := createComputer(t)

		app := tview.NewApplication()
		appConfig := &terminal.ApplicationConfig{}

		computer.Init(app, appConfig)

		if cpuWindow := GetWindow[ui.CpuWindow](computer.console, "cpu"); cpuWindow != nil {
			textview := cpuWindow.GetDrawArea().(*tview.TextView)

			text := textview.GetText(true)
			assert.Empty(t, text, "TextView should be empty initially")

			computer.Draw(context)

			text = textview.GetText(true)
			assert.NotEmpty(t, text, "TextView should not be empty after call to draw")
		} else {
			t.Fatal("CPU window should be initialized")
		}

	})
}

func TestStop(t *testing.T) {
	t.Run("Stop computer", func(t *testing.T) {
		computer, context := createComputer(t)
		computer.Stop(context)
		assert.True(t, context.Stop, "Context Stop flag should be set")
	})
}

func TestReset(t *testing.T) {
	t.Run("Reset computer", func(t *testing.T) {
		computer, context := createComputer(t)
		computer.Reset(context)
		assert.True(t, computer.mustReset, "Must reset flag should be set")
	})
}

func TestSpeedUp(t *testing.T) {
	t.Run("Speed up from below 0.5 MHz", func(t *testing.T) {
		computer, context := createComputer(t)
		computer.appConfig.TargetSpeedMhz = 0.1
		initialSpeed := computer.appConfig.TargetSpeedMhz

		computer.SpeedUp(context)

		expectedIncrease := initialSpeed * 0.2
		assert.Equal(t, initialSpeed+expectedIncrease, computer.appConfig.TargetSpeedMhz,
			"Speed should increase by 20% when below 0.5 MHz")
	})

	t.Run("Speed up from very low speed", func(t *testing.T) {
		computer, context := createComputer(t)
		computer.appConfig.TargetSpeedMhz = 0.000000001

		computer.SpeedUp(context)

		assert.Equal(t, 0.000001001, computer.appConfig.TargetSpeedMhz,
			"Speed should increase by minimum increment when very low")
	})

	t.Run("Speed up above 0.5 MHz", func(t *testing.T) {
		computer, context := createComputer(t)
		computer.appConfig.TargetSpeedMhz = 1.0
		initialSpeed := computer.appConfig.TargetSpeedMhz

		computer.SpeedUp(context)

		assert.Equal(t, initialSpeed+0.1, computer.appConfig.TargetSpeedMhz,
			"Speed should increase linearly by 0.1 MHz when above 0.5 MHz")
	})
}

func TestSpeedDown(t *testing.T) {
	t.Run("Speed down from above 0.5 MHz", func(t *testing.T) {
		computer, context := createComputer(t)
		computer.appConfig.TargetSpeedMhz = 1.0
		initialSpeed := computer.appConfig.TargetSpeedMhz

		computer.SpeedDown(context)

		assert.Equal(t, initialSpeed-0.1, computer.appConfig.TargetSpeedMhz,
			"Speed should decrease linearly by 0.1 MHz when above 0.5 MHz")
	})

	t.Run("Speed down from below 0.5 MHz", func(t *testing.T) {
		computer, context := createComputer(t)
		computer.appConfig.TargetSpeedMhz = 0.1
		initialSpeed := computer.appConfig.TargetSpeedMhz

		computer.SpeedDown(context)

		expectedReduction := initialSpeed * 0.2
		assert.Equal(t, initialSpeed-expectedReduction, computer.appConfig.TargetSpeedMhz,
			"Speed should decrease by 20% when below 0.5 MHz")
	})

	t.Run("Speed down from very low speed", func(t *testing.T) {
		computer, context := createComputer(t)
		computer.appConfig.TargetSpeedMhz = 0.000002

		computer.SpeedDown(context)

		assert.Equal(t, 0.000001, computer.appConfig.TargetSpeedMhz,
			"Speed should not go below minimum threshold")
	})

	t.Run("Speed down at minimum threshold", func(t *testing.T) {
		computer, context := createComputer(t)
		computer.appConfig.TargetSpeedMhz = 0.000001

		computer.SpeedDown(context)

		assert.Equal(t, 0.000001, computer.appConfig.TargetSpeedMhz,
			"Speed should not go below minimum threshold")
	})
}
