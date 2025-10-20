package clementina

import (
	"github.com/fran150/clementina-6502/pkg/computers"
	"github.com/fran150/clementina-6502/pkg/core/interfaces"
	"github.com/fran150/clementina-6502/pkg/core/managers"
	"github.com/fran150/clementina-6502/pkg/terminal"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/rivo/tview"
)

type console struct {
	*computers.Console

	grid              *tview.Grid
	app               *tview.Application
	windowsManager    terminal.WindowManager
	navigationManager interfaces.NavigationManager
}

// NewClementinaEmulationConsole creates and initializes a new console for the Clementina computer.
//
// Parameters:
//   - computer: The ClementinaComputer instance to create the console for
//
// Returns:
//   - A configured console ready for use
func NewClementinaEmulationConsole(computer *ClementinaComputer, emulator interfaces.Emulator) *console {
	wm := terminal.NewWindowManager()

	config := &computers.ConsoleBuildConfig{
		WindowManager:     wm,
		NavigationManager: managers.NewDefaultNavigationManager(),
		InputHandler:      terminal.NewDefaultInputHandler(wm),
		App:               tview.NewApplication(),
	}

	console := &console{
		Console:        computers.NewConsole(config),
		grid:           tview.NewGrid(),
		app:            config.App,
		windowsManager: config.WindowManager,
	}

	console.initializeMainGrid()

	menuOptions := createMenuOptions(console, emulator)

	// Initialize all windows
	wm.AddWindow("code", ui.NewCodeWindow(computer.chips.cpu, computer.getPotentialOperators))
	wm.AddWindow("speed", ui.NewSpeedWindow(emulator.GetSpeedController()))
	wm.AddWindow("cpu", ui.NewCpuWindow(computer.chips.cpu))
	wm.AddWindow("via", ui.NewViaWindow(computer.chips.via))
	wm.AddWindow("baseram", ui.NewMemoryWindow(computer.chips.baseram))
	wm.AddWindow("exram", ui.NewMemoryWindow(computer.chips.exram))
	wm.AddWindow("hiram", ui.NewMemoryWindow(computer.chips.hiram))
	wm.AddWindow("goto", ui.NewMemoryWindowGoToForm())
	busWindow := ui.NewBusWindow()
	wm.AddWindow("bus", busWindow)
	wm.AddWindow("breakpoint", ui.NewBreakPointForm(emulator.GetBreakpointManager()))
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
func (c *console) initializeMainGrid() {
	c.grid.SetRows(3, 0, 3).
		SetColumns(25, 0).
		SetBorder(true).
		SetTitle("Clementina 6502 Computer")

	// Get the tview app from the framework and set the grid as root
	c.app.SetRoot(c.grid, true)
}

// initializeBusWindow configures the bus window with the computer's buses.
//
// Parameters:
//   - computer: The ClementinaComputer instance to get buses from
//   - busWindow: The bus window to configure
func initializeBusWindow(computer *ClementinaComputer, busWindow *ui.BusWindow) {
	busWindow.AddBus16("Address Bus", computer.circuit.addressBus)
	busWindow.AddBus8("Data Bus", computer.circuit.dataBus)
	busWindow.AddBus8("Port A", computer.circuit.portABus)
	busWindow.AddBus8("Port B", computer.circuit.portBBus)
}

// initializeLayout sets up the initial layout of console windows.
func (c *console) initializeLayout() {
	// Setup initial grid layout
	c.grid.AddItem(c.windowsManager.GetWindow("speed").GetDrawArea(), 0, 0, 1, 1, 0, 0, false).
		AddItem(c.windowsManager.GetWindow("code").GetDrawArea(), 1, 0, 1, 1, 0, 0, false).
		AddItem(c.windowsManager.GetWindow("options").GetDrawArea(), 2, 0, 1, 2, 0, 0, false).
		AddItem(c.windowsManager.GetPages(), 0, 1, 2, 1, 0, 0, true)
}

/************************************************************************************
* Window switching methods
*************************************************************************************/

// ShowGotoForm shows the go to form for memory navigation allowing to navigate back.
func (c *console) ShowGotoForm() {
	activeKey := c.navigationManager.GetCurrent()

	if memoryWindow := terminal.GetWindow[ui.MemoryWindow](c.windowsManager, activeKey); memoryWindow != nil {
		if gotoWindow := terminal.GetWindow[ui.MemoryWindowGoToForm](c.windowsManager, "goto"); gotoWindow != nil {
			if optionsWindow := terminal.GetWindow[ui.OptionsWindow](c.windowsManager, "options"); optionsWindow != nil {
				gotoWindow.InitForm(memoryWindow, func() {
					optionsWindow.GoToPreviousMenu()
				})

				c.SetBreakpointConfigMode()
			}
		}
	}
}
