package lcd

import "github.com/fran150/clementina6502/buses"

const DDRAM_SIZE uint8 = 80
const CGRAM_SIZE uint8 = 64
const SPACE_CHAR uint8 = 0x20

type LcdHD44780U struct {
	dataRegisterSelected *buses.ConnectorEnabledHigh // 0: Instruction Register / 1: Data Register
	write                *buses.ConnectorEnabledLow
	enable               *buses.ConnectorEnabledHigh
	dataBus              *buses.BusConnector[uint8]

	shiftValue     uint8
	addressCounter uint8

	instructionRegister uint8
	dataRegister        uint8

	entryModeCursorDirectionR bool // D/R: Increments DDRAM or CGRAM when R or W.
	entryModeShiftDisplay     bool // S: Shifts the entire display
	displayOn                 bool // D: Display is on / off
	displayCursor             bool // C: Shows cursor (line under current DDRAM address)
	characterBlink            bool // B: Character blink (all dots alternates with character)
	dataLength8Bits           bool // DL: If false, data length is 4 bits so a full byte must be sent in 2 messages
	is2LineDisplay            bool // N: Number of lines
	is5x10Font                bool // F: Font size
	isBusy                    bool // BF: Busy Flag

	ram [CGRAM_SIZE + DDRAM_SIZE]uint8 // 0x00..0x3F CGRAM / 0x40..0x8F DDRAM
}

func (l *LcdHD44780U) tick(pbLine6Status bool) {
	if l.enable.Enabled() {
		if l.dataRegisterSelected.Enabled() {
			if l.write.Enabled() {
				l.dataRegister = l.dataBus.Read()
				l.ram[l.addressCounter] = l.dataRegister
			} else {
				l.dataRegister = l.ram[l.addressCounter]
				l.dataBus.Write(l.dataRegister)
			}
		} else {
			if l.write.Enabled() {
				l.instructionRegister = l.dataBus.Read()
			} else {
				l.instructionRegister = l.addressCounter & 0x7F

				if l.isBusy {
					l.instructionRegister |= 0x80
				}

				l.dataBus.Write(l.instructionRegister)
			}
		}
	}
}

func checkBit(value uint8, mask uint8) bool {
	return value&mask == mask
}

func clearDisplay(l *LcdHD44780U, params uint8) {
	for i := range DDRAM_SIZE {
		l.ram[i] = SPACE_CHAR
	}

	returnHome(l, params)
}

func returnHome(l *LcdHD44780U, params uint8) {
	l.shiftValue = 0x00
	l.addressCounter = 0x00
}

func entryModeSet(l *LcdHD44780U, params uint8) {
	l.entryModeCursorDirectionR = checkBit(params, 0x02)
	l.entryModeShiftDisplay = checkBit(params, 0x01)
}

func displayOnOff(l *LcdHD44780U, params uint8) {
	l.displayOn = checkBit(params, 0x04)
	l.displayCursor = checkBit(params, 0x02)
	l.characterBlink = checkBit(params, 0x01)
}

func (l *LcdHD44780U) setDDRAMAddressOnAC(address uint8) {
	l.addressCounter = address & DDRAM_SIZE
}

func (l *LcdHD44780U) setCGRAMAddressOnAC(address uint8) {
	l.addressCounter = address & CGRAM_SIZE
}

func cursorDisplayShift(l *LcdHD44780U, params uint8) {
	var target *uint8
	displayShift := checkBit(params, 0x08)

	if displayShift {
		target = &l.shiftValue
	} else {
		target = &l.addressCounter
	}

	moveRight := checkBit(params, 0x04)
	if moveRight {
		*target++
	} else {
		*target--
	}
}

func functionSet(l *LcdHD44780U, params uint8) {
	l.dataLength8Bits = checkBit(params, 0x10)
	l.is2LineDisplay = checkBit(params, 0x08)
	l.is5x10Font = checkBit(params, 0x04)
}

func setAddressCounter(l *LcdHD44780U, params uint8) {
	l.addressCounter = params & 0x7F
}

// 0000 0000 = 0x00 = 00
// 0010 0111 = 0x27 = 20
// 0100 0000 = 0x40 = 64
// 0110 0111 = 0x67 = 103

// 0000 0000 = 0x00 = 00
// 0100 0000 = 0x40 = 64
// 0011 1111 = 0x3F = 63
// 0111 1111 = 0x7F = 127
