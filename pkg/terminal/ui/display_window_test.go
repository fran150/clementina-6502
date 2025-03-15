package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fran150/clementina6502/pkg/common"
	"github.com/fran150/clementina6502/pkg/components/buses"
	"github.com/fran150/clementina6502/pkg/components/lcd"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLCDController is a mock implementation of the LCDControllerChip interface
type MockLCDController struct {
	mock.Mock
}

func (m *MockLCDController) Enable() *buses.ConnectorEnabledHigh {
	args := m.Called()
	return args.Get(0).(*buses.ConnectorEnabledHigh)
}

func (m *MockLCDController) ReadWrite() *buses.ConnectorEnabledLow {
	args := m.Called()
	return args.Get(0).(*buses.ConnectorEnabledLow)
}

func (m *MockLCDController) RegisterSelect() *buses.ConnectorEnabledHigh {
	args := m.Called()
	return args.Get(0).(*buses.ConnectorEnabledHigh)
}

func (m *MockLCDController) DataBus() *buses.BusConnector[uint8] {
	args := m.Called()
	return args.Get(0).(*buses.BusConnector[uint8])
}

func (m *MockLCDController) GetDisplayStatus() lcd.DisplayStatus {
	args := m.Called()
	return args.Get(0).(lcd.DisplayStatus)
}

func (m *MockLCDController) GetCursorStatus() lcd.CursorStatus {
	args := m.Called()
	return args.Get(0).(lcd.CursorStatus)
}

func (m *MockLCDController) Tick(context *common.StepContext) {
	m.Called()
}

func TestNewDisplayWindow(t *testing.T) {
	mockLCD := new(MockLCDController)
	window := NewDisplayWindow(mockLCD)

	assert.NotNil(t, window)
	assert.NotNil(t, window.text)
	assert.Equal(t, mockLCD, window.controller)
}

func TestDrawLcdLineOff(t *testing.T) {
	var buf bytes.Buffer
	drawLcdLineOff(&buf)
	result := buf.String()

	assert.Contains(t, result, "[black:grey]")
	assert.Equal(t, 16, len(result)-len("[black:grey]"))
}

