package beneater

import (
	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/fran150/clementina6502/pkg/ui/terminal"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type console struct {
	computer *BenEaterComputer
	app      *tview.Application
	grid     *tview.Grid

	lcdDisplay  *terminal.DisplayWindow
	codeWindow  *terminal.CodeWindow
	speedWindow *terminal.SpeedWindow

	cpuWindow *terminal.CpuWindow
	viaWindow *terminal.ViaWindow
	lcdWindow *terminal.LcdWindow

	options *terminal.OptionsWindow
}

func createMainConsole(computer *BenEaterComputer) *console {
	app := tview.NewApplication()

	grid := tview.NewGrid()
	grid.SetRows(4, 3, 0, 3).
		SetColumns(25, 0).
		SetBorder(true).
		SetTitle("Ben Eater 6502 Computer")

	displayWindow := terminal.CreateDisplayWindow(computer.chips.lcd)
	codeWindow := terminal.CreateCodeWindow(computer.chips.cpu, computer.peekNext2Operands)

	speedWindow := terminal.CreateSpeedWindow()

	cpuWindow := terminal.CreateCpuWindow(computer.chips.cpu)
	viaWindow := terminal.CreateViaWindow(computer.chips.via)
	lcdWindow := terminal.CreateLcdWindow(computer.chips.lcd)

	options := terminal.CreateOptionsWindow([]terminal.OptionsWindowConfig{
		{KeyName: "ESC", KeyDescription: "Quit"},
		{KeyName: "R", KeyDescription: "Reset CPU"},
		{KeyName: "P", KeyDescription: "Pause Execution"},
		{KeyName: "S", KeyDescription: "Next Step"},
	})

	// Layout for screens narrower than 100 cells (menu and side bar are hidden).
	grid.AddItem(displayWindow.GetDrawArea(), 0, 0, 1, 1, 0, 0, false).
		AddItem(speedWindow.GetDrawArea(), 1, 0, 1, 1, 0, 0, false).
		AddItem(codeWindow.GetDrawArea(), 2, 0, 1, 1, 0, 0, false).
		AddItem(lcdWindow.GetDrawArea(), 0, 1, 3, 1, 0, 0, false).
		AddItem(options.GetDrawArea(), 3, 0, 1, 2, 0, 0, false)

	return &console{
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

func (c *console) Tick(context *common.StepContext) {
	c.codeWindow.Tick(context)
}

func (c *console) Draw(context *common.StepContext) {
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

func (m *console) Run(inputCapture func(event *tcell.EventKey) *tcell.EventKey) {
	m.app.SetInputCapture(inputCapture)

	if err := m.app.SetRoot(m.grid, true).SetFocus(m.grid).Run(); err != nil {
		panic(err)
	}
}
