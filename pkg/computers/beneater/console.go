package beneater

import (
	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/fran150/clementina6502/pkg/terminal/ui"
	"github.com/gdamore/tcell/v2"
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
	ramWindow *ui.MemoryWindow
	romWindow *ui.MemoryWindow

	breakpointForm *ui.BreakPointForm

	active   tview.Primitive
	previous []tview.Primitive

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
	ramWindow := ui.NewMemoryWindow(computer.chips.ram)
	romWindow := ui.NewMemoryWindow(computer.chips.rom)

	breakPointForm := ui.NewBreakPointForm()

	options := ui.NewOptionsWindow([]*ui.OptionsWindowMenuOption{
		{
			Rune:           'r',
			KeyName:        "R",
			KeyDescription: "Reset",
			Action:         computer.Reset,
		},
		{
			Rune:           'b',
			KeyName:        "B",
			KeyDescription: "Breakpoints",
			Action:         console.SetBreakpointConfigMode,
			BackAction:     console.ReturnToPreviousWindow,

			SubMenu: []*ui.OptionsWindowMenuOption{
				{
					Rune:           'r',
					KeyName:        "R",
					KeyDescription: "Remove Selected Breakpoint",

					Action: breakPointForm.RemoveSelectedItem,
				},
			},
		},
		{
			Rune:           's',
			KeyName:        "S",
			KeyDescription: "Speed",
			SubMenu: []*ui.OptionsWindowMenuOption{
				{
					Rune:           '=',
					KeyName:        "+",
					KeyDescription: "Speed Up",
					Action:         computer.SpeedUp,
				},
				{
					Rune:           '-',
					KeyName:        "-",
					KeyDescription: "Speed Down",
					Action:         computer.SpeedDown,
				},
			},
		},
		{
			Rune:           'w',
			KeyName:        "W",
			KeyDescription: "Windows",
			SubMenu: []*ui.OptionsWindowMenuOption{
				{
					Key:            tcell.KeyF1,
					KeyName:        "F1",
					KeyDescription: "CPU",
					Action:         console.ShowCPUWindow,
				},
				{
					Key:            tcell.KeyF2,
					KeyName:        "F2",
					KeyDescription: "VIA",
					Action:         console.ShowVIAWindow,
				},
				{
					Key:            tcell.KeyF3,
					KeyName:        "F3",
					KeyDescription: "LCD",
					Action:         console.ShowLCDWindow,
				},
				{
					Key:            tcell.KeyF4,
					KeyName:        "F4",
					KeyDescription: "ROM",
					Action:         console.ShowROMWindow,
				},
				{
					Key:            tcell.KeyF5,
					KeyName:        "F5",
					KeyDescription: "RAM",
					Action:         console.ShowRAMWindow,
				},
			},
		},
		{
			Rune:           'q',
			KeyName:        "Q",
			KeyDescription: "Quit",
			Action:         computer.Stop,
		},
		{
			Rune:           'p',
			KeyName:        "P",
			KeyDescription: "Pause",
			Action:         computer.Pause,
		},
		{
			Rune:           'o',
			KeyName:        "O",
			KeyDescription: "Resume",
			Action:         computer.Resume,
		},
		{
			Rune:           'i',
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
	console.ramWindow = ramWindow
	console.romWindow = romWindow
	console.breakpointForm = breakPointForm
	console.options = options
	console.previous = make([]tview.Primitive, 2)

	console.SetActiveWindow(romWindow.GetDrawArea())

	return console
}

func (c *console) SetBreakpointConfigMode(context *common.StepContext) {
	c.AppendActiveWindow(c.breakpointForm.GetDrawArea())
}

func (c *console) ShowCPUWindow(context *common.StepContext) {
	c.SetActiveWindow(c.cpuWindow.GetDrawArea())
}

func (c *console) ShowVIAWindow(context *common.StepContext) {
	c.SetActiveWindow(c.viaWindow.GetDrawArea())
}

func (c *console) ShowLCDWindow(context *common.StepContext) {
	c.SetActiveWindow(c.lcdWindow.GetDrawArea())
}

func (c *console) ShowRAMWindow(context *common.StepContext) {
	c.SetActiveWindow(c.ramWindow.GetDrawArea())
}

func (c *console) ShowROMWindow(context *common.StepContext) {
	c.SetActiveWindow(c.romWindow.GetDrawArea())
}

func (c *console) SetActiveWindow(value tview.Primitive) {
	c.active = value
	c.setActiveWindowOnGrid()
}

func (c *console) AppendActiveWindow(value tview.Primitive) {
	c.previous = append(c.previous, c.active)
	c.SetActiveWindow(value)
}

func (c *console) ReturnToPreviousWindow(context *common.StepContext) {
	if c.previous != nil {
		previous, active := common.SlicePop(c.previous)
		c.previous = previous
		c.active = active
		c.setActiveWindowOnGrid()
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

	c.ramWindow.Clear()
	c.ramWindow.Draw(context)

	c.romWindow.Clear()
	c.romWindow.Draw(context)

	c.options.Clear()
	c.options.Draw(context)
}

func (c *console) setActiveWindowOnGrid() {
	c.grid.RemoveItem(c.active)
	c.grid.AddItem(c.active, 0, 1, 3, 1, 0, 0, false)
}
