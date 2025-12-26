//go:build (linux && arm) || (linux && arm64)

package ui

import (
	"fmt"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/pico"
	"github.com/rivo/tview"
)

type MiaWindow struct {
	text *tview.TextView
	mia  *pico.MiaConnector
}

func NewMiaWindow(mia *pico.MiaConnector) *MiaWindow {
	text := tview.NewTextView()
	text.SetScrollable(false).
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle("MIA Status")

	return &MiaWindow{
		text: text,
		mia:  mia,
	}
}

func (d *MiaWindow) Clear() {
	d.text.Clear()
}

func (d *MiaWindow) Draw(context *common.StepContext) {
	d.text.Clear()

	// Display status section with better formatting
	fmt.Fprintf(d.text, "[yellow]MIA Status:[white]\n")
	fmt.Fprintf(d.text, "├─ AddressBus:  [green]%04X[white]\n", d.mia.AddressBus().Read())
	fmt.Fprintf(d.text, "├─ DataBus:     [green]%04X[white]\n", d.mia.DataBus().Read())
	fmt.Fprintf(d.text, "├─ HIRAME:      [green]%v[white]\n", d.mia.HiRAMEnable().Enabled())
	fmt.Fprintf(d.text, "├─ RESET:       [green]%v[white]\n", d.mia.Reset().Enabled())
	fmt.Fprintf(d.text, "├─ WE:          [green]%v[white]\n", d.mia.WriteEnable().Enabled())
	fmt.Fprintf(d.text, "├─ OE:          [green]%v[white]\n", d.mia.OutputEnable().Enabled())
	fmt.Fprintf(d.text, "├─ HIRAMCS:     [green]%v[white]\n", d.mia.HiRAMCS().Enabled())
	fmt.Fprintf(d.text, "├─ IO0CS:       [green]%v[white]\n", d.mia.IO0CS().Enabled())
	fmt.Fprintf(d.text, "└─ IRQ:         [green]%v[white]\n", d.mia.IrqOut().Enabled())
	fmt.Fprintf(d.text, "\n")
}

// GetDrawArea returns the primitive that represents this window in the UI.
// This is used by the layout manager to position and render the window.
//
// Returns:
//   - The tview primitive for this window
func (d *MiaWindow) GetDrawArea() tview.Primitive {
	return d.text
}
