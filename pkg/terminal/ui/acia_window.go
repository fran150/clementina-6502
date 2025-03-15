package ui

import (
	"fmt"

	"github.com/fran150/clementina6502/pkg/common"
	"github.com/fran150/clementina6502/pkg/components"
	"github.com/rivo/tview"
)

type AciaWindow struct {
	text *tview.TextView
	acia components.Acia6522Chip
}

func NewAciaWindow(acia components.Acia6522Chip) *AciaWindow {
	text := tview.NewTextView()
	text.SetScrollable(false).
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle("VIA Registers")

	return &AciaWindow{
		text: text,
		acia: acia,
	}
}

func (w *AciaWindow) Clear() {
	w.text.Clear()
}

type bitStautsDetail struct {
	value uint8
	name  string
}

func (w *AciaWindow) drawBitStatusDetail(value uint8, mask uint8, color string, title string, details []bitStautsDetail) {

	fmt.Fprintf(w.text, "[%s]%s: ", color, title)
	value &= mask

	for _, detail := range details {
		if value == detail.value {
			fmt.Fprintf(w.text, "[white]%s\n", detail.name)
		}
	}
}

func (w *AciaWindow) drawStatusRegisterDetails() {
	const color = "blue"
	status := w.acia.GetStatusRegister()

	fmt.Fprintf(w.text, "\n[%s]Status Register Details:\n", color)
	w.drawBitStatusDetail(status, 0x80, color, "Interrupt (IRQ)", []bitStautsDetail{
		{0x00, "No Interrupt"},
		{0x80, "Interrupt occurred"},
	})
	w.drawBitStatusDetail(status, 0x40, color, "Data Set Ready (DSR)", []bitStautsDetail{
		{0x00, "Ready"},
		{0x40, "Not Ready"},
	})
	w.drawBitStatusDetail(status, 0x20, color, "Data Carrier Detect (DCD)", []bitStautsDetail{
		{0x00, "Detected"},
		{0x20, "Not Detected"},
	})
	w.drawBitStatusDetail(status, 0x10, color, "TX Data Register Empty", []bitStautsDetail{
		{0x00, "Not Empty"},
		{0x10, "Empty"},
	})
	w.drawBitStatusDetail(status, 0x08, color, "RX Data Register Full", []bitStautsDetail{
		{0x00, "Not Full"},
		{0x08, "Full"},
	})
	w.drawBitStatusDetail(status, 0x04, color, "Overrun", []bitStautsDetail{
		{0x00, "No Overrun"},
		{0x04, "Overrun"},
	})
	w.drawBitStatusDetail(status, 0x02, color, "Framing Error", []bitStautsDetail{
		{0x00, "No Error"},
		{0x02, "Framing Error Detected"},
	})
	w.drawBitStatusDetail(status, 0x01, color, "Parity Error", []bitStautsDetail{
		{0x00, "No Error"},
		{0x01, "Parity Error Detected"},
	})
}

func (w *AciaWindow) drawControlRegisterDetails() {
	const color = "green"
	control := w.acia.GetControlRegister()

	fmt.Fprintf(w.text, "\n[%s]Control Register Details:\n", color)
	w.drawBitStatusDetail(control, 0x80, color, "Stop Bit Number (SBN)", []bitStautsDetail{
		{0x00, "1 Stop bit"},
		{0x80, "2 Stop bits / 1.5 when WL = 5"},
	})
	w.drawBitStatusDetail(control, 0x60, color, "Word Length (WL)", []bitStautsDetail{
		{0x00, "8"},
		{0x20, "7"},
		{0x40, "6"},
		{0x60, "5"},
	})
	// Not Implemented
	w.drawBitStatusDetail(control, 0x10, color, "RX Clock Source (RCS)", []bitStautsDetail{
		{0x00, "Baud Rate"},
		{0x10, "Baud Rate"},
	})
	w.drawBitStatusDetail(control, 0x0F, color, "Selected Baud Rate (SBR)", []bitStautsDetail{
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

func (w *AciaWindow) drawCommandRegisterDetails() {
	const color = "orange"
	command := w.acia.GetCommandRegister()

	fmt.Fprintf(w.text, "\n[%s]Command Register Details:\n", color)
	w.drawBitStatusDetail(command, 0xC0, color, "Parity Mode Control (PMC)", []bitStautsDetail{
		{0x00, "00 - No Parity"},
		{0x40, "01 - No Parity"},
		{0x80, "10 - No Parity"},
		{0xC0, "11 - No Parity"},
	})
	w.drawBitStatusDetail(command, 0x20, color, "Parity Mode Enabled (PME)", []bitStautsDetail{
		{0x00, "0 - No Parity"},
		{0x20, "1 - No Parity"},
	})
	w.drawBitStatusDetail(command, 0x10, color, "Receiver Echo Mode (REM)", []bitStautsDetail{
		{0x00, "0 - Normal"},
		{0x10, "1 - Enabled"},
	})
	w.drawBitStatusDetail(command, 0x0C, color, "TX Interrupt Control (TIC)", []bitStautsDetail{
		{0x00, "00 - No Parity"},
		{0x04, "01 - Do Not Use"},
		{0x08, "10 - TX Interrupt disabled"},
		{0x0C, "11 - TX Interrupt disabled, Transmit Break"},
	})
	w.drawBitStatusDetail(command, 0x02, color, "RX Interrupt Disabled (IRD)", []bitStautsDetail{
		{0x00, "Enabled"},
		{0x02, "Disabled"},
	})
	w.drawBitStatusDetail(command, 0x01, color, "Data Terminal Ready (DTR)", []bitStautsDetail{
		{0x00, "Not Ready"},
		{0x01, "Ready"},
	})
}

func (w *AciaWindow) Draw(context *common.StepContext) {
	fmt.Fprintf(w.text, "[yellow] ACIA Registers:\n")
	fmt.Fprintf(w.text, "[yellow] Status Reg:   [white]$%02X\n", w.acia.GetStatusRegister())
	fmt.Fprintf(w.text, "[yellow] Control Reg:  [white]$%02X\n", w.acia.GetControlRegister())
	fmt.Fprintf(w.text, "[yellow] Command Reg:  [white]$%02X\n", w.acia.GetCommandRegister())
	fmt.Fprintf(w.text, "[yellow] TX Reg:       [white]$%02X\n", w.acia.GetTXRegister())
	fmt.Fprintf(w.text, "[yellow] RX Reg:       [white]$%02X\n", w.acia.GetRXRegister())

	w.drawStatusRegisterDetails()
	w.drawControlRegisterDetails()
	w.drawCommandRegisterDetails()
}

func (d *AciaWindow) GetDrawArea() tview.Primitive {
	return d.text
}
