package beneater

import (
	"github.com/fran150/clementina-6502/pkg/computers"
	"github.com/fran150/clementina-6502/pkg/core/interfaces"
	"github.com/fran150/clementina-6502/pkg/core/managers"
	"github.com/fran150/clementina-6502/pkg/terminal"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/rivo/tview"
)

type BenEaterComputerConsole struct {
	*computers.Console

	computer       *BenEaterComputer
	grid           *tview.Grid
	app            *tview.Application
	windowsManager terminal.WindowManager
}

// newMainConsole creates and initializes a new console for the Ben Eater computer.
func NewBenEaterEmulationConsole(computer *BenEaterComputer) *BenEaterComputerConsole {
	wm := terminal.NewWindowManager()

	config := &computers.ConsoleBuildConfig{
		WindowManager:     wm,
		NavigationManager: managers.NewNavigationManager(),
		InputHandler:      terminal.NewDefaultInputHandler(wm),
		App:               tview.NewApplication(),
	}

	console := &BenEaterComputerConsole{
		Console:        computers.NewConsole(config),
		computer:       computer,
		grid:           tview.NewGrid(),
		app:            config.App,
		windowsManager: config.WindowManager,
	}

	console.initializeMainGrid()

	return console
}

/************************************************************************************
* Initialization methods
*************************************************************************************/

func (c *BenEaterComputerConsole) SetEmulator(emulator interfaces.Emulator) {
	menuOptions := createMenuOptions(c, emulator)

	// Initialize all windows
	c.windowsManager.AddWindow("lcd", ui.NewDisplayWindow(c.computer.chips.lcd))
	c.windowsManager.AddWindow("code", ui.NewCodeWindow(c.computer.chips.cpu, c.computer.getPotentialOperators))
	c.windowsManager.AddWindow("speed", ui.NewSpeedWindow(emulator.GetSpeedController()))
	c.windowsManager.AddWindow("cpu", ui.NewCpuWindow(c.computer.chips.cpu))
	c.windowsManager.AddWindow("via", ui.NewViaWindow(c.computer.chips.via))
	c.windowsManager.AddWindow("lcd_controller", ui.NewLcdWindow(c.computer.chips.lcd))
	c.windowsManager.AddWindow("acia", ui.NewAciaWindow(c.computer.chips.acia))
	c.windowsManager.AddWindow("ram", ui.NewMemoryWindow(c.computer.chips.ram))
	c.windowsManager.AddWindow("rom", ui.NewMemoryWindow(c.computer.chips.rom))
	busWindow := ui.NewBusWindow()
	c.windowsManager.AddWindow("bus", busWindow)
	c.windowsManager.AddWindow("breakpoint", ui.NewBreakPointForm(emulator.GetBreakpointManager()))
	c.windowsManager.AddWindow("options", ui.NewOptionsWindow(menuOptions))

	initializeBusWindow(c.computer, busWindow)

	c.initializeLayout()

	// Set initial active window
	c.ShowWindow("cpu")

}

// initializeMainGrid sets up the main grid layout for the console.
func (c *BenEaterComputerConsole) initializeMainGrid() {
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
func (c *BenEaterComputerConsole) initializeLayout() {
	// Setup initial grid layout
	c.grid.AddItem(c.windowsManager.GetWindow("lcd").GetDrawArea(), 0, 0, 1, 1, 0, 0, false).
		AddItem(c.windowsManager.GetWindow("speed").GetDrawArea(), 1, 0, 1, 1, 0, 0, false).
		AddItem(c.windowsManager.GetWindow("code").GetDrawArea(), 2, 0, 1, 1, 0, 0, false).
		AddItem(c.windowsManager.GetWindow("options").GetDrawArea(), 3, 0, 1, 2, 0, 0, false).
		AddItem(c.windowsManager.GetPages(), 0, 1, 3, 1, 0, 0, true)
}
