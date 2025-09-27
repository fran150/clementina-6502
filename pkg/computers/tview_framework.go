package computers

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TViewFramework provides a tview-based implementation of the UIFramework interface.
type TViewFramework struct {
	app *tview.Application
}

// NewTViewFramework creates a new tview framework wrapper.
//
// Returns:
//   - A pointer to the initialized TViewFramework
func NewTViewFramework() *TViewFramework {
	return &TViewFramework{
		app: tview.NewApplication(),
	}
}

// Run starts the UI application.
//
// Returns:
//   - An error if the application fails to start
func (tf *TViewFramework) Run() error {
	return tf.app.Run()
}

// Stop stops the UI application.
func (tf *TViewFramework) Stop() {
	tf.app.Stop()
}

// Draw refreshes the display.
func (tf *TViewFramework) Draw() {
	tf.app.Draw()
}

// SetInputCapture sets the global input handler.
//
// Parameters:
//   - handler: The function to handle key events
func (tf *TViewFramework) SetInputCapture(handler func(*tcell.EventKey) *tcell.EventKey) {
	tf.app.SetInputCapture(handler)
}

// EnableMouse enables or disables mouse support.
//
// Parameters:
//   - enable: Whether to enable mouse support
func (tf *TViewFramework) EnableMouse(enable bool) {
	tf.app.EnableMouse(enable)
}

// EnablePaste enables or disables paste support.
//
// Parameters:
//   - enable: Whether to enable paste support
func (tf *TViewFramework) EnablePaste(enable bool) {
	tf.app.EnablePaste(enable)
}

// GetApp returns the underlying tview application for advanced usage.
//
// Returns:
//   - The tview.Application instance
func (tf *TViewFramework) GetApp() *tview.Application {
	return tf.app
}
