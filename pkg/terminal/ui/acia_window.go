package ui

import (
	"fmt"

	"github.com/fran150/clementina6502/pkg/common"
	"github.com/fran150/clementina6502/pkg/components"
	"github.com/rivo/tview"
)

// AciaWindow represents a window that displays ACIA (Asynchronous Communications Interface Adapter) information
type AciaWindow struct {
	// text is the TextView component used to display ACIA information
	text *tview.TextView
	// acia is the reference to the ACIA chip being monitored
	acia components.Acia6522Chip
}

// NewAciaWindow creates and initializes a new ACIA window with the provided ACIA chip
func NewAciaWindow(acia components.Acia6522Chip) *AciaWindow {
	text := tview.NewTextView()
	text.SetScrollable(false).
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle("ACIA Registers")

	return &AciaWindow{
		text: text,
		acia: acia,
	}
}

// Clear removes all content from the ACIA window
func (w *AciaWindow) Clear() {
	w.text.Clear()
}

// bitStatusDetail represents a single bit status with its value and descriptive name
type bitStatusDetail struct {
	// value represents the bit pattern to match for this status
	value uint8
	// name is the human-readable description of this status
	name string
}

// drawBitStatusDetail renders a specific bit status with its details using the provided color scheme
func (w *AciaWindow) drawBitStatusDetail(value uint8, mask uint8, color string, title string, details []bitStatusDetail) {
	value &= mask
	activeDetail := ""
	for _, detail := range details {
		if value == detail.value {
			activeDetail = detail.name
			break
		}
	}

	indicator := "○"
	if value != 0 {
		indicator = "●"
	}

	fmt.Fprintf(w.text, "[%s]%s [-]%-30s: [%s]%s\n", color, indicator, title, color, activeDetail)
}

// drawStatusRegisterDetails displays the detailed status of each bit in the status register
func (w *AciaWindow) drawStatusRegisterDetails() {
	const color = "blue"
	status := w.acia.GetStatusRegister()

	fmt.Fprintf(w.text, "\n[%s]Status Register Details:\n", color)
	w.drawBitStatusDetail(status, 0x80, color, "Interrupt (IRQ)", []bitStatusDetail{
		{0x00, "No Interrupt"},
		{0x80, "Interrupt occurred"},
	})
	w.drawBitStatusDetail(status, 0x40, color, "Data Set Ready (DSR)", []bitStatusDetail{
		{0x00, "Ready"},
		{0x40, "Not Ready"},
	})
	w.drawBitStatusDetail(status, 0x20, color, "Data Carrier Detect (DCD)", []bitStatusDetail{
		{0x00, "Detected"},
		{0x20, "Not Detected"},
	})
	w.drawBitStatusDetail(status, 0x10, color, "TX Data Register Empty", []bitStatusDetail{
		{0x00, "Not Empty"},
		{0x10, "Empty"},
	})
	w.drawBitStatusDetail(status, 0x08, color, "RX Data Register Full", []bitStatusDetail{
		{0x00, "Not Full"},
		{0x08, "Full"},
	})
	w.drawBitStatusDetail(status, 0x04, color, "Overrun", []bitStatusDetail{
		{0x00, "No Overrun"},
		{0x04, "Overrun"},
	})
	w.drawBitStatusDetail(status, 0x02, color, "Framing Error", []bitStatusDetail{
		{0x00, "No Error"},
		{0x02, "Framing Error Detected"},
	})
	w.drawBitStatusDetail(status, 0x01, color, "Parity Error", []bitStatusDetail{
		{0x00, "No Error"},
		{0x01, "Parity Error Detected"},
	})
}

// drawControlRegisterDetails displays the detailed status of each bit in the control register
func (w *AciaWindow) drawControlRegisterDetails() {
	const color = "green"
	control := w.acia.GetControlRegister()

	fmt.Fprintf(w.text, "\n[%s]Control Register Details:\n", color)
	w.drawBitStatusDetail(control, 0x80, color, "Stop Bit Number (SBN)", []bitStatusDetail{
		{0x00, "1 Stop bit"},
		{0x80, "2 Stop bits / 1.5 when WL = 5"},
	})
	w.drawBitStatusDetail(control, 0x60, color, "Word Length (WL)", []bitStatusDetail{
		{0x00, "8"},
		{0x20, "7"},
		{0x40, "6"},
		{0x60, "5"},
	})
	// Not Implemented
	w.drawBitStatusDetail(control, 0x10, color, "RX Clock Source (RCS)", []bitStatusDetail{
		{0x00, "Baud Rate"},
		{0x10, "Baud Rate"},
	})
	w.drawBitStatusDetail(control, 0x0F, color, "Selected Baud Rate (SBR)", []bitStatusDetail{
		{0x00, "115200"},
		{0x01, "50"},
		{0x02, "75"},
		{0x03, "109.92"},
		{0x04, "134.58"},
		{0x05, "150"},
		{0x06, "300"},
		{0x07, "600"},
		{0x08, "1200"},
		{0x09, "1800"},
		{0x0A, "2400"},
		{0x0B, "3600"},
		{0x0C, "4800"},
		{0x0D, "7200"},
		{0x0E, "9600"},
		{0x0F, "19200"},
	})
}

