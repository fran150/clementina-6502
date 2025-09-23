package beneater

import (
	"fmt"
	"testing"
	"time"

	"github.com/fran150/clementina-6502/internal/testutils"
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/computers"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"go.bug.st/serial"
)

const testDirectory string = "../../../assets/computer/beneater/"

func createConfig() *BenEaterComputerConfig {
	mock := testutils.NewPortMock(&serial.Mode{
		BaudRate: 19200,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	})

	return &BenEaterComputerConfig{
		Port:              mock, // No port specified for testing
		EmulateModemLines: false,
		EmulationLoopConfig: computers.EmulationLoopConfig{
			TargetSpeedMhz: 1.1,
			DisplayFps:     20,
		},
	}
}

func createComputer(t *testing.T) *BenEaterComputer {
	config := createConfig()

	computer, err := NewBenEaterComputer(config)
	if err != nil {
		t.Fatal("Failed to create computer:", err)
	}

	return computer
}

func TestNewBenEaterComputer(t *testing.T) {
	t.Run("No port specified should work correctly", func(t *testing.T) {
		config := createConfig()
		config.Port = nil // No port specified for testing

		_, err := NewBenEaterComputer(config)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Port specified should initialize and connect to the port correctly", func(t *testing.T) {
		config := createConfig()
		_, err := NewBenEaterComputer(config)

		if err != nil {
			t.Error(err)
		}
	})
	t.Run("Port specified but fails on connection should return the error", func(t *testing.T) {
		config := createConfig()

		mock := config.Port.(*testutils.SerialPortMock)
		mock.MakeCallsFailFrom = testutils.FailInSetMode
		_, err := NewBenEaterComputer(config)
		assert.Error(t, err, "Expected error when connecting to a failing port")
	})
}

func TestLoadRom(t *testing.T) {
	computer := createComputer(t)
	defer computer.Close()

	t.Run("Successfully load ROM file", func(t *testing.T) {
		err := computer.LoadRom(testDirectory + "wozmon.bin")
		assert.NoError(t, err, "Expected no error when loading valid ROM file")
	})

	t.Run("Fail to load non-existent ROM file", func(t *testing.T) {
		err := computer.LoadRom(testDirectory + "nonexistent.bin")
		assert.Error(t, err, "Expected error when loading non-existent ROM file")
	})
}

