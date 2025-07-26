package clementina

import (
	"github.com/fran150/clementina-6502/pkg/computers"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/rivo/tview"
)

type console struct {
	computers.BaseConsole

	grid *tview.Grid
}

// newMainConsole creates and initializes a new console for the Clementina computer.
//
// Parameters:
//   - computer: The ClementinaComputer instance to create the console for
//
// Returns:
//   - A configured console ready for use
func newMainConsole(computer *ClementinaComputer) *console {
	console := &console{
		grid:        tview.NewGrid(),
		BaseConsole: *computers.NewBaseConsole(),
	}

	console.initializeMainGrid(computer.ConsoleApp())

	menuOptions := createMenuOptions(computer, console)

	// Initialize all windows
	console.AddWindow("code", ui.NewCodeWindow(computer.chips.cpu, computer.getPotentialOperators))
	console.AddWindow("speed", ui.NewSpeedWindow(&computer.Loop().GetConfig().TargetSpeedMhz))
	console.AddWindow("cpu", ui.NewCpuWindow(computer.chips.cpu))
	console.AddWindow("via", ui.NewViaWindow(computer.chips.via))
	console.AddWindow("baseram", ui.NewMemoryWindow(computer.chips.baseram))
	console.AddWindow("exram", ui.NewMemoryWindow(computer.chips.exram))
	console.AddWindow("hiram", ui.NewMemoryWindow(computer.chips.hiram))
	console.AddWindow("goto", ui.NewMemoryWindowGoToForm())
	busWindow := ui.NewBusWindow()
	console.AddWindow("bus", busWindow)
	console.AddWindow("breakpoint", ui.NewBreakPointForm())
	console.AddWindow("options", ui.NewOptionsWindow(menuOptions))

	initializeBusWindow(computer, busWindow)

	console.initializeLayout()

	// Set initial active window
	console.SetActiveWindow("cpu")

	return console
}

/************************************************************************************
* Initialization methods
*************************************************************************************/

// initializeMainGrid sets up the main grid layout for the console.
//
// Parameters:
//   - tvApp: The tview application to set the root for
func (c *console) initializeMainGrid(tvApp *tview.Application) {
	c.grid.SetRows(3, 0, 3).
		SetColumns(25, 0).
		SetBorder(true).
		SetTitle("Clementina 6502 Computer")

	tvApp.SetRoot(c.grid, true)
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
	c.grid.AddItem(c.GetWindow("speed").GetDrawArea(), 0, 0, 1, 1, 0, 0, false).
		AddItem(c.GetWindow("code").GetDrawArea(), 1, 0, 1, 1, 0, 0, false).
		AddItem(c.GetWindow("options").GetDrawArea(), 2, 0, 1, 2, 0, 0, false).
		AddItem(c.GetPages(), 0, 1, 2, 1, 0, 0, true)
}

/************************************************************************************
* Window switching methods
*************************************************************************************/

// ShowGotoForm shows the go to form for memory navigation allowing to navigate back.
func (c *console) ShowGotoForm() {
	if memoryWin := computers.GetWindow[ui.MemoryWindow](&c.BaseConsole, c.GetActiveWindow()); memoryWin != nil {
		if form := computers.GetWindow[ui.MemoryWindowGoToForm](&c.BaseConsole, "goto"); form != nil {
			if options := computers.GetWindow[ui.OptionsWindow](&c.BaseConsole, "options"); options != nil {
				form.InitForm(memoryWin, func() { options.GoToPreviousMenu() })
				c.AppendActiveWindow("goto")
			}
		}
	}
}
