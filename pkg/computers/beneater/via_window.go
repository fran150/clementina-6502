package beneater

import (
	"fmt"

	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/rivo/tview"
)

type viaWindow struct {
	text     *tview.TextView
	computer *BenEaterComputer
}

func createViaWindow(computer *BenEaterComputer) *viaWindow {
	text := tview.NewTextView()
	text.SetScrollable(false).
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle("VIA Registers")

	return &viaWindow{
		computer: computer,
		text:     text,
	}
}

func (d *viaWindow) Clear() {
	d.text.Clear()
}

func (d *viaWindow) Draw(context *common.StepContext) {
	via := d.computer.chips.via

	fmt.Fprintf(d.text, "[yellow]VIA Registers:\n")
	fmt.Fprintf(d.text, "[yellow] ORA:  [white]$%02X\n", via.GetOutputRegisterA())
	fmt.Fprintf(d.text, "[yellow] ORB:  [white]$%02X\n", via.GetOutputRegisterB())
	fmt.Fprintf(d.text, "[yellow] IRA:  [white]$%02X\n", via.GetInputRegisterA())
	fmt.Fprintf(d.text, "[yellow] IRB:  [white]$%02X\n", via.GetInputRegisterB())
	fmt.Fprintf(d.text, "[yellow] DDRA: [white]$%02X\n", via.GetDataDirectionRegisterA())
	fmt.Fprintf(d.text, "[yellow] DDRB: [white]$%02X\n", via.GetDataDirectionRegisterB())
	fmt.Fprintf(d.text, "[yellow] LL1:  [white]$%02X\n", via.GetLowLatches1())
	fmt.Fprintf(d.text, "[yellow] HL1:  [white]$%02X\n", via.GetHighLatches1())
	fmt.Fprintf(d.text, "[yellow] CTR1: [white]$%04X\n", via.GetCounter1())
	fmt.Fprintf(d.text, "[yellow] LL2:  [white]$%02X\n", via.GetLowLatches2())
	fmt.Fprintf(d.text, "[yellow] HL2:  [white]$%02X\n", via.GetHighLatches2())
	fmt.Fprintf(d.text, "[yellow] CTR2: [white]$%04X\n", via.GetCounter2())
	fmt.Fprintf(d.text, "[yellow] SR:   [white]$%02X\n", via.GetShiftRegister())
	fmt.Fprintf(d.text, "[yellow] ACR:  [white]$%02X\n", via.GetAuxiliaryControl())
	fmt.Fprintf(d.text, "[yellow] PCR:  [white]$%02X\n", via.GetPeripheralControl())
	fmt.Fprintf(d.text, "[yellow] IFR:  [white]$%02X\n", via.GetInterruptFlagValue())
	fmt.Fprintf(d.text, "[yellow] IER:  [white]$%02X\n", via.GetInterruptEnabledFlag())
	fmt.Fprintf(d.text, "[yellow] Bus:  [white]$%04X\n", via.DataBus().Read())
}
