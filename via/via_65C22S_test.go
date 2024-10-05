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

/****************************************************************************************************************
* Test utilities
****************************************************************************************************************/

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

	*t = *t + 1
}

func readFromVia(via *Via65C22S, circuit *testCircuit, register viaRegisterCode, t *uint64) uint8 {
	circuit.setRegisterSelectValue(register)
	circuit.rw.Set(true)

	via.Tick(*t)

	*t = *t + 1

	return circuit.dataBus.Read()
}

func disableChipAndStepTime(via *Via65C22S, circuit *testCircuit, t *uint64) {
	circuit.cs1.Set(false)
	circuit.cs2.Set(true)

	via.Tick(*t)

	*t = *t + 1
}

func enableChip(circuit *testCircuit) {
	circuit.cs1.Set(true)
	circuit.cs2.Set(false)
}

type coutingTestConfiguration struct {
	via               *Via65C22S
	circuit           *testCircuit
	lcRegister        viaRegisterCode
	hcRegister        viaRegisterCode
	counterLSB        uint8
	counterMSB        uint8
	cyclesToExecute   int
	assertPB7         bool
	pB7expectedStatus bool
	expectedIRQStatus bool
}

func setupAndCountFrom(t *testing.T, config *coutingTestConfiguration, step *uint64) {
	// Will count down from 10 decimal
	writeToVia(config.via, config.circuit, config.lcRegister, config.counterLSB, step)

	// Starts the timer
	writeToVia(config.via, config.circuit, config.hcRegister, config.counterMSB, step)

	// At this point IRQ is clear (high)
	assert.Equal(t, true, config.circuit.irq.Status())

	countToTarget(t, config, step)
}

func countToTarget(t *testing.T, config *coutingTestConfiguration, step *uint64) {
	for range config.cyclesToExecute {
		disableChipAndStepTime(config.via, config.circuit, step)

		// IRQ remains high when counting
		assert.Equal(t, config.expectedIRQStatus, config.circuit.irq.Status())

		if config.assertPB7 {
			assert.Equal(t, config.pB7expectedStatus, config.circuit.portB.GetBusLine(7).Status())
		}
	}
}

/****************************************************************************************************************
* Latching tests
****************************************************************************************************************/

func TestOutputToPortAChangingTheDirectionAndDeselectingChip(t *testing.T) {
	var step uint64

	via := CreateVia65C22()
	circuit := createTestCircuit()

	circuit.wire(via)

	// Set all pins on Port A to output
	writeToVia(via, circuit, regDDRA, 0xFF, &step)

	// Set output on port A to 0x55
	writeToVia(via, circuit, regORAIRA, 0x55, &step)

	// Validate Port A output
	assert.Equal(t, uint8(0x55), circuit.portA.Read())

	// Do something else not related to check that Port A output stays the same (Read ACR)
	readFromVia(via, circuit, regACR, &step)

	// Validate that data is still 55
	assert.Equal(t, uint8(0x55), circuit.portA.Read())

	// Clear port A input
	circuit.portA.Write(0x00)

	// Set data direction register to have upper 4 bits as input
	writeToVia(via, circuit, regDDRA, 0x0F, &step)

	// After clearing the Port B and setting the first 4 bits as input,
	// A 5 should be still be set in the first 4 bits.
	assert.Equal(t, uint8(0x05), circuit.portA.Read())

	// Writing to pins configured for input does not affect their values
	writeToVia(via, circuit, regORAIRA, 0xF5, &step)

	// Port A remains unchanged
	assert.Equal(t, uint8(0x05), circuit.portA.Read())

	// Deselecting the chip does not affect the output
	circuit.cs1.Toggle()
	circuit.cs2.Toggle()

	// Do something else not related to check that Port A output stays the same (Read ACR)
	readFromVia(via, circuit, regACR, &step)

	// Port A remains unchanged
	assert.Equal(t, uint8(0x05), circuit.portA.Read())
}

