package computers

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/terminal"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TViewConsole provides a tview-based console implementation.
type TViewConsole struct {
	console       *Console
	pages         *tview.Pages
	inputHandler  InputHandler
	windowManager WindowManager
	app           *tview.Application
}

// NewTViewConsole creates a new tview-based console.
//
// Returns:
//   - A pointer to the initialized TViewConsole
func NewTViewConsole() *TViewConsole {
	windowManager := NewWindowManager()
	navigationManager := NewNavigationManager()
	console := NewConsole(windowManager, navigationManager)
	app := tview.NewApplication()

	tviewConsole := &TViewConsole{
		console:       console,
		pages:         tview.NewPages(),
		windowManager: windowManager,
		app:           app,
	}

	// Create input handler that delegates to the console
	tviewConsole.inputHandler = &DefaultInputHandler{
		windowManager: windowManager,
		console:       tviewConsole,
	}

	// Configure the framework
	app.SetInputCapture(tviewConsole.inputHandler.HandleKey)
	app.EnableMouse(true)
	app.EnablePaste(true)

	// Set the pages as the root of the tview app
	app.SetRoot(tviewConsole.pages, true)

	return tviewConsole
}

func (tc *TViewConsole) GetWindowManager() WindowManager {
	return tc.windowManager
}

func (tc *TViewConsole) SetRoot(root tview.Primitive) {
	tc.app.SetRoot(root, true)
}

// AddWindow adds a new window to the console.
//
// Parameters:
//   - key: The unique identifier for the window
//   - window: The window instance to add
func (tc *TViewConsole) AddWindow(key string, window terminal.Window) {
	tc.console.AddWindow(key, window)
	tc.pages.AddPage(key, window.GetDrawArea(), true, true)
}

// GetWindow retrieves a window by its key.
//
// Parameters:
//   - key: The unique identifier of the window to retrieve
//
// Returns:
//   - The window instance, or nil if not found
func (tc *TViewConsole) GetWindow(key string) terminal.Window {
	return tc.console.GetWindow(key)
}

// RemoveWindow removes a window by its key.
//
// Parameters:
//   - key: The unique identifier of the window to remove
func (tc *TViewConsole) RemoveWindow(key string) {
	tc.console.RemoveWindow(key)
	tc.pages.RemovePage(key)
}

// ShowWindow activates the specified window.
//
// Parameters:
//   - windowKey: The key identifying the window to show
func (tc *TViewConsole) ShowWindow(windowKey string) {
	tc.console.ShowWindow(windowKey)
	tc.pages.SwitchToPage(windowKey)
}

// SetBreakpointConfigMode activates the breakpoint configuration window.
func (tc *TViewConsole) SetBreakpointConfigMode() {
	tc.console.SetBreakpointConfigMode()
	tc.pages.SwitchToPage("breakpoint")
}

// ReturnToPreviousWindow returns to the previous window.
func (tc *TViewConsole) ReturnToPreviousWindow() {
	tc.console.ReturnToPreviousWindow()
	tc.pages.SwitchToPage(tc.console.GetActiveWindow())
}

// ScrollUp scrolls the active memory window up by the specified number of lines.
//
// Parameters:
//   - step: The number of lines to scroll up
func (tc *TViewConsole) ScrollUp(step uint32) {
	tc.console.ScrollUp(step)
}

// ScrollDown scrolls the active memory window down by the specified number of lines.
//
// Parameters:
//   - step: The number of lines to scroll down
func (tc *TViewConsole) ScrollDown(step uint32) {
	tc.console.ScrollDown(step)
}

// ShowEmulationSpeed displays the emulation speed configuration window.
func (tc *TViewConsole) ShowEmulationSpeed() {
	tc.console.ShowEmulationSpeed()
}

// Run starts the console application.
//
// Returns:
//   - An error if the application fails to start
func (tc *TViewConsole) Run() error {
	return tc.app.Run()
}

// Stop stops the console application.
func (tc *TViewConsole) Stop() {
	tc.app.Stop()
}

// Draw clears and draws all windows in the console.
//
// Parameters:
//   - context: The current step context
func (tc *TViewConsole) Draw(context *common.StepContext) {
	tc.console.Draw(context)
	tc.app.Draw()
}

// Tick updates the console components that need to be updated every cycle.
//
// Parameters:
//   - context: The current step context
func (tc *TViewConsole) Tick(context *common.StepContext) {
	tc.console.Tick(context)
}

// GetActiveWindow returns the key of the currently active window.
//
// Returns:
//   - The key of the currently active window
func (tc *TViewConsole) GetActiveWindow() string {
	return tc.console.GetActiveWindow()
}

// GetPages returns the tview.Pages component for advanced usage.
//
// Returns:
//   - The tview.Pages instance
func (tc *TViewConsole) GetPages() *tview.Pages {
	return tc.pages
}

// GetConsole returns the underlying console for direct access.
//
// Returns:
//   - The Console instance
func (tc *TViewConsole) GetConsole() *Console {
	return tc.console
}

// DefaultInputHandler provides default input handling for the console.
type DefaultInputHandler struct {
	windowManager WindowManager
	console       *TViewConsole
}

// HandleKey processes a key event and returns the modified event or nil.
//
// Parameters:
//   - event: The key event to process
//
// Returns:
//   - The processed key event or nil
func (dih *DefaultInputHandler) HandleKey(event *tcell.EventKey) *tcell.EventKey {
	// Delegate to the options window if it exists
	// This maintains compatibility with the original implementation

	if window := GetWindow[ui.OptionsWindow](dih.windowManager, "options"); window != nil {
		window.ProcessKey(event)
	}

	return event
}
