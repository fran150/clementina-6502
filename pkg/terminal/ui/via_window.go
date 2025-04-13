package ui

import (
	"fmt"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/rivo/tview"
)

// ViaWindow represents a UI component that displays the VIA (Versatile Interface Adapter) state.
// It shows the current register values, timer status, and I/O port configurations.
type ViaWindow struct {
	text *tview.TextView
	via  components.ViaChip
}

// NewViaWindow creates a new VIA state display window.
// It initializes the UI component and connects it to the provided VIA chip.
//
// Parameters:
//   - via: The VIA chip to monitor and display
//
// Returns:
//   - A pointer to the initialized ViaWindow
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
	fmt.Fprintf(d.text, "[white]╔══════════════════════════════════╗\n")
	// Port A Section
	fmt.Fprintf(d.text, "[white]║[yellow] Port A                           [white]║\n")
	fmt.Fprintf(d.text, "[white]║ [green]ORA:[-]  $%02X   [green]DDRA:[-] $%02X            ║\n",
		d.via.GetOutputRegisterA(), d.via.GetDataDirectionRegisterA())
	fmt.Fprintf(d.text, "[white]║ [green]IRA:[-]  $%02X                        ║\n",
		d.via.GetInputRegisterA())

	// Port B Section
	fmt.Fprintf(d.text, "[white]╟──────────────────────────────────╢\n")
	fmt.Fprintf(d.text, "[white]║[yellow] Port B                           [white]║\n")
	fmt.Fprintf(d.text, "[white]║ [blue]ORB:[-]  $%02X   [blue]DDRB:[-] $%02X            ║\n",
		d.via.GetOutputRegisterB(), d.via.GetDataDirectionRegisterB())
	fmt.Fprintf(d.text, "[white]║ [blue]IRB:[-]  $%02X                        ║\n",
		d.via.GetInputRegisterB())

	// Timer 1 Section
	fmt.Fprintf(d.text, "[white]╟──────────────────────────────────╢\n")
	fmt.Fprintf(d.text, "[white]║[yellow] Timer 1                          [white]║\n")
	fmt.Fprintf(d.text, "[white]║ [blue]Latches:[-] $%02X/$%02X  [blue]Counter:[-] $%04X ║\n",
		d.via.GetLowLatches1(), d.via.GetHighLatches1(), d.via.GetCounter1())

	// Timer 2 Section
	fmt.Fprintf(d.text, "[white]╟──────────────────────────────────╢\n")
	fmt.Fprintf(d.text, "[white]║[yellow] Timer 2                          [white]║\n")
	fmt.Fprintf(d.text, "[white]║ [green]Latches:[-] $%02X/$%02X  [green]Counter:[-] $%04X ║\n",
		d.via.GetLowLatches2(), d.via.GetHighLatches2(), d.via.GetCounter2())

	// Control Registers Section
	fmt.Fprintf(d.text, "[white]╟──────────────────────────────────╢\n")
	fmt.Fprintf(d.text, "[white]║[yellow] Control Registers                [white]║\n")
	fmt.Fprintf(d.text, "[white]║ [orange]SR:[-]  $%02X   [orange]ACR:[-] $%02X   [orange]PCR:[-] $%02X   ║\n",
		d.via.GetShiftRegister(), d.via.GetAuxiliaryControl(), d.via.GetPeripheralControl())

	// Interrupt Section
	fmt.Fprintf(d.text, "[white]╟──────────────────────────────────╢\n")
	fmt.Fprintf(d.text, "[white]║[yellow] Interrupts                       [white]║\n")
	fmt.Fprintf(d.text, "[white]║ [red]IFR:[-] $%02X   [red]IER:[-] $%02X              ║\n",
		d.via.GetInterruptFlagValue(), d.via.GetInterruptEnabledFlag())

	// Bus Value
	fmt.Fprintf(d.text, "[white]╟──────────────────────────────────╢\n")
	fmt.Fprintf(d.text, "[white]║ Data Bus: $%04X                  ║\n",
		d.via.DataBus().Read())
	fmt.Fprintf(d.text, "[white]╚══════════════════════════════════╝\n")
}

func (d *ViaWindow) GetDrawArea() tview.Primitive {
	return d.text
}
