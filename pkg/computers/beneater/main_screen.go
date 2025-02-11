package beneater

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type mainConsole struct {
	app        *tview.Application
	grid       *tview.Grid
	lcdDisplay *displayWindow
	codeWindow *codeWindow
	other      *tview.TextView
	options    *optionsWindow
}

func createMainConsole() *mainConsole {
	app := tview.NewApplication()

	grid := tview.NewGrid()
	grid.SetRows(4, 0, 3).
		SetColumns(25, 0).
		SetBorder(true).
		SetTitle("Ben Eater 6502 Computer")

	displayWindow := createDisplayWindow()

	codeWindow := createCodeWindow()

	other := tview.NewTextView()
	other.SetTextAlign(tview.AlignLeft).
		SetScrollable(false).
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle("LCD")

	options := createOptionsWindow([]options{
		{"ESC", "Quit"},
		{"R", "Reset CPU"},
		{"P", "Pause Execution"},
		{"S", "Next Step"},
	})

	// Layout for screens narrower than 100 cells (menu and side bar are hidden).
	grid.AddItem(displayWindow.text, 0, 0, 1, 1, 0, 0, false).
		AddItem(codeWindow.code, 1, 0, 1, 1, 0, 0, false).
		AddItem(other, 0, 1, 2, 1, 0, 0, false).
		AddItem(options.text, 2, 0, 1, 2, 0, 0, false)

	return &mainConsole{
		app:        app,
		grid:       grid,
		lcdDisplay: displayWindow,
		codeWindow: codeWindow,
		other:      other,
		options:    options,
	}
}

func (m *mainConsole) Run(inputCapture func(event *tcell.EventKey) *tcell.EventKey) {
	m.app.SetInputCapture(inputCapture)

	if err := m.app.SetRoot(m.grid, true).SetFocus(m.grid).Run(); err != nil {
		panic(err)
	}
}
