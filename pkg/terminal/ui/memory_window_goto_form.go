package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/rivo/tview"
)

// MemoryWindowGoToForm represents a form dialog for navigating to a specific memory address.
// It provides an input field for entering hexadecimal addresses and handles validation.
type MemoryWindowGoToForm struct {
	grid *tview.Grid
	form *tview.Form

	selectedMemoryWindow *MemoryWindow
	size                 int
	onSelect             func()
}

// NewMemoryWindowGoToForm creates and initializes a new memory window goto form.
// It sets up the UI components for navigating to a specific memory address within
// a memory window
//
// Returns:
//   - A pointer to the initialized MemoryWindowGoToForm
func NewMemoryWindowGoToForm() *MemoryWindowGoToForm {
	gotoForm := &MemoryWindowGoToForm{}

	form := tview.NewForm().
		AddInputField("Address", "", 32, gotoForm.validateHexInput, nil).
		AddButton("Go To", gotoForm.selectValue).
		SetFocus(0)

	grid := tview.NewGrid().
		SetColumns(0).
		SetRows(7).
		AddItem(form, 0, 0, 1, 1, 0, 0, true)

	gotoForm.grid = grid
	gotoForm.form = form

	return gotoForm
}

// validateHexInput validates that the input contains only valid hexadecimal characters
// and does not exceed the maximum address size for the memory window.
//
// Parameters:
//   - textToCheck: The current text in the input field
//   - lastChar: The last character entered
//
// Returns:
//   - true if the input is valid, false otherwise
func (d *MemoryWindowGoToForm) validateHexInput(textToCheck string, lastChar rune) bool {
	const allowedChars string = "0123456789ABCDEFabcdef"

	if len(textToCheck) > d.size {
		return false
	}

	return strings.ContainsRune(allowedChars, lastChar)
}

// InitForm initializes the form with a memory window and callback function.
// It sets up the input field width and initial address value.
// The specified memory window will be updated when the new address is selected.
//
// Parameters:
//   - memoryWindow: The memory window to navigate
//   - onSelect: Callback function to execute when an address is selected
func (d *MemoryWindowGoToForm) InitForm(memoryWindow *MemoryWindow, onSelect func()) {
	d.selectedMemoryWindow = memoryWindow
	d.size = len(fmt.Sprintf("%X", memoryWindow.Size()-1))

	input := d.form.GetFormItemByLabel("Address").(*tview.InputField)
	input.SetFieldWidth(d.size + 1)
	input.SetText(fmt.Sprintf("%X", memoryWindow.GetStartAddress()))

	d.onSelect = onSelect
}

// selectValue processes the entered address and updates the memory window's start position.
// It parses the hexadecimal input and calls the onSelect callback if provided.
func (d *MemoryWindowGoToForm) selectValue() {
	input := d.form.GetFormItemByLabel("Address").(*tview.InputField)
	text := input.GetText()

	value, err := strconv.ParseUint(text, 16, 32)
	if err != nil {
		panic(err)
	}

	// Find the closest value that is multiple of 8
	for value%8 != 0 {
		value--
	}

	d.selectedMemoryWindow.start = uint32(value)

	if d.onSelect != nil {
		d.onSelect()
	}
}

// Draw updates the form display.
// This is a placeholder implementation as the form is static.
//
// Parameters:
//   - context: The current step context
func (d *MemoryWindowGoToForm) Draw(context *common.StepContext) {
}

// Clear resets the form.
// This is a placeholder implementation as clearing is handled elsewhere.
func (d *MemoryWindowGoToForm) Clear() {
}

// GetDrawArea returns the primitive that represents this form in the UI.
// This is used by the layout manager to position and render the form.
//
// Returns:
//   - The tview primitive for this form
func (d *MemoryWindowGoToForm) GetDrawArea() tview.Primitive {
	return d.grid
}
