package via

import (
	"math"
	"testing"

	"github.com/fran150/clementina6502/buses"
	"github.com/stretchr/testify/assert"
)

type testCircuit struct {
	cs1     buses.Line
	cs2     buses.Line
	ca1     buses.Line
	ca2     buses.Line
	cb1     buses.Line
	cb2     buses.Line
	dataBus *buses.Bus[uint8]
	irq     buses.Line
	portA   *buses.Bus[uint8]
	portB   *buses.Bus[uint8]
	reset   buses.Line
	rs      [4]buses.Line
	rw      buses.Line
}

func createTestCircuit() *testCircuit {
	var rsLines [4]buses.Line

	for i := range 4 {
		rsLines[i] = buses.CreateStandaloneLine(false)
	}

	return &testCircuit{
		cs1:     buses.CreateStandaloneLine(true),
		cs2:     buses.CreateStandaloneLine(false),
		ca1:     buses.CreateStandaloneLine(false),
		ca2:     buses.CreateStandaloneLine(false),
		cb1:     buses.CreateStandaloneLine(false),
		cb2:     buses.CreateStandaloneLine(false),
		dataBus: buses.CreateBus[uint8](),
		irq:     buses.CreateStandaloneLine(true),
		portA:   buses.CreateBus[uint8](),
		portB:   buses.CreateBus[uint8](),
		reset:   buses.CreateStandaloneLine(true),
		rs:      rsLines,
		rw:      buses.CreateStandaloneLine(true),
	}
}

func (circuit *testCircuit) wire(via *Via65C22S) {
	via.ChipSelect1().Connect(circuit.cs1)
	via.ChipSelect2().Connect(circuit.cs2)

	via.PeripheralAControlLines(0).Connect(circuit.ca1)
	via.PeripheralAControlLines(1).Connect(circuit.ca2)

	via.PeripheralBControlLines(0).Connect(circuit.cb1)
	via.PeripheralBControlLines(1).Connect(circuit.cb2)

	via.DataBus().Connect(circuit.dataBus)

	via.IrqRequest().Connect(circuit.irq)

	via.PeripheralPortA().Connect(circuit.portA)
	via.PeripheralPortB().Connect(circuit.portB)

	via.Reset().Connect(circuit.reset)

	via.ConnectRegisterSelectLines(circuit.rs)

	via.ReadWrite().Connect(circuit.rw)
}

func (circuit *testCircuit) setRegisterSelectValue(value viaRegisterCode) {
	value &= 0x0F

	for i := range 4 {
		if uint8(value)&uint8(math.Pow(2, float64(i))) > 0 {
			circuit.rs[i].Set(true)
		} else {
			circuit.rs[i].Set(false)
		}
	}
}

func writeToVia(via *Via65C22S, circuit *testCircuit, register viaRegisterCode, value uint8, t uint64) {
	circuit.setRegisterSelectValue(register)
	circuit.rw.Set(false)
	circuit.dataBus.Write(value)

	via.Tick(t)
	via.PostTick(t)
}

func readFromVia(via *Via65C22S, circuit *testCircuit, register viaRegisterCode, t uint64) uint8 {
	circuit.setRegisterSelectValue(register)
	circuit.rw.Set(true)

	via.Tick(t)
	via.PostTick(t)

	return circuit.dataBus.Read()
}

func TestLatchingRecord(t *testing.T) {
	via := CreateVia65C22()
	circuit := createTestCircuit()

	circuit.wire(via)

	// Set all pins on Port B to output
	writeToVia(via, circuit, dataDirectionRegisterB, 0xFF, 0)
	// Set PCR to latch on CB high
	writeToVia(via, circuit, peripheralControl, 0x10, 2)

	// Set output on port B to 0xAA
	writeToVia(via, circuit, ioRegisterB, 0xAA, 3)

	// Validate Port B output
	assert.Equal(t, uint8(0xAA), circuit.portB.Read())

	// Do something else not related to check that Port B output stays the same (Read ACR)
	acr := readFromVia(via, circuit, auxiliaryControl, 4)

	assert.Equal(t, uint8(0xAA), circuit.portB.Read())
	assert.Equal(t, uint8(0x00), acr)

	writeToVia(via, circuit, dataDirectionRegisterB, 0x0F, 5)

	assert.Equal(t, uint8(0x0A), circuit.portB.Read())
}
