package clementina

import (
	"github.com/fran150/clementina-6502/internal/slicesext"
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/terminal"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/rivo/tview"
)

type console struct {
	grid     *tview.Grid
	pages    *tview.Pages
	windows  map[string]terminal.Window
	active   string
	previous []string
}

func newMainConsole(computer *ClementinaComputer, tvApp *tview.Application) *console {
	console := &console{
		grid:     tview.NewGrid(),
		windows:  make(map[string]terminal.Window),
		previous: make([]string, 2),
	}

	console.initializeMainGrid(tvApp)

	menuOptions := createMenuOptions(computer, console)

	// Initialize all windows
	console.windows["code"] = ui.NewCodeWindow(computer.chips.cpu, computer.getPotentialOperators)
	console.windows["speed"] = ui.NewSpeedWindow(&computer.appConfig.EmulationLoopConfig)
	console.windows["cpu"] = ui.NewCpuWindow(computer.chips.cpu)
	console.windows["via"] = ui.NewViaWindow(computer.chips.via)
	busWindow := ui.NewBusWindow()
	console.windows["bus"] = busWindow
	console.windows["breakpoint"] = ui.NewBreakPointForm()
	console.windows["options"] = ui.NewOptionsWindow(menuOptions)

	console.pages = tview.NewPages()
	for key, window := range console.windows {
		console.pages.AddPage(key, window.GetDrawArea(), true, true)
	}

	initializeBusWindow(computer, busWindow)

	console.initializeLayout()

	// Set initial active window
	console.SetActiveWindow("cpu")

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

func initializeBusWindow(computer *ClementinaComputer, busWindow *ui.BusWindow) {
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
		AddItem(c.windows["options"].GetDrawArea(), 3, 0, 1, 2, 0, 0, false).
		AddItem(c.pages, 0, 1, 3, 1, 0, 0, true)
}

/************************************************************************************
* Window switching methods
*************************************************************************************/

func (c *console) SetBreakpointConfigMode(context *common.StepContext) {
	c.AppendActiveWindow("breakpoint")
}

// ShowWindow activates the specified window in the console.
//
// Parameters:
//   - windowKey: The key identifying the window to show
//   - context: The current step context
func (c *console) ShowWindow(windowKey string, context *common.StepContext) {
	c.SetActiveWindow(windowKey)
}

/************************************************************************************
* Menu methods
*************************************************************************************/

func (c *console) ScrollUp(context *common.StepContext, step uint16) {
	if explorer, ok := c.windows[c.active].(*ui.MemoryWindow); ok {
		explorer.ScrollUp(step)
	}
}

// ScrollDown scrolls the active memory window down by the specified number of lines.
// This only has an effect if the active window is a memory window.
//
// Parameters:
//   - context: The current step context
//   - step: The number of lines to scroll down
func (c *console) ScrollDown(context *common.StepContext, step uint16) {
	if explorer, ok := c.windows[c.active].(*ui.MemoryWindow); ok {
		explorer.ScrollDown(step)
	}
}

// ShowEmulationSpeed displays the emulation speed configuration window.
// This allows the user to view and adjust the current emulation speed.
//
// Parameters:
//   - context: The current step context
func (c *console) ShowEmulationSpeed(context *common.StepContext) {
	if speedWindow := GetWindow[ui.SpeedWindow](c, "speed"); speedWindow != nil {
		speedWindow.ShowConfig(context)
	}
}

/************************************************************************************
* Internal Functions
*************************************************************************************/

func (c *console) SetActiveWindow(key string) {
	c.active = key
	c.pages.SwitchToPage(key)
}

// AppendActiveWindow adds the current active window to the history stack
// and activates the specified window.
//
// Parameters:
//   - key: The key of the window to activate
func (c *console) AppendActiveWindow(key string) {
	c.previous = append(c.previous, c.active)
	c.SetActiveWindow(key)
}

// ReturnToPreviousWindow restores the previously active window from the history stack.
// If there is no previous window in the stack, this method has no effect.
//
// Parameters:
//   - context: The current step context
func (c *console) ReturnToPreviousWindow(context *common.StepContext) {
	if c.previous != nil {
		previous, active := slicesext.SlicePop(c.previous)
		c.previous = previous
		c.SetActiveWindow(active)
	}
}

/************************************************************************************
* Public methods
*************************************************************************************/

// Draw clears and draws all windows in the console.
// This method is typically called to refresh the display after changes have been made.
func (c *console) Draw(context *common.StepContext) {
	// Clear and draw all windows
	for _, window := range c.windows {
		window.Clear()
		window.Draw(context)
	}
}

// Tick updates the console components that need to be updated every cycle.
// Currently, this updates the code window to reflect the current execution state.
//
// Parameters:
//   - context: The current step context
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
