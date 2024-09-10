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

func writeToVia(via *Via65C22S, circuit *testCircuit, register viaRegisterCode, value uint8, t *uint64) {
	circuit.setRegisterSelectValue(register)
	circuit.rw.Set(false)
	circuit.dataBus.Write(value)

	via.Tick(*t)
	via.PostTick(*t)

	*t = *t + 1
}

func readFromVia(via *Via65C22S, circuit *testCircuit, register viaRegisterCode, t *uint64) uint8 {
	circuit.setRegisterSelectValue(register)
	circuit.rw.Set(true)

	via.Tick(*t)
	via.PostTick(*t)

	*t = *t + 1

	return circuit.dataBus.Read()
}

func TestOutputToPortAChangingTheDirectionAndDeselectingChip(t *testing.T) {
	var step uint64

	via := CreateVia65C22()
	circuit := createTestCircuit()

	circuit.wire(via)

	// Set all pins on Port A to output
	writeToVia(via, circuit, dataDirectionRegisterA, 0xFF, &step)

	// Set output on port A to 0x55
	writeToVia(via, circuit, outputRegisterA, 0x55, &step)

	// Validate Port A output
	assert.Equal(t, uint8(0x55), circuit.portA.Read())

	// Do something else not related to check that Port A output stays the same (Read ACR)
	readFromVia(via, circuit, auxiliaryControl, &step)

	// Validate that data is still 55
	assert.Equal(t, uint8(0x55), circuit.portA.Read())

	// Clear port A input
	circuit.portA.Write(0x00)

	// Set data direction register to have upper 4 bits as input
	writeToVia(via, circuit, dataDirectionRegisterA, 0x0F, &step)

	// After clearing the Port B and setting the first 4 bits as input,
	// A 5 should be still be set in the first 4 bits.
	assert.Equal(t, uint8(0x05), circuit.portA.Read())

	// Writing to pins configured for input does not affect their values
	writeToVia(via, circuit, outputRegisterA, 0xF5, &step)

	// Port A remains unchanged
	assert.Equal(t, uint8(0x05), circuit.portA.Read())

	// Deselecting the chip does not affect the output
	circuit.cs1.Toggle()
	circuit.cs2.Toggle()

	// Do something else not related to check that Port A output stays the same (Read ACR)
	readFromVia(via, circuit, auxiliaryControl, &step)

	// Port A remains unchanged
	assert.Equal(t, uint8(0x05), circuit.portA.Read())
}

func TestOutputToPortBChangingTheDirectionAndDeselectingChip(t *testing.T) {
	var step uint64

	via := CreateVia65C22()
	circuit := createTestCircuit()

	circuit.wire(via)

	// Set all pins on Port B to output
	writeToVia(via, circuit, dataDirectionRegisterB, 0xFF, &step)

	// Set output on port B to 0xAA
	writeToVia(via, circuit, outputRegisterB, 0xAA, &step)

	// Validate Port B output
	assert.Equal(t, uint8(0xAA), circuit.portB.Read())

	// Do something else not related to check that Port B output stays the same (Read ACR)
	readFromVia(via, circuit, auxiliaryControl, &step)

	// Validate that data is still AA
	assert.Equal(t, uint8(0xAA), circuit.portB.Read())

	// Clear port B input
	circuit.portB.Write(0x00)

	// Set data direction register to have upper 4 bits as input
	writeToVia(via, circuit, dataDirectionRegisterB, 0x0F, &step)

	// Now port B reflects but still works even with chip not being selected
	assert.Equal(t, uint8(0x0A), circuit.portB.Read())

	// Writing to pins configured for input does not affect their values
	writeToVia(via, circuit, outputRegisterB, 0xFA, &step)

	// Port B remains unchanged
	assert.Equal(t, uint8(0x0A), circuit.portB.Read())

	// Deselecting the chip does not affect the output
	circuit.cs1.Toggle()
	circuit.cs2.Toggle()

	// Do something else not related to check that Port B output stays the same (Read ACR)
	readFromVia(via, circuit, auxiliaryControl, &step)

	// Port B remains unchanged
	assert.Equal(t, uint8(0x0A), circuit.portB.Read())
}

