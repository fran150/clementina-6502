package beneater

import (
	"github.com/fran150/clementina-6502/pkg/computers"
	"github.com/fran150/clementina-6502/pkg/core/interfaces"
	"github.com/fran150/clementina-6502/pkg/core/managers"
	"github.com/fran150/clementina-6502/pkg/terminal"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/rivo/tview"
)

type BenEaterComputerConsoleConfig struct {
	*computers.BaseTerminalEmulatorConsole
}

type BenEaterComputerConsole struct {
	*computers.BaseTerminalEmulatorConsole
	computer *BenEaterComputer
	grid     *tview.Grid
}

// newMainConsole creates and initializes a new console for the Ben Eater computer.
func NewBenEaterEmulationConsole(computer *BenEaterComputer) *BenEaterComputerConsole {
	wm := terminal.NewWindowManager()

	config := computers.BaseTerminalEmulatorConsoleConfig{
		WindowManager:     wm,
		NavigationManager: managers.NewDefaultNavigationManager(),
		InputHandler:      terminal.NewDefaultInputHandler(wm),
		App:               tview.NewApplication(),
	}

	console := &BenEaterComputerConsole{
		BaseTerminalEmulatorConsole: computers.NewBaseTerminalEmulatorConsole(config),
		computer:                    computer,
		grid:                        tview.NewGrid(),
	}

	console.initializeMainGrid()

	return console
}

/************************************************************************************
* Initialization methods
*************************************************************************************/

func (c *BenEaterComputerConsole) SetEmulator(emulator interfaces.Emulator) {
	menuOptions := createMenuOptions(c, emulator)

	wm := c.GetWindowManager()

	// Initialize all windows
	wm.AddWindow("lcd", ui.NewDisplayWindow(c.computer.chips.lcd))
	wm.AddWindow("code", ui.NewCodeWindow(c.computer.chips.cpu, c.computer.getPotentialOperators))
	wm.AddWindow("speed", ui.NewSpeedWindow(emulator.GetSpeedController()))
	wm.AddWindow("cpu", ui.NewCpuWindow(c.computer.chips.cpu))
	wm.AddWindow("via", ui.NewViaWindow(c.computer.chips.via))
	wm.AddWindow("lcd_controller", ui.NewLcdWindow(c.computer.chips.lcd))
	wm.AddWindow("acia", ui.NewAciaWindow(c.computer.chips.acia))
	wm.AddWindow("ram", ui.NewMemoryWindow(c.computer.chips.ram))
	wm.AddWindow("rom", ui.NewMemoryWindow(c.computer.chips.rom))
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
func (c *BenEaterComputerConsole) initializeMainGrid() {
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
func (c *BenEaterComputerConsole) initializeLayout() {
	// Setup initial grid layout
	wm := c.GetWindowManager()
	c.grid.AddItem(wm.GetWindow("lcd").GetDrawArea(), 0, 0, 1, 1, 0, 0, false).
		AddItem(wm.GetWindow("speed").GetDrawArea(), 1, 0, 1, 1, 0, 0, false).
		AddItem(wm.GetWindow("code").GetDrawArea(), 2, 0, 1, 1, 0, 0, false).
		AddItem(wm.GetWindow("options").GetDrawArea(), 3, 0, 1, 2, 0, 0, false).
		AddItem(wm.GetPages(), 0, 1, 3, 1, 0, 0, true)
}
