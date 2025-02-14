package beneater

import (
	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/fran150/clementina6502/pkg/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type mainConsole struct {
	computer    *BenEaterComputer
	app         *tview.Application
	grid        *tview.Grid
	lcdDisplay  *displayWindow
	codeWindow  *codeWindow
	speedWindow *speedWindow
	other       *tview.TextView
	options     *optionsWindow
}

func createMainConsole(computer *BenEaterComputer) *mainConsole {
	app := tview.NewApplication()

	grid := tview.NewGrid()
	grid.SetRows(4, 3, 0, 3).
		SetColumns(25, 0).
		SetBorder(true).
		SetTitle("Ben Eater 6502 Computer")

	displayWindow := createDisplayWindow(computer)

	codeWindow := createCodeWindow()

	speedWindow := createSpeedWindow()

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
		AddItem(speedWindow.text, 1, 0, 1, 1, 0, 0, false).
		AddItem(codeWindow.text, 2, 0, 1, 1, 0, 0, false).
		AddItem(other, 0, 1, 3, 1, 0, 0, false).
		AddItem(options.text, 3, 0, 1, 2, 0, 0, false)

	return &mainConsole{
		computer:    computer,
		app:         app,
		grid:        grid,
		lcdDisplay:  displayWindow,
		codeWindow:  codeWindow,
		speedWindow: speedWindow,
		other:       other,
		options:     options,
	}
}

func (c *mainConsole) Draw(context *common.StepContext) {
	c.lcdDisplay.Clear()
	c.lcdDisplay.Draw()

	c.codeWindow.Clear()
	c.codeWindow.Draw()

	c.speedWindow.text.Clear()
	c.speedWindow.Draw(context)

	c.other.Clear()
	ui.ShowLCDState(c.other, c.computer.chips.lcd)

	c.app.Draw()
}

func (m *mainConsole) Run(inputCapture func(event *tcell.EventKey) *tcell.EventKey) {
	m.app.SetInputCapture(inputCapture)

	if err := m.app.SetRoot(m.grid, true).SetFocus(m.grid).Run(); err != nil {
		panic(err)
	}
}
