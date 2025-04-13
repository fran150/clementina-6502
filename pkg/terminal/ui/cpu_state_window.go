package ui

import (
	"fmt"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/fran150/clementina-6502/pkg/components/cpu"
	"github.com/rivo/tview"
)

// CpuWindow represents a UI component that displays the current state of the CPU.
// It shows register values, flags, and the current instruction being executed.
type CpuWindow struct {
	text      *tview.TextView
	processor components.Cpu6502Chip
}

// NewCpuWindow creates a new CPU state display window.
// It initializes the UI component and connects it to the provided CPU.
//
// Parameters:
//   - processor: The CPU chip to monitor and display
//
// Returns:
//   - A pointer to the initialized CpuWindow
func NewCpuWindow(processor components.Cpu6502Chip) *CpuWindow {
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
	// Create a consistent layout with aligned columns
	fmt.Fprintf(d.text, "┌────────────────────────┐\n")
	fmt.Fprintf(d.text, "│ [yellow]Registers              [white]│\n")
	fmt.Fprintf(d.text, "│ [yellow]A :   [white]%3d [grey]($%02X)        [white]│\n",
		d.processor.GetAccumulatorRegister(),
		d.processor.GetAccumulatorRegister())
	fmt.Fprintf(d.text, "│ [yellow]X :   [white]%3d [grey]($%02X)        [white]│\n",
		d.processor.GetXRegister(),
		d.processor.GetXRegister())
	fmt.Fprintf(d.text, "│ [yellow]Y :   [white]%3d [grey]($%02X)        [white]│\n",
		d.processor.GetYRegister(),
		d.processor.GetYRegister())
	fmt.Fprintf(d.text, "│ [yellow]SP:   [white]%3d [grey]($%02X)        [white]│\n",
		d.processor.GetStackPointer(),
		d.processor.GetStackPointer())
	fmt.Fprintf(d.text, "│ [yellow]PC: [white]$%04X [grey](%5d)      [white]│\n",
		d.processor.GetProgramCounter(),
		d.processor.GetProgramCounter())
	fmt.Fprintf(d.text, "├────────────────────────┤\n")

	// Status flags with better visual separation
	status := d.processor.GetProcessorStatusRegister()
	fmt.Fprintf(d.text, "│ [yellow]Status Flags:          [white]│\n")
	fmt.Fprintf(d.text, "│ ")

	// Create flag display with descriptions
	flags := []struct {
		name string
		bit  cpu.StatusBit
		desc string
	}{
		{"N", cpu.NegativeFlagBit, "Negative"},
		{"V", cpu.OverflowFlagBit, "Overflow"},
		{"-", cpu.UnusedFlagBit, "Unused"},
		{"B", cpu.BreakCommandFlagBit, "Break"},
		{"D", cpu.DecimalModeFlagBit, "Decimal"},
		{"I", cpu.IrqDisableFlagBit, "IRQ Disable"},
		{"Z", cpu.ZeroFlagBit, "Zero"},
		{"C", cpu.CarryFlagBit, "Carry"},
	}

	// Print flags with colors
	for _, flag := range flags {
		fmt.Fprintf(d.text, "%s%s[white:-]", getFlagStatusColor(status, flag.bit), flag.name)
		if flag.name != "C" {
			fmt.Fprint(d.text, " ")
		}
	}
	fmt.Fprintf(d.text, "        [white]│\n")
	fmt.Fprintf(d.text, "└────────────────────────┘\n")
}

func (d *CpuWindow) GetDrawArea() tview.Primitive {
	return d.text
}
