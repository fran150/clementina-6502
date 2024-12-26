package lcd

const SECOND_LINE_BIT uint8 = 0x40        // Bit 6 of DDRAM can be used to distinguish between the 1st and 2nd lines
const BUSY_FLAG_BIT uint8 = 0x80          // When reading bit 7 is the busy flag
const SIX_BIT_ADDRESS_MASK uint8 = 0x3F   // Mask to read a 6 bit address for CGRAM
const SEVEN_BIT_ADDRESS_MASK uint8 = 0x7F // Mast to read a 7 bit address for DDRAM
const CGRAM_MIN_ADDR = 0x40               // Min address for CGRAM (for CGRAM bit 6 is always set)
const CGRAM_MAX_ADDR = 0x7F               // Max address for CGRAM (for CGRAM bit 6 is always set)
const DDRAM_MIN_ADDR = 0x00               // Min DDRAM address in 1 line mode
const DDRAM_MAX_ADDR = 0x4F               // Max DDRAM address in 1 line mode
const DDRAM_2LINE_MIN_ADDR = 0x00         // Min DDRAM address (of 1st line) when configured in 2 line mode
const DDRAM_2LINE_MAX_ADDR = 0x27         // Max DDRAM address (of 1st line) when configured in 2 line mode

// The address counter points to the next instruction that will be written in DDRAM or CGRAM.
// For every read or write it increments on decrements depending on the register configuration.
// It also handles the shifting of the pointers of the first character visible for each line in the
// LCD screen
type lcdAddressCounter struct {
	toCGRAM        bool // Returns if the next opertaion will be executed against CGRAM (or DDRAM if false)
	mustMoveRight  bool // Indicates if the pointer must be increased (move right) or decreased (move left)
	is2LineDisplay bool // N: Number of lines
	displayShift   bool // S: Shifts the entire display

	value      uint8 // Current value of the address counter
	line1Shift uint8 // Pointer to the first address visible on the LCD
	line2Shift uint8 // Pinter to the second line (if available) on the LCD

	instructionRegister *uint8 // Instruction register on the main chip
	dataRegister        *uint8 // Data register on the main chip
	busy                *bool  // Busy flag (the chip is executing the desired instruction)

	ddram *[DDRAM_SIZE]uint8 // Pointer to the DDRAM
	cgram *[CGRAM_SIZE]uint8 // Pointer to the CGRAM
}

// Creates an address counter for the specified chip
func createLCDAddressCounter(lcd *LcdHD44780U) *lcdAddressCounter {
	return &lcdAddressCounter{
		instructionRegister: &lcd.instructionRegister,
		dataRegister:        &lcd.dataRegister,

		toCGRAM:        false,
		mustMoveRight:  true,
		is2LineDisplay: false,
		displayShift:   false,

		value:      DDRAM_MIN_ADDR,
		line1Shift: DDRAM_MIN_ADDR,
		line2Shift: DDRAM_MIN_ADDR | SECOND_LINE_BIT,

		ddram: &lcd.ddram,
		cgram: &lcd.cgram,
		busy:  &lcd.isBusy,
	}
}

// Some addresses like the ones for CGRAM are limited to 6 bits. This function returns
// the value represented by lower 6 bits of the specified value
func get6BitAddress(value uint8) uint8 {
	return value & SIX_BIT_ADDRESS_MASK
}

// In 2 line mode, bit 6 of the address indicates if the address is on the first (bit clear) or
// in the second line (bit set). This function returns true if the bit 6 of the specified value
// is set.
func isAddressInSecondLine(value uint8) bool {
	return checkBit(value, SECOND_LINE_BIT)
}

// In 2 line mode, bit 6 of the address indicates if the address is on the first (bit clear) or
// in the second line (bit set). This function toggles the bit 6 of the specified value making
// the cursor jump between lines.
func toggleLine(value uint8) uint8 {
	return value ^ SECOND_LINE_BIT
}

// Returns the index of the current address counter value in CGRAM memory array
func (ac *lcdAddressCounter) getCGRAMIndex() uint8 {
	return get6BitAddress(ac.value)
}

// Returns the index of the current address counter value in DDRAM memory array
func (ac *lcdAddressCounter) getDDRAMIndex() uint8 {
	if ac.is2LineDisplay {
		// Getting a 6 bit address in 2 line mode will always give a line 1 address
		value := get6BitAddress(ac.value)

		// If original value is in second line index is in the second half
		if isAddressInSecondLine(ac.value) {
			value += (DDRAM_SIZE / 2)
		}

		return value
	} else {
		return ac.value
	}
}

// Increments the address counter effectively moving cursor to the right.
// When reaching the right limit it jumps to the beginning or the next line depending
// on the controller configuration and if reading from DDRAM or CGRAM
func (ac *lcdAddressCounter) moveRight() {
	value := ac.value

	// Gets the limits on the current configuration
	min, max := ac.getMinAndMax(ac.toCGRAM, value)

	// If its at rightmost value jump to the beginning (min)
	if value == max {
		value = min

		// If is not CGRAM and is a 2 line display, cursor must jump to
		// next line
		if !ac.toCGRAM && ac.is2LineDisplay {
			value = toggleLine(value)
		}
	} else {
		value++
	}

	ac.write(value)
}

