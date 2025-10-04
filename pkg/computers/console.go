package computers

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/core/interfaces"
	"github.com/fran150/clementina-6502/pkg/core/managers"
	"github.com/fran150/clementina-6502/pkg/terminal"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/rivo/tview"
)

// Console provides the core console functionality without UI framework dependencies.
type Console struct {
	windowManager     interfaces.WindowManager
	navigationManager interfaces.NavigationManager
	inputHandler      terminal.InputHandler
	pages             *tview.Pages
	app               *tview.Application
}

// ConsoleBuildConfig contains the objects required to build the console.
type ConsoleBuildConfig struct {
	WindowManager     interfaces.WindowManager
	NavigationManager interfaces.NavigationManager
	InputHandler      terminal.InputHandler
	Pages             *tview.Pages
	App               *tview.Application
}

// NewConsole creates a new console with the specified managers.
//
// Parameters:
//   - windowManager: The window manager to use
//   - navigationManager: The navigation manager to use
//
// Returns:
//   - A pointer to the initialized Console
func NewConsole(config *ConsoleBuildConfig) *Console {
	console := &Console{
		windowManager:     config.WindowManager,
		navigationManager: config.NavigationManager,
		inputHandler:      config.InputHandler,
		pages:             config.Pages,
		app:               config.App,
	}

	// Configure the framework
	console.app.SetInputCapture(console.inputHandler.HandleKey)
	console.app.EnableMouse(true)
	console.app.EnablePaste(true)

	// Set the pages as the root of the tview app
	console.app.SetRoot(console.pages, true)

	return console
}

// AddWindow adds a new window to the console.
//
// Parameters:
//   - key: The unique identifier for the window
//   - window: The window instance to add
func (c *Console) AddWindow(key string, window terminal.Window) {
	c.windowManager.AddWindow(key, window)
	c.pages.AddPage(key, window.GetDrawArea(), true, true)

}

// GetWindow retrieves a window by its key.
//
// Parameters:
//   - key: The unique identifier of the window to retrieve
//
// Returns:
//   - The window instance, or nil if not found
func (c *Console) GetWindow(key string) terminal.Window {
	return c.windowManager.GetWindow(key)
}

// RemoveWindow removes a window by its key.
//
// Parameters:
//   - key: The unique identifier of the window to remove
func (c *Console) RemoveWindow(key string) {
	c.windowManager.RemoveWindow(key)
	c.pages.RemovePage(key)
}

// ShowWindow activates the specified window.
//
// Parameters:
//   - windowKey: The key identifying the window to show
func (c *Console) ShowWindow(windowKey string) {
	c.navigationManager.NavigateTo(windowKey)
	c.pages.SwitchToPage(windowKey)
}

// SetBreakpointConfigMode activates the breakpoint configuration window.
func (c *Console) SetBreakpointConfigMode() {
	c.navigationManager.PushToHistory("breakpoint")
	c.pages.SwitchToPage("breakpoint")
}

// ReturnToPreviousWindow returns to the previous window.
func (c *Console) ReturnToPreviousWindow() {
	c.navigationManager.GoBack()
	c.pages.SwitchToPage(c.GetActiveWindow())
}

// ScrollUp scrolls the active memory window up by the specified number of lines.
//
// Parameters:
//   - step: The number of lines to scroll up
func (c *Console) ScrollUp(step uint32) {
	activeKey := c.navigationManager.GetCurrent()

	if window := managers.GetWindow[ui.MemoryWindow](c.windowManager, activeKey); window != nil {
		window.ScrollUp(step)
	}
}

// ScrollDown scrolls the active memory window down by the specified number of lines.
//
// Parameters:
//   - step: The number of lines to scroll down
func (c *Console) ScrollDown(step uint32) {
	activeKey := c.navigationManager.GetCurrent()

	if window := managers.GetWindow[ui.MemoryWindow](c.windowManager, activeKey); window != nil {
		window.ScrollDown(step)
	}
}

// RemoveSelectedItem removes the currently selected item from the breakpoint form window.
func (c *Console) RemoveSelectedItem() {
	if window := managers.GetWindow[ui.BreakPointForm](c.windowManager, "breakpoint"); window != nil {
		window.RemoveSelectedItem()
	}
}

// ShowEmulationSpeed displays the emulation speed configuration window.
func (c *Console) ShowEmulationSpeed() {
	if window := managers.GetWindow[ui.SpeedWindow](c.windowManager, "speed"); window != nil {
		window.ShowConfig()
	}
}

// GetActiveWindow returns the key of the currently active window.
//
// Returns:
//   - The key of the currently active window
func (c *Console) GetActiveWindow() string {
	return c.navigationManager.GetCurrent()
}

// Draw clears and draws all windows in the console.
//
// Parameters:
//   - context: The current step context
func (c *Console) Draw(context *common.StepContext) {
	for _, window := range c.windowManager.GetAllWindows() {
		window.Clear()
		window.Draw(context)
	}

	c.app.Draw()
}

// Tick updates the console components that need to be updated every cycle.
//
// Parameters:
//   - context: The current step context
func (c *Console) Tick(context *common.StepContext) {
	for _, ticker := range c.windowManager.GetTickers() {
		ticker.Tick(context)
	}
}

// Run starts the console application.
//
// Returns:
//   - An error if the application fails to start
func (c *Console) Run() error {
	return c.app.Run()
}

// Stop stops the console application.
func (c *Console) Stop() {
	c.app.Stop()
}

// GetWindowManager returns the window manager associated with this console.
func (c *Console) GetWindowManager() interfaces.WindowManager {
	return c.windowManager
}

// GetPages returns the tview.Pages instance associated with this console.
func (c *Console) GetPages() *tview.Pages {
	return c.pages
}

// SetRoot sets the root primitive for the console application.
//
// Parameters:
//   - root: The tview primitive to set as the application root
func (c *Console) SetRoot(root tview.Primitive) {
	c.app.SetRoot(root, true)
}
