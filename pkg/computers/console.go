package computers

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/core/interfaces"
	"github.com/fran150/clementina-6502/pkg/terminal"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/rivo/tview"
)

// Console provides the core console functionality without UI framework dependencies.
type Console struct {
	windowManager     terminal.WindowManager
	navigationManager interfaces.NavigationManager
	inputHandler      terminal.InputHandler
	app               *tview.Application
}

// ConsoleBuildConfig contains the objects required to build the console.
type ConsoleBuildConfig struct {
	WindowManager     terminal.WindowManager
	NavigationManager interfaces.NavigationManager
	InputHandler      terminal.InputHandler
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
		app:               config.App,
	}

	// Configure the framework
	console.app.SetInputCapture(console.inputHandler.HandleKey)
	console.app.EnableMouse(true)
	console.app.EnablePaste(true)

	// Set the pages as the root of the tview app
	console.app.SetRoot(console.windowManager.GetPages(), true)

	return console
}

// ShowWindow activates the specified window.
//
// Parameters:
//   - windowKey: The key identifying the window to show
func (c *Console) ShowWindow(windowKey string) {
	c.navigationManager.NavigateTo(windowKey)
	c.windowManager.SwitchToPage(windowKey)
}

// SetBreakpointConfigMode activates the breakpoint configuration window.
func (c *Console) SetBreakpointConfigMode() {
	c.navigationManager.PushToHistory("breakpoint")
	c.windowManager.SwitchToPage("breakpoint")
}

// ReturnToPreviousWindow returns to the previous window.
func (c *Console) ReturnToPreviousWindow() {
	c.navigationManager.GoBack()
	c.windowManager.SwitchToPage(c.navigationManager.GetCurrent())
}

// ScrollUp scrolls the active memory window up by the specified number of lines.
//
// Parameters:
//   - step: The number of lines to scroll up
func (c *Console) ScrollUp(step uint32) {
	activeKey := c.navigationManager.GetCurrent()

	if window := terminal.GetWindow[ui.MemoryWindow](c.windowManager, activeKey); window != nil {
		window.ScrollUp(step)
	}
}

// ScrollDown scrolls the active memory window down by the specified number of lines.
//
// Parameters:
//   - step: The number of lines to scroll down
func (c *Console) ScrollDown(step uint32) {
	activeKey := c.navigationManager.GetCurrent()

	if window := terminal.GetWindow[ui.MemoryWindow](c.windowManager, activeKey); window != nil {
		window.ScrollDown(step)
	}
}

// RemoveSelectedItem removes the currently selected item from the breakpoint form window.
func (c *Console) RemoveSelectedItem() {
	if window := terminal.GetWindow[ui.BreakPointForm](c.windowManager, "breakpoint"); window != nil {
		window.RemoveSelectedItem()
	}
}

// ShowEmulationSpeed displays the emulation speed configuration window.
func (c *Console) ShowEmulationSpeed() {
	if window := terminal.GetWindow[ui.SpeedWindow](c.windowManager, "speed"); window != nil {
		window.ShowConfig()
	}
}

// Draw clears and draws all windows in the console.
//
// Parameters:
//   - context: The current step context
func (c *Console) Draw(context *common.StepContext) {
	c.windowManager.GetAllWindows(func(key string, window terminal.Window) bool {
		window.Clear()
		window.Draw(context)
		return true // continue iteration
	})

	c.app.Draw()
}

// Tick updates the console components that need to be updated every cycle.
//
// Parameters:
//   - context: The current step context
func (c *Console) Tick(context *common.StepContext) {
	c.windowManager.GetTickers(func(key string, ticker terminal.TickerWindow) bool {
		ticker.Tick(context)
		return true // continue iteration
	})
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
