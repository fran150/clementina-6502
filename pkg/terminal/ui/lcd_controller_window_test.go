package ui

import (
	"strings"
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/fran150/clementina-6502/pkg/components/lcd"
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
	if _, err := window.text.Write([]byte("Test content")); err != nil {
		t.Fatalf("Failed to write to window: %v", err)
	}
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
	assert.Contains(t, content, "Display ON:     true")
	assert.Contains(t, content, "8 Bit Mode:     true")
	assert.Contains(t, content, "2 Line Display: true")
	assert.Contains(t, content, "Bus:             $00")
	assert.Contains(t, content, "Enable:          true")
	assert.Contains(t, content, "Read/Write:      false")
	assert.Contains(t, content, "Reg Select:      true")

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
		wantContains  string
	}{
		{
			name: "Empty DDRAM",
			displayStatus: lcd.DisplayStatus{
				DisplayOn:      true,
				Is2LineDisplay: true,
				Is8BitMode:     true,
				Line1Start:     0x00,
				Line2Start:     0x40,
				DDRAM:          make([]uint8, 8),
			},
			wantContains: `     ┌─────────────────────────┐
     │ [blue] 0 [white][blue] 1 [white][blue] 2 [white][blue] 3 [white][blue] 4 [white][blue] 5 [white][blue] 6 [white][blue] 7[white] │
     ├─────────────────────────┤
 [blue]00:[white] │ [yellow]00 [white][yellow]00 [white][yellow]00 [white][yellow]00 [white][yellow]00 [white][yellow]00 [white][yellow]00 [white][yellow]00[white] │
     └─────────────────────────┘
`,
		},
		{
			name: "Single line of data",
			displayStatus: lcd.DisplayStatus{
				DisplayOn:      true,
				Is2LineDisplay: true,
				Is8BitMode:     true,
				Line1Start:     0x00,
				Line2Start:     0x40,
				DDRAM: func() []uint8 {
					ddram := make([]uint8, 8)
					copy(ddram, []uint8{'H', 'e', 'l', 'l', 'o'})
					return ddram
				}(),
			},
			wantContains: `     ┌─────────────────────────┐
     │ [blue] 0 [white][blue] 1 [white][blue] 2 [white][blue] 3 [white][blue] 4 [white][blue] 5 [white][blue] 6 [white][blue] 7[white] │
     ├─────────────────────────┤
 [blue]00:[white] │ [green] H [white][green] e [white][green] l [white][green] l [white][green] o [white][yellow]00 [white][yellow]00 [white][yellow]00[white] │
     └─────────────────────────┘
`,
		},
		{
			name: "Multiple lines of data",
			displayStatus: lcd.DisplayStatus{
				DisplayOn:      true,
				Is2LineDisplay: true,
				Is8BitMode:     true,
				Line1Start:     0x00,
				Line2Start:     0x40,
				DDRAM: func() []uint8 {
					ddram := make([]uint8, 16)
					copy(ddram, []uint8{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'A', 'B', 'C', 'D', 'E', 'F'})
					return ddram
				}(),
			},
			wantContains: `     ┌─────────────────────────┐
     │ [blue] 0 [white][blue] 1 [white][blue] 2 [white][blue] 3 [white][blue] 4 [white][blue] 5 [white][blue] 6 [white][blue] 7[white] │
     ├─────────────────────────┤
 [blue]00:[white] │ [green] 0 [white][green] 1 [white][green] 2 [white][green] 3 [white][green] 4 [white][green] 5 [white][green] 6 [white][green] 7[white] │
 [blue]08:[white] │ [green] 8 [white][green] 9 [white][green] A [white][green] B [white][green] C [white][green] D [white][green] E [white][green] F[white] │
     └─────────────────────────┘
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			drawLcdDDRAM(&buf, tt.displayStatus)
			content := buf.String()
			assert.Equal(t, tt.wantContains, content)
		})
	}
}
