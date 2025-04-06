package lcd

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/stretchr/testify/assert"
)

func TestLcdHD44780U_Enable(t *testing.T) {
	// Arrange
	enableConnector := &buses.ConnectorEnabledHigh{}
	ctrl := &LcdHD44780U{
		enable: enableConnector,
	}

	// Act
	result := ctrl.Enable()

	// Assert
	assert.Equal(t, enableConnector, result)
}

func TestLcdHD44780U_ReadWrite(t *testing.T) {
	// Arrange
	writeConnector := &buses.ConnectorEnabledLow{}
	ctrl := &LcdHD44780U{
		write: writeConnector,
	}

	// Act
	result := ctrl.ReadWrite()

	// Assert
	assert.Equal(t, writeConnector, result)
}

func TestLcdHD44780U_RegisterSelect(t *testing.T) {
	// Arrange
	registerSelector := &buses.ConnectorEnabledHigh{}
	ctrl := &LcdHD44780U{
		dataRegisterSelected: registerSelector,
	}

	// Act
	result := ctrl.RegisterSelect()

	// Assert
	assert.Equal(t, registerSelector, result)
}

func TestLcdHD44780U_DataBus(t *testing.T) {
	// Arrange
	dataBusConnector := &buses.BusConnector[uint8]{}
	ctrl := &LcdHD44780U{
		dataBus: dataBusConnector,
	}

	// Act
	result := ctrl.DataBus()

	// Assert
	assert.Equal(t, dataBusConnector, result)
}

func TestLcdHD44780U_GetCursorStatus(t *testing.T) {
	tests := []struct {
		name               string
		displayCursor      bool
		addressValue       uint8
		blinkingVisible    bool
		expectedDDRAMIndex uint8
		want               CursorStatus
	}{
		{
			name:               "Cursor visible and blinking",
			displayCursor:      true,
			addressValue:       0x00,
			blinkingVisible:    true,
			expectedDDRAMIndex: 0x00,
			want: CursorStatus{
				CursorVisible:      true,
				CursorPosition:     0x00,
				BlinkStatusShowing: true,
			},
		},
		{
			name:               "Cursor hidden and not blinking",
			displayCursor:      false,
			addressValue:       0x40,
			blinkingVisible:    false,
			expectedDDRAMIndex: 0x40,
			want: CursorStatus{
				CursorVisible:      false,
				CursorPosition:     0x40,
				BlinkStatusShowing: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ctrl := &LcdHD44780U{}
			addressCounter := newLCDAddressCounter(ctrl)
			addressCounter.value = tt.addressValue

			ctrl.addressCounter = addressCounter
			ctrl.displayCursor = tt.displayCursor
			ctrl.blinkingVisible = tt.blinkingVisible

			// Act
			got := ctrl.GetCursorStatus()

			// Assert
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLcdHD44780U_GetDisplayStatus(t *testing.T) {
	tests := []struct {
		name           string
		displayOn      bool
		is2LineDisplay bool
		is5x10Font     bool
		line1Shift     uint8
		line2Shift     uint8
		is8BitMode     bool
		cgram          [64]uint8
		ddram          [80]uint8
		want           DisplayStatus
	}{
		{
			name:           "Display on, 2-line, 5x8 font",
			displayOn:      true,
			is2LineDisplay: true,
			is5x10Font:     false,
			line1Shift:     0x00,
			line2Shift:     0x28,
			is8BitMode:     true,
			cgram:          [64]uint8{},
			ddram:          [80]uint8{},
			want: DisplayStatus{
				DisplayOn:      true,
				Is2LineDisplay: true,
				Is5x10Font:     false,
				Line1Start:     0x00,
				Line2Start:     0x28,
				Is8BitMode:     true,
				CGRAM:          make([]uint8, 64),
				DDRAM:          make([]uint8, 80),
			},
		},
		{
			name:           "Display off, 1-line, 5x10 font",
			displayOn:      false,
			is2LineDisplay: false,
			is5x10Font:     true,
			line1Shift:     0x00,
			line2Shift:     0x00, // Changed to 0x00 for 1-line display
			is8BitMode:     false,
			cgram:          [64]uint8{},
			ddram:          [80]uint8{},
			want: DisplayStatus{
				DisplayOn:      false,
				Is2LineDisplay: false,
				Is5x10Font:     true,
				Line1Start:     0x00,
				Line2Start:     0x00, // Changed to 0x00 for 1-line display
				Is8BitMode:     false,
				CGRAM:          make([]uint8, 64),
				DDRAM:          make([]uint8, 80),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ctrl := &LcdHD44780U{
				displayOn:  tt.displayOn,
				is5x10Font: tt.is5x10Font,
				cgram:      tt.cgram,
				ddram:      tt.ddram,
			}

			addressCounter := newLCDAddressCounter(ctrl)
			addressCounter.is2LineDisplay = tt.is2LineDisplay
			addressCounter.line1Shift = tt.line1Shift
			addressCounter.line2Shift = tt.line2Shift
			ctrl.addressCounter = addressCounter

			buffer := newLcdBuffer()
			buffer.is8BitMode = tt.is8BitMode
			ctrl.buffer = buffer

			// Act
			got := ctrl.GetDisplayStatus()

			// Assert
			assert.Equal(t, tt.want, got)
		})
	}
}