func TestOutputToPortBChangingTheDirectionAndDeselectingChip(t *testing.T) {
	var step uint64

	via := CreateVia65C22()
	circuit := createTestCircuit()

	circuit.wire(via)

	// Set all pins on Port B to output
	writeToVia(via, circuit, regDDRB, 0xFF, &step)

	// Set output on port B to 0xAA
	writeToVia(via, circuit, regORBIRB, 0xAA, &step)

	// Validate Port B output
	assert.Equal(t, uint8(0xAA), circuit.portB.Read())

	// Do something else not related to check that Port B output stays the same (Read ACR)
	readFromVia(via, circuit, regACR, &step)

	// Validate that data is still AA
	assert.Equal(t, uint8(0xAA), circuit.portB.Read())

	// Clear port B input
	circuit.portB.Write(0x00)

	// Set data direction register to have upper 4 bits as input
	writeToVia(via, circuit, regDDRB, 0x0F, &step)

	// Now port B reflects but still works even with chip not being selected
	assert.Equal(t, uint8(0x0A), circuit.portB.Read())

	// Writing to pins configured for input does not affect their values
	writeToVia(via, circuit, regORBIRB, 0xFA, &step)

	// Port B remains unchanged
	assert.Equal(t, uint8(0x0A), circuit.portB.Read())

	// Deselecting the chip does not affect the output
	circuit.cs1.Toggle()
	circuit.cs2.Toggle()

	// Do something else not related to check that Port B output stays the same (Read ACR)
	readFromVia(via, circuit, regACR, &step)

	// Port B remains unchanged
	assert.Equal(t, uint8(0x0A), circuit.portB.Read())
}

func TestInputFromPortANoLatching(t *testing.T) {
	var step uint64

	via := CreateVia65C22()
	circuit := createTestCircuit()

	circuit.wire(via)

	// Set all pins on Port A to input
	writeToVia(via, circuit, regDDRA, 0x00, &step)

	// Set 0xAA on Port A
	circuit.portA.Write(0xAA)

	// Read IRA
	value := readFromVia(via, circuit, regORAIRA, &step)

	// Value must reflect the current status of the pins
	assert.Equal(t, uint8(0xAA), value)

	// Write 0xFF to ORA
	writeToVia(via, circuit, regORAIRA, 0xFF, &step)

	// Port A must remain unchanged until DDR is changed
	assert.Equal(t, uint8(0xAA), circuit.portA.Read())

	// Read IRA
	value = readFromVia(via, circuit, regORAIRA, &step)

	// Value must remain unaffected by the write to ORA
	assert.Equal(t, uint8(0xAA), value)

	// Change first 4 bits to output, this will make the value in ORA's first 4 bits be
	// put in Port A
	writeToVia(via, circuit, regDDRA, 0xF0, &step)

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
	writeToVia(via, circuit, regACR, 0x01, &step)

	// Set all pins on Port A to input
	writeToVia(via, circuit, regDDRA, 0x00, &step)

	// Set interrupt on CA1 transition enabled
	writeToVia(via, circuit, regIER, 0x82, &step)

	// Set interrupt on positive edge of CA1
	writeToVia(via, circuit, regPCR, 0x01, &step)

	// Set 0xAA on Port A
	circuit.portA.Write(0xAA)

	// Read IRA
	value := readFromVia(via, circuit, regORAIRA, &step)

	// Value is unaffected by the pin status
	assert.Equal(t, uint8(0x00), value)
	// At this point IRQ is clear (high)
	assert.Equal(t, true, circuit.irq.Status())

	// Raise CA1 to latch the record
	circuit.ca1.Set(true)

	// Read IRA
	value = readFromVia(via, circuit, regORAIRA, &step)

	// IRA must hold the latched value
	assert.Equal(t, uint8(0xAA), value)
	// IRQ must be triggering (Low)
	assert.Equal(t, false, circuit.irq.Status())

	// Change input on port A
	circuit.portA.Write(0xFF)

	// Read IRA
	value = readFromVia(via, circuit, regORAIRA, &step)

	// IRA must still hold the latched value
	assert.Equal(t, uint8(0xAA), value)

	// Change first 4 bits to output, this will make the value in ORA's first 4 bits be
	// put in Port A
	writeToVia(via, circuit, regDDRA, 0xF0, &step)

	// As we never wrote to ORA value should be 0 on all 4 pins. So output should be 0x0? with ? being F
	// set in the previous steps as input
	assert.Equal(t, uint8(0x0F), via.peripheralPortA.getConnector().Read())

	// Read IRA
	value = readFromVia(via, circuit, regORAIRA, &step)

	// If latching is enabled it always reads the latched value on IRA
	assert.Equal(t, uint8(0xAA), value)
}

