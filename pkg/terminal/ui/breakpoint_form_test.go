package ui

import (
	"testing"

	"github.com/fran150/clementina6502/pkg/common"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func TestNewBreakPointForm(t *testing.T) {
	form := NewBreakPointForm()

	assert.NotNil(t, form.grid)
	assert.NotNil(t, form.form)
	assert.NotNil(t, form.list)
	assert.Empty(t, form.breakpointAddresses)
}

func TestValidateHexInput(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		lastChar rune
		want     bool
	}{
		{"Valid hex digit", "1234", '4', true},
		{"Valid hex letter uppercase", "12AH", 'B', true},
		{"Valid hex letter lowercase", "12af", 'f', true},
		{"Invalid character", "123G", 'G', false},
		{"Text too long", "FFFF5", '5', false},
		{"Special character", "123$", '$', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateHexInput(tt.text, tt.lastChar)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestCheckBreakpoint(t *testing.T) {
	form := NewBreakPointForm()
	form.breakpointAddresses = []uint16{0x1234, 0x5678}

	tests := []struct {
		name    string
		address uint16
		want    bool
	}{
		{"Existing breakpoint 1", 0x1234, true},
		{"Existing breakpoint 2", 0x5678, true},
		{"Non-existing breakpoint", 0x9ABC, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := form.CheckBreakpoint(tt.address)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestRemoveSelectedItem(t *testing.T) {
	form := NewBreakPointForm()

	// Test empty list case first
	t.Run("Remove from empty list", func(t *testing.T) {
		context := &common.StepContext{}
		// List should be empty by default after initialization
		assert.Equal(t, 0, form.list.GetItemCount())
		// This should not panic or modify anything
		form.RemoveSelectedItem(context)
		// Verify it's still empty
		assert.Equal(t, 0, form.list.GetItemCount())
		assert.Empty(t, form.breakpointAddresses)
	})

	// Setup for the rest of the tests
	form.breakpointAddresses = []uint16{0x1234, 0x5678, 0x9ABC}

	// Add items to the list
	form.list.AddItem("$1234", "", ' ', nil)
	form.list.AddItem("$5678", "", ' ', nil)
	form.list.AddItem("$9ABC", "", ' ', nil)

	tests := []struct {
		name          string
		indexToRemove int
		expectedLen   int
	}{
		{"Remove first item", 0, 2},
		{"Remove from remaining items", 0, 1},
		{"Remove last item", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context := &common.StepContext{}
			form.RemoveSelectedItem(context)
			assert.Equal(t, tt.expectedLen, len(form.breakpointAddresses))
			assert.Equal(t, tt.expectedLen, form.list.GetItemCount())
		})
	}
}

func TestAddBreakpointAddress(t *testing.T) {
	form := NewBreakPointForm()
	input := form.form.GetFormItemByLabel("Address").(*tview.InputField)

	tests := []struct {
		name          string
		inputValue    string
		expectedValue uint16
		expectedText  string
		shouldPanic   bool
	}{
		{"Add simple hex", "1234", 0x1234, "$1234", false},
		{"Add lowercase hex", "abcd", 0xABCD, "$ABCD", false},
		{"Add mixed case hex", "Ef12", 0xEF12, "$EF12", false},
		{"Non parseable value", "ZZZZ", 0x0000, "$0000", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input.SetText(tt.inputValue)

			if tt.shouldPanic {
				assert.Panics(t, func() {
					form.addBreakpointAddress()
				}, "Expected panic for invalid hex value")
				return
			}

			form.addBreakpointAddress()

			// Check if breakpoint was added to the addresses slice
			lastAddr := form.breakpointAddresses[len(form.breakpointAddresses)-1]
			assert.Equal(t, tt.expectedValue, lastAddr)

			// Check if list item was added with correct format
			lastIndex := form.list.GetItemCount() - 1
			text, _ := form.list.GetItemText(lastIndex)
			assert.Equal(t, tt.expectedText, text)

			// Check if input was cleared
			assert.Empty(t, input.GetText())
		})
	}
}

func TestBreakPointForm_Draw(t *testing.T) {
	form := NewBreakPointForm()
	context := &common.StepContext{}

	// We just verify it doesn't panic
	form.Draw(context)
	form.Clear()
	grid := form.GetDrawArea()

	assert.Equal(t, form.grid, grid)
}
