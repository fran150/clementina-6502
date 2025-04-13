package ui

import (
	"fmt"
	"io"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/fran150/clementina-6502/pkg/components/lcd"
	"github.com/rivo/tview"
)

// LcdControllerWindow represents a UI component that displays the LCD controller state.
// It shows the current content of the LCD display and controller status.
type LcdControllerWindow struct {
	text *tview.TextView
	lcd  components.LCDControllerChip
}

// NewLcdWindow creates a new LCD controller display window.
// It initializes the UI component and connects it to the provided LCD controller.
//
// Parameters:
//   - lcd: The LCD controller chip to display
//
// Returns:
//   - A pointer to the initialized LcdControllerWindow
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

// Clear resets the LCD controller window, removing all text content.
func (d *LcdControllerWindow) Clear() {
	d.text.Clear()
}

// Draw updates the LCD controller window with the current LCD state.
// It displays the LCD memory contents, display status, and control signals.
//
// Parameters:
//   - context: The current step context containing system state information
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

// GetDrawArea returns the primitive that represents this window in the UI.
// This is used by the layout manager to position and render the window.
//
// Returns:
//   - The tview primitive for this window
func (d *LcdControllerWindow) GetDrawArea() tview.Primitive {
	return d.text
}
