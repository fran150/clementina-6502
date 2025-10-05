package terminal

import "github.com/rivo/tview"

// DefaultWindowManager manages console windows and their lifecycle.
type DefaultWindowManager struct {
	windows map[string]Window
	tickers map[string]TickerWindow
	pages   *tview.Pages
}

// NewWindowManager creates a new window manager.
//
// Returns:
//   - A pointer to the initialized DefaultWindowManager
func NewWindowManager() *DefaultWindowManager {
	return &DefaultWindowManager{
		windows: make(map[string]Window),
		tickers: make(map[string]TickerWindow),
		pages:   tview.NewPages(),
	}
}

// AddWindow adds a new window to the manager.
//
// Parameters:
//   - key: The unique identifier for the window
//   - window: The window instance to add
func (wm *DefaultWindowManager) AddWindow(key string, window Window) {
	if _, exists := wm.windows[key]; !exists {
		wm.windows[key] = window

		if ticker, ok := window.(TickerWindow); ok {
			wm.tickers[key] = ticker
		}

		wm.pages.AddPage(key, window.GetDrawArea(), true, true)
	}
}

// GetWindow retrieves a window by its key.
//
// Parameters:
//   - key: The unique identifier of the window to retrieve
//
// Returns:
//   - The window instance, or nil if not found
func (wm *DefaultWindowManager) GetWindow(key string) Window {
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
		wm.pages.RemovePage(key)
	}
}

// SwitchToPage changes the currently active page to the window with the
// specified key.
func (wm *DefaultWindowManager) SwitchToPage(key string) {
	wm.pages.SwitchToPage(key)
}

// GetAllWindows returns all windows.
//
// Returns:
//   - A map of all windows keyed by their identifiers
func (wm *DefaultWindowManager) GetAllWindows() map[string]Window {
	// Return a copy to prevent external modification
	result := make(map[string]Window)
	for k, v := range wm.windows {
		result[k] = v
	}
	return result
}

// GetTickers returns all ticker windows.
//
// Returns:
//   - A map of all ticker windows keyed by their identifiers
func (wm *DefaultWindowManager) GetTickers() map[string]TickerWindow {
	// Return a copy to prevent external modification
	result := make(map[string]TickerWindow)
	for k, v := range wm.tickers {
		result[k] = v
	}
	return result
}

func (wm *DefaultWindowManager) GetPages() *tview.Pages {
	return wm.pages
}

// GetWindow is a generic function that retrieves and type-casts a window from the window manager.
//
// Type Parameters:
//   - T: The expected window type to cast to
//
// Parameters:
//   - wm: The WindowManager instance to search in
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
