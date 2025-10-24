package terminal

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/core"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/rivo/tview"
)

// defaultEmulatorConsole provides a base implementation for terminal-based emulator consoles.
// It manages the terminal UI components including windows, navigation, and input handling.
type defaultEmulatorConsole struct {
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

func newEmulatorConsole(config EmulatorConsoleConfig) *defaultEmulatorConsole {
	console := &defaultEmulatorConsole{
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

// NewEmulatorConsole creates a new BaseTerminalEmulatorConsole instance with the provided configuration.
// It sets up the terminal UI framework by configuring input handling, mouse support, and setting the root component.
//
// Parameters:
//   - config: Configuration containing all necessary components for terminal UI operations
//
// Returns:
//   - A pointer to the newly created BaseTerminalEmulatorConsole instance
func NewEmulatorConsole(config EmulatorConsoleConfig) EmulatorConsole {
	return newEmulatorConsole(config)
}

/*********************************************************************************************************
* Window manipulation and UI related functions
*********************************************************************************************************/

// ShowWindow navigates to and displays the specified window.
//
// Parameters:
//   - windowKey: The key identifier of the window to show
func (c *defaultEmulatorConsole) ShowWindow(windowKey string) {
	c.config.NavigationManager.NavigateTo(windowKey)
	c.config.WindowManager.SwitchToPage(windowKey)
}

// ReturnToPreviousWindow navigates back to the previous window in the navigation history.
// It uses the navigation manager to go back one step and then switches the window manager
// to display the current window from the navigation history.
func (c *defaultEmulatorConsole) ReturnToPreviousWindow() {
	c.config.NavigationManager.GoBack()
	c.config.WindowManager.SwitchToPage(c.config.NavigationManager.GetCurrent())
}

// SwitchToBreakpointConfigMode switches the console to breakpoint configuration mode.
// It pushes "breakpoint" to the navigation history and switches to the breakpoint window.
func (c *defaultEmulatorConsole) SwitchToBreakpointConfigMode() {
	c.config.NavigationManager.PushToHistory("breakpoint")
	c.config.WindowManager.SwitchToPage("breakpoint")
}

// RemoveSelectedBreakpointAddress removes the currently selected breakpoint address from the breakpoint configuration window.
// It retrieves the breakpoint window and calls its RemoveSelectedItem method to remove the selected breakpoint.
func (c *defaultEmulatorConsole) RemoveSelectedBreakpointAddress() {
	if window := GetWindow[ui.BreakPointForm](c.config.WindowManager, "breakpoint"); window != nil {
		window.RemoveSelectedItem()
	}
}

// ShowEmulationSpeedPopup displays the emulation speed configuration popup window.
// It retrieves the speed window and calls its ShowConfig method to display the speed configuration interface.
func (c *defaultEmulatorConsole) ShowEmulationSpeedPopup() {
	if window := GetWindow[ui.SpeedWindow](c.config.WindowManager, "speed"); window != nil {
		window.ShowConfig()
	}
}

// ScrollActiveWindowUp scrolls the active memory window up by the specified number of lines.
//
// Parameters:
//   - step: The number of lines to scroll up
func (c *defaultEmulatorConsole) ScrollActiveWindowUp(step uint32) {
	activeKey := c.config.NavigationManager.GetCurrent()

	if window := GetWindow[ui.MemoryWindow](c.config.WindowManager, activeKey); window != nil {
		window.ScrollUp(step)
	}
}

// ScrollActiveWindowDown scrolls the active memory window down by the specified number of lines.
//
// Parameters:
//   - step: The number of lines to scroll down
func (c *defaultEmulatorConsole) ScrollActiveWindowDown(step uint32) {
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
func (c *defaultEmulatorConsole) Tick(context *common.StepContext) {
	c.config.WindowManager.GetTickerWindows(func(key string, ticker TickerWindow) bool {
		ticker.Tick(context)
		return true // continue iteration
	})
}

// Draw renders all windows in the console.
//
// Parameters:
//   - context: The current step context containing state information for rendering

func (c *defaultEmulatorConsole) Draw(context *common.StepContext) {
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
func (c *defaultEmulatorConsole) Run() error {
	return c.config.App.Run()
}

// Stop stops the console application.
func (c *defaultEmulatorConsole) Stop() {
	c.config.App.Stop()
}
