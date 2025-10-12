package ui

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/core/interfaces"
	"github.com/fran150/clementina-6502/pkg/core/managers"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func TestNewBreakPointForm(t *testing.T) {
	var bm interfaces.BreakpointManager = managers.NewBreakpointManager()
	form := NewBreakPointForm(bm)

	assert.NotNil(t, form.grid)
	assert.NotNil(t, form.form)
	assert.NotNil(t, form.list)
	assert.NotNil(t, form.breakpointManager)
	assert.Equal(t, 0, form.breakpointManager.GetBreakpointCount())
}

func TestValidateHexInput(t *testing.T) {
	var bm interfaces.BreakpointManager = managers.NewBreakpointManager()
	form := NewBreakPointForm(bm)

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
			result := form.validateHexInput(tt.text, tt.lastChar)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestCheckBreakpoint(t *testing.T) {
	var bm interfaces.BreakpointManager = managers.NewBreakpointManager()
	form := NewBreakPointForm(bm)
	bm.AddBreakpoint(0x1234)
	bm.AddBreakpoint(0x5678)

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
	var bm interfaces.BreakpointManager = managers.NewBreakpointManager()
	form := NewBreakPointForm(bm)

	// Test empty list case first
	t.Run("Remove from empty list", func(t *testing.T) {
		// List should be empty by default after initialization
		assert.Equal(t, 0, form.list.GetItemCount())
		// This should not panic or modify anything
		form.RemoveSelectedItem()
		// Verify it's still empty
		assert.Equal(t, 0, form.list.GetItemCount())
		assert.Equal(t, 0, form.breakpointManager.GetBreakpointCount())
	})

	// Setup for the rest of the tests
	bm.AddBreakpoint(0x1234)
	bm.AddBreakpoint(0x5678)
	bm.AddBreakpoint(0x9ABC)

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
			form.RemoveSelectedItem()
			assert.Equal(t, tt.expectedLen, form.breakpointManager.GetBreakpointCount())
			assert.Equal(t, tt.expectedLen, form.list.GetItemCount())
		})
	}
}

func TestAddBreakpointAddress(t *testing.T) {
	var bm interfaces.BreakpointManager = managers.NewBreakpointManager()
	form := NewBreakPointForm(bm)
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
					form.AddSelectedBreakpointAddress()
				}, "Expected panic for invalid hex value")
				return
			}

			initialCount := form.breakpointManager.GetBreakpointCount()
			form.AddSelectedBreakpointAddress()

			// Check if breakpoint was added to the manager
			assert.Equal(t, initialCount+1, form.breakpointManager.GetBreakpointCount())
			assert.True(t, form.breakpointManager.HasBreakpoint(tt.expectedValue))

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
	var bm interfaces.BreakpointManager = managers.NewBreakpointManager()
	form := NewBreakPointForm(bm)
	context := &common.StepContext{}

	// We just verify it doesn't panic
	form.Draw(context)
	form.Clear()
	grid := form.GetDrawArea()

	assert.Equal(t, form.grid, grid)
}
