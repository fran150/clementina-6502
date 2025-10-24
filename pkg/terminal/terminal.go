package terminal

import (
	"github.com/fran150/clementina-6502/pkg/core"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Window defines the interface for UI components that can be drawn in the terminal.
// It provides methods for clearing, drawing, and retrieving the drawable area.
type Window interface {
	core.Renderer

	// Clear resets the window content.
	Clear()

	// GetDrawArea returns the primitive that represents this window in the UI.
	//
	// Returns:
	//   - The tview primitive for this window
	GetDrawArea() tview.Primitive
}

// TickerWindow extends the Window interface with tick functionality.
// It represents windows that need to be updated every emulation cycle.
type TickerWindow interface {
	Window
	core.Ticker
}

// WindowManager defines the interface for managing console windows.
type WindowManager interface {
	// AddWindow adds a new window to the manager.
	AddWindow(key string, window Window)

	// GetWindow retrieves a window by its key.
	GetWindow(key string) Window

	// RemoveWindow removes a window by its key.
	RemoveWindow(key string)

	// GetAllWindows iterates over all windows for read-only access.
	// The callback function receives each key-value pair and should return true to continue iteration.
	GetAllWindows(fn func(key string, window Window) bool)

	// GetTickerWindows iterates over all ticker windows for read-only access.
	// The callback function receives each key-value pair and should return true to continue iteration.
	GetTickerWindows(fn func(key string, ticker TickerWindow) bool)

	// SwitchToPage makes the specified window active
	SwitchToPage(key string)

	// GetPages returns the tview.Pages container that manages all windows.
	//
	// Returns:
	//   - A pointer to the tview.Pages container
	GetPages() *tview.Pages
}

// InputHandler defines the interface for handling user input.
type InputHandler interface {
	// HandleKey processes a key event and returns the modified event or nil.
	HandleKey(event *tcell.EventKey) *tcell.EventKey
}

// TerminalApplication defines the interface for terminal application lifecycle management.
// It provides methods for starting and stopping the terminal application.
type TerminalApplication interface {
	// Run starts the terminal application and blocks until it exits.
	//
	// Returns:
	//   - An error if the application fails to start or encounters a runtime error
	Run() error

	// Stop gracefully shuts down the terminal application.
	Stop()
}

// ScrollableActiveWindow defines the interface for scrolling operations on the active window.
// It provides methods to scroll the currently active window up or down by a specified step.
type ScrollableActiveWindow interface {
	// ScrollActiveWindowUp scrolls the active window upward by the specified number of steps.
	//
	// Parameters:
	//   - step: The number of steps to scroll up
	ScrollActiveWindowUp(step uint32)

	// ScrollActiveWindowDown scrolls the active window downward by the specified number of steps.
	//
	// Parameters:
	//   - step: The number of steps to scroll down
	ScrollActiveWindowDown(step uint32)
}

// BreakpointConfigurator defines the interface for managing breakpoint configuration.
// It provides methods for entering breakpoint configuration mode and removing breakpoints.
type BreakpointConfigurator interface {
	// SwitchToBreakpointConfigMode enters the breakpoint configuration mode,
	// allowing the user to add, remove, or modify breakpoints.
	SwitchToBreakpointConfigMode()

	// RemoveSelectedBreakpointAddress removes the currently selected breakpoint address.
	RemoveSelectedBreakpointAddress()
}

// WindowManipulator defines the interface for window navigation and management.
// It provides methods for showing specific windows and navigating between them.
type WindowManipulator interface {
	// ShowWindow displays the window identified by the specified key.
	//
	// Parameters:
	//   - windowKey: The unique identifier for the window to display
	ShowWindow(windowKey string)

	// ReturnToPreviousWindow navigates back to the previously active window.
	ReturnToPreviousWindow()
}

// SpeedConfigurator defines the interface for emulation speed configuration.
// It provides methods for displaying and configuring emulation speed settings.
type SpeedConfigurator interface {
	// ShowEmulationSpeedPopup displays a popup dialog for configuring emulation speed.
	ShowEmulationSpeedPopup()
}

// EmulatorConsole defines the interface for terminal-based emulator consoles.
// It provides methods for window management, UI operations, and application lifecycle control.
type EmulatorConsole interface {
	core.Ticker
	core.Renderer
	TerminalApplication
	ScrollableActiveWindow
	BreakpointConfigurator
	WindowManipulator
	SpeedConfigurator
}
