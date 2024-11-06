package lcd

import (
	"testing"
	"time"

	"github.com/fran150/clementina6502/buses"
	"github.com/stretchr/testify/assert"
)

type testCircuit struct {
	bus            *buses.Bus[uint8]
	registerSelect *buses.StandaloneLine
	enable         *buses.StandaloneLine
	readWrite      *buses.StandaloneLine
}

func createTestCircuitCorrectTiming() (*LcdHD44780U, *testCircuit) {
	lcd := CreateLCD()

	circuit := testCircuit{
		bus:            buses.CreateBus[uint8](),
		registerSelect: buses.CreateStandaloneLine(false),
		enable:         buses.CreateStandaloneLine(false),
		readWrite:      buses.CreateStandaloneLine(false),
	}

	lcd.dataBus.Connect(circuit.bus)
	lcd.dataRegisterSelected.Connect(circuit.registerSelect)
	lcd.enable.Connect(circuit.enable)
	lcd.write.Connect(circuit.readWrite)

	return lcd, &circuit
}

func createTestCircuit() (*LcdHD44780U, *testCircuit) {
	lcd, circuit := createTestCircuitCorrectTiming()

	lcd.timingConfig.clearDisplayMicro = 0
	lcd.timingConfig.returnHomeMicro = 0
	lcd.timingConfig.instructionMicro = 0

	return lcd, circuit
}

func readInstruction(lcd *LcdHD44780U, circuit *testCircuit) uint8 {
	circuit.registerSelect.Set(false)
	circuit.enable.Set(true)
	circuit.readWrite.Set(true)

	t := time.Now()
	lcd.Tick(0, t, 0)

	return circuit.bus.Read()
}

func sendInstruction(lcd *LcdHD44780U, circuit *testCircuit, instruction uint8) {
	circuit.registerSelect.Set(false)
	circuit.enable.Set(true)
	circuit.bus.Write(instruction)
	circuit.readWrite.Set(false)

	t := time.Now()
	lcd.Tick(0, t, 0)
}

func writeValue(lcd *LcdHD44780U, circuit *testCircuit, value uint8) {
	circuit.registerSelect.Set(true)
	circuit.enable.Set(true)
	circuit.bus.Write(value)
	circuit.readWrite.Set(false)

	t := time.Now()
	lcd.Tick(0, t, 0)
}

func readValue(lcd *LcdHD44780U, circuit *testCircuit) uint8 {
	circuit.registerSelect.Set(true)
	circuit.enable.Set(true)
	circuit.readWrite.Set(true)

	t := time.Now()
	lcd.Tick(0, t, 0)

	return circuit.bus.Read()
}

type instructionValidator[T bool | uint8] struct {
	t       *testing.T
	lcd     *LcdHD44780U
	circuit *testCircuit
	fields  []*T
}

func createInstructionValidator[T bool | uint8](t *testing.T, lcd *LcdHD44780U, circuit *testCircuit, fields ...*T) *instructionValidator[T] {
	return &instructionValidator[T]{
		t,
		lcd,
		circuit,
		fields,
	}
}

func (val *instructionValidator[T]) send(instruction uint8) {
	sendInstruction(val.lcd, val.circuit, instruction)
}

func (val *instructionValidator[T]) validate(values ...T) {
	for i := range values {
		assert.Equal(val.t, values[i], *val.fields[i])
	}
}

func TestClearDisplayInstruction(t *testing.T) {
	lcd, circuit := createTestCircuit()

	// Fill DDRAM with 1s
	for i := range DDRAM_SIZE {
		lcd.ddram[i] = 0xFF
	}

	// Move cursor to the end of first line of a 1 line mode display
	lcd.addressCounter.value = 0x3F
	// Enable display shift
	lcd.addressCounter.displayShift = true
	// Simulate display shifted 2 positions
	lcd.addressCounter.line1Shift = 0x02
	lcd.addressCounter.line2Shift = 0x42
	// Set cursor to move left
	lcd.addressCounter.mustMoveRight = false

	// Send clear instruction
	sendInstruction(lcd, circuit, 0x01)

	// All memory must be filled with spaces
	for i := range DDRAM_SIZE {
		assert.Equal(t, SPACE_CHAR, lcd.ddram[i], "The value in memory position %v is incorrect", i)
	}

	// Cursor and shift are reset, cursos direction is move right. Shift configuration remains untouched.
	assert.Equal(t, uint8(0x00), lcd.addressCounter.value)
	assert.Equal(t, uint8(0x00), lcd.addressCounter.line1Shift)
	assert.Equal(t, uint8(0x40), lcd.addressCounter.line2Shift)
	assert.Equal(t, true, lcd.addressCounter.mustMoveRight)
	assert.Equal(t, true, lcd.addressCounter.displayShift)
}

