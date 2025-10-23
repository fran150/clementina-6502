package computers

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/core"
	"github.com/fran150/clementina-6502/pkg/terminal"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/rivo/tview"
)

// BaseTerminalEmulatorConsole provides a base implementation for terminal-based emulator consoles.
// It manages the terminal UI components including windows, navigation, and input handling.
type BaseTerminalEmulatorConsole struct {
	config *BaseTerminalEmulatorConsoleConfig
}

// BaseTerminalEmulatorConsoleConfig holds the configuration for a BaseTerminalEmulatorConsole.
// It contains all the necessary components for managing terminal UI operations.
type BaseTerminalEmulatorConsoleConfig struct {
	WindowManager     terminal.WindowManager
	NavigationManager core.NavigationManager
	InputHandler      terminal.InputHandler
	App               *tview.Application
}

/*********************************************************************************************************
* Constructor
**********************************************************************************************************/

// NewBaseTerminalEmulatorConsole creates a new BaseTerminalEmulatorConsole instance with the provided configuration.
// It sets up the terminal UI framework by configuring input handling, mouse support, and setting the root component.
//
// Parameters:
//   - config: Configuration containing all necessary components for terminal UI operations
//
// Returns:
//   - A pointer to the newly created BaseTerminalEmulatorConsole instance
func NewBaseTerminalEmulatorConsole(config BaseTerminalEmulatorConsoleConfig) *BaseTerminalEmulatorConsole {
	console := &BaseTerminalEmulatorConsole{
		config: &config,
	}

	// Configure the framework
	app := console.config.App
	app.SetInputCapture(console.config.InputHandler.HandleKey)
	app.EnableMouse(true)
	app.EnablePaste(true)

	// Set the pages as the root of the tview app
	app.SetRoot(console.config.WindowManager.GetPages(), true)

	return console
}

/*********************************************************************************************************
* Window manipulation and UI related functions
*********************************************************************************************************/

// ShowWindow navigates to and displays the specified window.
//
// Parameters:
//   - windowKey: The key identifier of the window to show
func (c *BaseTerminalEmulatorConsole) ShowWindow(windowKey string) {
	c.config.NavigationManager.NavigateTo(windowKey)
	c.config.WindowManager.SwitchToPage(windowKey)
}

// ReturnToPreviousWindow navigates back to the previous window in the navigation history.
// It uses the navigation manager to go back one step and then switches the window manager
// to display the current window from the navigation history.
func (c *BaseTerminalEmulatorConsole) ReturnToPreviousWindow() {
	c.config.NavigationManager.GoBack()
	c.config.WindowManager.SwitchToPage(c.config.NavigationManager.GetCurrent())
}

// SwitchToBreakpointConfigMode switches the console to breakpoint configuration mode.
// It pushes "breakpoint" to the navigation history and switches to the breakpoint window.
func (c *BaseTerminalEmulatorConsole) SwitchToBreakpointConfigMode() {
	c.config.NavigationManager.PushToHistory("breakpoint")
	c.config.WindowManager.SwitchToPage("breakpoint")
}

// RemoveSelectedBreakpointAddress removes the currently selected breakpoint address from the breakpoint configuration window.
// It retrieves the breakpoint window and calls its RemoveSelectedItem method to remove the selected breakpoint.
func (c *BaseTerminalEmulatorConsole) RemoveSelectedBreakpointAddress() {
	if window := terminal.GetWindow[ui.BreakPointForm](c.config.WindowManager, "breakpoint"); window != nil {
		window.RemoveSelectedItem()
	}
}

// ShowEmulationSpeedPopup displays the emulation speed configuration popup window.
// It retrieves the speed window and calls its ShowConfig method to display the speed configuration interface.
func (c *BaseTerminalEmulatorConsole) ShowEmulationSpeedPopup() {
	if window := terminal.GetWindow[ui.SpeedWindow](c.config.WindowManager, "speed"); window != nil {
		window.ShowConfig()
	}
}

// ScrollMemoryWindowUp scrolls the active memory window up by the specified number of lines.
//
// Parameters:
//   - step: The number of lines to scroll up
func (c *BaseTerminalEmulatorConsole) ScrollMemoryWindowUp(step uint32) {
	activeKey := c.config.NavigationManager.GetCurrent()

	if window := terminal.GetWindow[ui.MemoryWindow](c.config.WindowManager, activeKey); window != nil {
		window.ScrollUp(step)
	}
}

// ScrollMemoryWindowDown scrolls the active memory window down by the specified number of lines.
//
// Parameters:
//   - step: The number of lines to scroll down
func (c *BaseTerminalEmulatorConsole) ScrollMemoryWindowDown(step uint32) {
	activeKey := c.config.NavigationManager.GetCurrent()

	if window := terminal.GetWindow[ui.MemoryWindow](c.config.WindowManager, activeKey); window != nil {
		window.ScrollDown(step)
	}
}

/*********************************************************************************************************
* Loop Methods
**********************************************************************************************************/

// Tick updates the console components that need to be updated every cycle.
//
// Parameters:
//   - context: The current step context
func (c *BaseTerminalEmulatorConsole) Tick(context *common.StepContext) {
	c.config.WindowManager.GetTickers(func(key string, ticker terminal.TickerWindow) bool {
		ticker.Tick(context)
		return true // continue iteration
	})
}

// Draw renders all windows in the console.
//
// Parameters:
//   - context: The current step context containing state information for rendering

func (c *BaseTerminalEmulatorConsole) Draw(context *common.StepContext) {
	c.config.WindowManager.GetAllWindows(func(key string, window terminal.Window) bool {
		window.Clear()
		window.Draw(context)
		return true // continue iteration
	})

	c.config.App.Draw()
}

/*********************************************************************************************************
* State Management
**********************************************************************************************************/

// Run starts the console application.
//
// Returns:
//   - An error if the application fails to start
func (c *BaseTerminalEmulatorConsole) Run() error {
	return c.config.App.Run()
}

// Stop stops the console application.
func (c *BaseTerminalEmulatorConsole) Stop() {
	c.config.App.Stop()
}

/*********************************************************************************************************
* Getters
**********************************************************************************************************/

// GetWindowManager returns the window manager.
func (c *BaseTerminalEmulatorConsole) GetWindowManager() terminal.WindowManager {
	return c.config.WindowManager
}

// GetNavigationManager returns the navigation manager.
func (c *BaseTerminalEmulatorConsole) GetNavigationManager() core.NavigationManager {
	return c.config.NavigationManager
}

// GetInputHandler returns the input handler.
func (c *BaseTerminalEmulatorConsole) GetInputHandler() terminal.InputHandler {
	return c.config.InputHandler
}

// GetApp returns the tview application.
func (c *BaseTerminalEmulatorConsole) GetApp() *tview.Application {
	return c.config.App
}