// drawCommandRegisterDetails displays the detailed status of each bit in the command register
func (w *AciaWindow) drawCommandRegisterDetails() {
	const color = "orange"
	command := w.acia.GetCommandRegister()

	fmt.Fprintf(w.text, "\n[%s]Command Register Details:\n", color)
	w.drawBitStatusDetail(command, 0xC0, color, "Parity Mode Control (PMC)", []bitStatusDetail{
		{0x00, "00 - No Parity"},
		{0x40, "01 - No Parity"},
		{0x80, "10 - No Parity"},
		{0xC0, "11 - No Parity"},
	})
	w.drawBitStatusDetail(command, 0x20, color, "Parity Mode Enabled (PME)", []bitStatusDetail{
		{0x00, "0 - No Parity"},
		{0x20, "1 - No Parity"},
	})
	w.drawBitStatusDetail(command, 0x10, color, "Receiver Echo Mode (REM)", []bitStatusDetail{
		{0x00, "0 - Normal"},
		{0x10, "1 - Enabled"},
	})
	w.drawBitStatusDetail(command, 0x0C, color, "TX Interrupt Control (TIC)", []bitStatusDetail{
		{0x00, "00 - No Parity"},
		{0x04, "01 - Do Not Use"},
		{0x08, "10 - TX Interrupt disabled"},
		{0x0C, "11 - TX Interrupt disabled, Transmit Break"},
	})
	w.drawBitStatusDetail(command, 0x02, color, "RX Interrupt Disabled (IRD)", []bitStatusDetail{
		{0x00, "Enabled"},
		{0x02, "Disabled"},
	})
	w.drawBitStatusDetail(command, 0x01, color, "Data Terminal Ready (DTR)", []bitStatusDetail{
		{0x00, "Not Ready"},
		{0x01, "Ready"},
	})
}

// Draw updates the ACIA window display with current register values and their interpretations
// The context parameter provides the current step information for the update
func (w *AciaWindow) Draw(context *common.StepContext) {
	w.text.Clear()

	// Header with timestamp
	fmt.Fprintf(w.text, "[yellow::b]ACIA Status at Step\n")
	fmt.Fprintf(w.text, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// Register values in compact format with binary representation
	status := w.acia.GetStatusRegister()
	control := w.acia.GetControlRegister()
	command := w.acia.GetCommandRegister()
	tx := w.acia.GetTXRegister()
	rx := w.acia.GetRXRegister()

	fmt.Fprintf(w.text, "[yellow]Register Summary:[-]\n")
	fmt.Fprintf(w.text, "┌─────────────┬────────┬──────────┐\n")
	fmt.Fprintf(w.text, "│ Status      │ $%02X    │ %08b │\n", status, status)
	fmt.Fprintf(w.text, "│ Control     │ $%02X    │ %08b │\n", control, control)
	fmt.Fprintf(w.text, "│ Command     │ $%02X    │ %08b │\n", command, command)
	fmt.Fprintf(w.text, "│ TX Data     │ $%02X    │ %08b │\n", tx, tx)
	fmt.Fprintf(w.text, "│ RX Data     │ $%02X    │ %08b │\n", rx, rx)
	fmt.Fprintf(w.text, "└─────────────┴────────┴──────────┘\n")

	// Status indicators with colored symbols
	fmt.Fprintf(w.text, "\n[::b]Quick Status:[-]\n")
	irqStatus := "[red]●[-] IRQ Inactive"
	if status&0x80 != 0 {
		irqStatus = "[green]●[-] IRQ Active"
	}
	txStatus := "[red]●[-] TX Buffer Full"
	if status&0x10 != 0 {
		txStatus = "[green]●[-] TX Buffer Empty"
	}
	rxStatus := "[red]●[-] RX Buffer Empty"
	if status&0x08 != 0 {
		rxStatus = "[green]●[-] RX Buffer Full"
	}

	fmt.Fprintf(w.text, "%s   %s   %s\n\n", irqStatus, txStatus, rxStatus)

	// Detailed register information with improved formatting
	w.drawStatusRegisterDetails()
	fmt.Fprintf(w.text, "\n")
	w.drawControlRegisterDetails()
	fmt.Fprintf(w.text, "\n")
	w.drawCommandRegisterDetails()
}

// GetDrawArea returns the underlying primitive used for rendering the ACIA window.
func (d *AciaWindow) GetDrawArea() tview.Primitive {
	return d.text
}
