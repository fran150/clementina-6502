package terminal

import (
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/gdamore/tcell/v2"
)

// DefaultInputHandler provides default input handling for the console.
type DefaultInputHandler struct {
	windowManager WindowManager
}

func NewDefaultInputHandler(windowManager WindowManager) *DefaultInputHandler {
	return &DefaultInputHandler{
		windowManager: windowManager,
	}
}

// HandleKey processes a key event and returns the modified event or nil.
//
// Parameters:
//   - event: The key event to process
//
// Returns:
//   - The processed key event or nil
func (dih *DefaultInputHandler) HandleKey(event *tcell.EventKey) *tcell.EventKey {
	// Delegate to the options window if it exists
	// This maintains compatibility with the original implementation

	if window := GetWindow[ui.OptionsWindow](dih.windowManager, "options"); window != nil {
		window.ProcessKey(event)
	}

	return event
}
