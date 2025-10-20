package managers

import (
	"github.com/fran150/clementina-6502/internal/slicesext"
)

// DefaultNavigationManager manages window navigation and history.
type DefaultNavigationManager struct {
	current string
	history []string
}

// NewDefaultNavigationManager creates a new navigation manager.
//
// Returns:
//   - A pointer to the initialized DefaultNavigationManager
func NewDefaultNavigationManager() *DefaultNavigationManager {
	return &DefaultNavigationManager{
		current: "",
		history: make([]string, 0, 10), // Pre-allocate with reasonable capacity
	}
}

// NavigateTo switches to the specified window.
//
// Parameters:
//   - key: The key of the window to navigate to
func (nm *DefaultNavigationManager) NavigateTo(key string) {
	nm.current = key
}

// GoBack returns to the previous window.
// If there is no previous window in the history, this method has no effect.
func (nm *DefaultNavigationManager) GoBack() {
	if len(nm.history) > 0 {
		previous, current := slicesext.SlicePop(nm.history)
		nm.history = previous
		nm.current = current
	}
}

// GetCurrent returns the currently active window key.
//
// Returns:
//   - The key of the currently active window
func (nm *DefaultNavigationManager) GetCurrent() string {
	return nm.current
}

// PushToHistory adds the current window to history and navigates to new window.
//
// Parameters:
//   - key: The key of the window to navigate to
func (nm *DefaultNavigationManager) PushToHistory(key string) {
	if nm.current != "" {
		nm.history = append(nm.history, nm.current)
	}
	nm.current = key
}
