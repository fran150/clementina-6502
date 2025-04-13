package terminal

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/computers"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Computer defines the interface for a computer system that can be controlled
// through the terminal UI. It extends the base Computer interface with UI-specific
// functionality for initialization and keyboard input handling.
type Computer interface {
	computers.Computer
	Init(app *tview.Application, config *ApplicationConfig)
	KeyPressed(event *tcell.EventKey, context *common.StepContext) *tcell.EventKey
}

// Window defines the interface for UI components that can be drawn in the terminal.
// It provides methods for clearing, drawing, and retrieving the drawable area.
type Window interface {
	Clear()
	Draw(context *common.StepContext)
	GetDrawArea() tview.Primitive
}
