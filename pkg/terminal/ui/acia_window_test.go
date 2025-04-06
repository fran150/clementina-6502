package ui

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/stretchr/testify/assert"
	"go.bug.st/serial"
)

type testAcia struct {
	statusRegister  uint8
	controlRegister uint8
	commandRegister uint8
	txRegister      uint8
	rxRegister      uint8
	txEmpty         bool
	rxEmpty         bool

	dataBus        *buses.BusConnector[uint8]
	irqRequest     *buses.ConnectorEnabledLow
	readWrite      *buses.ConnectorEnabledLow
	chipSelect0    *buses.ConnectorEnabledHigh
	chipSelect1    *buses.ConnectorEnabledLow
	registerSelect [2]*buses.ConnectorEnabledHigh
	reset          *buses.ConnectorEnabledLow
}

func newTestAcia() *testAcia {
	return &testAcia{
		dataBus:     buses.NewBusConnector[uint8](),
		irqRequest:  buses.NewConnectorEnabledLow(),
		readWrite:   buses.NewConnectorEnabledLow(),
		chipSelect0: buses.NewConnectorEnabledHigh(),
		chipSelect1: buses.NewConnectorEnabledLow(),
		registerSelect: [2]*buses.ConnectorEnabledHigh{
			buses.NewConnectorEnabledHigh(),
			buses.NewConnectorEnabledHigh(),
		},
		reset: buses.NewConnectorEnabledLow(),
	}
}

// Pin connections
func (t *testAcia) DataBus() *buses.BusConnector[uint8] {
	return t.dataBus
}

func (t *testAcia) IrqRequest() *buses.ConnectorEnabledLow {
	return t.irqRequest
}

func (t *testAcia) ReadWrite() *buses.ConnectorEnabledLow {
	return t.readWrite
}

func (t *testAcia) ChipSelect0() *buses.ConnectorEnabledHigh {
	return t.chipSelect0
}

func (t *testAcia) ChipSelect1() *buses.ConnectorEnabledLow {
	return t.chipSelect1
}

func (t *testAcia) RegisterSelect(num uint8) *buses.ConnectorEnabledHigh {
	return t.registerSelect[num]
}

func (t *testAcia) Reset() *buses.ConnectorEnabledLow {
	return t.reset
}

// Configuration methods
func (t *testAcia) ConnectToPort(port serial.Port) error {
	return nil
}

func (t *testAcia) ConnectRegisterSelectLines(lines [2]buses.Line) {
	t.registerSelect[0].Connect(lines[0])
	t.registerSelect[1].Connect(lines[1])
}

func (t *testAcia) Close() {}

// Emulation methods
func (t *testAcia) Tick(context *common.StepContext) {}

// Register getters
func (t *testAcia) GetStatusRegister() uint8  { return t.statusRegister }
func (t *testAcia) GetControlRegister() uint8 { return t.controlRegister }
func (t *testAcia) GetCommandRegister() uint8 { return t.commandRegister }
func (t *testAcia) GetTXRegister() uint8      { return t.txRegister }
func (t *testAcia) GetRXRegister() uint8      { return t.rxRegister }
func (t *testAcia) GetTXRegisterEmpty() bool  { return t.txEmpty }
func (t *testAcia) GetRXRegisterEmpty() bool  { return t.rxEmpty }

func TestAciaWindowRegistersDisplay(t *testing.T) {
	testAcia := newTestAcia()

	// Set up test values
	testAcia.statusRegister = 0xFF
	testAcia.controlRegister = 0xAA
	testAcia.commandRegister = 0x55
	testAcia.txRegister = 0x12
	testAcia.rxRegister = 0x34
	testAcia.txEmpty = true
	testAcia.rxEmpty = false

	window := NewAciaWindow(testAcia)
	window.Draw(&common.StepContext{})

	// Add your assertions here to test the window display
	// For example:
	// assert.Contains(t, window.text.GetText(true), "Status Register: FF")
}

func TestStatusRegisterDetails(t *testing.T) {
	tests := []struct {
		name     string
		status   uint8
		expected []string
	}{
		{
			name:   "All flags set",
			status: 0xFF,
			expected: []string{
				"Interrupt occurred",
				"Not Ready",
				"Not Detected",
				"Empty",
				"Full",
				"Overrun",
				"Framing Error Detected",
				"Parity Error Detected",
			},
		},
		{
			name:   "No flags set",
			status: 0x00,
			expected: []string{
				"No Interrupt",
				"Ready",
				"Detected",
				"Not Empty",
				"Not Full",
				"No Overrun",
				"No Error",
				"No Error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testAcia := &testAcia{statusRegister: tt.status}
			window := NewAciaWindow(testAcia)
			window.Draw(&common.StepContext{})

			text := window.text.GetText(true)
			for _, expected := range tt.expected {
				assert.Contains(t, text, expected)
			}
		})
	}
}

func TestControlRegisterDetails(t *testing.T) {
	tests := []struct {
		name     string
		control  uint8
		expected []string
	}{
		{
			name:    "Word length 8, 1 stop bit, 115200 baud",
			control: 0x00,
			expected: []string{
				"1 Stop bit",
				"8",
				"Baud Rate",
				"115200",
			},
		},
		{
			name:    "Word length 5, 2 stop bits, 19200 baud",
			control: 0xEF,
			expected: []string{
				"2 Stop bits / 1.5 when WL = 5",
				"5",
				"Baud Rate",
				"19200",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testAcia := &testAcia{controlRegister: tt.control}
			window := NewAciaWindow(testAcia)
			window.Draw(&common.StepContext{})

			text := window.text.GetText(true)
			for _, expected := range tt.expected {
				assert.Contains(t, text, expected)
			}
		})
	}
}

func TestCommandRegisterDetails(t *testing.T) {
	tests := []struct {
		name     string
		command  uint8
		expected []string
	}{
		{
			name:    "All features disabled",
			command: 0x00,
			expected: []string{
				"00 - No Parity",
				"0 - No Parity",
				"0 - Normal",
				"00 - No Parity",
				"Enabled",
				"Not Ready",
			},
		},
		{
			name:    "All features enabled",
			command: 0xFF,
			expected: []string{
				"11 - No Parity",
				"1 - No Parity",
				"1 - Enabled",
				"11 - TX Interrupt disabled, Transmit Break",
				"Disabled",
				"Ready",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testAcia := &testAcia{commandRegister: tt.command}
			window := NewAciaWindow(testAcia)
			window.Draw(&common.StepContext{})

			text := window.text.GetText(true)
			for _, expected := range tt.expected {
				assert.Contains(t, text, expected)
			}
		})
	}
}

func TestWindowClear(t *testing.T) {
	testAcia := &testAcia{statusRegister: 0xFF}
	window := NewAciaWindow(testAcia)

	// Draw something first
	window.Draw(&common.StepContext{})
	assert.NotEmpty(t, window.text.GetText(true))

	// Clear the window
	window.Clear()
	assert.Empty(t, window.text.GetText(true))
}

func TestGetDrawArea(t *testing.T) {
	testAcia := &testAcia{}
	window := NewAciaWindow(testAcia)

	assert.Equal(t, window.text, window.GetDrawArea())
}
