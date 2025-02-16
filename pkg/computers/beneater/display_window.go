package beneater

import (
	"fmt"
	"io"

	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/fran150/clementina6502/pkg/components/lcd"
	"github.com/rivo/tview"
)

type displayWindow struct {
	computer *BenEaterComputer
	text     *tview.TextView
}

func createDisplayWindow(computer *BenEaterComputer) *displayWindow {
	text := tview.NewTextView()
	text.SetTextAlign(tview.AlignCenter).
		SetScrollable(false).
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle("LCD Display")

	return &displayWindow{
		computer: computer,
		text:     text,
	}
}

func (d *displayWindow) Clear() {
	d.text.Clear()
}

func (d *displayWindow) Draw(context *common.StepContext) {
	lcd := d.computer.chips.lcd
	const line1MinIndex, line1MaxIndex = 0, 40
	const line2MinIndex, line2MaxIndex = 40, 80

	displayStatus := lcd.GetDisplayStatus()
	cursorStatus := lcd.GetCursorStatus()

	if !displayStatus.DisplayOn {
		drawLcdLineOff(d.text)
		fmt.Fprint(d.text, "\n")
		drawLcdLineOff(d.text)
		return
	}

	if !displayStatus.Is2LineDisplay {
		fmt.Fprint(d.text, "[red]Not in two\nline mode")
		return
	}

	drawLcdLine(d.text, lcd.GetDisplayStatus().Line1Start, displayStatus, cursorStatus, line1MinIndex, line1MaxIndex)
	fmt.Fprint(d.text, "\n")
	drawLcdLine(d.text, lcd.GetDisplayStatus().Line2Start, displayStatus, cursorStatus, line2MinIndex, line2MaxIndex)
}

func drawLcdLineOff(writer io.Writer) {
	fmt.Fprintf(writer, "[black:grey]")

	for range 16 {
		fmt.Fprint(writer, " ")
	}
}

func drawLcdLine(writer io.Writer, lineStart uint8, displayStatus lcd.DisplayStatus, cursorStatus lcd.CursorStatus, min uint8, max uint8) {
	var count uint8 = 0
	var index uint8 = lineStart

	fmt.Fprintf(writer, "[black:green]")

	for count < 16 {
		if index >= max {
			index = min
		}

		char := string(displayStatus.DDRAM[index])

		if cursorStatus.BlinkStatusShowing && index == cursorStatus.CursorPosition {
			char = "█"
		}

		if cursorStatus.CursorVisible && index == cursorStatus.CursorPosition {
			fmt.Fprintf(writer, "[::u]%s[::-]", char)
		} else {
			fmt.Fprint(writer, char)
		}

		index++
		count++
	}
}
