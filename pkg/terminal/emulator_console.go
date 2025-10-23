package terminal

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/core"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/rivo/tview"
)

// EmulatorConsole provides a base implementation for terminal-based emulator consoles.
// It manages the terminal UI components including windows, navigation, and input handling.
type EmulatorConsole struct {
	config *EmulatorConsoleConfig
}

// EmulatorConsoleConfig holds the configuration for a BaseTerminalEmulatorConsole.
// It contains all the necessary components for managing terminal UI operations.
type EmulatorConsoleConfig struct {
	WindowManager     WindowManager
	NavigationManager core.NavigationManager
	InputHandler      InputHandler
	App               *tview.Application
}

/*********************************************************************************************************
* Constructor
**********************************************************************************************************/

// NewEmulatorConsole creates a new BaseTerminalEmulatorConsole instance with the provided configuration.
// It sets up the terminal UI framework by configuring input handling, mouse support, and setting the root component.
//
// Parameters:
//   - config: Configuration containing all necessary components for terminal UI operations
//
// Returns:
//   - A pointer to the newly created BaseTerminalEmulatorConsole instance
func NewEmulatorConsole(config EmulatorConsoleConfig) *EmulatorConsole {
	console := &EmulatorConsole{
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
func (c *EmulatorConsole) ShowWindow(windowKey string) {
	c.config.NavigationManager.NavigateTo(windowKey)
	c.config.WindowManager.SwitchToPage(windowKey)
}

// ReturnToPreviousWindow navigates back to the previous window in the navigation history.
// It uses the navigation manager to go back one step and then switches the window manager
// to display the current window from the navigation history.
func (c *EmulatorConsole) ReturnToPreviousWindow() {
	c.config.NavigationManager.GoBack()
	c.config.WindowManager.SwitchToPage(c.config.NavigationManager.GetCurrent())
}

// SwitchToBreakpointConfigMode switches the console to breakpoint configuration mode.
// It pushes "breakpoint" to the navigation history and switches to the breakpoint window.
func (c *EmulatorConsole) SwitchToBreakpointConfigMode() {
	c.config.NavigationManager.PushToHistory("breakpoint")
	c.config.WindowManager.SwitchToPage("breakpoint")
}

// RemoveSelectedBreakpointAddress removes the currently selected breakpoint address from the breakpoint configuration window.
// It retrieves the breakpoint window and calls its RemoveSelectedItem method to remove the selected breakpoint.
func (c *EmulatorConsole) RemoveSelectedBreakpointAddress() {
	if window := GetWindow[ui.BreakPointForm](c.config.WindowManager, "breakpoint"); window != nil {
		window.RemoveSelectedItem()
	}
}

// ShowEmulationSpeedPopup displays the emulation speed configuration popup window.
// It retrieves the speed window and calls its ShowConfig method to display the speed configuration interface.
func (c *EmulatorConsole) ShowEmulationSpeedPopup() {
	if window := GetWindow[ui.SpeedWindow](c.config.WindowManager, "speed"); window != nil {
		window.ShowConfig()
	}
}

// ScrollMemoryWindowUp scrolls the active memory window up by the specified number of lines.
//
// Parameters:
//   - step: The number of lines to scroll up
func (c *EmulatorConsole) ScrollMemoryWindowUp(step uint32) {
	activeKey := c.config.NavigationManager.GetCurrent()

	if window := GetWindow[ui.MemoryWindow](c.config.WindowManager, activeKey); window != nil {
		window.ScrollUp(step)
	}
}

// ScrollMemoryWindowDown scrolls the active memory window down by the specified number of lines.
//
// Parameters:
//   - step: The number of lines to scroll down
func (c *EmulatorConsole) ScrollMemoryWindowDown(step uint32) {
	activeKey := c.config.NavigationManager.GetCurrent()

	if window := GetWindow[ui.MemoryWindow](c.config.WindowManager, activeKey); window != nil {
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
func (c *EmulatorConsole) Tick(context *common.StepContext) {
	c.config.WindowManager.GetTickers(func(key string, ticker TickerWindow) bool {
		ticker.Tick(context)
		return true // continue iteration
	})
}

// Draw renders all windows in the console.
//
// Parameters:
//   - context: The current step context containing state information for rendering

func (c *EmulatorConsole) Draw(context *common.StepContext) {
	c.config.WindowManager.GetAllWindows(func(key string, window Window) bool {
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
func (c *EmulatorConsole) Run() error {
	return c.config.App.Run()
}

// Stop stops the console application.
func (c *EmulatorConsole) Stop() {
	c.config.App.Stop()
}

/*********************************************************************************************************
* Getters
**********************************************************************************************************/

// GetWindowManager returns the window manager.
func (c *EmulatorConsole) GetWindowManager() WindowManager {
	return c.config.WindowManager
}

// GetNavigationManager returns the navigation manager.
func (c *EmulatorConsole) GetNavigationManager() core.NavigationManager {
	return c.config.NavigationManager
}

// GetInputHandler returns the input handler.
func (c *EmulatorConsole) GetInputHandler() InputHandler {
	return c.config.InputHandler
}

// GetApp returns the tview application.
func (c *EmulatorConsole) GetApp() *tview.Application {
	return c.config.App
}
