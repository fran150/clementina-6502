package clementina

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/computers"
	"github.com/fran150/clementina-6502/pkg/terminal"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func createComputer(t *testing.T) (*ClementinaComputer, *common.StepContext) {
	computer, err := NewClementinaComputer()
	if err != nil {
		t.Fatal("Failed to create computer:", err)
	}

	appConfig := &terminal.ApplicationConfig{}
	app := tview.NewApplication()
	computer.Init(app, appConfig)

	context := common.NewStepContext()

	return computer, &context
}

func TestNewClementinaComputer(t *testing.T) {
	t.Run("Successfully create computer", func(t *testing.T) {
		computer, err := NewClementinaComputer()
		assert.NoError(t, err, "Expected no error when creating computer")
		assert.NotNil(t, computer, "Computer should not be nil")
		assert.NotNil(t, computer.chips, "Chips should be initialized")
		assert.NotNil(t, computer.circuit, "Circuit should be initialized")
		assert.False(t, computer.mustReset, "Must reset flag should be false by default")
		assert.False(t, computer.pause, "Computer should not be paused by default")
		assert.False(t, computer.step, "Step flag should be false by default")
	})
}

func TestInit(t *testing.T) {
	t.Run("Successfully initialize computer", func(t *testing.T) {
		computer, err := NewClementinaComputer()
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
		computer, err := NewClementinaComputer()
		if err != nil {
			t.Fatal("Failed to create computer:", err)
		}

		assert.Panics(t, func() {
			computer.Init(nil, &terminal.ApplicationConfig{})
		}, "Init should panic when app is nil")
	})

	t.Run("Initialize with nil config", func(t *testing.T) {
		computer, err := NewClementinaComputer()
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

		assert.False(t, computer.pause, "Computer should not be paused by default")
	})

	t.Run("Paused computer should not tick", func(t *testing.T) {
		computer, context := createComputer(t)

		computer.pause = true
		initialPC := computer.chips.cpu.GetProgramCounter()

		computer.Tick(context)

		assert.Equal(t, initialPC, computer.chips.cpu.GetProgramCounter(),
			"Program counter should not change when computer is paused")
	})

	t.Run("Single step operation", func(t *testing.T) {
		computer, context := createComputer(t)
		computer.pause = true
		computer.step = true
		initialPC := computer.chips.cpu.GetProgramCounter()

		computer.Tick(context)

		assert.False(t, computer.step, "Step flag should be reset after tick")
		assert.NotEqual(t, initialPC, computer.chips.cpu.GetProgramCounter(),
			"Program counter should change after single step")
	})

	t.Run("Breakpoint handling", func(t *testing.T) {
		computer, context := createComputer(t)

		if breakpointForm := GetWindow[ui.BreakPointForm](computer.console, "breakpoint"); breakpointForm != nil {
			pc := computer.chips.cpu.GetProgramCounter()
			breakpointForm.AddBreakpointAddress("FFFC")

			computer.Tick(context)

			if computer.pause {
				assert.True(t, breakpointForm.CheckBreakpoint(pc),
					"Computer should be paused when breakpoint is hit")
			}
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

		computer.Pause(context)
		assert.True(t, computer.pause, "Computer should be paused initially")

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

		assert.False(t, computer.mustReset, "Must reset flag should be unset by default")

		computer.Reset(context)
		assert.True(t, computer.mustReset, "Must reset flag should be set")

		for range 5 {
			computer.Tick(context)
			assert.True(t, computer.mustReset, "Must reset flag should be set for 5 cycles")
			assert.False(t, computer.circuit.cpuReset.Status(), "CPU reset must be held down for 5 cycles")
		}

		computer.Tick(context)
		computer.Tick(context)
		assert.False(t, computer.mustReset, "Flag should be unset after 5 cycles")
		assert.True(t, computer.circuit.cpuReset.Status(), "CPU reset must be released after 5 cycles")
	})
}

func TestSkipCycles(t *testing.T) {
	t.Run("Skip up increases cycles", func(t *testing.T) {
		computer, context := createComputer(t)
		computer.appConfig.SkipCycles = 0

		computer.SkipUp(context, 10)
		assert.Equal(t, int64(10), computer.appConfig.SkipCycles, "Skip cycles should increase by 10")
	})

	t.Run("Skip down decreases cycles", func(t *testing.T) {
		computer, context := createComputer(t)
		computer.appConfig.SkipCycles = 20

		computer.SkipDown(context, 10)
		assert.Equal(t, int64(10), computer.appConfig.SkipCycles, "Skip cycles should decrease by 10")
	})

	t.Run("Skip down cannot go below zero", func(t *testing.T) {
		computer, context := createComputer(t)
		computer.appConfig.SkipCycles = 5

		computer.SkipDown(context, 10)
		assert.Equal(t, int64(0), computer.appConfig.SkipCycles, "Skip cycles should not go below 0")
	})
}

func TestMemoryPoke(t *testing.T) {
	t.Run("Base RAM poke", func(t *testing.T) {
		computer, _ := createComputer(t)

		computer.BaseRamPoke(0x1000, 0xAB)
		value := computer.chips.baseram.Peek(0x1000)
		assert.Equal(t, uint8(0xAB), value, "Base RAM should contain poked value")
	})

	t.Run("Extended RAM poke", func(t *testing.T) {
		computer, _ := createComputer(t)

		computer.ExRamPoke(0x1000, 5, 0xCD)
		mapped := computer.mappers.exRam.MapFromSource([]uint16{0x1000, 5})
		value := computer.chips.exram.Peek(uint32(mapped))
		assert.Equal(t, uint8(0xCD), value, "Extended RAM should contain poked value")
	})

	t.Run("Hi RAM poke", func(t *testing.T) {
		computer, _ := createComputer(t)

		computer.HiRamPoke(0x1000, 2, 0xEF)
		bank := uint16((2 & 0x03) << 5)
		mapped := computer.mappers.hiRam.MapFromSource([]uint16{0x1000, bank})
		value := computer.chips.hiram.Peek(uint32(mapped))
		assert.Equal(t, uint8(0xEF), value, "Hi RAM should contain poked value")
	})
}

func TestKeyPressed(t *testing.T) {
	t.Run("Key forwarded when no options menu", func(t *testing.T) {
		computer, context := createComputer(t)

		delete(computer.console.windows, "options")
		event := tcell.NewEventKey(tcell.KeyF1, ' ', tcell.ModNone)
		response := computer.KeyPressed(event, context)

		assert.Equal(t, event, response, "Event should be forwarded when no options menu")
	})

	t.Run("Key processed by options menu", func(t *testing.T) {
		computer, context := createComputer(t)

		event := tcell.NewEventKey(tcell.KeyRune, 'v', tcell.ModNone)
		computer.KeyPressed(event, context)
		computer.Tick(context)

		event = tcell.NewEventKey(tcell.KeyF1, ' ', tcell.ModNone)
		response := computer.KeyPressed(event, context)
		computer.Tick(context)

		assert.Equal(t, "cpu", computer.console.active, "CPU window should be active")
		assert.Equal(t, event, response, "Event should be returned after processing")
	})
}

func TestConsole(t *testing.T) {
	t.Run("Console initialization", func(t *testing.T) {
		computer, _ := createComputer(t)

		assert.NotNil(t, computer.console, "Console should be initialized")
		assert.NotNil(t, computer.console.grid, "Grid should be initialized")
		assert.NotNil(t, computer.console.pages, "Pages should be initialized")
		assert.NotNil(t, computer.console.windows, "Windows should be initialized")
		assert.Equal(t, "cpu", computer.console.active, "CPU should be active window by default")
	})

	t.Run("Window switching", func(t *testing.T) {
		computer, context := createComputer(t)

		computer.console.ShowWindow("via", context)
		assert.Equal(t, "via", computer.console.active, "VIA window should be active")

		computer.console.ShowWindow("baseram", context)
		assert.Equal(t, "baseram", computer.console.active, "Base RAM window should be active")
	})

	t.Run("Window history navigation", func(t *testing.T) {
		computer, context := createComputer(t)

		computer.console.AppendActiveWindow("via")
		assert.Equal(t, "via", computer.console.active, "VIA window should be active")

		computer.console.ReturnToPreviousWindow(context)
		assert.Equal(t, "cpu", computer.console.active, "Should return to CPU window")
	})

	t.Run("Memory window scrolling", func(t *testing.T) {
		computer, context := createComputer(t)

		computer.console.ShowWindow("baseram", context)
		if memoryWindow := GetWindow[ui.MemoryWindow](computer.console, "baseram"); memoryWindow != nil {
			initialAddress := memoryWindow.GetStartAddress()

			computer.console.ScrollDown(context, 8)
			newAddress := memoryWindow.GetStartAddress()
			assert.Greater(t, newAddress, initialAddress, "Address should increase after scrolling down")

			computer.console.ScrollUp(context, 8)
			finalAddress := memoryWindow.GetStartAddress()
			assert.Equal(t, initialAddress, finalAddress, "Address should return to initial value")
		}
	})

	t.Run("Speed window configuration", func(t *testing.T) {
		computer, context := createComputer(t)

		computer.console.ShowEmulationSpeed(context)
		if speedWindow := GetWindow[ui.SpeedWindow](computer.console, "speed"); speedWindow != nil {
			assert.True(t, speedWindow.IsConfigVisible(), "Speed window should show config")
		}
	})

	t.Run("Breakpoint configuration", func(t *testing.T) {
		computer, context := createComputer(t)

		computer.console.SetBreakpointConfigMode(context)
		assert.Equal(t, "breakpoint", computer.console.active, "Breakpoint window should be active")
	})

	t.Run("Go to form", func(t *testing.T) {
		computer, context := createComputer(t)

		computer.console.ShowWindow("baseram", context)
		computer.console.ShowGotoForm(context)
		assert.Equal(t, "goto", computer.console.active, "Go to form should be active")
	})
}

func TestMenuOptions(t *testing.T) {
	t.Run("Menu structure validation", func(t *testing.T) {
		computer, _ := createComputer(t)
		console := computer.console

		menuOptions := createMenuOptions(computer, console)

		assert.Len(t, menuOptions, 3, "Should have 3 main menu options")

		// Test emulation menu
		emulationMenu := menuOptions[0]
		assert.Equal(t, 'e', emulationMenu.Rune, "First menu should be emulation")
		assert.NotNil(t, emulationMenu.SubMenu, "Emulation should have submenu")

		// Test view menu
		viewMenu := menuOptions[1]
		assert.Equal(t, 'v', viewMenu.Rune, "Second menu should be view")
		assert.NotNil(t, viewMenu.SubMenu, "View should have submenu")

		// Test quit menu
		quitMenu := menuOptions[2]
		assert.Equal(t, 'q', quitMenu.Rune, "Third menu should be quit")
		assert.NotNil(t, quitMenu.Action, "Quit should have action")
	})

	t.Run("Memory window submenu", func(t *testing.T) {
		computer, _ := createComputer(t)
		console := computer.console

		subMenu := createMemoryWindowSubMenu(console)

		assert.Len(t, subMenu, 5, "Memory window submenu should have 5 options")
		assert.Equal(t, tcell.KeyUp, subMenu[0].Key, "First option should be Up key")
		assert.Equal(t, tcell.KeyDown, subMenu[1].Key, "Second option should be Down key")
		assert.Equal(t, tcell.KeyPgUp, subMenu[2].Key, "Third option should be Page Up key")
		assert.Equal(t, tcell.KeyPgDn, subMenu[3].Key, "Fourth option should be Page Down key")
		assert.Equal(t, 'g', subMenu[4].Rune, "Fifth option should be Go To")
	})
}

func simulateKeyPress(computer *ClementinaComputer, context *common.StepContext, key tcell.Key, ch rune) {
	event := tcell.NewEventKey(key, ch, tcell.ModNone)
	computer.KeyPressed(event, context)
	computer.Tick(context)
	computer.Draw(context)
}

func TestMenuNavigation(t *testing.T) {
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

	// Test direct window switching (not through menu)
	computer.console.ShowWindow("via", context)
	assert.Equal(t, "via", computer.console.active, "Should show VIA")

	computer.console.ShowWindow("baseram", context)
	assert.Equal(t, "baseram", computer.console.active, "Should show Base RAM")

	computer.console.ShowWindow("exram", context)
	assert.Equal(t, "exram", computer.console.active, "Should show Extended RAM")

	computer.console.ShowWindow("hiram", context)
	assert.Equal(t, "hiram", computer.console.active, "Should show Hi RAM")

	computer.console.ShowWindow("bus", context)
	assert.Equal(t, "bus", computer.console.active, "Should show Buses")

	// Test skip cycles functionality
	assert.Equal(t, int64(0), computer.appConfig.SkipCycles)

	computer.SkipUp(context, 10)
	assert.Equal(t, int64(10), computer.appConfig.SkipCycles, "Should increase skip cycles by 10")

	computer.SkipDown(context, 10)
	assert.Equal(t, int64(0), computer.appConfig.SkipCycles, "Should decrease skip cycles by 10")

	computer.SkipUp(context, 100)
	assert.Equal(t, int64(100), computer.appConfig.SkipCycles, "Should increase skip cycles by 100")

	computer.SkipDown(context, 100)
	assert.Equal(t, int64(0), computer.appConfig.SkipCycles, "Should decrease skip cycles by 100")
}
