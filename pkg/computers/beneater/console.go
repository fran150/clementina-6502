package beneater

import (
	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/fran150/clementina6502/pkg/terminal/ui"
	"github.com/rivo/tview"
)

type console struct {
	lcdDisplay  *ui.Lcd16x2Window
	codeWindow  *ui.CodeWindow
	speedWindow *ui.SpeedWindow

	cpuWindow *ui.CpuWindow
	viaWindow *ui.ViaWindow
	lcdWindow *ui.LcdControllerWindow

	options *ui.OptionsWindow
}

func newMainConsole(computer *BenEaterComputer, tvApp *tview.Application) *console {
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
		operand1Address := (programCounter + 1) & 0x7FFF
		operand2Address := (programCounter + 2) & 0x7FFF

		return [2]uint8{rom.Peek(operand1Address), rom.Peek(operand2Address)}
	})

	speedWindow := ui.NewSpeedWindow()

	cpuWindow := ui.NewCpuWindow(computer.chips.cpu)
	viaWindow := ui.NewViaWindow(computer.chips.via)
	lcdWindow := ui.NewLcdWindow(computer.chips.lcd)

	options := ui.NewOptionsWindow([]ui.OptionsWindowConfig{
		{KeyName: "ESC", KeyDescription: "Quit"},
		{KeyName: "R", KeyDescription: "Reset CPU"},
		{KeyName: "P", KeyDescription: "Pause Execution"},
		{KeyName: "S", KeyDescription: "Next Step"},
	})

	grid.AddItem(displayWindow.GetDrawArea(), 0, 0, 1, 1, 0, 0, false).
		AddItem(speedWindow.GetDrawArea(), 1, 0, 1, 1, 0, 0, false).
		AddItem(codeWindow.GetDrawArea(), 2, 0, 1, 1, 0, 0, false).
		AddItem(lcdWindow.GetDrawArea(), 0, 1, 3, 1, 0, 0, false).
		AddItem(options.GetDrawArea(), 3, 0, 1, 2, 0, 0, false)

	return &console{
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
}
