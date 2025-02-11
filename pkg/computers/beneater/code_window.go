package beneater

import (
	"strings"

	"github.com/fran150/clementina6502/internal/queue"
	"github.com/fran150/clementina6502/pkg/components/cpu"
	"github.com/fran150/clementina6502/pkg/ui"
	"github.com/rivo/tview"
)

const maxLinesOfCode = 30

type codeWindow struct {
	code      *tview.TextView
	codeLines *queue.SimpleQueue[string]
}

func createCodeWindow() *codeWindow {
	code := tview.NewTextView()
	code.SetTextAlign(tview.AlignLeft)
	code.SetScrollable(false)
	code.SetDynamicColors(true)
	code.SetTitle("Code")
	code.SetBorder(true)

	return &codeWindow{
		code:      code,
		codeLines: queue.CreateQueue[string](),
	}
}

func (d *codeWindow) AddLineOfCode(programCounter uint16, instruction *cpu.CpuInstructionData, potentialOperands [2]uint8) {
	codeLine := ui.ShowCurrentInstruction(programCounter, instruction, potentialOperands)

	d.codeLines.Queue(codeLine)

	if d.codeLines.Size() > maxLinesOfCode {
		d.codeLines.DeQueue()
	}
}

func (d *codeWindow) ShowCode() {
	values := d.codeLines.GetValues()
	d.code.SetText(strings.Join(values, ""))
}
