package terminal

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/computers"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Computer interface {
	computers.Computer
	Init(app *tview.Application, config *ApplicationConfig)
	KeyPressed(event *tcell.EventKey, context *common.StepContext) *tcell.EventKey
}

type Window interface {
	Clear()
	Draw(context *common.StepContext)
	GetDrawArea() tview.Primitive
}
