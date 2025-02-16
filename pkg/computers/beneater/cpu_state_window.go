package beneater

import (
	"fmt"

	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/fran150/clementina6502/pkg/components/cpu"
	"github.com/rivo/tview"
)

type cpuWindow struct {
	text     *tview.TextView
	computer *BenEaterComputer
}

func createCpuWindow(computer *BenEaterComputer) *cpuWindow {
	text := tview.NewTextView()
	text.SetScrollable(false).
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle("CPU State")

	return &cpuWindow{
		text:     text,
		computer: computer,
	}
}

func getFlagStatusColor(status cpu.StatusRegister, bit cpu.StatusBit) string {
	if status.Flag(bit) {
		return "[green]"
	}

	return "[red]"
}

func (d *cpuWindow) Clear() {
	d.text.Clear()
}

func (d *cpuWindow) Draw(context *common.StepContext) {
	processor := d.computer.chips.cpu

	fmt.Fprintf(d.text, "[yellow] A: [white]%5d [grey]($%02X)\n", processor.GetAccumulatorRegister(), processor.GetAccumulatorRegister())
	fmt.Fprintf(d.text, "[yellow] X: [white]%5d [grey]($%02X)\n", processor.GetXRegister(), processor.GetXRegister())
	fmt.Fprintf(d.text, "[yellow] Y: [white]%5d [grey]($%02X)\n", processor.GetYRegister(), processor.GetYRegister())
	fmt.Fprintf(d.text, "[yellow]SP: [white]%5d [grey]($%02X)\n", processor.GetStackPointer(), processor.GetStackPointer())
	fmt.Fprintf(d.text, "[yellow]PC: [white]$%04X [grey](%v)\n", processor.GetProgramCounter(), processor.GetProgramCounter())

	status := processor.GetProcessorStatusRegister()

	fmt.Fprint(d.text, "[yellow]Flags: ")
	fmt.Fprintf(d.text, "%sN", getFlagStatusColor(status, cpu.NegativeFlagBit))
	fmt.Fprintf(d.text, "%sV", getFlagStatusColor(status, cpu.OverflowFlagBit))
	fmt.Fprintf(d.text, "%s-", getFlagStatusColor(status, cpu.UnusedFlagBit))
	fmt.Fprintf(d.text, "%sB", getFlagStatusColor(status, cpu.BreakCommandFlagBit))
	fmt.Fprintf(d.text, "%sD", getFlagStatusColor(status, cpu.DecimalModeFlagBit))
	fmt.Fprintf(d.text, "%sI", getFlagStatusColor(status, cpu.IrqDisableFlagBit))
	fmt.Fprintf(d.text, "%sZ", getFlagStatusColor(status, cpu.ZeroFlagBit))
	fmt.Fprintf(d.text, "%sC", getFlagStatusColor(status, cpu.CarryFlagBit))
	fmt.Fprint(d.text, "\n")
}