func TestInputFromPortBNoLatching(t *testing.T) {
	var step uint64

	via := CreateVia65C22()
	circuit := createTestCircuit()

	circuit.wire(via)

	// Set all pins on Port B to input
	writeToVia(via, circuit, regDDRB, 0x00, &step)

	// Set 0xAA on Port B
	circuit.portB.Write(0xAA)

	// Read IRB
	value := readFromVia(via, circuit, regORBIRB, &step)

	// Value must reflect the current status of the pins
	assert.Equal(t, uint8(0xAA), value)

	// Write 0xFF to ORB
	writeToVia(via, circuit, regORBIRB, 0xFF, &step)

	// Port B must remain unchanged until DDR is changed
	assert.Equal(t, uint8(0xAA), circuit.portB.Read())

	// Read IRB
	value = readFromVia(via, circuit, regORBIRB, &step)

	// Value must remain unaffected by the write to ORB
	assert.Equal(t, uint8(0xAA), value)

	// Change first 4 bits to output, this will make the value in ORB's first 4 bits be
	// put in Port B
	writeToVia(via, circuit, regDDRB, 0xF0, &step)

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
	writeToVia(via, circuit, regACR, 0x02, &step)

	// Set all pins on Port B to input
	writeToVia(via, circuit, regDDRB, 0x00, &step)

	// Set interrupt on CB1 transition enabled
	writeToVia(via, circuit, regIER, 0x90, &step)

	// Set interrupt on positive edge of CB1
	writeToVia(via, circuit, regPCR, 0x10, &step)

	// Set 0xAA on Port B
	circuit.portB.Write(0xAA)

	// Read IRA
	value := readFromVia(via, circuit, regORBIRB, &step)

	// Value is unaffected by the pin status
	assert.Equal(t, uint8(0x00), value)
	// At this point IRQ is clear (high)
	assert.Equal(t, true, circuit.irq.Status())

	// Raise CB1 to latch the record
	circuit.cb1.Set(true)

	// Read IRB
	value = readFromVia(via, circuit, regORBIRB, &step)

	// IRA must hold the latched value
	assert.Equal(t, uint8(0xAA), value)
	// IRQ must be triggering (Low)
	assert.Equal(t, false, circuit.irq.Status())

	// Change input on port B
	circuit.portB.Write(0xFF)

	// Read IRB
	value = readFromVia(via, circuit, regORBIRB, &step)

	// IRA must still hold the latched value
	assert.Equal(t, uint8(0xAA), value)
	// IRQ must be cleared by the read
	assert.Equal(t, true, circuit.irq.Status())

	// Change first 4 bits to output, this will make the value in ORB's first 4 bits be
	// put in Port B
	writeToVia(via, circuit, regDDRB, 0xF0, &step)

	// As we never wrote to ORB value should be 0 on all 4 pins. So output should be 0x0? with ? being F
	// set in the previous steps as input
	assert.Equal(t, uint8(0x0F), via.peripheralPortB.getConnector().Read())

	// Write 0x5A on the output register
	writeToVia(via, circuit, regORBIRB, 0x5A, &step)

	// Force port B output to 0xFF again
	circuit.portB.Write(0xFF)

	// Read IRA
	value = readFromVia(via, circuit, regORBIRB, &step)

	// First 4 bits are the ORB value = 5 as MPU reads from ORB, pin level has no effect.
	// Last 4 bits are from IRA at the last CB1 transition = A
	assert.Equal(t, uint8(0x5A), value)
}

