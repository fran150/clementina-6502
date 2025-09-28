package computers

import (
	"github.com/fran150/clementina-6502/pkg/terminal"
)

// DefaultWindowManager manages console windows and their lifecycle.
type DefaultWindowManager struct {
	windows map[string]terminal.Window
	tickers map[string]terminal.TickerWindow
}

// NewWindowManager creates a new window manager.
//
// Returns:
//   - A pointer to the initialized DefaultWindowManager
func NewWindowManager() *DefaultWindowManager {
	return &DefaultWindowManager{
		windows: make(map[string]terminal.Window),
		tickers: make(map[string]terminal.TickerWindow),
	}
}

// AddWindow adds a new window to the manager.
//
// Parameters:
//   - key: The unique identifier for the window
//   - window: The window instance to add
func (wm *DefaultWindowManager) AddWindow(key string, window terminal.Window) {
	if _, exists := wm.windows[key]; !exists {
		wm.windows[key] = window

		if ticker, ok := window.(terminal.TickerWindow); ok {
			wm.tickers[key] = ticker
		}
	}
}

// GetWindow retrieves a window by its key.
//
// Parameters:
//   - key: The unique identifier of the window to retrieve
//
// Returns:
//   - The window instance, or nil if not found
func (wm *DefaultWindowManager) GetWindow(key string) terminal.Window {
	if window, exists := wm.windows[key]; exists {
		return window
	}
	return nil
}

// RemoveWindow removes a window by its key.
//
// Parameters:
//   - key: The unique identifier of the window to remove
func (wm *DefaultWindowManager) RemoveWindow(key string) {
	if _, exists := wm.windows[key]; exists {
		delete(wm.windows, key)
		delete(wm.tickers, key)
	}
}

// GetAllWindows returns all windows.
//
// Returns:
//   - A map of all windows keyed by their identifiers
func (wm *DefaultWindowManager) GetAllWindows() map[string]terminal.Window {
	// Return a copy to prevent external modification
	result := make(map[string]terminal.Window)
	for k, v := range wm.windows {
		result[k] = v
	}
	return result
}

// GetTickers returns all ticker windows.
//
// Returns:
//   - A map of all ticker windows keyed by their identifiers
func (wm *DefaultWindowManager) GetTickers() map[string]terminal.TickerWindow {
	// Return a copy to prevent external modification
	result := make(map[string]terminal.TickerWindow)
	for k, v := range wm.tickers {
		result[k] = v
	}
	return result
}

// GetWindow is a generic function that retrieves and type-casts a window from the console's window map.
//
// Parameters:
//   - c: The BaseConsole instance to search in
//   - key: The unique identifier of the window to retrieve
//
// Returns:
//   - A pointer to the typed window, or nil if not found or type mismatch
func GetWindow[T any](wm WindowManager, key string) *T {
	if window := wm.GetWindow(key); window != nil {
		if typed, ok := any(window).(*T); ok {
			return typed
		}
	}

	return nil
}
