package ui

import (
	"testing"

	"github.com/fran150/clementina6502/pkg/common"
	"github.com/fran150/clementina6502/pkg/components/buses"
	"github.com/stretchr/testify/assert"
)

// MockMemoryChip implements the MemoryChip interface for testing
type MockMemoryChip struct {
	size         int
	data         []uint8
	addressBus   *buses.BusConnector[uint16]
	dataBus      *buses.BusConnector[uint8]
	writeEnable  *buses.ConnectorEnabledLow
	chipSelect   *buses.ConnectorEnabledLow
	outputEnable *buses.ConnectorEnabledLow
}

func NewMockMemoryChip(size int) *MockMemoryChip {
	return &MockMemoryChip{
		size:         size,
		data:         make([]uint8, size),
		addressBus:   buses.NewBusConnector[uint16](),
		dataBus:      buses.NewBusConnector[uint8](),
		writeEnable:  buses.NewConnectorEnabledLow(),
		chipSelect:   buses.NewConnectorEnabledLow(),
		outputEnable: buses.NewConnectorEnabledLow(),
	}
}

// Bus and control signal connections
func (m *MockMemoryChip) AddressBus() *buses.BusConnector[uint16] {
	return m.addressBus
}

func (m *MockMemoryChip) DataBus() *buses.BusConnector[uint8] {
	return m.dataBus
}

func (m *MockMemoryChip) WriteEnable() *buses.ConnectorEnabledLow {
	return m.writeEnable
}

func (m *MockMemoryChip) ChipSelect() *buses.ConnectorEnabledLow {
	return m.chipSelect
}

func (m *MockMemoryChip) OutputEnable() *buses.ConnectorEnabledLow {
	return m.outputEnable
}

// Utility methods
func (m *MockMemoryChip) Peek(address uint16) uint8 {
	if int(address) >= m.size {
		return 0
	}
	return m.data[address]
}

func (m *MockMemoryChip) PeekRange(startAddress uint16, endAddress uint16) []uint8 {
	if int(startAddress) >= m.size {
		return []uint8{}
	}
	if int(endAddress) >= m.size {
		endAddress = uint16(m.size - 1)
	}
	return m.data[startAddress : endAddress+1]
}

func (m *MockMemoryChip) Poke(address uint16, value uint8) {
	if int(address) < m.size {
		m.data[address] = value
	}
}

func (m *MockMemoryChip) Load(binFilePath string) error {
	// Mock implementation - return nil as we don't need file loading in tests
	return nil
}

func (m *MockMemoryChip) Size() int {
	return m.size
}

// Emulation method
func (m *MockMemoryChip) Tick(context *common.StepContext) {
	// Mock implementation - do nothing
}

func TestNewMemoryWindow(t *testing.T) {
	memory := NewMockMemoryChip(0x1000)
	window := NewMemoryWindow(memory)

	assert.NotNil(t, window)
	assert.Equal(t, "Memory Explorer", window.GetTitle())
	assert.Equal(t, uint16(0), window.GetStartAddress())
}

func TestMemoryWindowSetTitle(t *testing.T) {
	memory := NewMockMemoryChip(0x1000)
	window := NewMemoryWindow(memory)

	window.SetTitle("New Title")
	assert.Equal(t, "New Title", window.GetTitle())
}

func TestMemoryWindowScrollDown(t *testing.T) {
	tests := []struct {
		name     string
		memSize  int
		lines    uint16
		start    uint16
		expected uint16
	}{
		{"normal scroll", 0x0400, 1, 0x0000, 0x0008},
		{"scroll near end", 0x0400, 1, 0x03F8, 0x0400},
		{"scroll past end", 0x0400, 1, 0x0400, 0x03F8},
		{"multiple lines", 0x0400, 2, 0x0000, 0x0010},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memory := NewMockMemoryChip(tt.memSize)
			window := NewMemoryWindow(memory)
			window.start = tt.start

			window.ScrollDown(tt.lines)
			assert.Equal(t, tt.expected, window.GetStartAddress())
		})
	}
}

func TestMemoryWindowScrollUp(t *testing.T) {
	tests := []struct {
		name     string
		memSize  int
		lines    uint16
		start    uint16
		expected uint16
	}{
		{"normal scroll", 0x1000, 1, 0x0010, 0x0008},
		{"scroll near start", 0x1000, 1, 0x0008, 0x0000},
		{"scroll past start", 0x1000, 1, 0x0000, 0x0000},
		{"multiple lines", 0x1000, 2, 0x0020, 0x0010},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memory := NewMockMemoryChip(tt.memSize)
			window := NewMemoryWindow(memory)
			window.start = tt.start

			window.ScrollUp(tt.lines)
			assert.Equal(t, tt.expected, window.GetStartAddress())
		})
	}
}

func TestMemoryWindowDraw(t *testing.T) {
	memory := NewMockMemoryChip(0x400)
	window := NewMemoryWindow(memory)

	// Set some test values in memory
	for i := uint16(0); i < 16; i++ {
		memory.Poke(i, uint8(i))
	}

	context := &common.StepContext{}

	// Test initial draw
	window.Clear()
	window.Draw(context)

	// Get the text content
	content := window.text.GetText(true)

	// Verify the content contains the expected memory values
	assert.Contains(t, content, "0000: 00 01 02 03 04 05 06 07")

	// Test scrolled draw
	window.ScrollDown(1)
	window.Clear()
	window.Draw(context)

	content = window.text.GetText(true)
	assert.Contains(t, content, "0008: 08 09 0A 0B 0C 0D 0E 0F")

	window.start = 0x03FA
	window.Clear()
	window.Draw(context)

	content = window.text.GetText(true)
	assert.Contains(t, content, "03FA: 00 00 00 00 00 00")
}

func TestMemoryWindowGetDrawArea(t *testing.T) {
	memory := NewMockMemoryChip(0x1000)
	window := NewMemoryWindow(memory)

	assert.NotNil(t, window.GetDrawArea())
	assert.Equal(t, window.text, window.GetDrawArea())
}