/***********************************************************************************************************************
* Handshake modes tests
*
* See page 5 of the link below to understand the timing of handshake modes. This test is copying and validating
* the behaviour described there.
*
* https://web.archive.org/web/20160108173129if_/http://archive.6502.org/datasheets/mos_6522_preliminary_nov_1977.pdf
************************************************************************************************************************/

func TestReadHandshakeOnPortA(t *testing.T) {
	readHandshakeOnPortA(t, 0x08)
}

func TestReadHandshakePulseOnPortA(t *testing.T) {
	readHandshakeOnPortA(t, 0x0A)
}

func TestWriteHandshakeOnPortA(t *testing.T) {
	writeHandshakeOnPortA(t, 0x08)
}

func TestWriteHandshakePulseOnPortA(t *testing.T) {
	writeHandshakeOnPortA(t, 0x0A)
}

func TestWriteHandshakeOnPortB(t *testing.T) {
	writeHandshakeOnPortB(t, 0x80)
}

func TestWriteHandshakePulseOnPortB(t *testing.T) {
	writeHandshakeOnPortB(t, 0xA0)
}

func readHandshakeOnPortA(t *testing.T, mode uint8) {
	var step uint64

	via := CreateVia65C22()
	circuit := createTestCircuit()

	circuit.wire(via)

	// Set all pins on Port A to input
	writeToVia(via, circuit, regDDRA, 0x00, &step)

	// Set interrupt on CA1 transition enabled
	writeToVia(via, circuit, regIER, 0x82, &step)

	// Set interrupt on positive edge of CA1 (0x01) and CA2 in handshake desired handshake mode
	writeToVia(via, circuit, regPCR, mode|0x01, &step)

	// In handshake mode CA2 is default high
	assert.Equal(t, true, circuit.ca2.Status())
	// At this point IRQ is clear (high)
	assert.Equal(t, true, circuit.irq.Status())

	// Signal Data Ready on CA1
	circuit.ca1.Set(true)

	// Step time and check that IRQ is now active (low)
	disableChipAndStepTime(via, circuit, &step)
	assert.Equal(t, false, circuit.irq.Status())

	// Simulate some more steps and check
	disableChipAndStepTime(via, circuit, &step)
	disableChipAndStepTime(via, circuit, &step)
	disableChipAndStepTime(via, circuit, &step)
	assert.Equal(t, false, circuit.irq.Status())

	// Clear the data ready signal in CA 1
	circuit.ca1.Set(false)
	disableChipAndStepTime(via, circuit, &step)
	disableChipAndStepTime(via, circuit, &step)
	// IRQ stays triggered
	assert.Equal(t, false, circuit.irq.Status())

	// Re-enable the chip and read IRA
	enableChip(circuit)
	readFromVia(via, circuit, regORAIRA, &step)

	// CA2 should have dropped to signal "data taken"
	assert.Equal(t, false, circuit.ca2.Status())
	// IRQ must be cleared
	assert.Equal(t, true, circuit.irq.Status())

	if mode == 0x08 {
		// Simulate some more steps and check, in this mode CA2 will stay
		// low until transition of CB1 happens
		disableChipAndStepTime(via, circuit, &step)
		disableChipAndStepTime(via, circuit, &step)
		disableChipAndStepTime(via, circuit, &step)
		assert.Equal(t, false, circuit.ca2.Status())
		assert.Equal(t, true, circuit.irq.Status())
	} else {
		// In this mode CA2 will stay low for only 1 cycle after read IRA
		// and return to high
		disableChipAndStepTime(via, circuit, &step)
		assert.Equal(t, false, circuit.ca2.Status())
		assert.Equal(t, true, circuit.irq.Status())

		disableChipAndStepTime(via, circuit, &step)
		assert.Equal(t, true, circuit.ca2.Status())
		assert.Equal(t, true, circuit.irq.Status())
		disableChipAndStepTime(via, circuit, &step)
		assert.Equal(t, true, circuit.ca2.Status())
		assert.Equal(t, true, circuit.irq.Status())
	}

	// Signaling data ready on CA1 should make CA2 reset to high
	circuit.ca1.Set(true)
	disableChipAndStepTime(via, circuit, &step)
	assert.Equal(t, true, circuit.ca2.Status())
	assert.Equal(t, false, circuit.irq.Status())
}

