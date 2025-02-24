package ui

import (
	"fmt"
	"strings"

	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/rivo/tview"
)

type BreakPointForm struct {
	grid *tview.Grid
	form *tview.Form
	list *tview.List
}

func NewBreakPointForm() *BreakPointForm {
	breakPointForm := &BreakPointForm{}

	form := tview.NewForm().
		AddInputField("Address", "", 5, validateHexInput, nil).
		AddButton("Add", breakPointForm.addBreakpointAddress).
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

func (d *BreakPointForm) RemoveSelectedItem(context *common.StepContext) {
	current := d.list.GetCurrentItem()
	d.list.RemoveItem(current)
}

func (d *BreakPointForm) addBreakpointAddress() {
	input := d.form.GetFormItemByLabel("Address").(*tview.InputField)
	text := input.GetText()
	text = strings.ToUpper(text)

	text = fmt.Sprintf("$%04s", text)

	d.list.AddItem(text, "", ' ', nil)

	input.SetText("")
}

func validateHexInput(textToCheck string, lastChar rune) bool {
	const allowedChars string = "0123456789ABCDEFabcdef"

	if len(textToCheck) >= 5 {
		return false
	}

	return strings.ContainsRune(allowedChars, lastChar)
}

func (d *BreakPointForm) GetDrawArea() *tview.Grid {
	return d.grid
}
