package acia

import (
	"math"
	"testing"
	"time"

	"github.com/fran150/clementina6502/buses"
	"github.com/stretchr/testify/assert"
)

const TEST_PORT_NAME string = "/dev/ttys009"

type testCircuit struct {
	dataBus *buses.Bus[uint8]
	irq     buses.Line
	rw      buses.Line
	cs0     buses.Line
	cs1     buses.Line
	rs      [NUM_OF_RS_LINES]buses.Line
	reset   buses.Line
}

func createTestCircuit() *testCircuit {
	var rsLines [NUM_OF_RS_LINES]buses.Line

	for i := range NUM_OF_RS_LINES {
		rsLines[i] = buses.CreateStandaloneLine(false)
	}

	return &testCircuit{
		dataBus: buses.CreateBus[uint8](),
		irq:     buses.CreateStandaloneLine(true),
		rw:      buses.CreateStandaloneLine(true),
		cs0:     buses.CreateStandaloneLine(true),
		cs1:     buses.CreateStandaloneLine(false),
		rs:      rsLines,
		reset:   buses.CreateStandaloneLine(true),
	}
}

func (circuit *testCircuit) wire(acia *Acia65C51N) {
	acia.DataBus().Connect(circuit.dataBus)

	acia.IrqRequest().Connect(circuit.irq)

	acia.ReadWrite().Connect(circuit.rw)

	acia.ChipSelect0().Connect(circuit.cs0)
	acia.ChipSelect1().Connect(circuit.cs1)

	acia.ConnectRegisterSelectLines(circuit.rs)

	acia.Reset().Connect(circuit.reset)
}

func (circuit *testCircuit) setRegisterSelectValue(value uint8) {
	value &= 0x03

	for i := range NUM_OF_RS_LINES {
		if uint8(value)&uint8(math.Pow(2, float64(i))) > 0 {
			circuit.rs[i].Set(true)
		} else {
			circuit.rs[i].Set(false)
		}
	}
}

func writeToAcia(acia *Acia65C51N, circuit *testCircuit, register uint8, value uint8, t *uint64) {
	circuit.setRegisterSelectValue(register)
	circuit.rw.Set(false)
	circuit.dataBus.Write(value)

	acia.Tick(*t, time.Now(), 0)

	*t = *t + 1
}

func readFromAcia(acia *Acia65C51N, circuit *testCircuit, register uint8, t *uint64) uint8 {
	circuit.setRegisterSelectValue(register)
	circuit.rw.Set(true)

	acia.Tick(*t, time.Now(), 0)

	*t = *t + 1

	return circuit.dataBus.Read()
}

func disableChipAndStepTime(acia *Acia65C51N, circuit *testCircuit, t *uint64) {
	circuit.cs0.Set(false)
	circuit.cs1.Set(true)

	acia.Tick(*t, time.Now(), 0)

	*t = *t + 1
}

func enableChip(circuit *testCircuit) {
	circuit.cs0.Set(true)
	circuit.cs1.Set(false)
}

func TestWriteToControlRegister(t *testing.T) {
	var step uint64

	acia := createAcia65C51N(TEST_PORT_NAME)
	circuit := createTestCircuit()

	circuit.wire(acia)

	writeToAcia(acia, circuit, 0x03, 0xFF, &step)

	assert.Equal(t, uint8(0xFF), acia.controlRegister)
}

func TestWriteToTX(t *testing.T) {
	var step uint64

	acia := createAcia65C51N(TEST_PORT_NAME)
	circuit := createTestCircuit()

	circuit.wire(acia)

	var data string = "Hello World!!!"

	for _, char := range data {
		writeToAcia(acia, circuit, 0x00, uint8(char), &step)
		time.Sleep(time.Duration(100 * time.Millisecond))
	}
}
