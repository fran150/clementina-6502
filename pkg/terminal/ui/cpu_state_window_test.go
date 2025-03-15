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

	// Verify the content contains all expected values
	assert.Contains(t, content, "A:    66 ($42)")
	assert.Contains(t, content, "X:    36 ($24)")
	assert.Contains(t, content, "Y:    18 ($12)")
	assert.Contains(t, content, "SP:   255 ($FF)")
	assert.Contains(t, content, "PC: $1234 (4660)")

	// Verify flags are present
	assert.Contains(t, content, "Flags:")

	// Count color markers to verify correct flag rendering
	greenCount := strings.Count(window.text.GetText(false), "[green]")
	redCount := strings.Count(window.text.GetText(false), "[red]")

	assert.Equal(t, 4, greenCount) // For bits that are 1
	assert.Equal(t, 4, redCount)   // For bits that are 0
}

func TestCpuWindow_GetDrawArea(t *testing.T) {
	mockCpu := &mockCpuExtension{}
	window := NewCpuWindow(mockCpu)

	drawArea := window.GetDrawArea()
	assert.NotNil(t, drawArea)
	assert.Equal(t, window.text, drawArea)
}
