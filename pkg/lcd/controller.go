package lcd

import (
	"time"

	"github.com/fran150/clementina6502/pkg/buses"
)

const (
	addressForInstructionMask uint8 = 0x80
	instructionBitID          uint8 = 0x02 // I/D: Increments (I/D = 1) or decrements (I/D = 0) the DDRAM address by 1 when a character code is written into or read
	instructionBitS           uint8 = 0x01 // S = 1: Display Shift, 0: Do Not Shift
	instructionBitD           uint8 = 0x04 // D = 1: Display On, 0: Display off
	instructionBitC           uint8 = 0x02 // C = 1: Show Cursor, 0: Do not show cursor
	instructionBitB           uint8 = 0x01 // B = 1: Character Blink, 0: Do not blink
	instructionBitSC          uint8 = 0x08 // S/C = 1: Display Shift, 0: Cursor Shift
	instructionBitRL          uint8 = 0x04 // R/L = 1: Shift to the Right, 1: Shift to the left
	instructionBitDL          uint8 = 0x10 // DL = 1: 8 bit mode, 0: 4 bit mode
	instructionBitN           uint8 = 0x08 // N = 1: 2 lines, 0: 1 lines
	instructionBitF           uint8 = 0x04 // F = 1: 5x10 font, 0: 5x8 font
)

const DDRAM_SIZE uint8 = 80   // DDRAM can store up to 80 characters
const CGRAM_SIZE uint8 = 64   // CGRAM can store 64 bytes for custome characters
const SPACE_CHAR uint8 = 0x20 // Value of the space character

// The HD44780U dot-matrix liquid crystal display controller and driver LSI displays alphanumerics and symbols.
// It can be configured to drive a dot-matrix liquid crystal display
// under the control of a 4- or 8-bit microprocessor. Since all the functions such as display RAM, character
// generator, and liquid crystal driver, required for driving a dot-matrix liquid crystal display are internally
// provided on one chip, a minimal system can be interfaced with this controller/driver.
type LcdHD44780U struct {
	dataRegisterSelected *buses.ConnectorEnabledHigh // 0: Instruction Register / 1: Data Register
	write                *buses.ConnectorEnabledLow  // R/W flag
	enable               *buses.ConnectorEnabledHigh // Chip enable
	dataBus              *buses.BusConnector[uint8]  // 8 Bit data bus

	addressCounter *lcdAddressCounter // Pointer to the internal address counter
	buffer         *LcdBuffer         // Pointer to the internal buffer

	instructionRegister uint8 // Store the last specified instruction
	dataRegister        uint8 // Store the last read or written data value

	displayOn      bool // D: Display is on / off
	displayCursor  bool // C: Shows cursor (line under current DDRAM address)
	characterBlink bool // B: Character blink (all dots alternates with character)
	is5x10Font     bool // F: Font size

	ddram [DDRAM_SIZE]uint8 // DDRAM stores the ASCII value of the character to display in the LCD
	cgram [CGRAM_SIZE]uint8 // Stores the data to define own custom characters

	timingConfig LcdTimingConfig // Allows to configure different value for the timing of the device operation

	isBusy       bool      // The LCD is busy
	busyStart    time.Time // Timestamp when the busy period started
	busyDuration int64     // Duration of the busy period

	blinkingVisible bool      // When blinking enabled this is true when the cursor must be shown. (Used to make cursor blinks)
	blinkingStart   time.Time // Time when the blinking period started

	instructions [8]func(time.Time) // Handlers for the different instructions that can be specified to the chip
}

// Creates the LCD controller chip
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

	lcd.addressCounter = createLCDAddressCounter(&lcd)

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

// Executes one emulation step
func (ctrl *LcdHD44780U) Tick(cycles uint64, t time.Time) {
	// Checks the status of the busy flag and the status of the blinking cursor
	ctrl.checkBusy(t)
	ctrl.cursorBlink(t)

	if ctrl.enable.Enabled() {
		if ctrl.write.Enabled() {
			// Push the value on the bus to the buffer
			ctrl.buffer.push(ctrl.dataBus.Read())

			// If the buffer is full it means we collected and entire 8 bit
			// instruction or data
			if ctrl.buffer.isFull() {
				if !ctrl.isBusy {
					// Execute action depending if the data is an instruction or data
					if ctrl.dataRegisterSelected.Enabled() {
						ctrl.dataRegister = ctrl.buffer.value
						ctrl.addressCounter.writeToRam()
					} else {
						ctrl.instructionRegister = ctrl.buffer.value
						ctrl.processInstruction(t)
					}
				}

				// Flush the buffer to wait for the next value
				ctrl.buffer.flush()
			}
		} else {
			// When reading, we wait for the value to be transferred to the bus
			// completely and the buffer to be empty
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

			// If buffer is not empty transfer to the data bus
			if !ctrl.buffer.isEmpty() {
				ctrl.dataBus.Write(ctrl.buffer.pull())
			}
		}
	}
}

// Puts the chip in "busy" state for the specified duration. While in busy state the chip will not
// respond to instructions or read / write requests
func (ctrl *LcdHD44780U) setBusy(duration int64, busyStart time.Time) {
	ctrl.isBusy = true
	ctrl.busyStart = busyStart
	ctrl.busyDuration = duration
}

// Checks if the busy period completed and if so, lowers the "busy" flag
func (ctrl *LcdHD44780U) checkBusy(t time.Time) {
	if ctrl.isBusy {
		elapsed := t.Sub(ctrl.busyStart).Microseconds()

		if elapsed >= ctrl.busyDuration {
			ctrl.isBusy = false
		}
	}
}

