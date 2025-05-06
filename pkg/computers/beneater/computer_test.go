package beneater

import (
	"fmt"
	"testing"

	"github.com/fran150/clementina-6502/internal/testutils"
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/computers"
	"github.com/fran150/clementina-6502/pkg/terminal"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/gdamore/tcell/v2"
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
	computer, err := NewBenEaterComputer(nil, false)
	if err != nil {
		t.Fatal("Failed to create computer:", err)
	}
	defer computer.Close()

	t.Run("Successfully load ROM file", func(t *testing.T) {
		if err != nil {
			t.Fatal("Failed to create computer:", err)
		}

		err = computer.LoadRom(testDirectory + "wozmon.bin")
		assert.NoError(t, err, "Expected no error when loading valid ROM file")
	})

	t.Run("Fail to load non-existent ROM file", func(t *testing.T) {
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
		app := tview.NewApplication()
		appConfig := &terminal.ApplicationConfig{}

		computer.Init(app, appConfig)

		assert.False(t, computer.mustReset, "Must reset flag should be unset by default")

		// Reset the computer
		computer.Reset(context)
		assert.True(t, computer.mustReset, "Must reset flag should be set")

		for range 5 {
			computer.Tick(context)
			assert.True(t, computer.mustReset, "Must reset flag should be set for 5 cycles")
			assert.False(t, computer.circuit.cpuReset.Status(), "CPU reset must be held down for 5 cycles for CPU to reset")
		}

		computer.Tick(context)
		computer.Tick(context)
		assert.False(t, computer.mustReset, "Flag should be unset after 5 cycles")
		assert.True(t, computer.circuit.cpuReset.Status(), "CPU reset must be released after 5 cycles")
	})
}

func TestKeyPressed(t *testing.T) {
	t.Run("Test select VIA window using key strokes", func(t *testing.T) {
		computer, context := createComputer(t)

		app := tview.NewApplication()
		appConfig := &terminal.ApplicationConfig{}

		computer.Init(app, appConfig)

		// Check that window is showing CPU by default
		assert.Equal(t, "cpu", computer.console.active)

		// Simulate pressing v and then F2 to go to "Views" and then select "VIA"
		event := tcell.NewEventKey(tcell.KeyRune, 'v', tcell.ModNone)
		computer.KeyPressed(event, context)
		computer.Tick(context)

		event = tcell.NewEventKey(tcell.KeyF2, ' ', tcell.ModNone)
		response := computer.KeyPressed(event, context)
		computer.Tick(context)

		// Check that VIA window is active and event is forwarded
		assert.Equal(t, "via", computer.console.active)
		assert.Equal(t, event, response)

		// If no options menu is pressed pressing keys will just forward the event
		delete(computer.console.windows, "options")
		// Try to switch to ACIA window
		event = tcell.NewEventKey(tcell.KeyF3, ' ', tcell.ModNone)
		response = computer.KeyPressed(event, context)
		computer.Tick(context)
		// Check that VIA window is active and event is forwarded
		assert.Equal(t, "via", computer.console.active)
		assert.Equal(t, event, response)
	})
}

func TestLCDBusMapping(t *testing.T) {
	computer, _ := createComputer(t)
	defer computer.Close()

	t.Run("Map from PortB to LCD of 0xF0 should be mapped to 0x0F", func(t *testing.T) {
		computer.circuit.portBBus.Write(0x0F)

		value := computer.circuit.lcdBus.Read()
		assert.Equal(t, uint8(0xF0), value, "LCD data should be mapped correctly from PortB")
	})

	t.Run("Map from LCD to PortB of 0xA0 should be mapped to 0x0A", func(t *testing.T) {
		computer.circuit.lcdBus.Write(0xA0)

		value := computer.circuit.portBBus.Read()
		assert.Equal(t, uint8(0x0A), value, "LCD data should be mapped correctly from PortB")
	})
}

func simulateKeyPress(computer *BenEaterComputer, context *common.StepContext, key tcell.Key, ch rune) {
	event := tcell.NewEventKey(key, ch, tcell.ModNone)
	computer.KeyPressed(event, context)
	computer.Tick(context)
	computer.Draw(context)
}

func validateMemoryWindow(t *testing.T, computer *BenEaterComputer, context *common.StepContext, memoryWindow *ui.MemoryWindow) {
	assert.Equal(t, uint16(0x0000), memoryWindow.GetStartAddress(), "Window should start at address 0x0000")

	simulateKeyPress(computer, context, tcell.KeyDown, ' ')
	assert.Equal(t, uint16(0x0008), memoryWindow.GetStartAddress(), "Window should scroll down to address 0x0008")

	simulateKeyPress(computer, context, tcell.KeyUp, ' ')
	assert.Equal(t, uint16(0x0000), memoryWindow.GetStartAddress(), "Window should scroll up to address 0x0000")

	simulateKeyPress(computer, context, tcell.KeyPgDn, ' ')
	assert.Equal(t, uint16(0x00A0), memoryWindow.GetStartAddress(), "Window should scroll down to address 0x00A0")

	simulateKeyPress(computer, context, tcell.KeyPgUp, ' ')
	assert.Equal(t, uint16(0x0000), memoryWindow.GetStartAddress(), "Window should scroll up to address 0x0000")
}