// Decrements the address counter effectively moving cursor to the left.
// When reaching the left limit it jumps to the end or the previous line depending
// on the controller configuration and if reading from DDRAM or CGRAM
func (ac *lcdAddressCounter) moveLeft() {
	value := ac.value

	// Gets the limits on the current configuration
	min, max := ac.getMinAndMax(ac.toCGRAM, value)

	// If its at leftmost value jump to the end (max)
	if value == min {
		value = max

		// If is not CGRAM and is a 2 line display, cursor must jump to
		// previous line
		if !ac.toCGRAM && ac.is2LineDisplay {
			value = toggleLine(value)
		}
	} else {
		value--
	}

	ac.write(value)
}

// There are 2 pointers pointing to the beginning of each line, this is used
// to draw the character starting from this pointers. Increasing the pointers will
// make the entire display shift right
func (ac *lcdAddressCounter) shiftRight() {
	min, max := ac.getMinAndMax(false, ac.line1Shift)

	// If possible increase the shift value, if not, jump to the beginning
	if ac.line1Shift < max {
		ac.line1Shift++
	} else {
		ac.line1Shift = min
	}

	// In a 2 line display 2nd line index is in the same position as line 1 with the bit 6 set
	ac.line2Shift = ((ac.line1Shift & SEVEN_BIT_ADDRESS_MASK) | SECOND_LINE_BIT)
}

// There are 2 pointers pointing to the beginning of each line, this is used
// to draw the character starting from this pointers. Decreasing the pointers will
// make the entire display shift left
func (ac *lcdAddressCounter) shiftLeft() {
	min, max := ac.getMinAndMax(false, ac.line1Shift)

	// If possible decrease the shift value, if not, jump to the end
	if ac.line1Shift > min {
		ac.line1Shift--
	} else {
		ac.line1Shift = max
	}

	// In a 2 line display 2nd line index is in the same position as line 1 with the bit 6 set
	ac.line2Shift = ((ac.line1Shift & SEVEN_BIT_ADDRESS_MASK) | SECOND_LINE_BIT)
}

// Gets the min and max possible address based on the controller configuration and
// the specified value.
// CGRAM addresses go from 0x40 to 0x7F as bit number 6 is always on.
// In 2 line displays DDRAM address for 1st line goes from 0x00 and 0x27
// Second line goes from 0x40 to 0x67 (same as line 1 but bit 6 set)
// In 1 line display DDRAM address goes from 0x00 to 0x4F
func (ac *lcdAddressCounter) getMinAndMax(isCGRAM bool, value uint8) (uint8, uint8) {
	var min, max uint8

	if isCGRAM {
		min, max = CGRAM_MIN_ADDR, CGRAM_MAX_ADDR
	} else {
		if ac.is2LineDisplay {
			min, max = DDRAM_2LINE_MIN_ADDR, DDRAM_2LINE_MAX_ADDR

			if isAddressInSecondLine(value) {
				min, max = min|SECOND_LINE_BIT, max|SECOND_LINE_BIT
			}
		} else {
			min, max = DDRAM_MIN_ADDR, DDRAM_MAX_ADDR
		}
	}

	return min, max
}

// Points the address counter to CGRAM and sets the value from the instruction register
func (ac *lcdAddressCounter) setCGRAMAddress() {
	ac.toCGRAM = true
	ac.write(*ac.instructionRegister)
}

// Points the address counter to DDRAM and sets the value from the instruction register
func (ac *lcdAddressCounter) setDDRAMAddress() {
	ac.toCGRAM = false
	ac.write(*ac.instructionRegister)
}

// Writes to RAM, depending on where the address counter is pointing this will write to
// the value from the data register to CGRAM or DDRAM.
func (ac *lcdAddressCounter) writeToRam() {
	if ac.toCGRAM {
		index := ac.getCGRAMIndex()
		ac.cgram[index] = *ac.dataRegister
	} else {
		index := ac.getDDRAMIndex()
		ac.ddram[index] = *ac.dataRegister
	}

	ac.moveCursorAndDisplay()
}

// Reads from DRAM, depending on where the address counter is pointing this will read from
// CGRAM or DDRAM to the data register.
func (ac *lcdAddressCounter) readFromRam() {
	if ac.toCGRAM {
		index := ac.getCGRAMIndex()
		*ac.dataRegister = ac.cgram[index]
	} else {
		index := ac.getDDRAMIndex()
		*ac.dataRegister = ac.ddram[index]
	}

	ac.moveCursorAndDisplay()
}

// Moves cursor and display in the configured direction
func (ac *lcdAddressCounter) moveCursorAndDisplay() {
	if ac.mustMoveRight {
		ac.moveRight()

		if ac.displayShift {
			ac.shiftRight()
		}
	} else {
		ac.moveLeft()

		if ac.displayShift {
			ac.shiftLeft()
		}
	}
}

// Writes the specified value in the address counter, if an invalid
// DDRAM address is specified it will default to 0.
func (ac *lcdAddressCounter) write(value uint8) {
	value &= SEVEN_BIT_ADDRESS_MASK

	// TODO: Will need to validate this with real hardware.
	min, max := ac.getMinAndMax(ac.toCGRAM, value)
	if value < min || value > max {
		value = min
	}

	ac.value = value
}

// Reads the current value of the address counter. The current address
// is always a 6 bit value, bit 7 is used for the busy flag.
func (ac *lcdAddressCounter) read() uint8 {
	value := ac.value

	if *ac.busy {
		value |= BUSY_FLAG_BIT
	}

	return value
}
