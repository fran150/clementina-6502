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

	addressCounter *lcdAddressCounter

	instructionRegister uint8
	dataRegister        uint8
	shift               uint8

	is4BitMode            bool
	entryModeShiftDisplay bool // S: Shifts the entire display
	displayOn             bool // D: Display is on / off
	displayCursor         bool // C: Shows cursor (line under current DDRAM address)
	characterBlink        bool // B: Character blink (all dots alternates with character)
	is5x10Font            bool // F: Font size
	isBusy                bool // BF: Busy Flag

	ddram [DDRAM_SIZE]uint8
	cgram [CGRAM_SIZE]uint8

	instructions [8]func()
}

func CreateLCD() *LcdHD44780U {
	lcd := LcdHD44780U{
		dataRegisterSelected: buses.CreateConnectorEnabledHigh(),
		write:                buses.CreateConnectorEnabledLow(),
		enable:               buses.CreateConnectorEnabledHigh(),
		dataBus:              buses.CreateBusConnector[uint8](),

		is4BitMode: false,
	}

	lcd.addressCounter = createLCDAdressCounter(&lcd)

	lcd.instructions = [8]func(){
		lcd.clearDisplay,
		lcd.returnHome,
		lcd.entryModeSet,
		lcd.displayOnOff,
		lcd.cursorDisplayShift,
		lcd.functionSet,
		lcd.setCGRAMAddress,
		lcd.setDDRAMAddress,
	}

	return &lcd
}

func (ctrl *LcdHD44780U) tick() {
	if ctrl.enable.Enabled() {
		if ctrl.dataRegisterSelected.Enabled() {
			if ctrl.write.Enabled() {
				ctrl.dataRegister = ctrl.dataBus.Read()
				ctrl.addressCounter.writeToRam()
			} else {
				ctrl.addressCounter.readFromRam()
				ctrl.dataBus.Write(ctrl.dataRegister)
			}
		} else {
			if ctrl.write.Enabled() {
				ctrl.instructionRegister = ctrl.dataBus.Read()
				ctrl.processInstruction()
			} else {
				ctrl.addressCounter.read()
				ctrl.dataBus.Write(ctrl.dataRegister)
			}
		}
	}
}

func (ctrl *LcdHD44780U) processInstruction() {
	var mask uint8 = 0x80
	i := 7

	for mask > 0 {
		if checkBit(ctrl.instructionRegister, mask) {
			instruction := ctrl.instructions[i]
			instruction()
			break
		}

		i = i - 1
		mask = mask >> 1
	}
}

func checkBit(value uint8, mask uint8) bool {
	return value&mask == mask
}

/*
Clear display writes space code 20H (character pattern for character code 20H must be a blank pattern) into
all DDRAM addresses. It then sets DDRAM address 0 into the address counter, and returns the display to
its original status if it was shifted. In other words, the display disappears and the cursor or blinking goes to
the left edge of the display (in the first line if 2 lines are displayed). It also sets I/D to 1 (increment mode)
in entry mode. S of entry mode does not change.
*/
func (ctrl *LcdHD44780U) clearDisplay() {
	for i := range 80 {
		ctrl.ddram[i] = SPACE_CHAR
	}

	ctrl.addressCounter.mustMoveRight = true

	ctrl.returnHome()
}

func (ctrl *LcdHD44780U) returnHome() {
	ctrl.instructionRegister = 0x00
	ctrl.shift = 0x00
}

func (ctrl *LcdHD44780U) entryModeSet() {
	ctrl.addressCounter.mustMoveRight = checkBit(ctrl.instructionRegister, 0x02)
	ctrl.entryModeShiftDisplay = checkBit(ctrl.instructionRegister, 0x01)
}

func (ctrl *LcdHD44780U) displayOnOff() {
	ctrl.displayOn = checkBit(ctrl.instructionRegister, 0x04)
	ctrl.displayCursor = checkBit(ctrl.instructionRegister, 0x02)
	ctrl.characterBlink = checkBit(ctrl.instructionRegister, 0x01)

}

func (ctrl *LcdHD44780U) cursorDisplayShift() {
	//displayShift := checkBit(ctrl.instructionRegister, 0x08)
	directionRight := checkBit(ctrl.instructionRegister, 0x04)

	if directionRight {
		ctrl.addressCounter.moveRight()
	} else {
		ctrl.addressCounter.moveLeft()
	}
}

func (ctrl *LcdHD44780U) functionSet() {
	ctrl.is4BitMode = !checkBit(ctrl.instructionRegister, 0x10)
	ctrl.addressCounter.is2LineDisplay = checkBit(ctrl.instructionRegister, 0x80)
	ctrl.is5x10Font = checkBit(ctrl.instructionRegister, 0x04)
}

func (ctrl *LcdHD44780U) setCGRAMAddress() {
	ctrl.addressCounter.setCGRAMAddress()
}

func (ctrl *LcdHD44780U) setDDRAMAddress() {
	ctrl.addressCounter.setDDRAMAddress()
}