func TestDrawLcdLine(t *testing.T) {
	tests := []struct {
		name          string
		displayStatus lcd.DisplayStatus
		cursorStatus  lcd.CursorStatus
		lineStart     uint8
		min           uint8
		max           uint8
		expected      string
	}{
		{
			name: "Normal line without wrapping",
			displayStatus: lcd.DisplayStatus{
				DisplayOn:      true,
				Is2LineDisplay: true,
				DDRAM: func() []uint8 {
					ddram := createEmptyDDRAM()
					copy(ddram[0:], []uint8("Hello"))
					return ddram
				}(),
			},
			cursorStatus: lcd.CursorStatus{},
			lineStart:    0,
			min:          0,
			max:          40,
			expected:     "[black:green]Hello           ",
		},
		{
			name: "Line with wrapping at max index",
			displayStatus: lcd.DisplayStatus{
				DisplayOn:      true,
				Is2LineDisplay: true,
				DDRAM: func() []uint8 {
					ddram := createEmptyDDRAM()
					// Place "He" at positions 38,39
					copy(ddram[38:], []uint8("He"))
					// Place "World" at the beginning (position 0)
					copy(ddram[0:], []uint8("World"))
					return ddram
				}(),
			},
			cursorStatus: lcd.CursorStatus{},
			lineStart:    38, // Start reading from position 38
			min:          0,  // Wrap back to beginning of line
			max:          40, // End of first line
			expected:     "[black:green]HeWorld         ",
		},
		{
			name: "Second line with wrapping",
			displayStatus: lcd.DisplayStatus{
				DisplayOn:      true,
				Is2LineDisplay: true,
				DDRAM: func() []uint8 {
					ddram := createEmptyDDRAM()
					copy(ddram[78:], []uint8("He"))  // Place text at end of second line
					copy(ddram[40:], []uint8("llo")) // And at start of second line
					return ddram
				}(),
			},
			cursorStatus: lcd.CursorStatus{},
			lineStart:    78, // Start near end of second line
			min:          40,
			max:          80,
			expected:     "[black:green]Hello           ", // Should wrap within second line
		},
		{
			name: "Cursor blinking - showing block",
			displayStatus: lcd.DisplayStatus{
				DisplayOn:      true,
				Is2LineDisplay: true,
				DDRAM:          createEmptyDDRAM(),
			},
			cursorStatus: lcd.CursorStatus{
				CursorVisible:      true,
				BlinkStatusShowing: true,
				CursorPosition:     5,
			},
			lineStart: 0,
			min:       0,
			max:       40,
			expected:  "[black:green]     [::u]█[::-]          ",
		},
		{
			name: "Cursor blinking - not showing block",
			displayStatus: lcd.DisplayStatus{
				DisplayOn:      true,
				Is2LineDisplay: true,
				DDRAM:          createEmptyDDRAM(),
			},
			cursorStatus: lcd.CursorStatus{
				CursorVisible:      true,
				BlinkStatusShowing: false,
				CursorPosition:     5,
			},
			lineStart: 0,
			min:       0,
			max:       40,
			expected:  "[black:green]     [::u] [::-]          ",
		},
		{
			name: "Cursor blinking with text",
			displayStatus: lcd.DisplayStatus{
				DisplayOn:      true,
				Is2LineDisplay: true,
				DDRAM: func() []uint8 {
					ddram := createEmptyDDRAM()
					copy(ddram[0:], []uint8("Hello World"))
					return ddram
				}(),
			},
			cursorStatus: lcd.CursorStatus{
				CursorVisible:      true,
				BlinkStatusShowing: true,
				CursorPosition:     5,
			},
			lineStart: 0,
			min:       0,
			max:       40,
			expected:  "[black:green]Hello[::u]█[::-]World     ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			drawLcdLine(&buf, tt.lineStart, tt.displayStatus, tt.cursorStatus, tt.min, tt.max)
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}

func TestLcd16x2Window_Clear(t *testing.T) {
	// Setup
	text := tview.NewTextView()
	window := &Lcd16x2Window{
		text: text,
	}

	// Execute
	window.Clear()

	// Assert
	assert.Empty(t, text.GetText(false))
}

func createEmptyDDRAM() []uint8 {
	ddram := make([]uint8, 80)
	for i := range ddram {
		ddram[i] = 0x20 // ASCII space character
	}
	return ddram
}

func TestLcd16x2Window_Draw(t *testing.T) {
	tests := []struct {
		name           string
		displayStatus  lcd.DisplayStatus
		cursorStatus   lcd.CursorStatus
		expectedOutput string
	}{
		{
			name: "Display Off",
			displayStatus: lcd.DisplayStatus{
				DisplayOn:      false,
				Is2LineDisplay: true,
			},
			cursorStatus:   lcd.CursorStatus{},
			expectedOutput: "[black:grey]                \n[black:grey]                ",
		},
		{
			name: "Not in 2 Line Mode",
			displayStatus: lcd.DisplayStatus{
				DisplayOn:      true,
				Is2LineDisplay: false,
			},
			cursorStatus:   lcd.CursorStatus{},
			expectedOutput: "[red]Not in two\nline mode",
		},
		{
			name: "Normal Display",
			displayStatus: lcd.DisplayStatus{
				DisplayOn:      true,
				Is2LineDisplay: true,
				DDRAM:          createEmptyDDRAM(),
			},
			cursorStatus: lcd.CursorStatus{
				CursorVisible: false,
			},
			expectedOutput: "[black:green]                \n[black:green]                ",
		},
		{
			name: "Display with Text",
			displayStatus: lcd.DisplayStatus{
				DisplayOn:      true,
				Is2LineDisplay: true,
				DDRAM: func() []uint8 {
					ddram := createEmptyDDRAM()
					copy(ddram[0:], []uint8("Hello World"))
					copy(ddram[40:], []uint8("Second Line"))
					return ddram
				}(),
				Line1Start: 0,
				Line2Start: 40,
			},
			cursorStatus: lcd.CursorStatus{
				CursorVisible: false,
			},
			expectedOutput: "[black:green]Hello World     \n[black:green]Second Line     ",
		},
		{
			name: "Display with Cursor",
			displayStatus: lcd.DisplayStatus{
				DisplayOn:      true,
				Is2LineDisplay: true,
				DDRAM: func() []uint8 {
					ddram := createEmptyDDRAM()
					copy(ddram[0:], []uint8("Hello World"))
					copy(ddram[40:], []uint8("Second Line"))
					return ddram
				}(),
				Line1Start: 0,
				Line2Start: 40,
			},
			cursorStatus: lcd.CursorStatus{
				CursorVisible:      true,
				CursorPosition:     5,
				BlinkStatusShowing: false,
			},
			expectedOutput: "[black:green]Hello[::u] [::-]World     \n[black:green]Second Line     ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			text := tview.NewTextView()
			mockController := new(MockLCDController)

			mockController.On("GetDisplayStatus").Return(tt.displayStatus)
			mockController.On("GetCursorStatus").Return(tt.cursorStatus)

			window := &Lcd16x2Window{
				text:       text,
				controller: mockController,
			}

			// Execute
			window.Draw(&common.StepContext{})

			// Assert
			assert.Equal(t, tt.expectedOutput, text.GetText(false))
			mockController.AssertExpectations(t)
		})
	}
}

func TestLcd16x2Window_GetDrawArea(t *testing.T) {
	// Setup
	expectedText := tview.NewTextView()
	window := &Lcd16x2Window{
		text: expectedText,
	}

	// Execute
	result := window.GetDrawArea()

	// Assert
	assert.Equal(t, expectedText, result, "GetDrawArea should return the text view primitive")
}