func TestReturnHomeInstruction(t *testing.T) {
	lcd, circuit := createTestCircuit()

	// Fill DDRAM with 1s
	for i := range DDRAM_SIZE {
		lcd.ddram[i] = 0xFF
	}

	// Move cursor to the end of first line of a 1 line mode display
	lcd.addressCounter.value = 0x3F
	// Enable display shift
	lcd.addressCounter.displayShift = true
	// Simulate display shifted 2 positions
	lcd.addressCounter.line1Shift = 0x02
	lcd.addressCounter.line2Shift = 0x42
	// Set cursor to move left
	lcd.addressCounter.mustMoveRight = false

	// Send return home instruction
	sendInstruction(lcd, circuit, 0x02)

	// Memory must not change
	for i := range DDRAM_SIZE {
		assert.Equal(t, uint8(0xFF), lcd.ddram[i], "The value in memory position %v is incorrect", i)
	}

	// Cursor and shift are reset, other parameters remain untouched
	assert.Equal(t, uint8(0x00), lcd.addressCounter.value)
	assert.Equal(t, uint8(0x00), lcd.addressCounter.line1Shift)
	assert.Equal(t, uint8(0x40), lcd.addressCounter.line2Shift)
	assert.Equal(t, false, lcd.addressCounter.mustMoveRight)
	assert.Equal(t, true, lcd.addressCounter.displayShift)
}

func TestEntryModeSetInstruction(t *testing.T) {
	lcd, circuit := createTestCircuit()

	// Fill the DDRAM with consecutive values
	for i := range DDRAM_SIZE {
		lcd.ddram[i] = i
	}

	// Check default values for I/D and S
	assert.Equal(t, true, lcd.addressCounter.mustMoveRight)
	assert.Equal(t, false, lcd.addressCounter.displayShift)

	// Reading from memory should move the cursor right
	value := readValue(lcd, circuit)
	assert.Equal(t, uint8(0x00), value)
	value = readValue(lcd, circuit)
	assert.Equal(t, uint8(0x01), value)

	// Send instruction to write left to right
	sendInstruction(lcd, circuit, 0x04)

	// Writing and reading values should bring the cursor back to 0
	writeValue(lcd, circuit, 0xFF)
	assert.Equal(t, uint8(0x01), lcd.addressCounter.value)
	value = readValue(lcd, circuit)
	assert.Equal(t, uint8(0x01), value)
	assert.Equal(t, uint8(0x00), lcd.addressCounter.value)
}

func TestDisplayControl(t *testing.T) {
	lcd, circuit := createTestCircuit()

	// Check default values for D, C and B flags
	assert.Equal(t, false, lcd.displayOn)
	assert.Equal(t, false, lcd.displayCursor)
	assert.Equal(t, false, lcd.characterBlink)

	validator := createInstructionValidator[bool](t, lcd, circuit,
		&lcd.displayOn,
		&lcd.displayCursor,
		&lcd.characterBlink,
	)

	// Set display on
	validator.send(0x0C)
	validator.validate(true, false, false)

	// Add cursor display
	validator.send(0x0E)
	validator.validate(true, true, false)

	// Add character blink
	validator.send(0x0F)
	validator.validate(true, true, true)

	// Turn off display and cursor
	validator.send(0x09)
	validator.validate(false, false, true)
}

func TestFunctionSet(t *testing.T) {
	lcd, circuit := createTestCircuit()

	// Check default values for DL, N and F flags
	assert.Equal(t, true, lcd.buffer.is8BitMode)
	assert.Equal(t, false, lcd.addressCounter.is2LineDisplay)
	assert.Equal(t, false, lcd.is5x10Font)

	validator := createInstructionValidator[bool](t, lcd, circuit,
		&lcd.buffer.is8BitMode,
		&lcd.addressCounter.is2LineDisplay,
		&lcd.is5x10Font,
	)

	// Keep only 8 bit mode on
	validator.send(0x30)
	validator.validate(true, false, false)

	// Make it 2 line display
	validator.send(0x38)
	validator.validate(true, true, false)

	// Add 5x10 fonts
	validator.send(0x3F)
	validator.validate(true, true, true)

	// Disable all
	validator.send(0x20)
	validator.validate(false, false, false)

	// Send 0x33 in 4 bit mode, returning to 8 bit mode
	// only high nibble is used
	validator.send(0x30)
	validator.send(0x30)
	validator.validate(true, false, false)
}

