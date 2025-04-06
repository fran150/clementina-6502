package ui

import (
	"strings"
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/stretchr/testify/assert"
)

func TestNewBusWindow(t *testing.T) {
	window := NewBusWindow()
	assert.NotNil(t, window)
	assert.NotNil(t, window.text)
	assert.Empty(t, window.busInfos)
}

func TestAddBus8(t *testing.T) {
	window := NewBusWindow()
	bus := buses.New8BitStandaloneBus()
	bus.Write(0xA5)
	window.AddBus8("Test Bus", bus)

	assert.Len(t, window.busInfos, 1)
	assert.Equal(t, "Test Bus", window.busInfos[0].name)
	assert.Equal(t, 8, window.busInfos[0].bitWidth)
}

func TestAddBus16(t *testing.T) {
	window := NewBusWindow()
	bus := buses.New16BitStandaloneBus()
	bus.Write(0x1234)
	window.AddBus16("Address Bus", bus)

	assert.Len(t, window.busInfos, 1)
	assert.Equal(t, "Address Bus", window.busInfos[0].name)
	assert.Equal(t, 16, window.busInfos[0].bitWidth)
}

func TestDraw(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *BusWindow
		expected []string
	}{
		{
			name: "8-bit bus with value 0xA5",
			setup: func() *BusWindow {
				window := NewBusWindow()
				bus := buses.New8BitStandaloneBus()
				bus.Write(0xA5)
				window.AddBus8("Data Bus", bus)
				return window
			},
			expected: []string{
				"Data Bus: $A5",
				"  7   6   5   4   3   2   1   0",
				"┏━━━┳━━━┳━━━┳━━━┳━━━┳━━━┳━━━┳━━━┓",
				"┃━━━┃───┃━━━┃───┃───┃━━━┃───┃━━━┃",
				"┗━━━┻━━━┻━━━┻━━━┻━━━┻━━━┻━━━┻━━━┛",
			},
		},
		{
			name: "16-bit bus with value 0x1234",
			setup: func() *BusWindow {
				window := NewBusWindow()
				bus := buses.New16BitStandaloneBus()
				bus.Write(0x1234)
				window.AddBus16("Address Bus", bus)
				return window
			},
			expected: []string{
				"Address Bus: $1234",
				" 15  14  13  12  11  10   9   8   7   6   5   4   3   2   1   0",
				"┏━━━┳━━━┳━━━┳━━━┳━━━┳━━━┳━━━┳━━━┳━━━┳━━━┳━━━┳━━━┳━━━┳━━━┳━━━┳━━━┓",
				"┃───┃───┃───┃━━━┃───┃───┃━━━┃───┃───┃───┃━━━┃━━━┃───┃━━━┃───┃───┃",
				"┗━━━┻━━━┻━━━┻━━━┻━━━┻━━━┻━━━┻━━━┻━━━┻━━━┻━━━┻━━━┻━━━┻━━━┻━━━┻━━━┛",
			},
		},
		{
			name: "Multiple buses",
			setup: func() *BusWindow {
				window := NewBusWindow()
				bus8 := buses.New8BitStandaloneBus()
				bus16 := buses.New16BitStandaloneBus()
				bus8.Write(0xFF)
				bus16.Write(0x0000)
				window.AddBus8("Control", bus8)
				window.AddBus16("Address", bus16)
				return window
			},
			expected: []string{
				"Control: $FF",
				"Address: $0000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			window := tt.setup()
			window.Draw(&common.StepContext{})

			// Get the content from the TextView
			content := window.text.GetText(true)
			lines := strings.Split(content, "\n")

			var currentLine string

			// Check that each expected line exists in the content
			for _, expectedLine := range tt.expected {
				found := false
				for _, line := range lines {
					currentLine = line
					if strings.Contains(line, expectedLine) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected line not found: %s, found: %s", expectedLine, currentLine)
			}
		})
	}
}

func TestClear(t *testing.T) {
	window := NewBusWindow()
	bus := buses.New8BitStandaloneBus()
	bus.Write(0xA5)
	window.AddBus8("Test Bus", bus)

	// Draw something
	window.Draw(&common.StepContext{})
	assert.NotEmpty(t, window.text.GetText(true))

	// Clear
	window.Clear()
	assert.Empty(t, window.text.GetText(true))
}

func TestBusGetDrawArea(t *testing.T) {
	window := NewBusWindow()
	assert.NotNil(t, window.GetDrawArea())
	assert.Equal(t, window.text, window.GetDrawArea())
}

func TestDrawBusLine(t *testing.T) {
	tests := []struct {
		name        string
		value       uint16
		bitPosition int
		expected    string
	}{
		{
			name:        "High bit",
			value:       0x01,
			bitPosition: 0,
			expected:    "[green]━━━",
		},
		{
			name:        "Low bit",
			value:       0x00,
			bitPosition: 0,
			expected:    "[red]───",
		},
		{
			name:        "High bit in position 7",
			value:       0x80,
			bitPosition: 7,
			expected:    "[green]━━━",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := drawBusLine(tt.value, tt.bitPosition)
			assert.Equal(t, tt.expected, result)
		})
	}
}
