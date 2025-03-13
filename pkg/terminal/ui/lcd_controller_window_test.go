package ui

import (
	"testing"

	"github.com/fran150/clementina6502/pkg/components/buses"
	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/fran150/clementina6502/pkg/components/lcd"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func TestNewLcdWindow(t *testing.T) {
	mockLcd := new(MockLCDController)
	window := NewLcdWindow(mockLcd)

	assert.NotNil(t, window)
	assert.NotNil(t, window.text)
	assert.NotNil(t, window.lcd)
}

func TestLcdControllerWindow_Clear(t *testing.T) {
	mockLcd := new(MockLCDController)
	window := NewLcdWindow(mockLcd)

	// Write some content first
	window.text.Write([]byte("Test content"))
	assert.NotEmpty(t, window.text.GetText(true))

	// Clear the window
	window.Clear()
	assert.Empty(t, window.text.GetText(true))
}

func TestLcdControllerWindow_Draw(t *testing.T) {
	mockLcd := new(MockLCDController)

	// Set up the mock expectations
	mockLcd.On("GetCursorStatus").Return(lcd.CursorStatus{
		CursorPosition: 5,
	})

	mockLcd.On("GetDisplayStatus").Return(lcd.DisplayStatus{
		DisplayOn:      true,
		Is8BitMode:     true,
		Is2LineDisplay: true,
		DDRAM:          []byte("Hello LCD"),
	})

	// Create signal lines with the desired values
	dataBus := buses.New8BitStandaloneBus()
	enable := buses.NewStandaloneLine(true)
	readWrite := buses.NewStandaloneLine(true)
	registerSelect := buses.NewStandaloneLine(true)

	dataBusConnector := buses.NewBusConnector[uint8]()
	enableConnector := buses.NewConnectorEnabledHigh()
	readWriteConnector := buses.NewConnectorEnabledLow()
	registerSelectConnector := buses.NewConnectorEnabledHigh()

	dataBusConnector.Connect(dataBus)
	enableConnector.Connect(enable)
	readWriteConnector.Connect(readWrite)
	registerSelectConnector.Connect(registerSelect)

	// Set up expectations for signal line getters
	mockLcd.On("DataBus").Return(dataBusConnector)
	mockLcd.On("Enable").Return(enableConnector)
	mockLcd.On("ReadWrite").Return(readWriteConnector)
	mockLcd.On("RegisterSelect").Return(registerSelectConnector)

	window := NewLcdWindow(mockLcd)
	context := &common.StepContext{}

	// Draw the window
	window.Draw(context)

	// Get the text content
	content := window.text.GetText(true)

	// Verify the content contains all expected information
	assert.Contains(t, content, "LCD Memory:")
	assert.Contains(t, content, "Display ON: true")
	assert.Contains(t, content, "8 Bit Mode: true")
	assert.Contains(t, content, "Line 2 display: true")
	assert.Contains(t, content, "Cursor Position: 5")
	assert.Contains(t, content, "Bus: 0")
	assert.Contains(t, content, "E: true")
	assert.Contains(t, content, "RW: false")
	assert.Contains(t, content, "RS: true")

	// Verify that all expected methods were called
	mockLcd.AssertExpectations(t)
}

func TestLcdControllerWindow_GetDrawArea(t *testing.T) {
	mockLcd := new(MockLCDController)
	window := NewLcdWindow(mockLcd)

	drawArea := window.GetDrawArea()
	assert.NotNil(t, drawArea)
	assert.Equal(t, window.text, drawArea)
}

func TestDrawLcdDDRAM(t *testing.T) {
	tests := []struct {
		name          string
		displayStatus lcd.DisplayStatus
		wantContains  []string
	}{
		{
			name: "Empty DDRAM",
			displayStatus: lcd.DisplayStatus{
				DDRAM: []byte{},
			},
			wantContains: []string{},
		},
		{
			name: "Single line of data",
			displayStatus: lcd.DisplayStatus{
				DDRAM: []byte("Hello"),
			},
			wantContains: []string{"00: H", "01: e", "02: l", "03: l", "04: o"},
		},
		{
			name: "Multiple lines of data",
			displayStatus: lcd.DisplayStatus{
				DDRAM: []byte("0123456789ABCDEF"),
			},
			wantContains: []string{
				"00: 0", "09: 9",
				"0A: A", "0F: F",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := tview.NewTextView()
			text.SetScrollable(false).
				SetDynamicColors(true)
			drawLcdDDRAM(text, tt.displayStatus)
			result := text.GetText(true)

			for _, want := range tt.wantContains {
				assert.Contains(t, result, want)
			}
		})
	}
}