func writeHandshakeOnPortA(t *testing.T, mode uint8) {
	var step uint64

	via := CreateVia65C22()
	circuit := createTestCircuit()

	circuit.wire(via)

	// Set all pins on Port A to output
	writeToVia(via, circuit, regDDRA, 0xFF, &step)

	// Set interrupt on CA1 transition enabled
	writeToVia(via, circuit, regIER, 0x82, &step)

	// Set interrupt on positive edge of CA1 (0x01) and CA2 in handshake desired handshake mode
	writeToVia(via, circuit, regPCR, mode|0x01, &step)

	// Data taken is low
	assert.Equal(t, false, circuit.ca1.Status())
	// In handshake mode CA2 is default high
	assert.Equal(t, true, circuit.ca2.Status())
	// At this point IRQ is clear (high)
	assert.Equal(t, true, circuit.irq.Status())

	// Write to ORA
	writeToVia(via, circuit, regORAIRA, 0xFF, &step)

	// CA2 will drop to signal "data ready"
	if mode == 0x08 {
		// Simulate some more steps and check, in this mode CA2 will stay
		// low until transition of CB1 happens
		disableChipAndStepTime(via, circuit, &step)
		disableChipAndStepTime(via, circuit, &step)
		disableChipAndStepTime(via, circuit, &step)
		assert.Equal(t, false, circuit.ca2.Status())
	} else {
		// In this mode CA2 will stay low for only 1 cycle
		// and return to high
		disableChipAndStepTime(via, circuit, &step)
		assert.Equal(t, false, circuit.ca2.Status())

		disableChipAndStepTime(via, circuit, &step)
		assert.Equal(t, true, circuit.ca2.Status())
		disableChipAndStepTime(via, circuit, &step)
		assert.Equal(t, true, circuit.ca2.Status())
	}

	// Signal Data Taken on CA1
	circuit.ca1.Set(true)

	// Step time and check that IRQ is now active (low)
	// And CA2 has been returned to high
	disableChipAndStepTime(via, circuit, &step)
	assert.Equal(t, false, circuit.irq.Status())
	assert.Equal(t, true, circuit.ca2.Status())

	// Do some steps and renable the Data Taken flag,
	// IRQ must stay triggered (low)
	disableChipAndStepTime(via, circuit, &step)
	disableChipAndStepTime(via, circuit, &step)
	circuit.ca1.Set(false)
	disableChipAndStepTime(via, circuit, &step)
	assert.Equal(t, false, circuit.irq.Status())

	// Re-enable the chip and write to ORA
	enableChip(circuit)
	writeToVia(via, circuit, regORAIRA, 0xFE, &step)

	// IRQ must be reset and CA2 goes low again
	assert.Equal(t, true, circuit.irq.Status())
	assert.Equal(t, false, circuit.ca2.Status())
}

