package computers

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/terminal"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
)

// Console provides the core console functionality without UI framework dependencies.
type Console struct {
	windowManager     WindowManager
	navigationManager NavigationManager
	windowRegistry    *WindowRegistry

	// Type-safe window controllers
	memoryControllers     map[string]*MemoryWindowController
	speedControllers      map[string]*SpeedWindowController
	optionsControllers    map[string]*OptionsWindowController
	breakpointControllers map[string]*BreakpointWindowController
}

// NewConsole creates a new console with the specified managers.
//
// Parameters:
//   - windowManager: The window manager to use
//   - navigationManager: The navigation manager to use
//
// Returns:
//   - A pointer to the initialized Console
func NewConsole(windowManager WindowManager, navigationManager NavigationManager) *Console {
	return &Console{
		windowManager:         windowManager,
		navigationManager:     navigationManager,
		windowRegistry:        NewWindowRegistry(windowManager),
		memoryControllers:     make(map[string]*MemoryWindowController),
		speedControllers:      make(map[string]*SpeedWindowController),
		optionsControllers:    make(map[string]*OptionsWindowController),
		breakpointControllers: make(map[string]*BreakpointWindowController),
	}
}

// AddWindow adds a new window to the console.
//
// Parameters:
//   - key: The unique identifier for the window
//   - window: The window instance to add
func (c *Console) AddWindow(key string, window terminal.Window) {
	c.windowManager.AddWindow(key, window)

	// Create type-safe controllers for specific window types
	if memWindow, ok := window.(*ui.MemoryWindow); ok {
		c.memoryControllers[key] = NewMemoryWindowController(memWindow)
	}
	if speedWindow, ok := window.(*ui.SpeedWindow); ok {
		c.speedControllers[key] = NewSpeedWindowController(speedWindow)
	}
	if optionsWindow, ok := window.(*ui.OptionsWindow); ok {
		c.optionsControllers[key] = NewOptionsWindowController(optionsWindow)
	}
	if breakpointWindow, ok := window.(*ui.BreakPointForm); ok {
		c.breakpointControllers[key] = NewBreakpointWindowController(breakpointWindow)
	}
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
	delete(c.memoryControllers, key)
	delete(c.speedControllers, key)
	delete(c.optionsControllers, key)
	delete(c.breakpointControllers, key)
}

// ShowWindow activates the specified window.
//
// Parameters:
//   - windowKey: The key identifying the window to show
func (c *Console) ShowWindow(windowKey string) {
	c.navigationManager.NavigateTo(windowKey)
}

// SetBreakpointConfigMode activates the breakpoint configuration window.
func (c *Console) SetBreakpointConfigMode() {
	c.navigationManager.PushToHistory("breakpoint")
}

// ReturnToPreviousWindow returns to the previous window.
func (c *Console) ReturnToPreviousWindow() {
	c.navigationManager.GoBack()
}

// ScrollUp scrolls the active memory window up by the specified number of lines.
//
// Parameters:
//   - step: The number of lines to scroll up
func (c *Console) ScrollUp(step uint32) {
	activeKey := c.navigationManager.GetCurrent()
	if controller, exists := c.memoryControllers[activeKey]; exists {
		controller.ScrollUp(step)
	}
}

// ScrollDown scrolls the active memory window down by the specified number of lines.
//
// Parameters:
//   - step: The number of lines to scroll down
func (c *Console) ScrollDown(step uint32) {
	activeKey := c.navigationManager.GetCurrent()
	if controller, exists := c.memoryControllers[activeKey]; exists {
		controller.ScrollDown(step)
	}
}

// ShowEmulationSpeed displays the emulation speed configuration window.
func (c *Console) ShowEmulationSpeed() {
	if controller, exists := c.speedControllers["speed"]; exists {
		controller.ShowConfig()
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

// GetMemoryController returns the memory window controller for the specified key.
//
// Parameters:
//   - key: The window key
//
// Returns:
//   - The memory window controller, or nil if not found
func (c *Console) GetMemoryController(key string) *MemoryWindowController {
	return c.memoryControllers[key]
}

// GetSpeedController returns the speed window controller for the specified key.
//
// Parameters:
//   - key: The window key
//
// Returns:
//   - The speed window controller, or nil if not found
func (c *Console) GetSpeedController(key string) *SpeedWindowController {
	return c.speedControllers[key]
}

// GetOptionsController returns the options window controller for the specified key.
//
// Parameters:
//   - key: The window key
//
// Returns:
//   - The options window controller, or nil if not found
func (c *Console) GetOptionsController(key string) *OptionsWindowController {
	return c.optionsControllers[key]
}

// GetBreakpointController returns the breakpoint window controller for the specified key.
//
// Parameters:
//   - key: The window key
//
// Returns:
//   - The breakpoint window controller, or nil if not found
func (c *Console) GetBreakpointController(key string) *BreakpointWindowController {
	return c.breakpointControllers[key]
}
