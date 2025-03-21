package beneater

import (
	"github.com/fran150/clementina6502/internal/slicesext"
	"github.com/fran150/clementina6502/pkg/common"
	"github.com/fran150/clementina6502/pkg/terminal"
	"github.com/fran150/clementina6502/pkg/terminal/ui"
	"github.com/rivo/tview"
)

type console struct {
	grid     *tview.Grid
	windows  map[string]terminal.Window
	active   terminal.Window
	previous []terminal.Window
}

func newMainConsole(computer *BenEaterComputer, tvApp *tview.Application) *console {
	console := &console{
		grid:     tview.NewGrid(),
		windows:  make(map[string]terminal.Window),
		previous: make([]terminal.Window, 2),
	}

	console.initializeMainGrid(tvApp)

	menuOptions := createMenuOptions(computer, console)

	// Initialize all windows
	console.windows["lcd"] = ui.NewDisplayWindow(computer.chips.lcd)
	console.windows["code"] = ui.NewCodeWindow(computer.chips.cpu, computer.getPotentialOperators)
	console.windows["speed"] = ui.NewSpeedWindow(&computer.appConfig.EmulationLoopConfig)
	console.windows["cpu"] = ui.NewCpuWindow(computer.chips.cpu)
	console.windows["via"] = ui.NewViaWindow(computer.chips.via)
	console.windows["lcd_controller"] = ui.NewLcdWindow(computer.chips.lcd)
	console.windows["acia"] = ui.NewAciaWindow(computer.chips.acia)
	console.windows["ram"] = ui.NewMemoryWindow(computer.chips.ram)
	console.windows["rom"] = ui.NewMemoryWindow(computer.chips.rom)
	busWindow := ui.NewBusWindow()
	console.windows["bus"] = busWindow
	console.windows["breakpoint"] = ui.NewBreakPointForm()
	console.windows["options"] = ui.NewOptionsWindow(menuOptions)

	initializeBusWindow(computer, busWindow)

	console.initializeLayout()

	// Set initial active window
	console.SetActiveWindow(console.windows["cpu"])

	return console
}

/************************************************************************************
* Initialization methods
*************************************************************************************/

func (c *console) initializeMainGrid(tvApp *tview.Application) {
	c.grid.SetRows(4, 3, 0, 3).
		SetColumns(25, 0).
		SetBorder(true).
		SetTitle("Ben Eater 6502 Computer")

	tvApp.SetRoot(c.grid, true)
}

func initializeBusWindow(computer *BenEaterComputer, busWindow *ui.BusWindow) {
	busWindow.AddBus16("Address Bus", computer.circuit.addressBus)
	busWindow.AddBus8("Data Bus", computer.circuit.dataBus)
	busWindow.AddBus8("Port A", computer.circuit.portABus)
	busWindow.AddBus8("Port B", computer.circuit.portBBus)
}

func (c *console) initializeLayout() {
	// Setup initial grid layout
	c.grid.AddItem(c.windows["lcd"].GetDrawArea(), 0, 0, 1, 1, 0, 0, false).
		AddItem(c.windows["speed"].GetDrawArea(), 1, 0, 1, 1, 0, 0, false).
		AddItem(c.windows["code"].GetDrawArea(), 2, 0, 1, 1, 0, 0, false).
		AddItem(c.windows["options"].GetDrawArea(), 3, 0, 1, 2, 0, 0, false)

}

/************************************************************************************
* Window switching methods
*************************************************************************************/

func (c *console) SetBreakpointConfigMode(context *common.StepContext) {
	c.AppendActiveWindow(c.windows["breakpoint"])
}

func (c *console) ShowWindow(windowKey string, context *common.StepContext) {
	if window, exists := c.windows[windowKey]; exists {
		c.SetActiveWindow(window)
	}
}

/************************************************************************************
* Menu methods
*************************************************************************************/

func (c *console) ScrollUp(context *common.StepContext, step uint16) {
	if explorer, ok := c.active.(*ui.MemoryWindow); ok {
		explorer.ScrollUp(step)
	}
}

func (c *console) ScrollDown(context *common.StepContext, step uint16) {
	if explorer, ok := c.active.(*ui.MemoryWindow); ok {
		explorer.ScrollDown(step)
	}
}

func (c *console) ShowEmulationSpeed(context *common.StepContext) {
	if speedWindow := GetWindow[ui.SpeedWindow](c, "speed"); speedWindow != nil {
		speedWindow.ShowConfig(context)
	}
}

/************************************************************************************
* Internal Functions
*************************************************************************************/

func (c *console) SetActiveWindow(value terminal.Window) {
	c.active = value
	c.setActiveWindowOnGrid()
}

func (c *console) AppendActiveWindow(value terminal.Window) {
	c.previous = append(c.previous, c.active)
	c.SetActiveWindow(value)
}

func (c *console) ReturnToPreviousWindow(context *common.StepContext) {
	if c.previous != nil {
		previous, active := slicesext.SlicePop(c.previous)
		c.previous = previous
		c.active = active
		c.setActiveWindowOnGrid()
	}
}

func (c *console) setActiveWindowOnGrid() {
	c.grid.RemoveItem(c.active.GetDrawArea())
	c.grid.AddItem(c.active.GetDrawArea(), 0, 1, 3, 1, 0, 0, false)
}

/************************************************************************************
* Public methods
*************************************************************************************/

func (c *console) Draw(context *common.StepContext) {
	// Clear and draw all windows
	for _, window := range c.windows {
		window.Clear()
		window.Draw(context)
	}
}

func (c *console) Tick(context *common.StepContext) {
	if codeWindow := GetWindow[ui.CodeWindow](c, "code"); codeWindow != nil {
		codeWindow.Tick(context)
	}
}

// GetWindow is a generic function that retrieves and type-casts a window from the console's window map
func GetWindow[T any](c *console, key string) *T {
	if window, ok := c.windows[key]; ok {
		if typed, ok := any(window).(*T); ok {
			return typed
		}
	}
	return nil
}
