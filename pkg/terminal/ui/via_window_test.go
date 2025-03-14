package ui

import (
	"testing"

	"github.com/fran150/clementina6502/pkg/components/buses"
	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

// MockViaChip implements the ViaChip interface for testing
type MockViaChip struct {
	outputRegisterA        uint8
	outputRegisterB        uint8
	inputRegisterA         uint8
	inputRegisterB         uint8
	dataDirectionRegisterA uint8
	dataDirectionRegisterB uint8
	lowLatches1            uint8
	highLatches1           uint8
	counter1               uint16
	lowLatches2            uint8
	highLatches2           uint8
	counter2               uint16
	shiftRegister          uint8
	auxiliaryControl       uint8
	peripheralControl      uint8
	interruptFlag          uint8
	interruptEnabled       uint8
	dataBusValue           uint8
}

func NewMockViaChip() *MockViaChip {
	return &MockViaChip{
		outputRegisterA:        0x01,
		outputRegisterB:        0x02,
		inputRegisterA:         0x03,
		inputRegisterB:         0x04,
		dataDirectionRegisterA: 0x05,
		dataDirectionRegisterB: 0x06,
		lowLatches1:            0x07,
		highLatches1:           0x08,
		counter1:               0x0910,
		lowLatches2:            0x0A,
		highLatches2:           0x0B,
		counter2:               0x0C0D,
		shiftRegister:          0x0E,
		auxiliaryControl:       0x0F,
		peripheralControl:      0x10,
		interruptFlag:          0x11,
		interruptEnabled:       0x12,
		dataBusValue:           0x13,
	}
}

// Implement necessary methods for the mock
func (m *MockViaChip) GetOutputRegisterA() uint8        { return m.outputRegisterA }
func (m *MockViaChip) GetOutputRegisterB() uint8        { return m.outputRegisterB }
func (m *MockViaChip) GetInputRegisterA() uint8         { return m.inputRegisterA }
func (m *MockViaChip) GetInputRegisterB() uint8         { return m.inputRegisterB }
func (m *MockViaChip) GetDataDirectionRegisterA() uint8 { return m.dataDirectionRegisterA }
func (m *MockViaChip) GetDataDirectionRegisterB() uint8 { return m.dataDirectionRegisterB }
func (m *MockViaChip) GetLowLatches1() uint8            { return m.lowLatches1 }
func (m *MockViaChip) GetHighLatches1() uint8           { return m.highLatches1 }
func (m *MockViaChip) GetCounter1() uint16              { return m.counter1 }
func (m *MockViaChip) GetLowLatches2() uint8            { return m.lowLatches2 }
func (m *MockViaChip) GetHighLatches2() uint8           { return m.highLatches2 }
func (m *MockViaChip) GetCounter2() uint16              { return m.counter2 }
func (m *MockViaChip) GetShiftRegister() uint8          { return m.shiftRegister }
func (m *MockViaChip) GetAuxiliaryControl() uint8       { return m.auxiliaryControl }
func (m *MockViaChip) GetPeripheralControl() uint8      { return m.peripheralControl }
func (m *MockViaChip) GetInterruptFlagValue() uint8     { return m.interruptFlag }
func (m *MockViaChip) GetInterruptEnabledFlag() uint8   { return m.interruptEnabled }

var connector *buses.BusConnector[uint8] = buses.NewBusConnector[uint8]()
var bus buses.Bus[uint8] = buses.New8BitStandaloneBus()

func (m *MockViaChip) DataBus() *buses.BusConnector[uint8] {
	connector.Connect(bus)
	bus.Write(m.dataBusValue)
	return connector
}

// Add these methods to your MockViaChip struct implementation

func (m *MockViaChip) PeripheralAControlLines(num int) *buses.ConnectorEnabledHigh {
	return buses.NewConnectorEnabledHigh()
}

func (m *MockViaChip) PeripheralBControlLines(num int) *buses.ConnectorEnabledHigh {
	return buses.NewConnectorEnabledHigh()
}

func (m *MockViaChip) ChipSelect1() *buses.ConnectorEnabledHigh {
	return buses.NewConnectorEnabledHigh()
}

func (m *MockViaChip) ChipSelect2() *buses.ConnectorEnabledLow {
	return buses.NewConnectorEnabledLow()
}

func (m *MockViaChip) IrqRequest() *buses.ConnectorEnabledLow {
	return buses.NewConnectorEnabledLow()
}

func (m *MockViaChip) PeripheralPortA() *buses.BusConnector[uint8] {
	return buses.NewBusConnector[uint8]()
}

func (m *MockViaChip) PeripheralPortB() *buses.BusConnector[uint8] {
	return buses.NewBusConnector[uint8]()
}

func (m *MockViaChip) Reset() *buses.ConnectorEnabledLow {
	return buses.NewConnectorEnabledLow()
}

func (m *MockViaChip) RegisterSelect(num uint8) *buses.ConnectorEnabledHigh {
	return buses.NewConnectorEnabledHigh()
}

func (m *MockViaChip) ReadWrite() *buses.ConnectorEnabledLow {
	return buses.NewConnectorEnabledLow()
}

func (m *MockViaChip) ConnectRegisterSelectLines(lines [4]buses.Line) {
	// Mock implementation - does nothing in the test
}

func (m *MockViaChip) Tick(context *common.StepContext) {
	// Mock implementation - does nothing in the test
}

func TestNewViaWindow(t *testing.T) {
	mockVia := NewMockViaChip()
	window := NewViaWindow(mockVia)

	assert.NotNil(t, window)
	assert.NotNil(t, window.text)
	assert.Equal(t, mockVia, window.via)

	assert.Equal(t, "VIA Registers", window.text.GetTitle())
}

func TestViaWindowClear(t *testing.T) {
	mockVia := NewMockViaChip()
	window := NewViaWindow(mockVia)

	// Write some content first
	window.text.SetText("Some test content")
	assert.NotEmpty(t, window.text.GetText(true))

	// Clear the window
	window.Clear()
	assert.Empty(t, window.text.GetText(true))
}

func TestViaWindowDraw(t *testing.T) {
	mockVia := NewMockViaChip()
	window := NewViaWindow(mockVia)
	context := &common.StepContext{}

	window.Draw(context)
	text := window.text.GetText(true)

	// Check if all register values are present in the output
	expectedValues := []string{
		"ORA:  $01",
		"ORB:  $02",
		"IRA:  $03",
		"IRB:  $04",
		"DDRA: $05",
		"DDRB: $06",
		"LL1:  $07",
		"HL1:  $08",
		"CTR1: $0910",
		"LL2:  $0A",
		"HL2:  $0B",
		"CTR2: $0C0D",
		"SR:   $0E",
		"ACR:  $0F",
		"PCR:  $10",
		"IFR:  $11",
		"IER:  $12",
		"Bus:  $0013",
	}

	for _, expected := range expectedValues {
		assert.Contains(t, text, expected)
	}
}

func TestViaWindowGetDrawArea(t *testing.T) {
	mockVia := NewMockViaChip()
	window := NewViaWindow(mockVia)

	primitive := window.GetDrawArea()
	assert.NotNil(t, primitive)
	assert.Equal(t, window.text, primitive)
	assert.Implements(t, (*tview.Primitive)(nil), primitive)
}
