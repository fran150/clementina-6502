package terminal

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Window defines the interface for UI components that can be drawn in the terminal.
// It provides methods for clearing, drawing, and retrieving the drawable area.
type Window interface {
	// Clear resets the window content.
	Clear()

	// Draw updates the window display with current state.
	//
	// Parameters:
	//   - context: The current step context for the emulation cycle
	Draw(context *common.StepContext)

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

	// Tick updates the window state for each emulation cycle.
	//
	// Parameters:
	//   - context: The current step context for the emulation cycle
	Tick(context *common.StepContext)
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

	// GetTickers iterates over all ticker windows for read-only access.
	// The callback function receives each key-value pair and should return true to continue iteration.
	GetTickers(fn func(key string, ticker TickerWindow) bool)

	// SwitchToPage makes the specified window active
	SwitchToPage(key string)

	GetPages() *tview.Pages
}

// InputHandler defines the interface for handling user input.
type InputHandler interface {
	// HandleKey processes a key event and returns the modified event or nil.
	HandleKey(event *tcell.EventKey) *tcell.EventKey
}