func TestTick(t *testing.T) {
	t.Run("Normal tick operation", func(t *testing.T) {
		computer := createComputer(t)
		context := common.NewStepContext()

		initialPC := computer.chips.cpu.GetProgramCounter()

		computer.Tick(&context)

		// TODO: Improve this test mocking componentes to validate that the have
		// been ticked
		assert.NotEqual(t, initialPC, computer.chips.cpu.GetProgramCounter())
		assert.False(t, computer.IsPaused(), "Computer should not be paused by default")
	})

	t.Run("Paused computer should not tick", func(t *testing.T) {
		computer := createComputer(t)
		context := common.NewStepContext()

		// Set computer to paused state
		computer.Pause()
		initialPC := computer.chips.cpu.GetProgramCounter()

		computer.Tick(&context)

		// Verify PC hasn't changed since computer is paused
		assert.Equal(t, initialPC, computer.chips.cpu.GetProgramCounter(),
			"Program counter should not change when computer is paused")
	})

	t.Run("Single step operation", func(t *testing.T) {
		computer := createComputer(t)
		context := common.NewStepContext()

		// Set computer to paused state but enable single step
		computer.Pause()

		initialPC := computer.chips.cpu.GetProgramCounter()
		computer.Tick(&context)

		// Verify PC hasn't changed since computer is paused
		assert.Equal(t, initialPC, computer.chips.cpu.GetProgramCounter(),
			"Program counter should not change when computer is paused")

		// Enable single step
		computer.Step()
		assert.True(t, computer.IsStepping(), "Computer should be in stepping mode")

		// Should perform a single step
		computer.Tick(&context)

		// Verify step flag was reset
		assert.False(t, computer.IsStepping(), "Step flag should be reset after tick")

		// Verify PC changed due to single step
		assert.NotEqual(t, initialPC, computer.chips.cpu.GetProgramCounter(),
			"Program counter should change after single step")
	})

	t.Run("Breakpoint handling", func(t *testing.T) {
		computer := createComputer(t)
		context := common.NewStepContext()

		if breakpointForm := computers.GetWindow[ui.BreakPointForm](&computer.console.BaseConsole, "breakpoint"); breakpointForm != nil {
			pc := computer.chips.cpu.GetProgramCounter()
			text := fmt.Sprintf("%04X", pc)
			breakpointForm.AddBreakpointAddress(text)

			computer.Tick(&context)

			if computer.IsPaused() {
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

func TestDraw(t *testing.T) {
	t.Run("Draw computer console", func(t *testing.T) {
		computer := createComputer(t)
		defer computer.Close()

		context := common.NewStepContext()

		if cpuWindow := computers.GetWindow[ui.CpuWindow](&computer.console.BaseConsole, "cpu"); cpuWindow != nil {
			textview := cpuWindow.GetDrawArea().(*tview.TextView)

			text := textview.GetText(true)
			assert.Empty(t, text, "TextView should be empty initially")

			// Run this test in background to avoid blocking issues with tview.
			go computer.Draw(&context)

			// Allow some time for the draw to complete
			time.Sleep(100 * time.Millisecond)

			text = textview.GetText(true)
			assert.NotEmpty(t, text, "TextView should not be empty after call to draw")
		} else {
			t.Fatal("CPU window should be initialized")
		}
	})
}

func TestMenuOptions(t *testing.T) {
	computer := createComputer(t)
	context := common.NewStepContext()
	defer computer.Close()

	assert.Equal(t, "cpu", computer.console.GetActiveWindow(), "Initial active window should be CPU")

	simulateKeyPress(computer, &context, tcell.KeyRune, 'v')
	simulateKeyPress(computer, &context, tcell.KeyF2, ' ')
	assert.Equal(t, "via", computer.console.GetActiveWindow(), "F2 option should be VIA")

	simulateKeyPress(computer, &context, tcell.KeyF3, ' ')
	assert.Equal(t, "acia", computer.console.GetActiveWindow(), "F3 option should be ACIA")

	simulateKeyPress(computer, &context, tcell.KeyF4, ' ')
	assert.Equal(t, "lcd_controller", computer.console.GetActiveWindow(), "F4 option should be LCD")

	simulateKeyPress(computer, &context, tcell.KeyF5, ' ')
	assert.Equal(t, "rom", computer.console.GetActiveWindow(), "F5 option should be ROM")
	if memoryWindow := computers.GetWindow[ui.MemoryWindow](&computer.console.BaseConsole, "rom"); memoryWindow != nil {
		validateMemoryWindow(t, computer, &context, memoryWindow)
	}
	simulateKeyPress(computer, &context, tcell.KeyESC, ' ')

	simulateKeyPress(computer, &context, tcell.KeyF6, ' ')
	assert.Equal(t, "ram", computer.console.GetActiveWindow(), "F6 option should be RAM")
	if memoryWindow := computers.GetWindow[ui.MemoryWindow](&computer.console.BaseConsole, "ram"); memoryWindow != nil {
		validateMemoryWindow(t, computer, &context, memoryWindow)
	}
	simulateKeyPress(computer, &context, tcell.KeyESC, ' ')

	simulateKeyPress(computer, &context, tcell.KeyF7, ' ')
	assert.Equal(t, "bus", computer.console.GetActiveWindow(), "F7 option should be Buses")

	// Back to CPU
	simulateKeyPress(computer, &context, tcell.KeyF1, ' ')
	assert.Equal(t, "cpu", computer.console.GetActiveWindow(), "F1 option should be CPU")

	// Back to main menu
	simulateKeyPress(computer, &context, tcell.KeyESC, ' ')

	// Go to emulation menu
	simulateKeyPress(computer, &context, tcell.KeyRune, 'e')
	// Go to speed menu
	simulateKeyPress(computer, &context, tcell.KeyRune, 's')

	config := computer.Loop().GetConfig()

	// Speed
	assert.Equal(t, 1.1, config.TargetSpeedMhz)

	// Speed up
	simulateKeyPress(computer, &context, tcell.KeyUp, ' ')
	assert.InDelta(t, 1.2, 0.001, config.TargetSpeedMhz, "Speed should increase")

	// Speed Down
	simulateKeyPress(computer, &context, tcell.KeyDown, ' ')
	assert.InDelta(t, 1.1, 0.001, config.TargetSpeedMhz, "Speed should decrease")

	if speedWindow := computers.GetWindow[ui.SpeedWindow](&computer.console.BaseConsole, "speed"); speedWindow != nil {
		assert.True(t, speedWindow.IsConfigVisible(), "Speed window should be in config mode")
	}

	// Back to emulation menu
	simulateKeyPress(computer, &context, tcell.KeyESC, ' ')
	// Go to execution menu
	simulateKeyPress(computer, &context, tcell.KeyRune, 'e')
	// Go to breakpoint menu
	simulateKeyPress(computer, &context, tcell.KeyRune, 'b')

	// Check that breakpoint window is active
	assert.Equal(t, "breakpoint", computer.console.GetActiveWindow(), "Breakpoint window should be active")

	if breakpointForm := computers.GetWindow[ui.BreakPointForm](&computer.console.BaseConsole, "breakpoint"); breakpointForm != nil {
		breakpointForm.AddBreakpointAddress("00FF")
		assert.True(t, breakpointForm.CheckBreakpoint(0x00FF), "Breakpoint should be added")

		// Remove selected breakpoint
		simulateKeyPress(computer, &context, tcell.KeyRune, 'r')
		assert.False(t, breakpointForm.CheckBreakpoint(0x00FF), "Breakpoint should be removed")
	}

	// Back to emulation menu must return the window to the previous one
	simulateKeyPress(computer, &context, tcell.KeyESC, ' ')
	assert.Equal(t, "cpu", computer.console.GetActiveWindow(), "CPU window should be active again")
}

func TestLCDBusMapping(t *testing.T) {
	computer := createComputer(t)
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

func TestLineIsHeldLowWhenResetting(t *testing.T) {
	computer := createComputer(t)
	defer computer.Close()

	// Check if the reset line is held high initially
	assert.True(t, computer.circuit.cpuReset.Status(), "CPU reset line is high ")

	// Reset the computer
	computer.Reset()

	// Reset line is held low for 6 cycles
	for i := 0; i <= 5; i++ {
		computer.Tick(&common.StepContext{})
		assert.False(t, computer.circuit.cpuReset.Status(), "CPU reset line should be released after a 5 ticks")
	}

	computer.Tick(&common.StepContext{})
	assert.True(t, computer.circuit.cpuReset.Status(), "CPU reset line is released")
}

func simulateKeyPress(computer *BenEaterComputer, context *common.StepContext, key tcell.Key, ch rune) {
	event := tcell.NewEventKey(key, ch, tcell.ModNone)
	computer.console.KeyPressed(event)
	computer.Tick(context)
}

func validateMemoryWindow(t *testing.T, computer *BenEaterComputer, context *common.StepContext, memoryWindow *ui.MemoryWindow) {
	assert.Equal(t, uint32(0x0000), memoryWindow.GetStartAddress(), "Window should start at address 0x0000")

	simulateKeyPress(computer, context, tcell.KeyDown, ' ')
	assert.Equal(t, uint32(0x0008), memoryWindow.GetStartAddress(), "Window should scroll down to address 0x0008")

	simulateKeyPress(computer, context, tcell.KeyUp, ' ')
	assert.Equal(t, uint32(0x0000), memoryWindow.GetStartAddress(), "Window should scroll up to address 0x0000")

	simulateKeyPress(computer, context, tcell.KeyPgDn, ' ')
	assert.Equal(t, uint32(0x00A0), memoryWindow.GetStartAddress(), "Window should scroll down to address 0x00A0")

	simulateKeyPress(computer, context, tcell.KeyPgUp, ' ')
	assert.Equal(t, uint32(0x0000), memoryWindow.GetStartAddress(), "Window should scroll up to address 0x0000")
}
