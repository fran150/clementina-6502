package beneater

import (
	"github.com/fran150/clementina-6502/pkg/computers"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/rivo/tview"
)

type benEaterEmulatorConsoleConfig struct {
	computers.BaseTerminalEmulatorConsoleConfig
	emulator *benEaterEmulator
}

type benEaterEmulatorConsole struct {
	*computers.BaseTerminalEmulatorConsole
	grid *tview.Grid
}

// newMainConsole creates and initializes a new console for the Ben Eater computer.
func newBenEaterEmulatorConsole(config benEaterEmulatorConsoleConfig) *benEaterEmulatorConsole {
	console := &benEaterEmulatorConsole{
		BaseTerminalEmulatorConsole: computers.NewBaseTerminalEmulatorConsole(config.BaseTerminalEmulatorConsoleConfig),
		grid:                        tview.NewGrid(),
	}

	console.initializeMainGrid()

	menuOptions := createMenuOptions(console, config.emulator)

	computer := config.emulator.computer
	wm := console.GetWindowManager()

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
	c.GetApp().SetRoot(c.grid, true)
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
	wm := c.GetWindowManager()
	c.grid.AddItem(wm.GetWindow("lcd").GetDrawArea(), 0, 0, 1, 1, 0, 0, false).
		AddItem(wm.GetWindow("speed").GetDrawArea(), 1, 0, 1, 1, 0, 0, false).
		AddItem(wm.GetWindow("code").GetDrawArea(), 2, 0, 1, 1, 0, 0, false).
		AddItem(wm.GetWindow("options").GetDrawArea(), 3, 0, 1, 2, 0, 0, false).
		AddItem(wm.GetPages(), 0, 1, 3, 1, 0, 0, true)
}
