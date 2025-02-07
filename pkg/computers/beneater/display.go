package beneater

import (
	"strings"

	"github.com/fran150/clementina6502/internal/queue"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type display struct {
	app          *tview.Application
	grid         *tview.Grid
	code         *tview.TextView
	other        *tview.TextView
	instructions *queue.SimpleQueue[string]
}

func CreateDisplay() *display {
	app := tview.NewApplication()

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			app.Stop()
		}

		return event
	})

	newPrimitive := func(text string) *tview.TextView {
		return tview.NewTextView().
			SetTextAlign(tview.AlignLeft).
			SetScrollable(false).
			SetDynamicColors(true).
			SetText(text)
	}

	code := newPrimitive("Main content")
	other := newPrimitive("Side Bar")

	grid := tview.NewGrid().
		SetRows(0).
		SetColumns(18, 0).
		SetBorders(true)

	// Layout for screens narrower than 100 cells (menu and side bar are hidden).
	grid.AddItem(code, 0, 0, 1, 1, 0, 0, false).
		AddItem(other, 0, 1, 1, 1, 0, 0, false)

	return &display{
		app:          app,
		grid:         grid,
		code:         code,
		other:        other,
		instructions: queue.CreateQueue[string](),
	}
}

func (d *display) AddInstruction(instruction string) {
	d.instructions.Queue(instruction)

	if d.instructions.Size() > 30 {
		d.instructions.DeQueue()
	}
}

func (d *display) ShowInstructions() {
	values := d.instructions.GetValues()
	d.code.SetText(strings.Join(values, ""))
}
