package managers

import (
	"github.com/fran150/clementina-6502/internal/slicesext"
	"github.com/fran150/clementina-6502/pkg/core"
)

// navigationManager manages window navigation and history.
type navigationManager struct {
	current string
	history []string
}

// newNavigationManager creates a new navigation manager.
//
// Returns:
//   - A pointer to the initialized DefaultNavigationManager
func newNavigationManager() *navigationManager {
	return &navigationManager{
		current: "",
		history: make([]string, 0, 10), // Pre-allocate with reasonable capacity
	}
}

// NewNavigationManager creates a new navigation manager.
//
// Returns:
//   - A pointer to the initialized DefaultNavigationManager
func NewNavigationManager() core.NavigationManager {
	return newNavigationManager()
}

// NavigateTo switches to the specified window.
//
// Parameters:
//   - key: The key of the window to navigate to
func (nm *navigationManager) NavigateTo(key string) {
	nm.current = key
}

// GoBack returns to the previous window.
// If there is no previous window in the history, this method has no effect.
func (nm *navigationManager) GoBack() {
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
func (nm *navigationManager) GetCurrent() string {
	return nm.current
}

// PushToHistory adds the current window to history and navigates to new window.
//
// Parameters:
//   - key: The key of the window to navigate to
func (nm *navigationManager) PushToHistory(key string) {
	if nm.current != "" {
		nm.history = append(nm.history, nm.current)
	}
	nm.current = key
}