func writeHandshakeOnPortB(t *testing.T, mode uint8) {
	var step uint64

	via := CreateVia65C22()
	circuit := createTestCircuit()

	circuit.wire(via)

	// Set all pins on Port B to output
	writeToVia(via, circuit, regDDRB, 0xFF, &step)

	// Set interrupt on CB1 transition enabled
	writeToVia(via, circuit, regIER, 0x90, &step)

	// Set interrupt on positive edge of CB1 (0x10) and CB2 in handshake desired handshake mode
	writeToVia(via, circuit, regPCR, mode|0x10, &step)

	// Data taken is low
	assert.Equal(t, false, circuit.cb1.Status())
	// In handshake mode CB2 is default high
	assert.Equal(t, true, circuit.cb2.Status())
	// At this point IRQ is clear (high)
	assert.Equal(t, true, circuit.irq.Status())

	// Write to ORB
	writeToVia(via, circuit, regORBIRB, 0xFF, &step)

	// CB2 will drop to signal "data ready"
	if mode == 0x80 {
		// Simulate some more steps and check, in this mode CB2 will stay
		// low until transition of CB1 happens
		disableChipAndStepTime(via, circuit, &step)
		disableChipAndStepTime(via, circuit, &step)
		disableChipAndStepTime(via, circuit, &step)
		assert.Equal(t, false, circuit.cb2.Status())
	} else {
		// In this mode CB2 will stay low for only 1 cycle
		// and return to high
		disableChipAndStepTime(via, circuit, &step)
		assert.Equal(t, false, circuit.cb2.Status())

		disableChipAndStepTime(via, circuit, &step)
		assert.Equal(t, true, circuit.cb2.Status())
		disableChipAndStepTime(via, circuit, &step)
		assert.Equal(t, true, circuit.cb2.Status())
	}

	// Signal Data Taken on CB1
	circuit.cb1.Set(true)

	// Step time and check that IRQ is now active (low)
	// And CB2 has been returned to high
	disableChipAndStepTime(via, circuit, &step)
	assert.Equal(t, false, circuit.irq.Status())
	assert.Equal(t, true, circuit.cb2.Status())

	// Do some steps and renable the Data Taken flag,
	// IRQ must stay triggered (low)
	disableChipAndStepTime(via, circuit, &step)
	disableChipAndStepTime(via, circuit, &step)
	circuit.cb1.Set(false)
	disableChipAndStepTime(via, circuit, &step)
	assert.Equal(t, false, circuit.irq.Status())

	// Re-enable the chip and write to ORB
	enableChip(circuit)
	writeToVia(via, circuit, regORBIRB, 0xFE, &step)

	// IRQ must be reset and CB2 goes low again
	assert.Equal(t, true, circuit.irq.Status())
	assert.Equal(t, false, circuit.cb2.Status())
}

/****************************************************************************************************************
* Timer tests
****************************************************************************************************************/

func TestTimer1OneShotMode(t *testing.T) {
	var step uint64

	via := CreateVia65C22()
	circuit := createTestCircuit()

	circuit.wire(via)

	config := coutingTestConfiguration{
		via:               via,
		circuit:           circuit,
		lcRegister:        regT1CL,
		hcRegister:        regT1CH,
		counterLSB:        10,
		counterMSB:        0,
		cyclesToExecute:   11,
		assertPB7:         false,
		pB7expectedStatus: false,
		expectedIRQStatus: true,
	}

	// Set ACR to 0x00, for this test is important bit 6 and 7 = 00 -> Timer 1 single shot PB7 disabled
	writeToVia(via, circuit, regACR, 0x00, &step)

	// Enable interrupts for T1 timeout (bit 7 -> enable, bit 6 -> T1)
	writeToVia(via, circuit, regIER, 0xC0, &step)

	// Counts down from 10, it takes N+1 cycles to count down
	// While counting PB7 is not driven and IRQ stays high
	setupAndCountFrom(t, &config, &step)

	// After counting to 0 requires extra 0.5 step
	// to trigger IRQ
	disableChipAndStepTime(via, circuit, &step)
	assert.Equal(t, false, circuit.irq.Status())

	// Counter keeps counting down.
	// When IRQ triggered counter was in the end of FFFF / beginning of FFFE
	// 2 extra cycles will move that to FFFC
	disableChipAndStepTime(via, circuit, &step)
	disableChipAndStepTime(via, circuit, &step)
	assert.Equal(t, uint16(0xFFFC), via.registers.counter1)

	// Reenable the chip
	enableChip(circuit)

	// Clear the interrupt flag by reading T1 low order counter
	counter := readFromVia(via, circuit, regT1CL, &step)
	assert.Equal(t, uint8(0xFB), counter)
	assert.Equal(t, true, circuit.irq.Status())

	// Repeats the couting from 10
	setupAndCountFrom(t, &config, &step)
	disableChipAndStepTime(via, circuit, &step)
	assert.Equal(t, false, circuit.irq.Status())

	// Reenable the chip
	enableChip(circuit)

	// Set ACR to 0x80, for this test is important bit 6 and 7 = 10 -> Timer 1 single shot PB7 enabled
	// When ACR sets PB7 as output, line goes high
	writeToVia(via, circuit, regACR, 0x80, &step)
	assert.Equal(t, true, circuit.portB.GetBusLine(7).Status())

	// Will now test PB7 behaviour
	config.assertPB7 = true
	config.pB7expectedStatus = false

	// Repeats the couting from 10, now evaluating the
	// PB7 flag to stay low while counting
	setupAndCountFrom(t, &config, &step)
	disableChipAndStepTime(via, circuit, &step)

	// At this point IRQ is set (low) and PB7 goes back to high
	assert.Equal(t, false, circuit.irq.Status())
	assert.Equal(t, true, circuit.portB.GetBusLine(7).Status())
}

