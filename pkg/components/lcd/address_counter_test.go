package lcd

import (
	"testing"
)

func TestNewLCDAddressCounter(t *testing.T) {
	lcd := &LcdHD44780U{}
	ac := newLCDAddressCounter(lcd)

	if ac.toCGRAM != false {
		t.Error("Expected toCGRAM to be false by default")
	}
	if ac.mustMoveRight != true {
		t.Error("Expected mustMoveRight to be true by default")
	}
	if ac.value != DDRAM_MIN_ADDR {
		t.Errorf("Expected initial value to be %d, got %d", DDRAM_MIN_ADDR, ac.value)
	}
}

func TestGet6BitAddress(t *testing.T) {
	tests := []struct {
		input    uint8
		expected uint8
	}{
		{0xFF, 0x3F},
		{0x40, 0x00},
		{0x3F, 0x3F},
		{0x00, 0x00},
	}

	for _, test := range tests {
		result := get6BitAddress(test.input)
		if result != test.expected {
			t.Errorf("get6BitAddress(%d) = %d; want %d", test.input, result, test.expected)
		}
	}
}

func TestIsAddressInSecondLine(t *testing.T) {
	tests := []struct {
		input    uint8
		expected bool
	}{
		{0x40, true},  // Second line
		{0x00, false}, // First line
		{0x67, true},  // Second line max address
		{0x27, false}, // First line max address
	}

	for _, test := range tests {
		result := isAddressInSecondLine(test.input)
		if result != test.expected {
			t.Errorf("isAddressInSecondLine(%d) = %v; want %v", test.input, result, test.expected)
		}
	}
}

func TestMoveRight(t *testing.T) {
	lcd := &LcdHD44780U{}
	ac := newLCDAddressCounter(lcd)

	// Test moving right in 1-line mode
	initialValue := ac.value
	ac.moveRight()
	if ac.value != initialValue+1 {
		t.Errorf("Expected value to increment from %d to %d, got %d", initialValue, initialValue+1, ac.value)
	}

	// Test wrapping in 1-line mode
	ac.value = DDRAM_MAX_ADDR
	ac.moveRight()
	if ac.value != DDRAM_MIN_ADDR {
		t.Errorf("Expected value to wrap to %d, got %d", DDRAM_MIN_ADDR, ac.value)
	}

	// Test 2-line mode
	ac.is2LineDisplay = true
	ac.value = DDRAM_2LINE_MAX_ADDR
	ac.moveRight()
	if !isAddressInSecondLine(ac.value) {
		t.Error("Expected to move to second line after reaching end of first line")
	}
}

func TestMoveLeft(t *testing.T) {
	lcd := &LcdHD44780U{}
	ac := newLCDAddressCounter(lcd)

	// Test moving left from position 1
	ac.value = 1
	ac.moveLeft()
	if ac.value != 0 {
		t.Errorf("Expected value to decrement to 0, got %d", ac.value)
	}

	// Test wrapping in 1-line mode
	ac.value = DDRAM_MIN_ADDR
	ac.moveLeft()
	if ac.value != DDRAM_MAX_ADDR {
		t.Errorf("Expected value to wrap to %d, got %d", DDRAM_MAX_ADDR, ac.value)
	}
}

func TestAddressCounterSetCGRAMAddress(t *testing.T) {
	lcd := &LcdHD44780U{}
	ac := newLCDAddressCounter(lcd)

	lcd.instructionRegister = CGRAM_MIN_ADDR
	ac.setCGRAMAddress()

	if !ac.toCGRAM {
		t.Error("Expected toCGRAM to be true after setCGRAMAddress")
	}
	if ac.value != CGRAM_MIN_ADDR {
		t.Errorf("Expected CGRAM address to be %d, got %d", CGRAM_MIN_ADDR, ac.value)
	}
}

func TestSetDDRAMAddress(t *testing.T) {
	lcd := &LcdHD44780U{}
	ac := newLCDAddressCounter(lcd)

	lcd.instructionRegister = DDRAM_MIN_ADDR
	ac.setDDRAMAddress()

	if ac.toCGRAM {
		t.Error("Expected toCGRAM to be false after setDDRAMAddress")
	}
	if ac.value != DDRAM_MIN_ADDR {
		t.Errorf("Expected DDRAM address to be %d, got %d", DDRAM_MIN_ADDR, ac.value)
	}
}

func TestShiftOperations(t *testing.T) {
	lcd := &LcdHD44780U{}
	ac := newLCDAddressCounter(lcd)

	initialShift := ac.line1Shift
	ac.shiftRight()
	if ac.line1Shift != initialShift+1 {
		t.Errorf("Expected line1Shift to increment from %d to %d, got %d",
			initialShift, initialShift+1, ac.line1Shift)
	}

	ac.shiftLeft()
	if ac.line1Shift != initialShift {
		t.Errorf("Expected line1Shift to return to %d, got %d",
			initialShift, ac.line1Shift)
	}
}

func TestReadWrite(t *testing.T) {
	lcd := &LcdHD44780U{}
	ac := newLCDAddressCounter(lcd)

	// Test write with valid address
	ac.write(0x20)
	if ac.value != 0x20 {
		t.Errorf("Expected value to be 0x20, got 0x%02X", ac.value)
	}

	// Test write with invalid address (should default to min)
	ac.write(0xFF)
	if ac.value != DDRAM_MIN_ADDR {
		t.Errorf("Expected invalid address to default to %d, got %d",
			DDRAM_MIN_ADDR, ac.value)
	}

	// Test read with busy flag
	*ac.busy = true
	readValue := ac.read()
	if readValue&BUSY_FLAG_BIT == 0 {
		t.Error("Expected busy flag to be set in read value")
	}
}
