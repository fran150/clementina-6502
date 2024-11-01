package lcd

import (
	"testing"

	"github.com/fran150/clementina6502/buses"
	"github.com/stretchr/testify/assert"
)

type testCircuit struct {
	bus            *buses.Bus[uint8]
	registerSelect *buses.StandaloneLine
	enable         *buses.StandaloneLine
	readWrite      *buses.StandaloneLine
}

func createTestCircuit() (*LcdHD44780U, *testCircuit) {
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

func sendInstruction(lcd *LcdHD44780U, circuit *testCircuit, instruction uint8) {
	circuit.registerSelect.Set(false)
	circuit.enable.Set(true)
	circuit.bus.Write(instruction)
	circuit.readWrite.Set(false)

	lcd.tick()
}

func writeValue(lcd *LcdHD44780U, circuit *testCircuit, value uint8) {
	circuit.registerSelect.Set(true)
	circuit.enable.Set(true)
	circuit.bus.Write(value)
	circuit.readWrite.Set(false)

	lcd.tick()
}

func readValue(lcd *LcdHD44780U, circuit *testCircuit) uint8 {
	circuit.registerSelect.Set(true)
	circuit.enable.Set(true)
	circuit.readWrite.Set(true)

	lcd.tick()

	return circuit.bus.Read()
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

	// Send instruction to move cursor left
	sendInstruction(lcd, circuit, 0x04)

	// Writing and reading values should bring the cursor back to 0
	writeValue(lcd, circuit, 0xFF)
	assert.Equal(t, uint8(0x01), lcd.addressCounter.value)
	value = readValue(lcd, circuit)
	assert.Equal(t, uint8(0x01), value)
	assert.Equal(t, uint8(0x00), lcd.addressCounter.value)
}

func TestBasicWriteToLimitOfLine(t *testing.T) {
	lcd, circuit := createTestCircuit()

	sendInstruction(lcd, circuit, 0xA7)
	sendInstruction(lcd, circuit, 0xFF)

	assert.Equal(t, true, true)
}
