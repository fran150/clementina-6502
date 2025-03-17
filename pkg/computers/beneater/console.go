package beneater

import (
	"github.com/fran150/clementina6502/internal/slicesext"
	"github.com/fran150/clementina6502/pkg/common"
	"github.com/fran150/clementina6502/pkg/terminal"
	"github.com/fran150/clementina6502/pkg/terminal/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type console struct {
	grid *tview.Grid

	lcdDisplay  *ui.Lcd16x2Window
	codeWindow  *ui.CodeWindow
	speedWindow *ui.SpeedWindow

	cpuWindow  *ui.CpuWindow
	viaWindow  *ui.ViaWindow
	lcdWindow  *ui.LcdControllerWindow
	aciaWindow *ui.AciaWindow
	ramWindow  *ui.MemoryWindow
	romWindow  *ui.MemoryWindow
	busWindow  *ui.BusWindow

	breakpointForm *ui.BreakPointForm

	active   terminal.Window
	previous []terminal.Window

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
	aciaWindow := ui.NewAciaWindow(computer.chips.acia)
	ramWindow := ui.NewMemoryWindow(computer.chips.ram)
	romWindow := ui.NewMemoryWindow(computer.chips.rom)

	busWindow := ui.NewBusWindow()
	busWindow.AddBus16("Address Bus", computer.circuit.addressBus)
	busWindow.AddBus8("Data Bus", computer.circuit.dataBus)
	busWindow.AddBus8("Port A", computer.circuit.portABus)
	busWindow.AddBus8("Port B", computer.circuit.portBBus)

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
					KeyDescription: "ACIA",
					Action:         console.ShowAciaWindow,
				},
				{
					Key:            tcell.KeyF4,
					KeyName:        "F4",
					KeyDescription: "LCD",
					Action:         console.ShowLCDWindow,
				},
				{
					Key:            tcell.KeyF5,
					KeyName:        "F5",
					KeyDescription: "ROM",
					Action:         console.ShowROMWindow,
					SubMenu: []*ui.OptionsWindowMenuOption{
						{
							Rune:           'w',
							KeyName:        "W",
							KeyDescription: "Scroll Up",
							Action:         console.ScrollUp,
						},
						{
							Rune:           's',
							KeyName:        "S",
							KeyDescription: "Scroll Down",
							Action:         console.ScrollDown,
						},
						{
							Rune:           'e',
							KeyName:        "E",
							KeyDescription: "Scroll Up Fast",
							Action:         console.ScrollUpFast,
						},
						{
							Rune:           'd',
							KeyName:        "D",
							KeyDescription: "Scroll Down Fast",
							Action:         console.ScrollDownFast,
						},
					},
				},
				{
					Key:            tcell.KeyF6,
					KeyName:        "F6",
					KeyDescription: "RAM",
					Action:         console.ShowRAMWindow,
					SubMenu: []*ui.OptionsWindowMenuOption{
						{
							Rune:           'w',
							KeyName:        "W",
							KeyDescription: "Scroll Up",
							Action:         console.ScrollUp,
						},
						{
							Rune:           's',
							KeyName:        "S",
							KeyDescription: "Scroll Down",
							Action:         console.ScrollDown,
						},
						{
							Rune:           'e',
							KeyName:        "E",
							KeyDescription: "Scroll Up Fast",
							Action:         console.ScrollUpFast,
						},
						{
							Rune:           'd',
							KeyName:        "D",
							KeyDescription: "Scroll Down Fast",
							Action:         console.ScrollDownFast,
						},
					},
				},
				{
					Key:            tcell.KeyF7,
					KeyName:        "F7",
					KeyDescription: "Buses",
					Action:         console.ShowBusWindow,
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
	console.aciaWindow = aciaWindow
	console.lcdWindow = lcdWindow
	console.ramWindow = ramWindow
	console.romWindow = romWindow
	console.busWindow = busWindow
	console.breakpointForm = breakPointForm
	console.options = options
	console.previous = make([]terminal.Window, 2)

	console.SetActiveWindow(cpuWindow)

	return console
}

/************************************************************************************
* Set Active Windows
*************************************************************************************/

func (c *console) SetBreakpointConfigMode(context *common.StepContext) {
	c.AppendActiveWindow(c.breakpointForm)
}

func (c *console) ShowCPUWindow(context *common.StepContext) {
	c.SetActiveWindow(c.cpuWindow)
}

func (c *console) ShowVIAWindow(context *common.StepContext) {
	c.SetActiveWindow(c.viaWindow)
}

func (c *console) ShowLCDWindow(context *common.StepContext) {
	c.SetActiveWindow(c.lcdWindow)
}

func (c *console) ShowAciaWindow(context *common.StepContext) {
	c.SetActiveWindow(c.aciaWindow)
}

func (c *console) ShowRAMWindow(context *common.StepContext) {
	c.SetActiveWindow(c.ramWindow)
}

func (c *console) ShowROMWindow(context *common.StepContext) {
	c.SetActiveWindow(c.romWindow)
}

func (c *console) ShowBusWindow(context *common.StepContext) {
	c.SetActiveWindow(c.busWindow)
}

/************************************************************************************
* Other key functions
*************************************************************************************/

func (c *console) ScrollUp(context *common.StepContext) {
	explorer := c.active.(*ui.MemoryWindow)
	explorer.ScrollUp(1)
}

func (c *console) ScrollUpFast(context *common.StepContext) {
	explorer := c.active.(*ui.MemoryWindow)
	explorer.ScrollUp(20)
}

func (c *console) ScrollDown(context *common.StepContext) {
	explorer := c.active.(*ui.MemoryWindow)
	explorer.ScrollDown(1)
}

func (c *console) ScrollDownFast(context *common.StepContext) {
	explorer := c.active.(*ui.MemoryWindow)
	explorer.ScrollDown(20)
}

/************************************************************************************
* Internal Functions
*************************************************************************************/

func (c *console) SetActiveWindow(value terminal.Window) {
	c.active = value
	c.setActiveWindowOnGrid()
}

func (c *console) AppendActiveWindow(value terminal.Window) {
	c.previous = append(c.previous, c.active)
	c.SetActiveWindow(value)
}

func (c *console) ReturnToPreviousWindow(context *common.StepContext) {
	if c.previous != nil {
		previous, active := slicesext.SlicePop(c.previous)
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

	c.speedWindow.Clear()
	c.speedWindow.Draw(context)

	c.codeWindow.Clear()
	c.codeWindow.Draw(context)

	c.cpuWindow.Clear()
	c.cpuWindow.Draw(context)

	c.viaWindow.Clear()
	c.viaWindow.Draw(context)

	c.aciaWindow.Clear()
	c.aciaWindow.Draw(context)

	c.lcdWindow.Clear()
	c.lcdWindow.Draw(context)

	c.ramWindow.Clear()
	c.ramWindow.Draw(context)

	c.romWindow.Clear()
	c.romWindow.Draw(context)

	c.busWindow.Clear()
	c.busWindow.Draw(context)

	c.options.Clear()
	c.options.Draw(context)
}

func (c *console) setActiveWindowOnGrid() {
	c.grid.RemoveItem(c.active.GetDrawArea())
	c.grid.AddItem(c.active.GetDrawArea(), 0, 1, 3, 1, 0, 0, false)
}
