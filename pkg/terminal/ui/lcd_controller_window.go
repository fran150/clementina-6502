package ui

import (
	"fmt"
	"io"

	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/fran150/clementina6502/pkg/components/lcd"
	"github.com/rivo/tview"
)

type LcdControllerWindow struct {
	text *tview.TextView
	lcd  *lcd.LcdHD44780U
}

func NewLcdWindow(lcd *lcd.LcdHD44780U) *LcdControllerWindow {
	text := tview.NewTextView()
	text.SetScrollable(false).
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle("LCD Controller Status")

	return &LcdControllerWindow{
		text: text,
		lcd:  lcd,
	}
}

func (d *LcdControllerWindow) Clear() {
	d.text.Clear()
}

func (d *LcdControllerWindow) Draw(context *common.StepContext) {
	cursorStatus := d.lcd.GetCursorStatus()
	displayStatus := d.lcd.GetDisplayStatus()

	fmt.Fprintf(d.text, "LCD Memory:\n")
	drawLcdDDRAM(d.text, displayStatus)

	fmt.Fprintf(d.text, "Display ON: %v\n", displayStatus.DisplayOn)
	fmt.Fprintf(d.text, "8 Bit Mode: %v\n", displayStatus.Is8BitMode)
	fmt.Fprintf(d.text, "Line 2 display: %v\n", displayStatus.Is2LineDisplay)
	fmt.Fprintf(d.text, "Cursor Position: %v\n", cursorStatus.CursorPosition)
	fmt.Fprintf(d.text, "Bus: %v\n", d.lcd.DataBus().Read())
	fmt.Fprintf(d.text, "E: %v\n", d.lcd.Enable().Enabled())
	fmt.Fprintf(d.text, "RW: %v\n", d.lcd.ReadWrite().Enabled())
	fmt.Fprintf(d.text, "RS: %v\n", d.lcd.RegisterSelect().Enabled())
}

func drawLcdDDRAM(writer io.Writer, displayStatus lcd.DisplayStatus) {
	const itemsPerLine = 10

	for i, data := range displayStatus.DDRAM {
		fmt.Fprintf(writer, "[yellow]%02v: [white]%s ", i, string(data))

		if i%itemsPerLine == (itemsPerLine - 1) {
			fmt.Fprintf(writer, "\n")
		}
	}
}

func (d *LcdControllerWindow) GetDrawArea() tview.Primitive {
	return d.text
}
