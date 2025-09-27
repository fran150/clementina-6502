package computers

import (
	"github.com/fran150/clementina-6502/pkg/terminal"
	"github.com/gdamore/tcell/v2"
)

// WindowManager defines the interface for managing console windows.
type WindowManager interface {
	// AddWindow adds a new window to the manager.
	AddWindow(key string, window terminal.Window)

	// GetWindow retrieves a window by its key.
	GetWindow(key string) terminal.Window

	// RemoveWindow removes a window by its key.
	RemoveWindow(key string)

	// GetAllWindows returns all windows.
	GetAllWindows() map[string]terminal.Window

	// GetTickers returns all ticker windows.
	GetTickers() map[string]terminal.TickerWindow
}

// NavigationManager defines the interface for managing window navigation.
type NavigationManager interface {
	// NavigateTo switches to the specified window.
	NavigateTo(key string)

	// GoBack returns to the previous window.
	GoBack()

	// GetCurrent returns the currently active window key.
	GetCurrent() string

	// PushToHistory adds the current window to history and navigates to new window.
	PushToHistory(key string)
}

// InputHandler defines the interface for handling user input.
type InputHandler interface {
	// HandleKey processes a key event and returns the modified event or nil.
	HandleKey(event *tcell.EventKey) *tcell.EventKey
}

// UIFramework defines the interface for UI framework abstraction.
type UIFramework interface {
	// Run starts the UI application.
	Run() error

	// Stop stops the UI application.
	Stop()

	// Draw refreshes the display.
	Draw()

	// SetInputCapture sets the global input handler.
	SetInputCapture(handler func(*tcell.EventKey) *tcell.EventKey)

	// EnableMouse enables mouse support.
	EnableMouse(enable bool)

	// EnablePaste enables paste support.
	EnablePaste(enable bool)
}

// ConsoleActions defines the interface for console operations.
type ConsoleActions interface {
	// ScrollUp scrolls the active memory window up.
	ScrollUp(step uint32)

	// ScrollDown scrolls the active memory window down.
	ScrollDown(step uint32)

	// ShowEmulationSpeed displays the emulation speed configuration.
	ShowEmulationSpeed()

	// SetBreakpointConfigMode activates the breakpoint configuration window.
	SetBreakpointConfigMode()

	// ShowWindow activates the specified window.
	ShowWindow(windowKey string)

	// ReturnToPreviousWindow returns to the previous window.
	ReturnToPreviousWindow()
}
