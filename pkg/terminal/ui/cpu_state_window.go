package ui

import (
	"fmt"

	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/fran150/clementina6502/pkg/components/cpu"
	"github.com/rivo/tview"
)

type CpuWindow struct {
	text      *tview.TextView
	processor *cpu.Cpu65C02S
}

func NewCpuWindow(processor *cpu.Cpu65C02S) *CpuWindow {
	text := tview.NewTextView()
	text.SetScrollable(false).
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle("CPU State")

	return &CpuWindow{
		text:      text,
		processor: processor,
	}
}

func getFlagStatusColor(status cpu.StatusRegister, bit cpu.StatusBit) string {
	if status.Flag(bit) {
		return "[green]"
	}

	return "[red]"
}

func (d *CpuWindow) Clear() {
	d.text.Clear()
}

func (d *CpuWindow) Draw(context *common.StepContext) {
	fmt.Fprintf(d.text, "[yellow] A: [white]%5d [grey]($%02X)\n", d.processor.GetAccumulatorRegister(), d.processor.GetAccumulatorRegister())
	fmt.Fprintf(d.text, "[yellow] X: [white]%5d [grey]($%02X)\n", d.processor.GetXRegister(), d.processor.GetXRegister())
	fmt.Fprintf(d.text, "[yellow] Y: [white]%5d [grey]($%02X)\n", d.processor.GetYRegister(), d.processor.GetYRegister())
	fmt.Fprintf(d.text, "[yellow]SP: [white]%5d [grey]($%02X)\n", d.processor.GetStackPointer(), d.processor.GetStackPointer())
	fmt.Fprintf(d.text, "[yellow]PC: [white]$%04X [grey](%v)\n", d.processor.GetProgramCounter(), d.processor.GetProgramCounter())

	status := d.processor.GetProcessorStatusRegister()

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

func (d *CpuWindow) GetDrawArea() tview.Primitive {
	return d.text
}
