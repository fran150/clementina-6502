package ui

import (
	"fmt"
	"io"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/fran150/clementina-6502/pkg/components/lcd"
	"github.com/rivo/tview"
)

type LcdControllerWindow struct {
	text *tview.TextView
	lcd  components.LCDControllerChip
}

func NewLcdWindow(lcd components.LCDControllerChip) *LcdControllerWindow {
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

	d.text.Clear()

	// Display LCD Memory section with a header
	fmt.Fprintf(d.text, "[yellow]LCD Memory:[white]\n")
	drawLcdDDRAM(d.text, displayStatus)
	fmt.Fprintf(d.text, "\n")

	// Display status section with better formatting
	fmt.Fprintf(d.text, "[yellow]Display Status:[white]\n")
	fmt.Fprintf(d.text, "├─ Display ON:     [green]%v[white]\n", displayStatus.DisplayOn)
	fmt.Fprintf(d.text, "├─ 8 Bit Mode:     [green]%v[white]\n", displayStatus.Is8BitMode)
	fmt.Fprintf(d.text, "└─ 2 Line Display: [green]%v[white]\n", displayStatus.Is2LineDisplay)
	fmt.Fprintf(d.text, "\n")

	// Cursor and bus information section
	fmt.Fprintf(d.text, "[yellow]Control Status:[white]\n")
	fmt.Fprintf(d.text, "├─ Cursor Position: [green]%v[white]\n", cursorStatus.CursorPosition)
	fmt.Fprintf(d.text, "├─ Bus:             [green]$%02X[white]\n", d.lcd.DataBus().Read())
	fmt.Fprintf(d.text, "├─ Enable:          [green]%v[white]\n", d.lcd.Enable().Enabled())
	fmt.Fprintf(d.text, "├─ Read/Write:      [green]%v[white]\n", d.lcd.ReadWrite().Enabled())
	fmt.Fprintf(d.text, "└─ Reg Select:      [green]%v[white]\n", d.lcd.RegisterSelect().Enabled())
}

func drawLcdDDRAM(writer io.Writer, displayStatus lcd.DisplayStatus) {
	const itemsPerLine = 8

	// Draw top border with header
	fmt.Fprintf(writer, "     ┌")
	for range (itemsPerLine * 3) + 1 {
		fmt.Fprintf(writer, "─")
	}
	fmt.Fprintf(writer, "┐\n")

	// Draw header row
	fmt.Fprintf(writer, "     │ ")
	for i := range itemsPerLine {
		if i == itemsPerLine-1 {
			fmt.Fprintf(writer, "[blue]%2X[white]", i)
		} else {
			fmt.Fprintf(writer, "[blue]%2X [white]", i)
		}
	}
	fmt.Fprintf(writer, " │\n")

	// Draw separator
	fmt.Fprintf(writer, "     ├")
	for range (itemsPerLine * 3) + 1 {
		fmt.Fprintf(writer, "─")
	}
	fmt.Fprintf(writer, "┤\n")

	// Draw memory contents
	for row := range len(displayStatus.DDRAM) / itemsPerLine {
		fmt.Fprintf(writer, " [blue]%02X:[white] │ ", row*itemsPerLine)

		for col := 0; col < itemsPerLine; col++ {
			index := row*itemsPerLine + col
			if index < len(displayStatus.DDRAM) {
				data := displayStatus.DDRAM[index]
				// Show printable characters in green, others in hex
				if data >= 32 && data <= 126 {
					if col == itemsPerLine-1 {
						fmt.Fprintf(writer, "[green]%2c[white]", data)
					} else {
						fmt.Fprintf(writer, "[green]%2c [white]", data)
					}
				} else {
					if col == itemsPerLine-1 {
						fmt.Fprintf(writer, "[yellow]%02X[white]", data)
					} else {
						fmt.Fprintf(writer, "[yellow]%02X [white]", data)
					}
				}
			}
		}
		fmt.Fprintf(writer, " │\n")
	}

	// Draw bottom border
	fmt.Fprintf(writer, "     └")
	for range (itemsPerLine * 3) + 1 {
		fmt.Fprintf(writer, "─")
	}
	fmt.Fprintf(writer, "┘\n")
}

func (d *LcdControllerWindow) GetDrawArea() tview.Primitive {
	return d.text
}
