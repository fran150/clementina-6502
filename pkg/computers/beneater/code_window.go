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
	text  *tview.TextView
	lines *queue.SimpleQueue[string]
}

func createCodeWindow() *codeWindow {
	code := tview.NewTextView()
	code.SetTextAlign(tview.AlignLeft)
	code.SetScrollable(false)
	code.SetDynamicColors(true)
	code.SetTitle("Code")
	code.SetBorder(true)

	return &codeWindow{
		text:  code,
		lines: queue.CreateQueue[string](),
	}
}

func (d *codeWindow) AddLineOfCode(programCounter uint16, instruction *cpu.CpuInstructionData, potentialOperands [2]uint8) {
	codeLine := ui.ShowCurrentInstruction(programCounter, instruction, potentialOperands)

	d.lines.Queue(codeLine)

	if d.lines.Size() > maxLinesOfCode {
		d.lines.DeQueue()
	}
}

func (d *codeWindow) Clear() {
	d.text.Clear()
}

func (d *codeWindow) Draw() {
	values := d.lines.GetValues()
	d.text.SetText(strings.Join(values, ""))
}
