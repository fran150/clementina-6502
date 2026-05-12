package clementina

import (
	"github.com/fran150/clementina-6502/pkg/core"
	"github.com/fran150/clementina-6502/pkg/terminal"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/rivo/tview"
)

// clementinaEmulatorConsoleConfig holds the configuration for creating a new Clementina emulator console.
// It embeds the base EmulatorConsoleConfig and adds a reference to the Clementina emulator instance.
type clementinaEmulatorConsoleConfig struct {
	terminal.BaseEmulatorConsoleConfig
	emulator *clementinaEmulator
}

// clementinaEmulatorConsole represents the main console interface for the Clementina 6502 emulator.
// It manages the terminal UI, window layout, and user interactions for the emulator.
type clementinaEmulatorConsole struct {
	terminal.EmulatorConsole

	app               *tview.Application
	windowManager     terminal.WindowManager
	navigationManager core.NavigationManager

	grid *tview.Grid
}

// newClementinaEmulatorConsole creates a new instance of the Clementina emulator console.
// It initializes the console with the provided configuration, sets up the main grid layout,
// creates menu options, initializes all windows (code, speed, CPU, VIA, memory windows, etc.),
// configures the bus window, and sets up the initial layout with the CPU window as active.
//
// Parameters:
//   - config: Configuration struct containing emulator console settings and emulator instance
//
// Returns:
//   - *clementinaEmulatorConsole: A fully initialized console ready for user interaction
func newClementinaEmulatorConsole(config clementinaEmulatorConsoleConfig) *clementinaEmulatorConsole {

	console := &clementinaEmulatorConsole{
		EmulatorConsole:   terminal.NewEmulatorConsole(config.BaseEmulatorConsoleConfig),
		app:               config.App,
		windowManager:     config.WindowManager,
		navigationManager: config.NavigationManager,
		grid:              tview.NewGrid(),
	}

	console.initializeMainGrid()

	menuOptions := createMenuOptions(console, config.emulator)

	computer := config.emulator.computer
	wm := config.WindowManager

	// Initialize all windows
	wm.AddWindow("code", ui.NewCodeWindow(computer.chips.cpu, computer.getPotentialOperators))
	wm.AddWindow("speed", ui.NewSpeedWindow(config.emulator.speedController))
	wm.AddWindow("cpu", ui.NewCpuWindow(computer.chips.cpu))
	wm.AddWindow("via", ui.NewViaWindow(computer.chips.via))
	wm.AddWindow("baseram", ui.NewMemoryWindow(computer.chips.baseram))
	wm.AddWindow("exram", ui.NewMemoryWindow(computer.chips.exram))
	wm.AddWindow("goto", ui.NewMemoryWindowGoToForm())
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
func (c *clementinaEmulatorConsole) initializeMainGrid() {
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
func (c *clementinaEmulatorConsole) initializeLayout() {
	// Setup initial grid layout
	wm := c.windowManager
	c.grid.AddItem(wm.GetWindow("speed").GetDrawArea(), 0, 0, 1, 1, 0, 0, false).
		AddItem(wm.GetWindow("code").GetDrawArea(), 1, 0, 1, 1, 0, 0, false).
		AddItem(wm.GetWindow("options").GetDrawArea(), 2, 0, 1, 2, 0, 0, false).
		AddItem(wm.GetPages(), 0, 1, 2, 1, 0, 0, true)
}

/************************************************************************************
* Window switching methods
*************************************************************************************/

// ShowGotoForm shows the go to form for memory navigation allowing to navigate back.
func (c *clementinaEmulatorConsole) ShowGotoForm() {
	wm := c.windowManager
	activeKey := c.navigationManager.GetCurrent()

	if memoryWindow := terminal.GetWindow[ui.MemoryWindow](wm, activeKey); memoryWindow != nil {
		if gotoWindow := terminal.GetWindow[ui.MemoryWindowGoToForm](wm, "goto"); gotoWindow != nil {
			if optionsWindow := terminal.GetWindow[ui.OptionsWindow](wm, "options"); optionsWindow != nil {
				gotoWindow.InitForm(memoryWindow, func() {
					optionsWindow.GoToPreviousMenu()
				})

				c.SwitchToBreakpointConfigMode()
			}
		}
	}
}