func TestCursorShift1Line(t *testing.T) {
	lcd, circuit := createTestCircuit()

	lcd.addressCounter.is2LineDisplay = false

	assert.Equal(t, uint8(0x00), lcd.addressCounter.value)
	assert.Equal(t, uint8(0x00), lcd.addressCounter.line1Shift)
	assert.Equal(t, uint8(0x40), lcd.addressCounter.line2Shift)

	validator := createInstructionValidator[uint8](t, lcd, circuit,
		&lcd.addressCounter.value,
		&lcd.addressCounter.line1Shift,
		&lcd.addressCounter.line2Shift,
	)

	// Move cursor right all the way to 0x4F
	for pos := range uint8(79) {
		validator.send(0x14)
		validator.validate((pos + 1), 0x00, 0x40)
	}

	// From 0x4F moving to the right goes to 0x00 on 1 line display
	validator.send(0x14)
	validator.validate(0x00, 0x00, 0x40)

	// From 0x00 moving to the left goes to 0x4F on 1 line display
	validator.send(0x10)
	validator.validate(0x4F, 0x00, 0x40)

	// Move cursor left all the way to 0x00
	for pos := range uint8(79) {
		validator.send(0x10)
		validator.validate((0x4E - pos), 0x00, 0x40)
	}
}

func TestCursorShift2Line(t *testing.T) {
	lcd, circuit := createTestCircuit()

	lcd.addressCounter.is2LineDisplay = true

	assert.Equal(t, uint8(0x00), lcd.addressCounter.value)
	assert.Equal(t, uint8(0x00), lcd.addressCounter.line1Shift)
	assert.Equal(t, uint8(0x40), lcd.addressCounter.line2Shift)

	validator := createInstructionValidator[uint8](t, lcd, circuit,
		&lcd.addressCounter.value,
		&lcd.addressCounter.line1Shift,
		&lcd.addressCounter.line2Shift,
	)

	// Move cursor right all the way to 0x27
	for pos := range uint8(39) {
		validator.send(0x14)
		validator.validate((pos + 1), 0x00, 0x40)
	}

	// From 0x27 moving to the right goes to 0x40 on 2 line display
	validator.send(0x14)
	validator.validate(0x40, 0x00, 0x40)

	// Move cursor right all the way to 0x67
	for pos := range uint8(39) {
		validator.send(0x14)
		validator.validate((pos + 0x41), 0x00, 0x40)
	}

	// From 0x67 moving to the right goes to 0x00 on 2 line display
	validator.send(0x14)
	validator.validate(0x00, 0x00, 0x40)

	// From 0x00 moving to the left goes to 0x67 on 2 line display
	validator.send(0x10)
	validator.validate(0x67, 0x00, 0x40)

	// Move cursor left all the way to 0x40
	for pos := range uint8(39) {
		validator.send(0x10)
		validator.validate((0x66 - pos), 0x00, 0x40)
	}

	// From 0x40 moving to the left goes to 0x27 on 2 line display
	validator.send(0x10)
	validator.validate(0x27, 0x00, 0x40)

	// Move cursor left all the way to 0x00
	for pos := range uint8(39) {
		validator.send(0x10)
		validator.validate((0x26 - pos), 0x00, 0x40)
	}
}

func TestDisplayShift1Line(t *testing.T) {
	lcd, circuit := createTestCircuit()

	lcd.addressCounter.is2LineDisplay = false

	assert.Equal(t, uint8(0x00), lcd.addressCounter.value)
	assert.Equal(t, uint8(0x00), lcd.addressCounter.line1Shift)
	assert.Equal(t, uint8(0x40), lcd.addressCounter.line2Shift)

	validator := createInstructionValidator[uint8](t, lcd, circuit,
		&lcd.addressCounter.value,
		&lcd.addressCounter.line1Shift,
	)

	// Move display right all the way to 0x4F
	for pos := range uint8(79) {
		validator.send(0x1C)
		validator.validate(0x00, (pos + 1))
	}

	// From 0x4F moving to the right goes to 0x00 on 1 line display
	validator.send(0x1C)
	validator.validate(0x00, 0x00)

	// From 0x00 moving to the left goes to 0x4F on 1 line display
	validator.send(0x18)
	validator.validate(0x00, 0x4F)

	// Move display left all the way to 0x00
	for pos := range uint8(79) {
		validator.send(0x18)
		validator.validate(0x00, (0x4E - pos))
	}
}

