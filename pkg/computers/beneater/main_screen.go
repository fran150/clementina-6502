package beneater

import (
	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type mainConsole struct {
	computer *BenEaterComputer
	app      *tview.Application
	grid     *tview.Grid

	lcdDisplay  *displayWindow
	codeWindow  *codeWindow
	speedWindow *speedWindow

	cpuWindow *cpuWindow
	viaWindow *viaWindow
	lcdWindow *lcdWindow

	options *optionsWindow
}

func createMainConsole(computer *BenEaterComputer) *mainConsole {
	app := tview.NewApplication()

	grid := tview.NewGrid()
	grid.SetRows(4, 3, 0, 3).
		SetColumns(25, 0).
		SetBorder(true).
		SetTitle("Ben Eater 6502 Computer")

	displayWindow := createDisplayWindow(computer)
	codeWindow := createCodeWindow(computer)
	speedWindow := createSpeedWindow()

	cpuWindow := createCpuWindow(computer)
	viaWindow := createViaWindow(computer)
	lcdWindow := createLcdWindow(computer)

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
		AddItem(lcdWindow.text, 0, 1, 3, 1, 0, 0, false).
		AddItem(options.text, 3, 0, 1, 2, 0, 0, false)

	return &mainConsole{
		computer:    computer,
		app:         app,
		grid:        grid,
		lcdDisplay:  displayWindow,
		codeWindow:  codeWindow,
		speedWindow: speedWindow,

		cpuWindow: cpuWindow,
		viaWindow: viaWindow,
		lcdWindow: lcdWindow,
		options:   options,
	}
}

func (c *mainConsole) Tick(context *common.StepContext) {
	c.codeWindow.Tick(context)
}

func (c *mainConsole) Draw(context *common.StepContext) {
	c.lcdDisplay.Clear()
	c.lcdDisplay.Draw(context)

	c.codeWindow.Clear()
	c.codeWindow.Draw(context)

	c.speedWindow.Clear()
	c.speedWindow.Draw(context)

	c.viaWindow.Clear()
	c.viaWindow.Draw(context)

	c.cpuWindow.Clear()
	c.cpuWindow.Draw(context)

	c.lcdWindow.Clear()
	c.lcdWindow.Draw(context)

	c.app.Draw()
}

func (m *mainConsole) Run(inputCapture func(event *tcell.EventKey) *tcell.EventKey) {
	m.app.SetInputCapture(inputCapture)

	if err := m.app.SetRoot(m.grid, true).SetFocus(m.grid).Run(); err != nil {
		panic(err)
	}
}
