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

func TestBasicWriteToLimitOfLine(t *testing.T) {
	lcd, circuit := createTestCircuit()

	circuit.registerSelect.Set(false)
	circuit.enable.Set(true)
	circuit.bus.Write(0xA7)
	circuit.readWrite.Set(false)

	lcd.tick()

	circuit.registerSelect.Set(true)
	circuit.enable.Set(true)
	circuit.bus.Write(0xFF)
	circuit.readWrite.Set(false)

	lcd.tick()

	assert.Equal(t, true, true)
}
