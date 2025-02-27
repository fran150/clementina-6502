package ui

import (
	"fmt"
	"strings"

	"github.com/fran150/clementina6502/internal/queue"
	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/fran150/clementina6502/pkg/components/cpu"
	"github.com/rivo/tview"
)

const maxLinesOfCode = 30

type CodeWindow struct {
	text      *tview.TextView
	lines     *queue.SimpleQueue[string]
	processor *cpu.Cpu65C02S

	operandsGetter func(programCounter uint16) [2]uint8
}

func NewCodeWindow(processor *cpu.Cpu65C02S, operandsGetter func(programCounter uint16) [2]uint8) *CodeWindow {
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
	var sb strings.Builder = strings.Builder{}

	addressModeDetails := cpu.GetAddressMode(instruction.AddressMode())

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
	default:
		fmt.Fprintf(&sb, "Unrecognized Instruction or Address Mode")
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

func (d *CodeWindow) Tick(context *common.StepContext) {
	pc := d.processor.GetProgramCounter()
	instruction := d.processor.GetCurrentInstruction()

	if d.processor.IsReadingOpcode() && instruction != nil {
		d.addLineOfCode(pc, instruction, d.operandsGetter(pc))
	}
}

func (d *CodeWindow) Clear() {
	d.text.Clear()
}

func (d *CodeWindow) Draw(context *common.StepContext) {
	values := d.lines.GetValues()
	d.text.SetText(strings.Join(values, ""))
}

func (d *CodeWindow) GetDrawArea() *tview.TextView {
	return d.text
}