func TestMenuOptions(t *testing.T) {
	computer, context := createComputer(t)
	defer computer.Close()

	app := tview.NewApplication()
	appConfig := &terminal.ApplicationConfig{
		EmulationLoopConfig: computers.EmulationLoopConfig{
			SkipCycles: 0,
			DisplayFps: 10,
		},
	}

	computer.Init(app, appConfig)

	assert.Equal(t, "cpu", computer.console.active, "Initial active window should be CPU")

	simulateKeyPress(computer, context, tcell.KeyRune, 'v')

	simulateKeyPress(computer, context, tcell.KeyF2, ' ')
	assert.Equal(t, "via", computer.console.active, "F2 option should be VIA")

	simulateKeyPress(computer, context, tcell.KeyF3, ' ')
	assert.Equal(t, "acia", computer.console.active, "F3 option should be ACIA")

	simulateKeyPress(computer, context, tcell.KeyF4, ' ')
	assert.Equal(t, "lcd_controller", computer.console.active, "F4 option should be LCD")

	simulateKeyPress(computer, context, tcell.KeyF5, ' ')
	assert.Equal(t, "rom", computer.console.active, "F5 option should be ROM")
	if memoryWindow := GetWindow[ui.MemoryWindow](computer.console, "rom"); memoryWindow != nil {
		validateMemoryWindow(t, computer, context, memoryWindow)
	}
	simulateKeyPress(computer, context, tcell.KeyESC, ' ')

	simulateKeyPress(computer, context, tcell.KeyF6, ' ')
	assert.Equal(t, "ram", computer.console.active, "F6 option should be RAM")
	if memoryWindow := GetWindow[ui.MemoryWindow](computer.console, "ram"); memoryWindow != nil {
		validateMemoryWindow(t, computer, context, memoryWindow)
	}
	simulateKeyPress(computer, context, tcell.KeyESC, ' ')

	simulateKeyPress(computer, context, tcell.KeyF7, ' ')
	assert.Equal(t, "bus", computer.console.active, "F7 option should be Buses")

	// Back to CPU
	simulateKeyPress(computer, context, tcell.KeyF1, ' ')
	assert.Equal(t, "cpu", computer.console.active, "F1 option should be CPU")

	// Back to main menu
	simulateKeyPress(computer, context, tcell.KeyESC, ' ')

	// Go to emulation menu
	simulateKeyPress(computer, context, tcell.KeyRune, 'e')
	// Go to speed menu
	simulateKeyPress(computer, context, tcell.KeyRune, 's')

	assert.Equal(t, int64(0), computer.appConfig.SkipCycles)

	// Skip Cycles increase by 10
	simulateKeyPress(computer, context, tcell.KeyRune, '=')
	assert.Equal(t, int64(10), computer.appConfig.SkipCycles, "Skipped cycles should increase by 10")

	// Skip Cycles decrease by 10
	simulateKeyPress(computer, context, tcell.KeyRune, '-')
	assert.Equal(t, int64(0), computer.appConfig.SkipCycles, "Skipped cycles should decrease by 10")

	// Skip Cycles increase by 100
	simulateKeyPress(computer, context, tcell.KeyRune, '+')
	assert.Equal(t, int64(100), computer.appConfig.SkipCycles, "Skipped cycles should increase by 100")

	// Skip Cycles decrease by 100
	simulateKeyPress(computer, context, tcell.KeyRune, '_')
	assert.Equal(t, int64(0), computer.appConfig.SkipCycles, "Skipped cycles should decrease by 100")

	// Speed should not go down more than 0
	simulateKeyPress(computer, context, tcell.KeyRune, '-')
	assert.Equal(t, int64(0), computer.appConfig.SkipCycles, "Speed should not decrease below 0")

	if speedWindow := GetWindow[ui.SpeedWindow](computer.console, "speed"); speedWindow != nil {
		assert.True(t, speedWindow.IsConfigVisible(), "Speed window should be in config mode")
	}

	// Back to emulation menu
	simulateKeyPress(computer, context, tcell.KeyESC, ' ')
	// Go to execution menu
	simulateKeyPress(computer, context, tcell.KeyRune, 'e')
	// Go to breakpoint menu
	simulateKeyPress(computer, context, tcell.KeyRune, 'b')

	// Check that breakpoint window is active
	assert.Equal(t, "breakpoint", computer.console.active, "Breakpoint window should be active")

	if breakpointForm := GetWindow[ui.BreakPointForm](computer.console, "breakpoint"); breakpointForm != nil {
		breakpointForm.AddBreakpointAddress("00FF")
		assert.True(t, breakpointForm.CheckBreakpoint(0x00FF), "Breakpoint should be added")

		// Remove selected breakpoint
		simulateKeyPress(computer, context, tcell.KeyRune, 'r')
		assert.False(t, breakpointForm.CheckBreakpoint(0x00FF), "Breakpoint should be removed")
	}

	// Back to emulation menu must return the window to the previous one
	simulateKeyPress(computer, context, tcell.KeyESC, ' ')
	assert.Equal(t, "cpu", computer.console.active, "CPU window should be active again")
}
