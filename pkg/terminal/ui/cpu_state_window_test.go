package ui

import (
	"strings"
	"testing"

	"github.com/fran150/clementina6502/pkg/common"
	"github.com/fran150/clementina6502/pkg/components/cpu"
	"github.com/stretchr/testify/assert"
)

// Add these fields to the existing mockCpu struct in code_window_test.go
type mockCpuExtension struct {
	mockCpu
	accumulator             uint8
	xRegister               uint8
	yRegister               uint8
	stackPointer            uint8
	processorStatusRegister cpu.StatusRegister
}

// Override the methods needed for CPU state window
func (m *mockCpuExtension) GetAccumulatorRegister() uint8 { return m.accumulator }
func (m *mockCpuExtension) GetXRegister() uint8           { return m.xRegister }
func (m *mockCpuExtension) GetYRegister() uint8           { return m.yRegister }
func (m *mockCpuExtension) GetStackPointer() uint8        { return m.stackPointer }
func (m *mockCpuExtension) GetProcessorStatusRegister() cpu.StatusRegister {
	return m.processorStatusRegister
}

func TestNewCpuWindow(t *testing.T) {
	mockCpu := &mockCpuExtension{}
	window := NewCpuWindow(mockCpu)

	assert.NotNil(t, window)
	assert.NotNil(t, window.text)
	assert.NotNil(t, window.processor)
}

func TestGetFlagStatusColor(t *testing.T) {
	tests := []struct {
		name     string
		status   cpu.StatusRegister
		bit      cpu.StatusBit
		expected string
	}{
		{
			name:     "Flag set",
			status:   cpu.StatusRegister(0xFF), // All bits set
			bit:      cpu.ZeroFlagBit,
			expected: "[green]",
		},
		{
			name:     "Flag not set",
			status:   cpu.StatusRegister(0x00), // All bits clear
			bit:      cpu.ZeroFlagBit,
			expected: "[red]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFlagStatusColor(tt.status, tt.bit)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCpuWindow_Draw(t *testing.T) {
	mockCpu := &mockCpuExtension{
		mockCpu: mockCpu{
			programCounter: 0x1234,
		},
		accumulator:             0x42,
		xRegister:               0x24,
		yRegister:               0x12,
		stackPointer:            0xFF,
		processorStatusRegister: cpu.StatusRegister(0b10101010), // Alternating bits
	}

	window := NewCpuWindow(mockCpu)
	context := &common.StepContext{}

	// Clear and draw
	window.Clear()
	window.Draw(context)

	// Get the text content
	content := window.text.GetText(true)

	// Verify the content contains all expected values with the new format
	assert.Contains(t, content, "A :    66 ($42)")
	assert.Contains(t, content, "X :    36 ($24)")
	assert.Contains(t, content, "Y :    18 ($12)")
	assert.Contains(t, content, "SP:   255 ($FF)")
	assert.Contains(t, content, "PC: $1234 ( 4660)")
	assert.Contains(t, content, "Status Flags:")

	// Verify the box drawing characters are present
	assert.Contains(t, content, "┌────────────────────────┐")
	assert.Contains(t, content, "└────────────────────────┘")

	// Count color markers to verify correct flag rendering
	text := window.text.GetText(false)
	greenCount := strings.Count(text, "[green]")
	redCount := strings.Count(text, "[red]")

	// Since the status register is 0b10101010, we should have 4 green and 4 red flags
	assert.Equal(t, 4, greenCount, "Should have 4 green (set) flags")
	assert.Equal(t, 4, redCount, "Should have 4 red (unset) flags")

	// Verify color usage
	assert.Contains(t, text, "[yellow]")
	assert.Contains(t, text, "[white]")
	assert.Contains(t, text, "[grey]")
}

func TestCpuWindow_GetDrawArea(t *testing.T) {
	mockCpu := &mockCpuExtension{}
	window := NewCpuWindow(mockCpu)

	drawArea := window.GetDrawArea()
	assert.NotNil(t, drawArea)
	assert.Equal(t, window.text, drawArea)
}
