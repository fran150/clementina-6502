package lcd

type lcdAddressCounter struct {
	toCGRAM        bool
	mustMoveRight  bool
	is2LineDisplay bool // N: Number of lines
	displayShift   bool // S: Shifts the entire display
	busy           bool

	value      uint8
	line1Shift uint8
	line2Shift uint8

	instructionRegister *uint8
	dataRegister        *uint8

	ddram *[DDRAM_SIZE]uint8
	cgram *[CGRAM_SIZE]uint8
}

/*For 4-bit interface data, only four bus lines (DB4 to DB7) are used for transfer. Bus lines DB0 to DB3
are disabled. The data transfer between the HD44780U and the MPU is completed after the 4-bit data
has been transferred twice. As for the order of data transfer, the four high order bits (for 8-bit operation,
DB4 to DB7) are transferred before the four low order bits (for 8-bit operation, DB0 to DB3).
The busy flag must be checked (one instruction) after the 4-bit data has been transferred twice. Two
more 4-bit operations then transfer the busy flag and address counter data.*/

func createLCDAdressCounter(lcd *LcdHD44780U) *lcdAddressCounter {
	return &lcdAddressCounter{
		instructionRegister: &lcd.instructionRegister,
		dataRegister:        &lcd.dataRegister,

		toCGRAM:        false,
		mustMoveRight:  true,
		is2LineDisplay: false,
		displayShift:   false,
		busy:           false,

		value:      0x00,
		line1Shift: 0x00,
		line2Shift: 0x40,

		ddram: &lcd.ddram,
		cgram: &lcd.cgram,
	}
}

func (ac *lcdAddressCounter) getCGRAMIndex() uint8 {
	return ac.value & 0x3F
}

func (ac *lcdAddressCounter) getDDRAMIndex() uint8 {
	if ac.is2LineDisplay {
		if ac.value&0x40 == 0x40 {
			return (ac.value & 0x3F) + 40
		} else {
			return ac.value & 0x7F
		}
	} else {
		return ac.value
	}
}

func (ac *lcdAddressCounter) moveRight() {
	ac.write(ac.value + 1)
}

func (ac *lcdAddressCounter) moveLeft() {
	// TODO: There is probably a better way of doing this.
	if ac.is2LineDisplay {
		if ac.value == 0 {
			ac.value = 0x68
		}

		if ac.value == 0x40 {
			ac.value = 0x28
		}
	} else {
		if ac.value == 0 {
			ac.value = 0x50
		}
	}

	ac.write(ac.value - 1)
}

func (ac *lcdAddressCounter) shiftRight() {
	var max uint8 = 0x50

	if ac.is2LineDisplay {
		max = 0x28
	}

	ac.line1Shift++

	if ac.line1Shift >= max {
		ac.line1Shift = 0x00
	}

	ac.line2Shift = ((ac.line1Shift & 0x7F) | 0x40)
}

func (ac *lcdAddressCounter) shiftLeft() {
	var max uint8 = 0x50

	if ac.is2LineDisplay {
		max = 0x28
	}

	if ac.line1Shift > 0x00 {
		ac.line1Shift--
	} else {
		ac.line1Shift = max - 1
	}

	ac.line2Shift = ((ac.line1Shift & 0x7F) | 0x40)
}

func (ac *lcdAddressCounter) setCGRAMAddress() {
	ac.toCGRAM = true
	ac.write(*ac.instructionRegister)
}

func (ac *lcdAddressCounter) setDDRAMAddress() {
	ac.toCGRAM = false
	ac.write(*ac.instructionRegister)
}

func (ac *lcdAddressCounter) writeToRam() {
	if ac.toCGRAM {
		address := ac.getCGRAMIndex()
		ac.cgram[address] = *ac.dataRegister
	} else {
		address := ac.getDDRAMIndex()
		ac.ddram[address] = *ac.dataRegister
	}

	ac.moveCursorAndDisplay()
}

func (ac *lcdAddressCounter) readFromRam() {
	if ac.toCGRAM {
		address := ac.getCGRAMIndex()
		*ac.dataRegister = ac.cgram[address]
	} else {
		address := ac.getDDRAMIndex()
		*ac.dataRegister = ac.ddram[address]
	}

	ac.moveCursorAndDisplay()
}

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

/*
When N is 0 (1-line display), AC can be 00H to 4FH. When N is 1 (2-line display),
AC can be 00H to 27H for the first line, and 40H to 67H for the second line.
*/
func (ac *lcdAddressCounter) write(value uint8) {
	value &= 0x7F

	if ac.toCGRAM {
		value = value % 0x80
	} else {
		if ac.is2LineDisplay {
			if value >= 0x28 && value < 0x40 {
				value = (value % 0x28) + 0x40
			}

			if value >= 0x67 && value < 0x7F {
				value %= 0x68
			}
		} else {
			value %= 0x50
		}
	}

	ac.value = value
}

func (ac *lcdAddressCounter) read() uint8 {
	value := ac.value & 0x7F

	if ac.busy {
		value |= 0x80
	}

	return value
}