func TestDisplayShift2Line(t *testing.T) {
	lcd, circuit := createTestCircuit()

	lcd.addressCounter.is2LineDisplay = true

	assert.Equal(t, uint8(0x00), lcd.addressCounter.value)
	assert.Equal(t, uint8(0x00), lcd.addressCounter.line1Shift)
	assert.Equal(t, uint8(0x40), lcd.addressCounter.line2Shift)

	validator := createInstructionValidator[uint8](t, lcd, circuit,
		&lcd.addressCounter.value,
		&lcd.addressCounter.line1Shift,
		&lcd.addressCounter.line2Shift,
	)

	// Move display right all the way to 0x27 and 0x67
	for pos := range uint8(39) {
		validator.send(0x1C)
		validator.validate(0x00, (pos + 1), (pos + 0x41))
	}

	// From 0x27 and 0x67 moving to the right goes to 0x00 and 0x40 on 2 line display
	validator.send(0x1C)
	validator.validate(0x00, 0x00, 0x40)

	// From 0x00 and 0x40 moving to the left goes to 0x27 and 0x67 on 2 line display
	validator.send(0x18)
	validator.validate(0x00, 0x27, 0x67)

	// Move cursor left all the way to 0x40
	for pos := range uint8(39) {
		validator.send(0x18)
		validator.validate(0x00, (0x26 - pos), (0x66 - pos))
	}
}

func TestSetCGRAMAddress(t *testing.T) {
	lcd, circuit := createTestCircuit()

	// Set CGRAM instruction bit
	const SET_CGRAM uint8 = 0x40

	for address := range CGRAM_SIZE {
		sendInstruction(lcd, circuit, SET_CGRAM|address)

		assert.Equal(t, 0x40+address, lcd.addressCounter.value)
		assert.Equal(t, true, lcd.addressCounter.toCGRAM)
	}

	// One more sends it back to 0x40
	sendInstruction(lcd, circuit, SET_CGRAM|CGRAM_SIZE)
	assert.Equal(t, uint8(0x40), lcd.addressCounter.value)
}

func TestSetDDRAMAddress1Line(t *testing.T) {
	lcd, circuit := createTestCircuit()

	// Set DDRAM instruction bit
	const SET_DDRAM uint8 = 0x80

	for address := range DDRAM_SIZE {
		sendInstruction(lcd, circuit, SET_DDRAM|address)

		assert.Equal(t, address, lcd.addressCounter.value)
		assert.Equal(t, false, lcd.addressCounter.toCGRAM)
	}

	// One more sends it back to 0x00
	sendInstruction(lcd, circuit, SET_DDRAM|DDRAM_SIZE)
	assert.Equal(t, uint8(0x00), lcd.addressCounter.value)
}

func TestSetDDRAMAddress2Line(t *testing.T) {
	lcd, circuit := createTestCircuit()

	lcd.addressCounter.is2LineDisplay = true

	// Set DDRAM instruction bit
	const SET_DDRAM uint8 = 0x80

	for address := range (DDRAM_SIZE / 2) - 1 {
		sendInstruction(lcd, circuit, SET_DDRAM|address)

		assert.Equal(t, address, lcd.addressCounter.value)
		assert.Equal(t, false, lcd.addressCounter.toCGRAM)
	}

	// One more sends it back to second line at 0x40
	sendInstruction(lcd, circuit, SET_DDRAM|0x28)
	assert.Equal(t, uint8(0x40), lcd.addressCounter.value)

	for address := range (DDRAM_SIZE / 2) - 1 {
		sendInstruction(lcd, circuit, SET_DDRAM|(address+0x40))

		assert.Equal(t, (address + 0x40), lcd.addressCounter.value)
		assert.Equal(t, false, lcd.addressCounter.toCGRAM)
	}

	// One more sends it back to 0x00
	sendInstruction(lcd, circuit, SET_DDRAM|0x68)
	assert.Equal(t, uint8(0x00), lcd.addressCounter.value)
}

func TestReadAddressCounter(t *testing.T) {
	lcd, circuit := createTestCircuit()

	// Set DDRAM instruction bit
	const SET_DDRAM uint8 = 0x80

	sendInstruction(lcd, circuit, SET_DDRAM|0x4F)

	value := readInstruction(lcd, circuit)

	assert.Equal(t, uint8(0x4F), value)
}

