package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fran150/clementina-6502/internal/slicesext"
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/rivo/tview"
)

// BreakPointForm represents a UI component for managing breakpoints in the debugger.
// It provides a form for adding new breakpoints and a list for displaying and removing
// existing breakpoints.
type BreakPointForm struct {
	grid *tview.Grid
	form *tview.Form
	list *tview.List

	breakpointAddresses []uint16
}

// NewBreakPointForm creates and initializes a new breakpoint management form.
// It sets up the UI components for adding, displaying, and removing breakpoints.
//
// Returns:
//   - A pointer to the initialized BreakPointForm
func NewBreakPointForm() *BreakPointForm {
	breakPointForm := &BreakPointForm{}

	form := tview.NewForm().
		AddInputField("Address", "", 5, breakPointForm.validateHexInput, nil).
		AddButton("Add", breakPointForm.AddSelectedBreakpointAddress).
		SetFocus(0)

	form.SetBorder(true).SetTitle("Add a new breakpoint")

	list := tview.NewList()

	list.SetBorder(true).SetTitle("Active Breakpoints")

	grid := tview.NewGrid().
		SetColumns(0).
		SetRows(7, 0).
		AddItem(form, 0, 0, 1, 1, 0, 0, true).
		AddItem(list, 1, 0, 1, 1, 0, 0, false)

	breakPointForm.grid = grid
	breakPointForm.form = form
	breakPointForm.list = list

	return breakPointForm
}

// RemoveSelectedItem removes the currently selected breakpoint from the list.
// If the list is empty, this method has no effect.
//
// Parameters:
//   - context: The current step context
func (d *BreakPointForm) RemoveSelectedItem(context *common.StepContext) {
	if d.list.GetItemCount() == 0 {
		return
	}

	current := d.list.GetCurrentItem()

	d.RemoveBreakpointAddress(current)
}

// RemoveBreakpointAddress removes a breakpoint at the specified index from the list.
//
// Parameters:
//   - index: The index of the breakpoint to remove
func (d *BreakPointForm) RemoveBreakpointAddress(index int) {
	d.breakpointAddresses = slicesext.SliceRemove(d.breakpointAddresses, index)

	d.list.RemoveItem(index)
}

// CheckBreakpoint checks if a breakpoint exists at the specified address.
//
// Parameters:
//   - address: The address to check
//
// Returns:
//   - true if a breakpoint exists at the address, false otherwise
func (d *BreakPointForm) CheckBreakpoint(address uint16) bool {
	for _, value := range d.breakpointAddresses {
		if value == address {
			return true
		}
	}

	return false
}

// AddBreakpointAddress adds a new breakpoint at the specified hexadecimal address.
// The address is converted to uppercase and displayed with a "$" prefix.
//
// Parameters:
//   - text: The hexadecimal address as a string
func (d *BreakPointForm) AddBreakpointAddress(text string) {
	text = strings.ToUpper(text)

	value, err := strconv.ParseUint(text, 16, 16)
	if err != nil {
		panic(err)
	}

	d.breakpointAddresses = append(d.breakpointAddresses, uint16(value))

	text = fmt.Sprintf("$%04s", text)

	d.list.AddItem(text, "", ' ', nil)
}

// AddSelectedBreakpointAddress adds the address currently entered in the form
// as a new breakpoint.
func (d *BreakPointForm) AddSelectedBreakpointAddress() {
	input := d.form.GetFormItemByLabel("Address").(*tview.InputField)
	text := input.GetText()

	d.AddBreakpointAddress(text)

	input.SetText("")
}

func (d *BreakPointForm) validateHexInput(textToCheck string, lastChar rune) bool {
	const allowedChars string = "0123456789ABCDEFabcdef"

	if len(textToCheck) >= 5 {
		return false
	}

	return strings.ContainsRune(allowedChars, lastChar)
}

// Draw updates the breakpoint form display.
// This is a placeholder implementation as the form is static.
//
// Parameters:
//   - context: The current step context
func (d *BreakPointForm) Draw(context *common.StepContext) {
}

// Clear resets the breakpoint form.
// This is a placeholder implementation as clearing is handled elsewhere.
func (d *BreakPointForm) Clear() {
}

// GetDrawArea returns the primitive that represents this form in the UI.
// This is used by the layout manager to position and render the form.
//
// Returns:
//   - The tview primitive for this form
func (d *BreakPointForm) GetDrawArea() tview.Primitive {
	return d.grid
}
