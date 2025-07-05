package ui

import (
	"fmt"
	"strings"

	"github.com/fran150/clementina-6502/internal/queue"
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/fran150/clementina-6502/pkg/components/cpu"
	"github.com/rivo/tview"
)

const maxLinesOfCode = 30

// CodeWindow represents a UI component that displays the disassembled code being executed.
// It shows the current instruction and recent execution history with syntax highlighting.
type CodeWindow struct {
	text      *tview.TextView
	lines     *queue.SimpleQueue[string]
	processor components.Cpu6502Chip

	operandsGetter func(programCounter uint16) [2]uint8
}

// NewCodeWindow creates a new code display window that shows disassembled instructions.
// It initializes the UI component and connects it to the provided CPU.
//
// Parameters:
//   - processor: The CPU chip to monitor
//   - operandsGetter: Function to retrieve potential operands for the current instruction
//
// Returns:
//   - A pointer to the initialized CodeWindow
func NewCodeWindow(processor components.Cpu6502Chip, operandsGetter func(programCounter uint16) [2]uint8) *CodeWindow {
	code := tview.NewTextView()
	code.SetTextAlign(tview.AlignLeft)
	code.SetScrollable(false)
	code.SetDynamicColors(true)
	code.SetTitle("Code")
	code.SetBorder(true)

	return &CodeWindow{
		text:           code,
		lines:          queue.NewQueue[string](),
		processor:      processor,
		operandsGetter: operandsGetter,
	}
}

func showCurrentInstruction(programCounter uint16, instruction *cpu.CpuInstructionData, potentialOperands [2]uint8) string {
	sb := strings.Builder{}

	addressMode := instruction.AddressMode()
	addressModeDetails := cpu.GetAddressMode(addressMode)

	var size uint8

	// BRK is internally a 2 byte instruction, but we show it as 1 byte.
	// So no operand reading is needed.
	if instruction.AddressMode() == cpu.AddressModeBreak {
		size = 0
	} else {
		size = addressModeDetails.MemSize() - 1
	}

	// Write current address
	fmt.Fprintf(&sb, "[blue]$%04X: [red]%s [white]", (programCounter - 1), instruction.Mnemonic())

	// Write operands
	switch size {
	case 0:
	case 1:
		fmt.Fprintf(&sb, addressModeDetails.Format(), potentialOperands[0])
	case 2:
		msb := uint16(potentialOperands[1]) << 8
		lsb := uint16(potentialOperands[0])
		fmt.Fprintf(&sb, addressModeDetails.Format(), msb|lsb)
	}

	// If the address mode is relative we will show the value to which the CPU will jump
	if addressMode == cpu.AddressModeRelative || addressMode == cpu.AddressModeRelativeExtended {
		// Get the operator value
		value := uint16(potentialOperands[0])

		// If bit 7 is set then we will perform subtraction by using 2's component
		if value&0x80 == 0x80 {
			value |= 0xFF00
		}

		// Add the value to the program counter
		value = programCounter + value + 1

		// Print the relative jump
		fmt.Fprintf(&sb, "[green] ($%04X)", value)
	}

	fmt.Fprint(&sb, "\r\n")

	return sb.String()
}

func (d *CodeWindow) addLineOfCode(programCounter uint16, instruction *cpu.CpuInstructionData, potentialOperands [2]uint8) {
	codeLine := showCurrentInstruction(programCounter, instruction, potentialOperands)

	d.lines.Queue(codeLine)

	if d.lines.Size() > maxLinesOfCode {
		d.lines.DeQueue()
	}
}

// Tick processes a CPU cycle and updates the code window if a new instruction is being read.
// This method is called on each CPU cycle to maintain the execution history.
//
// Parameters:
//   - context: The current step context containing CPU state information
func (d *CodeWindow) Tick(context *common.StepContext) {
	pc := d.processor.GetProgramCounter()
	instruction := d.processor.GetCurrentInstruction()

	if d.processor.IsReadingOpcode() && instruction != nil {
		d.addLineOfCode(pc, instruction, d.operandsGetter(pc))
	}
}

// Clear resets the code window, removing all text content and execution history.
func (d *CodeWindow) Clear() {
	d.text.Clear()
}

// Draw updates the code window with the current execution history.
// It displays the disassembled instructions that have been executed.
//
// Parameters:
//   - context: The current step context containing system state information
func (d *CodeWindow) Draw(context *common.StepContext) {
	values := d.lines.GetValues()

	if values == nil {
		values = []string{}
	}

	d.text.SetText(strings.Join(values, ""))
}

// GetDrawArea returns the primitive that represents this window in the UI.
// This is used by the layout manager to position and render the window.
//
// Returns:
//   - The tview primitive for this window
func (d *CodeWindow) GetDrawArea() tview.Primitive {
	return d.text
}