func TestWriteAndReadCGRAM(t *testing.T) {
	lcd, circuit := createTestCircuit()

	// Set AC to write to CGRAM address 0x00
	sendInstruction(lcd, circuit, 0x40)

	for value := range uint8(CGRAM_SIZE) {
		writeValue(lcd, circuit, value)
	}

	for value := range uint8(CGRAM_SIZE) {
		assert.Equal(t, value, lcd.cgram[value])
	}

	// Set AC to set cursor to CGRAM address 0x00
	sendInstruction(lcd, circuit, 0x40)

	for expected := range uint8(CGRAM_SIZE) {
		value := readValue(lcd, circuit)
		assert.Equal(t, expected, value)
	}
}

func TestWriteAndReadDDRAM1Line(t *testing.T) {
	lcd, circuit := createTestCircuit()

	// Set AC to write to DDRAM address 0x00
	sendInstruction(lcd, circuit, 0x80)

	for value := range uint8(DDRAM_SIZE) {
		writeValue(lcd, circuit, value)
	}

	for value := range uint8(DDRAM_SIZE) {
		assert.Equal(t, value, lcd.ddram[value])
	}

	// Set AC to set cursor to DDRAM address 0x00
	sendInstruction(lcd, circuit, 0x80)

	for expected := range uint8(DDRAM_SIZE) {
		value := readValue(lcd, circuit)
		assert.Equal(t, expected, value)
	}
}

func TestWriteAndReadDDRAM2Lines(t *testing.T) {
	lcd, circuit := createTestCircuit()

	lcd.addressCounter.is2LineDisplay = true

	// Set AC to write to DDRAM address 0x00
	sendInstruction(lcd, circuit, 0x80)

	for value := range uint8(DDRAM_SIZE) {
		writeValue(lcd, circuit, value)
	}

	for value := range uint8(DDRAM_SIZE) {
		assert.Equal(t, value, lcd.ddram[value])
	}

	// Set AC to set cursor to DDRAM address 0x00
	sendInstruction(lcd, circuit, 0x80)

	for expected := range uint8(DDRAM_SIZE) {
		value := readValue(lcd, circuit)
		assert.Equal(t, expected, value)
	}

	// In a 2 line display, 0x40 should contain the 40th value
	sendInstruction(lcd, circuit, 0xC0)
	value := readValue(lcd, circuit)
	assert.Equal(t, uint8(0x28), value)
}

func TestBusyFlag(t *testing.T) {
	lcd, circuit := createTestCircuitCorrectTiming()

	// Send clear display instruction
	ti := time.Now()
	sendInstruction(lcd, circuit, 0x01)

	isBusy := true

	for isBusy {
		value := readInstruction(lcd, circuit)
		isBusy = (value & 0x80) == 0x80
	}

	elapsed := time.Since(ti).Microseconds()
	assert.GreaterOrEqual(t, elapsed, lcd.timingConfig.clearDisplayMicro)

	// Send return home instruction
	ti = time.Now()
	sendInstruction(lcd, circuit, 0x02)

	isBusy = true

	for isBusy {
		value := readInstruction(lcd, circuit)
		isBusy = (value & 0x80) == 0x80
	}

	elapsed = time.Since(ti).Microseconds()
	assert.GreaterOrEqual(t, elapsed, lcd.timingConfig.clearDisplayMicro)

	// Regular instruction timing test
	ti = time.Now()
	sendInstruction(lcd, circuit, 0x02)

	isBusy = true

	for isBusy {
		value := readInstruction(lcd, circuit)
		isBusy = (value & 0x80) == 0x80
	}

	elapsed = time.Since(ti).Microseconds()
	assert.GreaterOrEqual(t, elapsed, lcd.timingConfig.clearDisplayMicro)
}

func TestCursorBlinking(t *testing.T) {
	lcd, _ := createTestCircuitCorrectTiming()

	ti := time.Now()

	for !lcd.blinkingVisible {
		lcd.Tick(0, time.Now(), time.Since(ti))
	}

	for lcd.blinkingVisible {
		lcd.Tick(0, time.Now(), time.Since(ti))
	}

	elapsed := time.Since(ti).Microseconds()
	assert.GreaterOrEqual(t, elapsed, lcd.timingConfig.blinkingMicro*2)
}
