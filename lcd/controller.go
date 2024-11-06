package lcd

import (
	"time"

	"github.com/fran150/clementina6502/buses"
)

const DDRAM_SIZE uint8 = 80
const CGRAM_SIZE uint8 = 64
const SPACE_CHAR uint8 = 0x20

type LcdHD44780U struct {
	dataRegisterSelected *buses.ConnectorEnabledHigh // 0: Instruction Register / 1: Data Register
	write                *buses.ConnectorEnabledLow
	enable               *buses.ConnectorEnabledHigh
	dataBus              *buses.BusConnector[uint8]

	addressCounter *lcdAddressCounter
	buffer         *LcdBuffer

	instructionRegister uint8
	dataRegister        uint8

	displayOn      bool // D: Display is on / off
	displayCursor  bool // C: Shows cursor (line under current DDRAM address)
	characterBlink bool // B: Character blink (all dots alternates with character)
	is5x10Font     bool // F: Font size

	ddram [DDRAM_SIZE]uint8
	cgram [CGRAM_SIZE]uint8

	timingConfig LcdTimingConfig

	isBusy       bool
	busyStart    time.Time
	busyDuration int64

	blinkingVisible bool
	blinkingStart   time.Time

	instructions [8]func(time.Time)
}

func CreateLCD() *LcdHD44780U {
	lcd := LcdHD44780U{
		dataRegisterSelected: buses.CreateConnectorEnabledHigh(),
		write:                buses.CreateConnectorEnabledLow(),
		enable:               buses.CreateConnectorEnabledHigh(),
		dataBus:              buses.CreateBusConnector[uint8](),
		buffer:               createLcdBuffer(),

		displayOn:      false,
		displayCursor:  false,
		characterBlink: false,
		is5x10Font:     false,

		timingConfig: LcdTimingConfig{
			clearDisplayMicro: 1520,   // 1.52 ms
			returnHomeMicro:   1520,   // 1.52 ms
			instructionMicro:  37,     // 37 us
			blinkingMicro:     400000, // 400 ms
		},

		isBusy:          false,
		blinkingVisible: false,
	}

	lcd.addressCounter = createLCDAdressCounter(&lcd)

	lcd.instructions = [8]func(time.Time){
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

func (ctrl *LcdHD44780U) Tick(cycles uint64, t time.Time, dt time.Duration) {
	ctrl.checkBusy(t)
	ctrl.cursorBlink(t)

	if ctrl.enable.Enabled() {
		if ctrl.write.Enabled() {
			ctrl.buffer.push(ctrl.dataBus.Read())

			if ctrl.buffer.isFull() {

				if !ctrl.isBusy {
					if ctrl.dataRegisterSelected.Enabled() {
						ctrl.dataRegister = ctrl.buffer.value
						ctrl.addressCounter.writeToRam()
					} else {
						ctrl.instructionRegister = ctrl.buffer.value
						ctrl.processInstruction(t)
					}
				}

				ctrl.buffer.flush()
			}
		} else {
			if ctrl.buffer.isEmpty() {
				if ctrl.dataRegisterSelected.Enabled() {
					if !ctrl.isBusy {
						ctrl.addressCounter.readFromRam()
						ctrl.buffer.load(ctrl.dataRegister)
					}
				} else {
					counter := ctrl.addressCounter.read()
					ctrl.buffer.load(counter)
				}
			}

			if !ctrl.buffer.isEmpty() {
				ctrl.dataBus.Write(ctrl.buffer.pull())
			}
		}
	}
}

func (ctrl *LcdHD44780U) setBusy(duration int64, busyStart time.Time) {
	ctrl.isBusy = true
	ctrl.busyStart = busyStart
	ctrl.busyDuration = duration
}

func (ctrl *LcdHD44780U) checkBusy(t time.Time) {
	if ctrl.isBusy {
		elapsed := t.Sub(ctrl.busyStart).Microseconds()

		if elapsed >= ctrl.busyDuration {
			ctrl.isBusy = false
		}
	}
}

func (ctrl *LcdHD44780U) cursorBlink(t time.Time) {
	if ctrl.blinkingStart.IsZero() {
		ctrl.blinkingStart = t
	}

	elapsed := t.Sub(ctrl.blinkingStart).Microseconds()

	if elapsed >= ctrl.timingConfig.blinkingMicro {
		expectedDuration := time.Microsecond * time.Duration(ctrl.timingConfig.blinkingMicro)
		ctrl.blinkingStart = ctrl.blinkingStart.Add(expectedDuration)

		ctrl.blinkingVisible = !ctrl.blinkingVisible
	}
}

func (ctrl *LcdHD44780U) processInstruction(t time.Time) {
	var mask uint8 = 0x80
	i := 7

	for mask > 0 {
		if checkBit(ctrl.instructionRegister, mask) {
			instruction := ctrl.instructions[i]
			instruction(t)
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
func (ctrl *LcdHD44780U) clearDisplay(t time.Time) {
	ctrl.setBusy(ctrl.timingConfig.clearDisplayMicro, t)

	for i := range 80 {
		ctrl.ddram[i] = SPACE_CHAR
	}

	ctrl.addressCounter.mustMoveRight = true

	ctrl.returnHome(t)
}

func (ctrl *LcdHD44780U) returnHome(t time.Time) {
	ctrl.setBusy(ctrl.timingConfig.returnHomeMicro, t)

	ctrl.addressCounter.value = 0x00
	ctrl.addressCounter.line1Shift = 0x00
	ctrl.addressCounter.line2Shift = 0x40
}

func (ctrl *LcdHD44780U) entryModeSet(t time.Time) {
	ctrl.setBusy(ctrl.timingConfig.instructionMicro, t)

	// I/D = 1: Increment, 0: Decrement
	ctrl.addressCounter.mustMoveRight = checkBit(ctrl.instructionRegister, 0x02)
	// S = 1: Display Shift, 0: Do Not Shift
	ctrl.addressCounter.displayShift = checkBit(ctrl.instructionRegister, 0x01)
}

func (ctrl *LcdHD44780U) displayOnOff(t time.Time) {
	ctrl.setBusy(ctrl.timingConfig.instructionMicro, t)

	// D = 1: Display On, 0: Display off
	ctrl.displayOn = checkBit(ctrl.instructionRegister, 0x04)
	// C = 1: Show Cursor, 0: Do not show cursor
	ctrl.displayCursor = checkBit(ctrl.instructionRegister, 0x02)
	// B = 1: Character Blink, 0: Do not blink
	ctrl.characterBlink = checkBit(ctrl.instructionRegister, 0x01)

}

func (ctrl *LcdHD44780U) cursorDisplayShift(t time.Time) {
	ctrl.setBusy(ctrl.timingConfig.instructionMicro, t)

	displayShift := checkBit(ctrl.instructionRegister, 0x08)
	directionRight := checkBit(ctrl.instructionRegister, 0x04)

	if displayShift {
		if directionRight {
			ctrl.addressCounter.shiftRight()
		} else {
			ctrl.addressCounter.shiftLeft()
		}
	} else {
		if directionRight {
			ctrl.addressCounter.moveRight()
		} else {
			ctrl.addressCounter.moveLeft()
		}
	}
}

func (ctrl *LcdHD44780U) functionSet(t time.Time) {
	ctrl.setBusy(ctrl.timingConfig.instructionMicro, t)

	ctrl.buffer.is8BitMode = checkBit(ctrl.instructionRegister, 0x10)
	ctrl.addressCounter.is2LineDisplay = checkBit(ctrl.instructionRegister, 0x08)
	ctrl.is5x10Font = checkBit(ctrl.instructionRegister, 0x04)
}

func (ctrl *LcdHD44780U) setCGRAMAddress(t time.Time) {
	ctrl.setBusy(ctrl.timingConfig.instructionMicro, t)

	ctrl.addressCounter.setCGRAMAddress()
}

func (ctrl *LcdHD44780U) setDDRAMAddress(t time.Time) {
	ctrl.setBusy(ctrl.timingConfig.instructionMicro, t)

	ctrl.addressCounter.setDDRAMAddress()
}