// Used to make the cursor blink, it changes the "blinkingVisible" value based on the
// configured blinking period
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

// Processes the specified instruction
func (ctrl *LcdHD44780U) processInstruction(t time.Time) {
	var mask uint8 = addressForInstructionMask
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

// Returns true if the value matches the specified mask
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

	for i := range DDRAM_SIZE {
		ctrl.ddram[i] = SPACE_CHAR
	}

	ctrl.addressCounter.mustMoveRight = true

	ctrl.returnHome(t)
}

// Return home sets DDRAM address 0 into the address counter, and returns the display to its original status
// if it was shifted. The DDRAM contents do not change.
// The cursor or blinking go to the left edge of the display (in the first line if 2 lines are displayed).
func (ctrl *LcdHD44780U) returnHome(t time.Time) {
	ctrl.setBusy(ctrl.timingConfig.returnHomeMicro, t)

	ctrl.addressCounter.value = 0x00
	ctrl.addressCounter.line1Shift = 0x00
	ctrl.addressCounter.line2Shift = 0x40
}

// I/D: Increments (I/D = 1) or decrements (I/D = 0) the DDRAM address by 1 when a character code is
// written into or read from DDRAM.
// The cursor or blinking moves to the right when incremented by 1 and to the left when decremented by 1.
// The same applies to writing and reading of CGRAM.
// S: Shifts the entire display either to the right (I/D = 0) or to the left (I/D = 1) when S is 1. The display does
// not shift if S is 0.
// If S is 1, it will seem as if the cursor does not move but the display does. The display does not shift when
// reading from DDRAM. Also, writing into or reading out from CGRAM does not shift the display.
func (ctrl *LcdHD44780U) entryModeSet(t time.Time) {
	ctrl.setBusy(ctrl.timingConfig.instructionMicro, t)

	// I/D = 1: Increment, 0: Decrement
	ctrl.addressCounter.mustMoveRight = checkBit(ctrl.instructionRegister, instructionBitID)
	// S = 1: Display Shift, 0: Do Not Shift
	ctrl.addressCounter.displayShift = checkBit(ctrl.instructionRegister, instructionBitS)
}

// D: The display is on when D is 1 and off when D is 0. When off, the display data remains in DDRAM, but
// can be displayed instantly by setting D to 1.
// C: The cursor is displayed when C is 1 and not displayed when C is 0. Even if the cursor disappears, the
// function of I/D or other specifications will not change during display data write.
// B: The character indicated by the cursor blinks when B is 1
func (ctrl *LcdHD44780U) displayOnOff(t time.Time) {
	ctrl.setBusy(ctrl.timingConfig.instructionMicro, t)

	// D = 1: Display On, 0: Display off
	ctrl.displayOn = checkBit(ctrl.instructionRegister, instructionBitD)
	// C = 1: Show Cursor, 0: Do not show cursor
	ctrl.displayCursor = checkBit(ctrl.instructionRegister, instructionBitC)
	// B = 1: Character Blink, 0: Do not blink
	ctrl.characterBlink = checkBit(ctrl.instructionRegister, instructionBitB)
}

// Cursor or display shift shifts the cursor position or display to the right or left without writing or reading
// display data. This function is used to correct or search the display. In a 2-line display, the cursor
// moves to the second line when it passes the 40th digit of the first line. Note that the first and second line
// displays will shift at the same time.
// When the displayed data is shifted repeatedly each line moves only horizontally. The second line display
// does not shift into the first line position.
// The address counter (AC) contents will not change if the only action performed is a display shift.
func (ctrl *LcdHD44780U) cursorDisplayShift(t time.Time) {
	ctrl.setBusy(ctrl.timingConfig.instructionMicro, t)

	displayShift := checkBit(ctrl.instructionRegister, instructionBitSC)
	directionRight := checkBit(ctrl.instructionRegister, instructionBitRL)

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

// DL: Sets the interface data length. Data is sent or received in 8-bit lengths (DB7 to DB0) when DL is 1,
// and in 4-bit lengths (DB7 to DB4) when DL is 0.When 4-bit length is selected, data must be sent or
// received twice.
// N: Sets the number of display lines.
// F: Sets the character font.
// Note: Perform the function at the head of the program before executing any instructions (except for the
// read busy flag and address instruction). From this point, the function set instruction cannot be
// executed unless the interface data length is changed.
func (ctrl *LcdHD44780U) functionSet(t time.Time) {
	ctrl.setBusy(ctrl.timingConfig.instructionMicro, t)

	ctrl.buffer.is8BitMode = checkBit(ctrl.instructionRegister, instructionBitDL)
	ctrl.addressCounter.is2LineDisplay = checkBit(ctrl.instructionRegister, instructionBitN)
	ctrl.is5x10Font = checkBit(ctrl.instructionRegister, instructionBitF)
}

// Sets the CGRAM address based on the value in the instruction register
func (ctrl *LcdHD44780U) setCGRAMAddress(t time.Time) {
	ctrl.setBusy(ctrl.timingConfig.instructionMicro, t)

	ctrl.addressCounter.setCGRAMAddress()
}

// Sets the DDRAM address based on the value in the instruction register
func (ctrl *LcdHD44780U) setDDRAMAddress(t time.Time) {
	ctrl.setBusy(ctrl.timingConfig.instructionMicro, t)

	ctrl.addressCounter.setDDRAMAddress()
}
