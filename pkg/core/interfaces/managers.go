package interfaces

import (
	"github.com/fran150/clementina-6502/pkg/terminal"
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
