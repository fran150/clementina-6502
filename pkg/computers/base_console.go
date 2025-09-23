package computers

import (
	"github.com/fran150/clementina-6502/internal/slicesext"
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/terminal"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// BaseConsole provides the base functionality for console interfaces in computer emulations.
// It manages multiple windows, handles window switching, and maintains navigation history.
type BaseConsole struct {
	pages    *tview.Pages
	windows  map[string]terminal.Window
	tickers  map[string]terminal.TickerWindow
	active   string
	previous []string
	tvApp    *tview.Application
}

// NewBaseConsole creates a new instance of BaseConsole with initialized components.
//
// Returns:
//   - A pointer to the initialized BaseConsole
func NewBaseConsole(tvApp *tview.Application) *BaseConsole {
	console := &BaseConsole{
		pages:    tview.NewPages(),
		windows:  make(map[string]terminal.Window),
		tickers:  make(map[string]terminal.TickerWindow),
		previous: make([]string, 2),
		tvApp:    tvApp,
	}

	tvApp.SetInputCapture(console.KeyPressed).
		EnableMouse(true).
		EnablePaste(true)

	return console

}

/************************************************************************************
* General methods
*************************************************************************************/
// Get the tview application used for the console.
func (c *BaseConsole) ConsoleApp() *tview.Application {
	return c.tvApp
}

// Runs the console application.
func (c *BaseConsole) Run() error {
	return c.tvApp.Run()
}

// Finishes the console application.
func (c *BaseConsole) Stop() {
	c.tvApp.Stop()
}

/************************************************************************************
* Window CRUD methods
*************************************************************************************/
// AddWindow adds a new window to the console's window map.
//
// Parameters:
//   - key: The unique identifier for the window
//   - window: The window instance to add
func (c *BaseConsole) AddWindow(key string, window terminal.Window) {
	if _, exists := c.windows[key]; !exists {
		c.windows[key] = window

		if ticker, ok := window.(terminal.TickerWindow); ok {
			c.tickers[key] = ticker
		}

		c.pages.AddPage(key, window.GetDrawArea(), true, true)
	}
}

// GetWindow retrieves a window by its key from the console's window map.
//
// Parameters:
//   - key: The unique identifier of the window to retrieve
//
// Returns:
//   - The window instance, or nil if not found
func (c *BaseConsole) GetWindow(key string) terminal.Window {
	if window, exists := c.windows[key]; exists {
		return window
	}
	return nil
}

// DeleteWindow removes a window from the console's window map by its key.
//
// Parameters:
//   - key: The unique identifier of the window to remove
func (c *BaseConsole) DeleteWindow(key string) {
	if _, exists := c.windows[key]; exists {
		delete(c.windows, key)
		delete(c.tickers, key)
		c.pages.RemovePage(key)
	}
}

/************************************************************************************
* Window switching methods
*************************************************************************************/
// SetBreakpointConfigMode activates the breakpoint configuration window.
func (c *BaseConsole) SetBreakpointConfigMode() {
	c.AppendActiveWindow("breakpoint")
}

// ShowWindow activates the specified window in the console.
//
// Parameters:
//   - windowKey: The key identifying the window to show
func (c *BaseConsole) ShowWindow(windowKey string) {
	c.SetActiveWindow(windowKey)
}

/************************************************************************************
* Menu methods
*************************************************************************************/
// KeyPressed handles key press events and routes them to the appropriate window.
func (c *BaseConsole) KeyPressed(event *tcell.EventKey) *tcell.EventKey {
	options := GetWindow[ui.OptionsWindow](c, "options")
	return options.ProcessKey(event)
}

// ScrollUp scrolls the active memory window up by the specified number of lines.
// This only has an effect if the active window is a memory window.
//
// Parameters:
//   - step: The number of lines to scroll up
func (c *BaseConsole) ScrollUp(step uint32) {
	if explorer := GetWindow[ui.MemoryWindow](c, c.active); explorer != nil {
		explorer.ScrollUp(step)
	}
}

// ScrollDown scrolls the active memory window down by the specified number of lines.
// This only has an effect if the active window is a memory window.
//
// Parameters:
//   - step: The number of lines to scroll down
func (c *BaseConsole) ScrollDown(step uint32) {
	if explorer := GetWindow[ui.MemoryWindow](c, c.active); explorer != nil {
		explorer.ScrollDown(step)
	}
}

// ShowEmulationSpeed displays the emulation speed configuration window.
// This allows the user to view and adjust the current emulation speed.
func (c *BaseConsole) ShowEmulationSpeed() {
	if speedWindow := GetWindow[ui.SpeedWindow](c, "speed"); speedWindow != nil {
		speedWindow.ShowConfig()
	}
}

// SetActiveWindow sets the active window in the console and switches to it in the pages component.
//
// Parameters:
//   - key: The unique identifier of the window to activate
func (c *BaseConsole) SetActiveWindow(key string) {
	c.active = key
	c.pages.SwitchToPage(key)
}

// AppendActiveWindow adds the current active window to the history stack
// and activates the specified window.
//
// Parameters:
//   - key: The key of the window to activate
func (c *BaseConsole) AppendActiveWindow(key string) {
	c.previous = append(c.previous, c.active)
	c.SetActiveWindow(key)
}

// ReturnToPreviousWindow restores the previously active window from the history stack.
// If there is no previous window in the stack, this method has no effect.
func (c *BaseConsole) ReturnToPreviousWindow() {
	if c.previous != nil {
		previous, active := slicesext.SlicePop(c.previous)
		c.previous = previous
		c.SetActiveWindow(active)
	}
}

// GetPages returns the tview.Pages component that manages the console's windows.
func (c *BaseConsole) GetPages() *tview.Pages {
	return c.pages
}

// GetWindow is a generic function that retrieves and type-casts a window from the console's window map.
//
// Parameters:
//   - c: The BaseConsole instance to search in
//   - key: The unique identifier of the window to retrieve
//
// Returns:
//   - A pointer to the typed window, or nil if not found or type mismatch
func GetWindow[T any](c *BaseConsole, key string) *T {
	if window, ok := c.windows[key]; ok {
		if typed, ok := any(window).(*T); ok {
			return typed
		}
	}
	return nil
}

// Draw clears and draws all windows in the console.
// This method is typically called to refresh the display after changes have been made.
//
// Parameters:
//   - context: The current step context
func (c *BaseConsole) Draw(context *common.StepContext) {
	// Clear and draw all windows
	for _, window := range c.windows {
		window.Clear()
		window.Draw(context)
	}

	c.tvApp.Draw()
}

// Tick updates the console components that need to be updated every cycle.
//
// Parameters:
//   - context: The current step context
func (c *BaseConsole) Tick(context *common.StepContext) {
	for _, ticker := range c.tickers {
		ticker.Tick(context)
	}
}

// GetActiveWindow returns the key of the currently active window in the console.
func (c *BaseConsole) GetActiveWindow() string {
	return c.active
}
