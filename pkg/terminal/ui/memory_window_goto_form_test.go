package ui

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func TestNewMemoryWindowGoToForm(t *testing.T) {
	form := NewMemoryWindowGoToForm()

	assert.NotNil(t, form.grid)
	assert.NotNil(t, form.form)
	assert.Nil(t, form.selectedMemoryWindow)
	assert.Equal(t, 0, form.size)
	assert.Nil(t, form.onSelect)
}

func TestMemoryWindowGoToForm_validateHexInput(t *testing.T) {
	form := NewMemoryWindowGoToForm()
	form.size = 4 // Set size for testing

	tests := []struct {
		name     string
		text     string
		lastChar rune
		want     bool
	}{
		{"Valid hex digit", "1234", '4', true},
		{"Valid hex letter uppercase", "12AB", 'B', true},
		{"Valid hex letter lowercase", "12af", 'f', true},
		{"Invalid character", "123G", 'G', false},
		{"Text too long", "12345", '5', false},
		{"Special character", "123$", '$', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := form.validateHexInput(tt.text, tt.lastChar)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestMemoryWindowGoToForm_InitForm(t *testing.T) {
	form := NewMemoryWindowGoToForm()
	memory := NewMockMemoryChip(0x10000) // 64KB memory
	memoryWindow := NewMemoryWindow(memory)
	memoryWindow.SetStartAddress(0x1234)

	callbackCalled := false
	onSelect := func() {
		callbackCalled = true
	}

	form.InitForm(memoryWindow, onSelect)

	assert.Equal(t, memoryWindow, form.selectedMemoryWindow)
	assert.Equal(t, 4, form.size) // len(fmt.Sprintf("%X", 0xFFFF)) = 4
	assert.NotNil(t, form.onSelect)

	// Test callback
	form.onSelect()
	assert.True(t, callbackCalled)

	// Check input field setup
	input := form.form.GetFormItemByLabel("Address").(*tview.InputField)
	assert.Equal(t, "1234", input.GetText())
}

func TestMemoryWindowGoToForm_selectValue(t *testing.T) {
	form := NewMemoryWindowGoToForm()
	memory := NewMockMemoryChip(0x10000)
	memoryWindow := NewMemoryWindow(memory)

	callbackCalled := false
	onSelect := func() {
		callbackCalled = true
	}

	form.InitForm(memoryWindow, onSelect)
	input := form.form.GetFormItemByLabel("Address").(*tview.InputField)

	tests := []struct {
		name         string
		inputValue   string
		expectedAddr uint32
		shouldPanic  bool
	}{
		{"Valid hex address", "ABCD", 0xABC8, false},
		{"Lowercase hex", "abcd", 0xABC8, false},
		{"Zero address", "0000", 0x0000, false},
		{"Address multiple of 8", "0010", 0x0010, false},
		{"Invalid hex", "ZZZZ", 0x0000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input.SetText(tt.inputValue)
			callbackCalled = false

			if tt.shouldPanic {
				assert.Panics(t, func() {
					form.selectValue()
				}, "Expected panic for invalid hex value")
				return
			}

			form.selectValue()

			assert.Equal(t, tt.expectedAddr, memoryWindow.start)
			assert.True(t, callbackCalled)
		})
	}
}

func TestMemoryWindowGoToForm_selectValue_NoCallback(t *testing.T) {
	form := NewMemoryWindowGoToForm()
	memory := NewMockMemoryChip(0x10000)
	memoryWindow := NewMemoryWindow(memory)

	form.selectedMemoryWindow = memoryWindow
	form.onSelect = nil

	input := form.form.GetFormItemByLabel("Address").(*tview.InputField)
	input.SetText("1234")

	// Should not panic when onSelect is nil
	assert.NotPanics(t, func() {
		form.selectValue()
	})

	assert.Equal(t, uint32(0x1230), memoryWindow.start)
}

func TestMemoryWindowGoToForm_Draw(t *testing.T) {
	form := NewMemoryWindowGoToForm()
	context := &common.StepContext{}

	// Should not panic
	assert.NotPanics(t, func() {
		form.Draw(context)
	})
}

func TestMemoryWindowGoToForm_Clear(t *testing.T) {
	form := NewMemoryWindowGoToForm()

	// Should not panic
	assert.NotPanics(t, func() {
		form.Clear()
	})
}

func TestMemoryWindowGoToForm_GetDrawArea(t *testing.T) {
	form := NewMemoryWindowGoToForm()

	drawArea := form.GetDrawArea()
	assert.NotNil(t, drawArea)
	assert.Equal(t, form.grid, drawArea)
}
