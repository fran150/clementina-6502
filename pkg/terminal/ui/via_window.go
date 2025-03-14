package ui

import (
	"fmt"

	"github.com/fran150/clementina6502/pkg/components"
	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/rivo/tview"
)

type ViaWindow struct {
	text *tview.TextView
	via  components.ViaChip
}

func NewViaWindow(via components.ViaChip) *ViaWindow {
	text := tview.NewTextView()
	text.SetScrollable(false).
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle("VIA Registers")

	return &ViaWindow{
		text: text,
		via:  via,
	}
}

func (d *ViaWindow) Clear() {
	d.text.Clear()
}

func (d *ViaWindow) Draw(context *common.StepContext) {
	fmt.Fprintf(d.text, "[yellow] VIA Registers:\n")
	fmt.Fprintf(d.text, "[yellow] ORA:  [white]$%02X\n", d.via.GetOutputRegisterA())
	fmt.Fprintf(d.text, "[yellow] ORB:  [white]$%02X\n", d.via.GetOutputRegisterB())
	fmt.Fprintf(d.text, "[yellow] IRA:  [white]$%02X\n", d.via.GetInputRegisterA())
	fmt.Fprintf(d.text, "[yellow] IRB:  [white]$%02X\n", d.via.GetInputRegisterB())
	fmt.Fprintf(d.text, "[yellow] DDRA: [white]$%02X\n", d.via.GetDataDirectionRegisterA())
	fmt.Fprintf(d.text, "[yellow] DDRB: [white]$%02X\n", d.via.GetDataDirectionRegisterB())
	fmt.Fprintf(d.text, "[yellow] LL1:  [white]$%02X\n", d.via.GetLowLatches1())
	fmt.Fprintf(d.text, "[yellow] HL1:  [white]$%02X\n", d.via.GetHighLatches1())
	fmt.Fprintf(d.text, "[yellow] CTR1: [white]$%04X\n", d.via.GetCounter1())
	fmt.Fprintf(d.text, "[yellow] LL2:  [white]$%02X\n", d.via.GetLowLatches2())
	fmt.Fprintf(d.text, "[yellow] HL2:  [white]$%02X\n", d.via.GetHighLatches2())
	fmt.Fprintf(d.text, "[yellow] CTR2: [white]$%04X\n", d.via.GetCounter2())
	fmt.Fprintf(d.text, "[yellow] SR:   [white]$%02X\n", d.via.GetShiftRegister())
	fmt.Fprintf(d.text, "[yellow] ACR:  [white]$%02X\n", d.via.GetAuxiliaryControl())
	fmt.Fprintf(d.text, "[yellow] PCR:  [white]$%02X\n", d.via.GetPeripheralControl())
	fmt.Fprintf(d.text, "[yellow] IFR:  [white]$%02X\n", d.via.GetInterruptFlagValue())
	fmt.Fprintf(d.text, "[yellow] IER:  [white]$%02X\n", d.via.GetInterruptEnabledFlag())
	fmt.Fprintf(d.text, "[yellow] Bus:  [white]$%04X\n", d.via.DataBus().Read())
}

func (d *ViaWindow) GetDrawArea() tview.Primitive {
	return d.text
}