func TestTimer1FreeRunMode(t *testing.T) {
	var step uint64

	via := CreateVia65C22()
	circuit := createTestCircuit()

	circuit.wire(via)

	config := coutingTestConfiguration{
		via:               via,
		circuit:           circuit,
		lcRegister:        regT1CL,
		hcRegister:        regT1CH,
		counterLSB:        10,
		counterMSB:        0,
		cyclesToExecute:   11,
		assertPB7:         true,
		pB7expectedStatus: false,
		expectedIRQStatus: true,
	}

	// Set ACR to 0x11, for this test is important bit 6 and 7 = 11 -> Timer 1 free run and PB7 enabled
	// Line 7 goes high when ACR is set to output
	writeToVia(via, circuit, regACR, 0xC0, &step)
	assert.Equal(t, true, circuit.portB.GetBusLine(7).Status())

	// Enable interrupts for T1 timeout (bit 7 -> enable, bit 6 -> T1)
	writeToVia(via, circuit, regIER, 0xC0, &step)

	// Counts down from 10, it takes N+1 cycles to count down
	// While counting PB7 is driven low and IRQ stays high
	setupAndCountFrom(t, &config, &step)

	// After counting to 0 requires extra 0.5 step
	// to trigger IRQ and port B toggles high.
	disableChipAndStepTime(via, circuit, &step)
	assert.Equal(t, false, circuit.irq.Status())
	assert.Equal(t, true, circuit.portB.GetBusLine(7).Status())

	// We'll keep counting. On Free-Run mode the PB7 is expected to toggle
	// between states, since last countdown was low, now is expected high
	// until timer reaches zero. Since we won't reset the interrupt, we
	// expect that one to remain low.
	config.pB7expectedStatus = true
	config.expectedIRQStatus = false
	countToTarget(t, &config, &step)

	// After couting to zero, we need one more step to transition.
	// PB7 will go low, IRQ remains low (unchanged)
	disableChipAndStepTime(via, circuit, &step)
	assert.Equal(t, false, circuit.irq.Status())
	assert.Equal(t, false, circuit.portB.GetBusLine(7).Status())

	// Clear the interrupt flag by reading T1 low order counter
	// This spent one cycle from the new counting so counter will be 9.
	enableChip(circuit)
	counter := readFromVia(via, circuit, regT1CL, &step)
	assert.Equal(t, uint8(0x09), counter)
	assert.Equal(t, true, circuit.irq.Status())

	// Since we consumed 1 cycle to read the counter and reset the IRQ
	// we will now count 10 cycles, also now IRQ is expected high now.
	// This cycle PB7 is expected low.
	config.cyclesToExecute = 10
	config.expectedIRQStatus = true
	config.pB7expectedStatus = false
	countToTarget(t, &config, &step)

	// One extra step to update status lines,
	// IRQ will trigger (go low) and PB7 will toggle high
	disableChipAndStepTime(via, circuit, &step)
	assert.Equal(t, false, circuit.irq.Status())
	assert.Equal(t, true, circuit.portB.GetBusLine(7).Status())
}

