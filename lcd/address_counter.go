package lcd

type lcdAddressCounter struct {
	toCGRAM        bool
	mustMoveRight  bool
	is2LineDisplay bool // N: Number of lines
	busy           bool
	value          uint8

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

		toCGRAM:       false,
		mustMoveRight: true,
		busy:          false,

		ddram: &lcd.ddram,
		cgram: &lcd.cgram,
	}
}

func (ac *lcdAddressCounter) valueToCGRAMIndex() uint8 {
	return ac.value & 0x3F
}

func (ac *lcdAddressCounter) valueToDDRMIndex() uint8 {
	if ac.value&0x40 == 0x40 {
		return (ac.value & 0x3F) + 40
	} else {
		return ac.value & 0x7F
	}
}

func (ac *lcdAddressCounter) moveRight() {
	ac.write(ac.value + 1)
}

func (ac *lcdAddressCounter) moveLeft() {
	ac.write(ac.value - 1)
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
		address := ac.valueToCGRAMIndex()
		ac.cgram[address] = *ac.dataRegister
	} else {
		address := ac.valueToDDRMIndex()
		ac.ddram[address] = *ac.dataRegister
	}

	if ac.mustMoveRight {
		ac.moveRight()
	} else {
		ac.moveLeft()
	}
}

func (ac *lcdAddressCounter) readFromRam() {
	if ac.toCGRAM {
		address := ac.valueToCGRAMIndex()
		*ac.dataRegister = ac.cgram[address]
	} else {
		address := ac.valueToDDRMIndex()
		*ac.dataRegister = ac.ddram[address]
	}

	if ac.mustMoveRight {
		ac.moveRight()
	} else {
		ac.moveLeft()
	}
}

/*
When N is 0 (1-line display), AC can be 00H to 4FH. When N is 1 (2-line display),
AC can be 00H to 27H for the first line, and 40H to 67H for the second line.
*/
func (ac *lcdAddressCounter) write(value uint8) {
	value &= 0x7F

	if ac.toCGRAM {
		value = value % 0x7F
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