func TestInputFromPortANoLatching(t *testing.T) {
	var step uint64

	via := CreateVia65C22()
	circuit := createTestCircuit()

	circuit.wire(via)

	// Set all pins on Port A to input
	writeToVia(via, circuit, dataDirectionRegisterA, 0x00, &step)

	// Set 0xAA on Port A
	circuit.portA.Write(0xAA)

	// Read IRA
	value := readFromVia(via, circuit, outputRegisterA, &step)

	// Value must reflect the current status of the pins
	assert.Equal(t, uint8(0xAA), value)

	// Write 0xFF to ORA
	writeToVia(via, circuit, outputRegisterA, 0xFF, &step)

	// Port A must remain unchanged until DDR is changed
	assert.Equal(t, uint8(0xAA), circuit.portA.Read())

	// Read IRA
	value = readFromVia(via, circuit, outputRegisterA, &step)

	// Value must remain unaffected by the write to ORA
	assert.Equal(t, uint8(0xAA), value)

	// Change first 4 bits to output, this will make the value in ORA's first 4 bits be
	// put in Port A
	writeToVia(via, circuit, dataDirectionRegisterA, 0xF0, &step)

	// Value is expected to be FA (F from the previous write of FF to ORA) and A that remains up in the
	// input bus
	assert.Equal(t, uint8(0xFA), circuit.portA.Read())
}

func TestInputFromPortALatching(t *testing.T) {
	var step uint64

	via := CreateVia65C22()
	circuit := createTestCircuit()

	circuit.wire(via)

	// Set latching enabled on Port A
	writeToVia(via, circuit, auxiliaryControl, 0x01, &step)

	// Set all pins on Port A to input
	writeToVia(via, circuit, dataDirectionRegisterA, 0x00, &step)

	// Set interrupt on CA1 transition enabled
	writeToVia(via, circuit, interruptEnable, 0x82, &step)

	// Set interrupt on positive edge of CA1
	writeToVia(via, circuit, peripheralControl, 0x01, &step)

	// Set 0xAA on Port A
	circuit.portA.Write(0xAA)

	// Read IRA
	value := readFromVia(via, circuit, outputRegisterA, &step)

	// Value is unaffected by the pin status
	assert.Equal(t, uint8(0x00), value)
	// At this point IRQ is clear (high)
	assert.Equal(t, true, circuit.irq.Status())

	// Raise CA1 to latch the record
	circuit.ca1.Set(true)

	// Read IRA
	value = readFromVia(via, circuit, outputRegisterA, &step)

	// IRA must hold the latched value
	assert.Equal(t, uint8(0xAA), value)
	// IRQ must be triggering (Low)
	assert.Equal(t, false, circuit.irq.Status())

	// Change input on port A
	circuit.portA.Write(0xFF)

	// Read IRA
	value = readFromVia(via, circuit, outputRegisterA, &step)

	// IRA must still hold the latched value
	assert.Equal(t, uint8(0xAA), value)

	// Change first 4 bits to output, this will make the value in ORA's first 4 bits be
	// put in Port A
	writeToVia(via, circuit, dataDirectionRegisterA, 0xF0, &step)

	// As we never wrote to ORA value should be 0 on all 4 pins. So output should be 0x0? with ? being F
	// set in the previous steps as input
	assert.Equal(t, uint8(0x0F), via.peripheralPortA.Read())

	// Read IRA
	value = readFromVia(via, circuit, outputRegisterA, &step)

	// If latching is enabled it always reads the latched value on IRA
	assert.Equal(t, uint8(0xAA), value)
}

