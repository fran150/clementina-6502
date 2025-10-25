package beneater

import (
	"github.com/fran150/clementina-6502/pkg/terminal"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/rivo/tview"
)

// benEaterEmulatorConsoleConfig holds the configuration needed to create a Ben Eater emulator console.
// It embeds the base EmulatorConsoleConfig and adds a reference to the specific Ben Eater emulator.
type benEaterEmulatorConsoleConfig struct {
	terminal.BaseEmulatorConsoleConfig
	emulator *benEaterEmulator
}

// benEaterEmulatorConsole represents the main console interface for the Ben Eater 6502 computer emulator.
// It provides a terminal-based UI with multiple windows for monitoring and controlling the emulated system.
type benEaterEmulatorConsole struct {
	terminal.EmulatorConsole

	app           *tview.Application
	windowManager terminal.WindowManager
	grid          *tview.Grid
}

// newMainConsole creates and initializes a new console for the Ben Eater computer.
func newBenEaterEmulatorConsole(config benEaterEmulatorConsoleConfig) *benEaterEmulatorConsole {
	console := &benEaterEmulatorConsole{
		EmulatorConsole: terminal.NewEmulatorConsole(config.BaseEmulatorConsoleConfig),
		app:             config.App,
		windowManager:   config.WindowManager,
		grid:            tview.NewGrid(),
	}

	console.initializeMainGrid()

	menuOptions := createMenuOptions(console, config.emulator)

	computer := config.emulator.computer
	wm := config.WindowManager

	// Initialize all windows
	wm.AddWindow("lcd", ui.NewDisplayWindow(computer.chips.lcd))
	wm.AddWindow("code", ui.NewCodeWindow(computer.chips.cpu, computer.getPotentialOperators))
	wm.AddWindow("speed", ui.NewSpeedWindow(config.emulator.speedController))
	wm.AddWindow("cpu", ui.NewCpuWindow(computer.chips.cpu))
	wm.AddWindow("via", ui.NewViaWindow(computer.chips.via))
	wm.AddWindow("lcd_controller", ui.NewLcdWindow(computer.chips.lcd))
	wm.AddWindow("acia", ui.NewAciaWindow(computer.chips.acia))
	wm.AddWindow("ram", ui.NewMemoryWindow(computer.chips.ram))
	wm.AddWindow("rom", ui.NewMemoryWindow(computer.chips.rom))
	busWindow := ui.NewBusWindow()
	wm.AddWindow("bus", busWindow)
	wm.AddWindow("breakpoint", ui.NewBreakPointForm(config.emulator.breakpointManager))
	wm.AddWindow("options", ui.NewOptionsWindow(menuOptions))

	initializeBusWindow(computer, busWindow)
	console.initializeLayout()

	// Set initial active window
	console.ShowWindow("cpu")

	return console
}

/************************************************************************************
* Initialization methods
*************************************************************************************/

// initializeMainGrid sets up the main grid layout for the console.
func (c *benEaterEmulatorConsole) initializeMainGrid() {
	c.grid.SetRows(4, 3, 0, 3).
		SetColumns(25, 0).
		SetBorder(true).
		SetTitle("Ben Eater 6502 Computer")

	// Get the tview app from the framework and set the grid as root
	c.app.SetRoot(c.grid, true)
}

// initializeBusWindow configures the bus window with the computer's buses.
func initializeBusWindow(computer *BenEaterComputer, busWindow *ui.BusWindow) {
	busWindow.AddBus16("Address Bus", computer.circuit.addressBus)
	busWindow.AddBus8("Data Bus", computer.circuit.dataBus)
	busWindow.AddBus8("Port A", computer.circuit.portABus)
	busWindow.AddBus8("Port B", computer.circuit.portBBus)
}

// initializeLayout sets up the initial layout of console windows.
func (c *benEaterEmulatorConsole) initializeLayout() {
	// Setup initial grid layout
	wm := c.windowManager
	c.grid.AddItem(wm.GetWindow("lcd").GetDrawArea(), 0, 0, 1, 1, 0, 0, false).
		AddItem(wm.GetWindow("speed").GetDrawArea(), 1, 0, 1, 1, 0, 0, false).
		AddItem(wm.GetWindow("code").GetDrawArea(), 2, 0, 1, 1, 0, 0, false).
		AddItem(wm.GetWindow("options").GetDrawArea(), 3, 0, 1, 2, 0, 0, false).
		AddItem(wm.GetPages(), 0, 1, 3, 1, 0, 0, true)
}
