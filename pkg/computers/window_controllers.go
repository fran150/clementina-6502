package computers

import (
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
)

// WindowRegistry provides type-safe access to specific window types.
type WindowRegistry struct {
	windowManager WindowManager
}

// NewWindowRegistry creates a new window registry.
//
// Parameters:
//   - windowManager: The window manager to use for window access
//
// Returns:
//   - A pointer to the initialized WindowRegistry
func NewWindowRegistry(windowManager WindowManager) *WindowRegistry {
	return &WindowRegistry{
		windowManager: windowManager,
	}
}

// MemoryWindowController provides type-safe operations for memory windows.
type MemoryWindowController struct {
	window *ui.MemoryWindow
}

// NewMemoryWindowController creates a new memory window controller.
//
// Parameters:
//   - window: The memory window to control
//
// Returns:
//   - A pointer to the initialized MemoryWindowController
func NewMemoryWindowController(window *ui.MemoryWindow) *MemoryWindowController {
	return &MemoryWindowController{
		window: window,
	}
}

// ScrollUp scrolls the memory window up by the specified number of lines.
//
// Parameters:
//   - step: The number of lines to scroll up
func (mwc *MemoryWindowController) ScrollUp(step uint32) {
	mwc.window.ScrollUp(step)
}

// ScrollDown scrolls the memory window down by the specified number of lines.
//
// Parameters:
//   - step: The number of lines to scroll down
func (mwc *MemoryWindowController) ScrollDown(step uint32) {
	mwc.window.ScrollDown(step)
}

// GetWindow returns the underlying memory window.
//
// Returns:
//   - The ui.MemoryWindow instance
func (mwc *MemoryWindowController) GetWindow() *ui.MemoryWindow {
	return mwc.window
}

// SpeedWindowController provides type-safe operations for speed windows.
type SpeedWindowController struct {
	window *ui.SpeedWindow
}

// NewSpeedWindowController creates a new speed window controller.
//
// Parameters:
//   - window: The speed window to control
//
// Returns:
//   - A pointer to the initialized SpeedWindowController
func NewSpeedWindowController(window *ui.SpeedWindow) *SpeedWindowController {
	return &SpeedWindowController{
		window: window,
	}
}

// ShowConfig displays the speed configuration.
func (swc *SpeedWindowController) ShowConfig() {
	swc.window.ShowConfig()
}

// OptionsWindowController provides type-safe operations for options windows.
type OptionsWindowController struct {
	window *ui.OptionsWindow
}

// NewOptionsWindowController creates a new options window controller.
//
// Parameters:
//   - window: The options window to control
//
// Returns:
//   - A pointer to the initialized OptionsWindowController
func NewOptionsWindowController(window *ui.OptionsWindow) *OptionsWindowController {
	return &OptionsWindowController{
		window: window,
	}
}

// GetWindow returns the underlying options window.
//
// Returns:
//   - The ui.OptionsWindow instance
func (owc *OptionsWindowController) GetWindow() *ui.OptionsWindow {
	return owc.window
}

// BreakpointWindowController provides type-safe operations for breakpoint windows.
type BreakpointWindowController struct {
	window *ui.BreakPointForm
}

// NewBreakpointWindowController creates a new breakpoint window controller.
//
// Parameters:
//   - window: The breakpoint window to control
//
// Returns:
//   - A pointer to the initialized BreakpointWindowController
func NewBreakpointWindowController(window *ui.BreakPointForm) *BreakpointWindowController {
	return &BreakpointWindowController{
		window: window,
	}
}

// CheckBreakpoint checks if a breakpoint exists at the specified address.
//
// Parameters:
//   - address: The address to check
//
// Returns:
//   - true if a breakpoint exists at the address, false otherwise
func (bwc *BreakpointWindowController) CheckBreakpoint(address uint16) bool {
	return bwc.window.CheckBreakpoint(address)
}

// GetWindow returns the underlying breakpoint window.
//
// Returns:
//   - The ui.BreakPointForm instance
func (bwc *BreakpointWindowController) GetWindow() *ui.BreakPointForm {
	return bwc.window
}