func TestTimer2OneShotMode(t *testing.T) {
	var step uint64

	via := CreateVia65C22()
	circuit := createTestCircuit()

	circuit.wire(via)

	config := coutingTestConfiguration{
		via:               via,
		circuit:           circuit,
		lcRegister:        regT2CL,
		hcRegister:        regT2CH,
		counterLSB:        10,
		counterMSB:        0,
		cyclesToExecute:   11,
		assertPB7:         false,
		pB7expectedStatus: false,
		expectedIRQStatus: true,
	}

	// Set ACR to 0x00, for this test is important bit 5 = 00 -> Timer 2 single shot
	writeToVia(via, circuit, regACR, 0x00, &step)

	// Enable interrupts for T2 timeout (bit 7 -> enable, bit 5 -> T2)
	writeToVia(via, circuit, regIER, 0xA0, &step)

	// Counts down from 10, it takes N+1 cycles to count down
	// While counting PB7 is not driven and IRQ stays high
	setupAndCountFrom(t, &config, &step)

	// After counting to 0 requires extra 0.5 step
	// to trigger IRQ
	disableChipAndStepTime(via, circuit, &step)
	assert.Equal(t, false, circuit.irq.Status())

	// Counter keeps counting down.
	// When IRQ triggered counter was in the end of FFFF / beginning of FFFE
	// 2 extra cycles will move that to FFFC
	disableChipAndStepTime(via, circuit, &step)
	disableChipAndStepTime(via, circuit, &step)
	assert.Equal(t, uint16(0xFFFC), via.registers.counter2)

	// Reenable the chip
	enableChip(circuit)

	// Clear the interrupt flag by reading T2 low order counter
	counter := readFromVia(via, circuit, regT2CL, &step)
	assert.Equal(t, uint8(0xFB), counter)
	assert.Equal(t, true, circuit.irq.Status())

	// Repeats the couting from 10
	setupAndCountFrom(t, &config, &step)
	disableChipAndStepTime(via, circuit, &step)
	assert.Equal(t, false, circuit.irq.Status())
}

func TestTimer2PulseCountingMode(t *testing.T) {
	var step uint64

	via := CreateVia65C22()
	circuit := createTestCircuit()

	circuit.wire(via)

	// Set ACR to 0x00, for this test is important bit 5 = 1 -> Timer 2 pulse counting
	writeToVia(via, circuit, regACR, 0x20, &step)

	// Enable interrupts for T2 timeout (bit 7 -> enable, bit 5 -> T2)
	writeToVia(via, circuit, regIER, 0xA0, &step)

	// Set PB6 high so it doesn't count down
	circuit.portB.GetBusLine(6).Set(true)

	// Set counter to 10
	writeToVia(via, circuit, regT2CL, 10, &step)
	writeToVia(via, circuit, regT2CH, 0x00, &step)

	// Pass 2 cycles, counter should still be in 10
	disableChipAndStepTime(via, circuit, &step)
	disableChipAndStepTime(via, circuit, &step)

	assert.Equal(t, uint16(10), via.registers.counter2)

	// According to https://web.archive.org/web/20220708103848if_/http://archive.6502.org/datasheets/synertek_sy6522.pdf
	// IRQ is set when counter rolls over to FFFF
	// TODO: I cannot find documentation online about this behaviour and official manual states that this happens on
	// beggining of cycle with 0 in the counter. Might need to test in real hardware
	for n := 11; n > 0; n-- {
		circuit.portB.GetBusLine(6).Set(false)
		disableChipAndStepTime(via, circuit, &step)
		assert.Equal(t, uint16(n-2), via.registers.counter2)

		circuit.portB.GetBusLine(6).Set(true)
		disableChipAndStepTime(via, circuit, &step)
		assert.Equal(t, uint16(n-2), via.registers.counter2)
	}

	// After counting to 0 requires extra 0.5 step
	// to trigger IRQ
	circuit.portB.GetBusLine(6).Set(false)
	disableChipAndStepTime(via, circuit, &step)
	assert.Equal(t, false, circuit.irq.Status())
}
