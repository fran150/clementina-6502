package clementina

import (
	"github.com/fran150/clementina-6502/pkg/computers"
	"github.com/fran150/clementina-6502/pkg/core"
	"github.com/fran150/clementina-6502/pkg/terminal"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/rivo/tview"
)

type ClementinaEmulatorConsoleConfig struct {
	computers.BaseTerminalEmulatorConsoleConfig

	Computer *ClementinaComputer
}

type ClementinaEmulatorConsole struct {
	*computers.BaseTerminalEmulatorConsole
	computer *ClementinaComputer
	grid     *tview.Grid
}

func NewClementinaEmulatorConsole(config ClementinaEmulatorConsoleConfig) *ClementinaEmulatorConsole {

	console := &ClementinaEmulatorConsole{
		BaseTerminalEmulatorConsole: computers.NewBaseTerminalEmulatorConsole(config.BaseTerminalEmulatorConsoleConfig),
		computer:                    config.Computer,
		grid:                        tview.NewGrid(),
	}

	console.initializeMainGrid()

	return console
}

/************************************************************************************
* Initialization methods
*************************************************************************************/

func (c *ClementinaEmulatorConsole) SetEmulator(emulator core.BaseEmulator) {
	menuOptions := createMenuOptions(c, emulator)

	wm := c.GetWindowManager()

	// Initialize all windows
	wm.AddWindow("code", ui.NewCodeWindow(c.computer.chips.cpu, c.computer.getPotentialOperators))
	wm.AddWindow("speed", ui.NewSpeedWindow(emulator.GetSpeedController()))
	wm.AddWindow("cpu", ui.NewCpuWindow(c.computer.chips.cpu))
	wm.AddWindow("via", ui.NewViaWindow(c.computer.chips.via))
	wm.AddWindow("baseram", ui.NewMemoryWindow(c.computer.chips.baseram))
	wm.AddWindow("exram", ui.NewMemoryWindow(c.computer.chips.exram))
	wm.AddWindow("hiram", ui.NewMemoryWindow(c.computer.chips.hiram))
	wm.AddWindow("goto", ui.NewMemoryWindowGoToForm())
	busWindow := ui.NewBusWindow()
	wm.AddWindow("bus", busWindow)
	wm.AddWindow("breakpoint", ui.NewBreakPointForm(emulator.GetBreakpointManager()))
	wm.AddWindow("options", ui.NewOptionsWindow(menuOptions))

	initializeBusWindow(c.computer, busWindow)

	c.initializeLayout()

	// Set initial active window
	c.ShowWindow("cpu")
}

// initializeMainGrid sets up the main grid layout for the console.
func (c *ClementinaEmulatorConsole) initializeMainGrid() {
	c.grid.SetRows(3, 0, 3).
		SetColumns(25, 0).
		SetBorder(true).
		SetTitle("Clementina 6502 Computer")

	// Get the tview app from the framework and set the grid as root
	c.GetApp().SetRoot(c.grid, true)
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
func (c *ClementinaEmulatorConsole) initializeLayout() {
	// Setup initial grid layout
	wm := c.GetWindowManager()
	c.grid.AddItem(wm.GetWindow("speed").GetDrawArea(), 0, 0, 1, 1, 0, 0, false).
		AddItem(wm.GetWindow("code").GetDrawArea(), 1, 0, 1, 1, 0, 0, false).
		AddItem(wm.GetWindow("options").GetDrawArea(), 2, 0, 1, 2, 0, 0, false).
		AddItem(wm.GetPages(), 0, 1, 2, 1, 0, 0, true)
}

/************************************************************************************
* Window switching methods
*************************************************************************************/

// ShowGotoForm shows the go to form for memory navigation allowing to navigate back.
func (c *ClementinaEmulatorConsole) ShowGotoForm() {
	activeKey := c.GetNavigationManager().GetCurrent()

	wm := c.GetWindowManager()

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
