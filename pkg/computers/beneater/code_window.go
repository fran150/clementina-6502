package beneater

import (
	"fmt"
	"strings"

	"github.com/fran150/clementina6502/internal/queue"
	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/fran150/clementina6502/pkg/components/cpu"
	"github.com/rivo/tview"
)

const maxLinesOfCode = 30

type codeWindow struct {
	text     *tview.TextView
	lines    *queue.SimpleQueue[string]
	computer *BenEaterComputer
}

func createCodeWindow(computer *BenEaterComputer) *codeWindow {
	code := tview.NewTextView()
	code.SetTextAlign(tview.AlignLeft)
	code.SetScrollable(false)
	code.SetDynamicColors(true)
	code.SetTitle("Code")
	code.SetBorder(true)

	return &codeWindow{
		text:     code,
		lines:    queue.CreateQueue[string](),
		computer: computer,
	}
}

func showCurrentInstruction(programCounter uint16, instruction *cpu.CpuInstructionData, potentialOperands [2]uint8) string {
	var sb strings.Builder = strings.Builder{}

	addressModeDetails := cpu.GetAddressMode(instruction.AddressMode())

	size := addressModeDetails.MemSize() - 1

	// Write current address
	fmt.Fprintf(&sb, "[blue]$%04X: [red]%s [white]", programCounter, instruction.Mnemonic())

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

func (d *codeWindow) addLineOfCode(programCounter uint16, instruction *cpu.CpuInstructionData, potentialOperands [2]uint8) {
	codeLine := showCurrentInstruction(programCounter, instruction, potentialOperands)

	d.lines.Queue(codeLine)

	if d.lines.Size() > maxLinesOfCode {
		d.lines.DeQueue()
	}
}

func (d *codeWindow) getPotentialOperands(programCounter uint16) [2]uint8 {
	rom := d.computer.chips.rom

	programCounter &= 0x7FFF
	operand1Address := (programCounter + 1) & 0x7FFF
	operand2Address := (programCounter + 2) & 0x7FFF

	return [2]uint8{rom.Peek(operand1Address), rom.Peek(operand2Address)}
}

func (d *codeWindow) Tick(context *common.StepContext) {
	cpu := d.computer.chips.cpu

	pc := cpu.GetProgramCounter()
	instruction := cpu.GetCurrentInstruction()

	if cpu.IsReadingOpcode() && instruction != nil {
		d.addLineOfCode(pc, instruction, d.getPotentialOperands(pc))
	}
}

func (d *codeWindow) Clear() {
	d.text.Clear()
}

func (d *codeWindow) Draw(context *common.StepContext) {
	values := d.lines.GetValues()
	d.text.SetText(strings.Join(values, ""))
}
