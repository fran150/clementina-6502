package beneater

import (
	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/fran150/clementina6502/pkg/terminal/ui"
	"github.com/rivo/tview"
)

type console struct {
	grid *tview.Grid

	lcdDisplay  *ui.Lcd16x2Window
	codeWindow  *ui.CodeWindow
	speedWindow *ui.SpeedWindow

	cpuWindow *ui.CpuWindow
	viaWindow *ui.ViaWindow
	lcdWindow *ui.LcdControllerWindow

	breakpointForm *ui.BreakPointForm

	active   tview.Primitive
	previous tview.Primitive

	options *ui.OptionsWindow
}

func newMainConsole(computer *BenEaterComputer, tvApp *tview.Application) *console {
	console := &console{}

	grid := tview.NewGrid()
	grid.SetRows(4, 3, 0, 3).
		SetColumns(25, 0).
		SetBorder(true).
		SetTitle("Ben Eater 6502 Computer")

	tvApp.SetRoot(grid, true)

	displayWindow := ui.NewDisplayWindow(computer.chips.lcd)

	codeWindow := ui.NewCodeWindow(computer.chips.cpu, func(programCounter uint16) [2]uint8 {
		// TODO: Might need to improve using address decoding logic
		rom := computer.chips.rom

		programCounter &= 0x7FFF
		operand1Address := programCounter & 0x7FFF
		operand2Address := (programCounter + 1) & 0x7FFF

		return [2]uint8{rom.Peek(operand1Address), rom.Peek(operand2Address)}
	})

	speedWindow := ui.NewSpeedWindow()

	cpuWindow := ui.NewCpuWindow(computer.chips.cpu)
	viaWindow := ui.NewViaWindow(computer.chips.via)
	lcdWindow := ui.NewLcdWindow(computer.chips.lcd)

	breakPointForm := ui.NewBreakPointForm()

	options := ui.NewOptionsWindow([]*ui.OptionsWindowMenuOption{
		{
			Key:            'r',
			KeyName:        "R",
			KeyDescription: "Reset",
			Action:         computer.Reset,
		},
		{
			Key:            'b',
			KeyName:        "B",
			KeyDescription: "Breakpoints",
			Action:         console.SetBreakpointConfigMode,
			BackAction:     console.ReturnToPreviousWindow,

			SubMenu: []*ui.OptionsWindowMenuOption{
				{
					Key:            'r',
					KeyName:        "R",
					KeyDescription: "Remove Selected Breakpoint",

					Action: breakPointForm.RemoveSelectedItem,
				},
			},
		},
		{
			Key:            'q',
			KeyName:        "Q",
			KeyDescription: "Quit",
			Action:         computer.Stop,
		},
		{
			Key:            'p',
			KeyName:        "P",
			KeyDescription: "Pause",
			Action:         computer.Pause,
		},
		{
			Key:            'o',
			KeyName:        "O",
			KeyDescription: "Resume",
			Action:         computer.Resume,
		},
		{
			Key:            's',
			KeyName:        "S",
			KeyDescription: "Speed",
			SubMenu: []*ui.OptionsWindowMenuOption{
				{
					Key:            '=',
					KeyName:        "+",
					KeyDescription: "Speed Up",
					Action:         computer.SpeedUp,
				},
				{
					Key:            '-',
					KeyName:        "-",
					KeyDescription: "Speed Down",
					Action:         computer.SpeedDown,
				},
			},
		},
		{
			Key:            'i',
			KeyName:        "I",
			KeyDescription: "Step",
			Action:         computer.Step,
		},
	})

	grid.AddItem(displayWindow.GetDrawArea(), 0, 0, 1, 1, 0, 0, false).
		AddItem(speedWindow.GetDrawArea(), 1, 0, 1, 1, 0, 0, false).
		AddItem(codeWindow.GetDrawArea(), 2, 0, 1, 1, 0, 0, false).
		AddItem(options.GetDrawArea(), 3, 0, 1, 2, 0, 0, false)

	console.grid = grid
	console.lcdDisplay = displayWindow
	console.codeWindow = codeWindow
	console.speedWindow = speedWindow
	console.cpuWindow = cpuWindow
	console.viaWindow = viaWindow
	console.lcdWindow = lcdWindow
	console.breakpointForm = breakPointForm
	console.options = options
	console.SetActiveWindow(cpuWindow.GetDrawArea())

	return console
}

func (c *console) SetBreakpointConfigMode(context *common.StepContext) {
	c.SetActiveWindow(c.breakpointForm.GetDrawArea())
}

func (c *console) SetActiveWindow(value tview.Primitive) {
	c.grid.RemoveItem(c.active)
	c.previous = c.active
	c.active = value
	c.grid.AddItem(c.active, 0, 1, 3, 1, 0, 0, false)
}

func (c *console) ReturnToPreviousWindow(context *common.StepContext) {
	if c.previous != nil {
		c.grid.RemoveItem(c.active)
		c.active = c.previous
		c.previous = nil
		c.grid.AddItem(c.active, 0, 1, 3, 1, 0, 0, false)
	}
}

func (c *console) Tick(context *common.StepContext) {
	c.codeWindow.Tick(context)
}

func (c *console) Draw(context *common.StepContext) {
	// TODO: Only draw the visible one
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

	c.options.Clear()
	c.options.Draw(context)
}
