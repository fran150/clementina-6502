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
	config *ConsoleConfig
}

// ConsoleConfig contains the objects required to build the console.
type ConsoleConfig struct {
	WindowManager     terminal.WindowManager
	NavigationManager interfaces.NavigationManager
	InputHandler      terminal.InputHandler
	App               *tview.Application
}

func NewConsole(config ConsoleConfig) *Console {
	console := &Console{
		config: &config,
	}

	// Configure the framework
	console.config.App.SetInputCapture(console.config.InputHandler.HandleKey)
	console.config.App.EnableMouse(true)
	console.config.App.EnablePaste(true)

	// Set the pages as the root of the tview app
	console.config.App.SetRoot(console.config.WindowManager.GetPages(), true)

	return console
}

// ShowWindow activates the specified window.
//
// Parameters:
//   - windowKey: The key identifying the window to show
func (c *Console) ShowWindow(windowKey string) {
	c.config.NavigationManager.NavigateTo(windowKey)
	c.config.WindowManager.SwitchToPage(windowKey)
}

// SetBreakpointConfigMode activates the breakpoint configuration window.
func (c *Console) SetBreakpointConfigMode() {
	c.config.NavigationManager.PushToHistory("breakpoint")
	c.config.WindowManager.SwitchToPage("breakpoint")
}

// ReturnToPreviousWindow returns to the previous window.
func (c *Console) ReturnToPreviousWindow() {
	c.config.NavigationManager.GoBack()
	c.config.WindowManager.SwitchToPage(c.config.NavigationManager.GetCurrent())
}

// ScrollUp scrolls the active memory window up by the specified number of lines.
//
// Parameters:
//   - step: The number of lines to scroll up
func (c *Console) ScrollUp(step uint32) {
	activeKey := c.config.NavigationManager.GetCurrent()

	if window := terminal.GetWindow[ui.MemoryWindow](c.config.WindowManager, activeKey); window != nil {
		window.ScrollUp(step)
	}
}

// ScrollDown scrolls the active memory window down by the specified number of lines.
//
// Parameters:
//   - step: The number of lines to scroll down
func (c *Console) ScrollDown(step uint32) {
	activeKey := c.config.NavigationManager.GetCurrent()

	if window := terminal.GetWindow[ui.MemoryWindow](c.config.WindowManager, activeKey); window != nil {
		window.ScrollDown(step)
	}
}

// RemoveSelectedItem removes the currently selected item from the breakpoint form window.
func (c *Console) RemoveSelectedItem() {
	if window := terminal.GetWindow[ui.BreakPointForm](c.config.WindowManager, "breakpoint"); window != nil {
		window.RemoveSelectedItem()
	}
}

// ShowEmulationSpeed displays the emulation speed configuration window.
func (c *Console) ShowEmulationSpeed() {
	if window := terminal.GetWindow[ui.SpeedWindow](c.config.WindowManager, "speed"); window != nil {
		window.ShowConfig()
	}
}

// Draw clears and draws all windows in the console.
//
// Parameters:
//   - context: The current step context
func (c *Console) Draw(context *common.StepContext) {
	c.config.WindowManager.GetAllWindows(func(key string, window terminal.Window) bool {
		window.Clear()
		window.Draw(context)
		return true // continue iteration
	})

	c.config.App.Draw()
}

// Tick updates the console components that need to be updated every cycle.
//
// Parameters:
//   - context: The current step context
func (c *Console) Tick(context *common.StepContext) {
	c.config.WindowManager.GetTickers(func(key string, ticker terminal.TickerWindow) bool {
		ticker.Tick(context)
		return true // continue iteration
	})
}

// Run starts the console application.
//
// Returns:
//   - An error if the application fails to start
func (c *Console) Run() error {
	return c.config.App.Run()
}

// Stop stops the console application.
func (c *Console) Stop() {
	c.config.App.Stop()
}

// GetWindowManager returns the window manager.
func (c *Console) GetWindowManager() terminal.WindowManager {
	return c.config.WindowManager
}

// GetNavigationManager returns the navigation manager.
func (c *Console) GetNavigationManager() interfaces.NavigationManager {
	return c.config.NavigationManager
}

// GetInputHandler returns the input handler.
func (c *Console) GetInputHandler() terminal.InputHandler {
	return c.config.InputHandler
}

// GetApp returns the tview application.
func (c *Console) GetApp() *tview.Application {
	return c.config.App
}