func TestInputFromPortBNoLatching(t *testing.T) {
	var step uint64

	via := CreateVia65C22()
	circuit := createTestCircuit()

	circuit.wire(via)

	// Set all pins on Port B to input
	writeToVia(via, circuit, dataDirectionRegisterB, 0x00, &step)

	// Set 0xAA on Port B
	circuit.portB.Write(0xAA)

	// Read IRB
	value := readFromVia(via, circuit, outputRegisterB, &step)

	// Value must reflect the current status of the pins
	assert.Equal(t, uint8(0xAA), value)

	// Write 0xFF to ORB
	writeToVia(via, circuit, outputRegisterB, 0xFF, &step)

	// Port B must remain unchanged until DDR is changed
	assert.Equal(t, uint8(0xAA), circuit.portB.Read())

	// Read IRB
	value = readFromVia(via, circuit, outputRegisterB, &step)

	// Value must remain unaffected by the write to ORB
	assert.Equal(t, uint8(0xAA), value)

	// Change first 4 bits to output, this will make the value in ORB's first 4 bits be
	// put in Port B
	writeToVia(via, circuit, dataDirectionRegisterB, 0xF0, &step)

	// Value is expected to be FA (F from the previous write of FF to ORA) and A that remains up in the
	// input bus
	assert.Equal(t, uint8(0xFA), circuit.portB.Read())
}

func TestInputFromPortBLatching(t *testing.T) {
	var step uint64

	via := CreateVia65C22()
	circuit := createTestCircuit()

	circuit.wire(via)

	// Set latching enabled on Port B
	writeToVia(via, circuit, auxiliaryControl, 0x02, &step)

	// Set all pins on Port B to input
	writeToVia(via, circuit, dataDirectionRegisterB, 0x00, &step)

	// Set interrupt on CB1 transition enabled
	writeToVia(via, circuit, interruptEnable, 0x90, &step)

	// Set interrupt on positive edge of CB1
	writeToVia(via, circuit, peripheralControl, 0x10, &step)

	// Set 0xAA on Port B
	circuit.portB.Write(0xAA)

	// Read IRA
	value := readFromVia(via, circuit, outputRegisterB, &step)

	// Value is unaffected by the pin status
	assert.Equal(t, uint8(0x00), value)
	// At this point IRQ is clear (high)
	assert.Equal(t, true, circuit.irq.Status())

	// Raise CB1 to latch the record
	circuit.cb1.Set(true)

	// Read IRB
	value = readFromVia(via, circuit, outputRegisterB, &step)

	// IRA must hold the latched value
	assert.Equal(t, uint8(0xAA), value)
	// IRQ must be triggering (Low)
	assert.Equal(t, false, circuit.irq.Status())

	// Change input on port B
	circuit.portB.Write(0xFF)

	// Read IRB
	value = readFromVia(via, circuit, outputRegisterB, &step)

	// IRA must still hold the latched value
	assert.Equal(t, uint8(0xAA), value)
	// IRQ must be cleared by the read
	assert.Equal(t, true, circuit.irq.Status())

	// Change first 4 bits to output, this will make the value in ORB's first 4 bits be
	// put in Port B
	writeToVia(via, circuit, dataDirectionRegisterB, 0xF0, &step)

	// As we never wrote to ORB value should be 0 on all 4 pins. So output should be 0x0? with ? being F
	// set in the previous steps as input
	assert.Equal(t, uint8(0x0F), via.peripheralPortB.Read())

	// Write 0x5A on the output register
	writeToVia(via, circuit, outputRegisterB, 0x5A, &step)

	// Force port B output to 0xFF again
	circuit.portB.Write(0xFF)

	// Read IRA
	value = readFromVia(via, circuit, outputRegisterB, &step)

	// First 4 bits are the ORB value = 5 as MPU reads from ORB, pin level has no effect.
	// Last 4 bits are from IRA at the last CB1 transition = A
	assert.Equal(t, uint8(0x5A), value)
}
