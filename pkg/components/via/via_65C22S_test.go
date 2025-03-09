package via

import (
	"math"
	"testing"

	"github.com/fran150/clementina6502/pkg/components/buses"
	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/stretchr/testify/assert"
)

// Represents a circuit used for testing. It contains all the required
// lines
type testCircuit struct {
	cs1     buses.Line
	cs2     buses.Line
	ca1     buses.Line
	ca2     buses.Line
	cb1     buses.Line
	cb2     buses.Line
	dataBus buses.Bus[uint8]
	irq     buses.Line
	portA   buses.Bus[uint8]
	portB   buses.Bus[uint8]
	reset   buses.Line
	rs      [4]buses.Line
	rw      buses.Line
}

// Creates and returns a reference to the test circuit
func newTestCircuit() *testCircuit {
	var rsLines [4]buses.Line

	for i := range 4 {
		rsLines[i] = buses.NewStandaloneLine(false)
	}

	return &testCircuit{
		cs1:     buses.NewStandaloneLine(true),
		cs2:     buses.NewStandaloneLine(false),
		ca1:     buses.NewStandaloneLine(false),
		ca2:     buses.NewStandaloneLine(false),
		cb1:     buses.NewStandaloneLine(false),
		cb2:     buses.NewStandaloneLine(false),
		dataBus: buses.New8BitStandaloneBus(),
		irq:     buses.NewStandaloneLine(true),
		portA:   buses.New8BitStandaloneBus(),
		portB:   buses.New8BitStandaloneBus(),
		reset:   buses.NewStandaloneLine(true),
		rs:      rsLines,
		rw:      buses.NewStandaloneLine(true),
	}
}

// Connects the specified VIA chip to the test circuit
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

// Sets the status of the RS lines based on the specified value
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

// Writes to the specified values to a register in the VIA chip
func writeToVia(via *Via65C22S, circuit *testCircuit, register viaRegisterCode, value uint8, context *common.StepContext) {
	circuit.setRegisterSelectValue(register)
	circuit.rw.Set(false)
	circuit.dataBus.Write(value)

	via.Tick(context)

	context.NextCycle()
}

// Reads the specified register from the VIA chip
func readFromVia(via *Via65C22S, circuit *testCircuit, register viaRegisterCode, context *common.StepContext) uint8 {
	circuit.setRegisterSelectValue(register)
	circuit.rw.Set(true)

	via.Tick(context)

	context.NextCycle()

	return circuit.dataBus.Read()
}

// Disables the chip and step time (used to wait for actions)
func disableChipAndStepTime(via *Via65C22S, circuit *testCircuit, context *common.StepContext) {
	circuit.cs1.Set(false)
	circuit.cs2.Set(true)

	via.Tick(context)

	context.NextCycle()
}

// Reneables the chip (does not time step)
func enableChip(circuit *testCircuit) {
	circuit.cs1.Set(true)
	circuit.cs2.Set(false)
}

// Configuration used for timer tests
type coutingTestConfiguration struct {
	via                           *Via65C22S
	circuit                       *testCircuit
	lcRegister                    viaRegisterCode
	hcRegister                    viaRegisterCode
	counterLSB                    uint8
	counterMSB                    uint8
	cyclesToExecute               int
	assertPB7                     bool
	pB7expectedStatus             bool
	expectedIRQStatusWhenCounting bool
	expectedInitialIRQStatus      bool
}

// Sets ups the VIA chip according to the test configuration and starts counting
func setupAndCountFrom(t *testing.T, config *coutingTestConfiguration, context *common.StepContext) {
	// Will count down from 10 decimal
	writeToVia(config.via, config.circuit, config.lcRegister, config.counterLSB, context)

	// Starts the timer
	writeToVia(config.via, config.circuit, config.hcRegister, config.counterMSB, context)

	// At this point IRQ is clear (high)
	assert.Equal(t, config.expectedInitialIRQStatus, config.circuit.irq.Status())

	countToTarget(t, config, context)
}

// Disables the chip and executes the number of time steps specified in the testing configuration.
func countToTarget(t *testing.T, config *coutingTestConfiguration, context *common.StepContext) {
	for range config.cyclesToExecute {
		disableChipAndStepTime(config.via, config.circuit, context)

		// IRQ remains high when counting
		assert.Equal(t, config.expectedIRQStatusWhenCounting, config.circuit.irq.Status())

		if config.assertPB7 {
			assert.Equal(t, config.pB7expectedStatus, config.circuit.portB.GetBusLine(7).Status())
		}
	}
}

/****************************************************************************************************************
* Latching tests
****************************************************************************************************************/

func TestOutputToPortAChangingTheDirectionAndDeselectingChip(t *testing.T) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	// Set all pins on Port A to output
	writeToVia(via, circuit, regDDRA, 0xFF, &context)

	// Set output on port A to 0x55
	writeToVia(via, circuit, regORAIRA, 0x55, &context)

	// Validate Port A output
	assert.Equal(t, uint8(0x55), circuit.portA.Read())

	// Do something else not related to check that Port A output stays the same (Read ACR)
	readFromVia(via, circuit, regACR, &context)

	// Validate that data is still 55
	assert.Equal(t, uint8(0x55), circuit.portA.Read())

	// Clear port A input
	circuit.portA.Write(0x00)

	// Set data direction register to have upper 4 bits as input
	writeToVia(via, circuit, regDDRA, 0x0F, &context)

	// After clearing the Port B and setting the first 4 bits as input,
	// A 5 should be still be set in the first 4 bits.
	assert.Equal(t, uint8(0x05), circuit.portA.Read())

	// Writing to pins configured for input does not affect their values
	writeToVia(via, circuit, regORAIRA, 0xF5, &context)

	// Port A remains unchanged
	assert.Equal(t, uint8(0x05), circuit.portA.Read())

	// Deselecting the chip does not affect the output
	circuit.cs1.Toggle()
	circuit.cs2.Toggle()

	// Do something else not related to check that Port A output stays the same (Read ACR)
	readFromVia(via, circuit, regACR, &context)

	// Port A remains unchanged
	assert.Equal(t, uint8(0x05), circuit.portA.Read())
}

func TestOutputToPortBChangingTheDirectionAndDeselectingChip(t *testing.T) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	// Set all pins on Port B to output
	writeToVia(via, circuit, regDDRB, 0xFF, &context)

	// Set output on port B to 0xAA
	writeToVia(via, circuit, regORBIRB, 0xAA, &context)

	// Validate Port B output
	assert.Equal(t, uint8(0xAA), circuit.portB.Read())

	// Do something else not related to check that Port B output stays the same (Read ACR)
	readFromVia(via, circuit, regACR, &context)

	// Validate that data is still AA
	assert.Equal(t, uint8(0xAA), circuit.portB.Read())

	// Clear port B input
	circuit.portB.Write(0x00)

	// Set data direction register to have upper 4 bits as input
	writeToVia(via, circuit, regDDRB, 0x0F, &context)

	// Now port B reflects but still works even with chip not being selected
	assert.Equal(t, uint8(0x0A), circuit.portB.Read())

	// Writing to pins configured for input does not affect their values
	writeToVia(via, circuit, regORBIRB, 0xFA, &context)

	// Port B remains unchanged
	assert.Equal(t, uint8(0x0A), circuit.portB.Read())

	// Deselecting the chip does not affect the output
	circuit.cs1.Toggle()
	circuit.cs2.Toggle()

	// Do something else not related to check that Port B output stays the same (Read ACR)
	readFromVia(via, circuit, regACR, &context)

	// Port B remains unchanged
	assert.Equal(t, uint8(0x0A), circuit.portB.Read())
}

func TestInputFromPortANoLatching(t *testing.T) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	// Set all pins on Port A to input
	writeToVia(via, circuit, regDDRA, 0x00, &context)

	// Set 0xAA on Port A
	circuit.portA.Write(0xAA)

	// Read IRA
	value := readFromVia(via, circuit, regORAIRA, &context)

	// Value must reflect the current status of the pins
	assert.Equal(t, uint8(0xAA), value)

	// Write 0xFF to ORA
	writeToVia(via, circuit, regORAIRA, 0xFF, &context)

	// Port A must remain unchanged until DDR is changed
	assert.Equal(t, uint8(0xAA), circuit.portA.Read())

	// Read IRA
	value = readFromVia(via, circuit, regORAIRA, &context)

	// Value must remain unaffected by the write to ORA
	assert.Equal(t, uint8(0xAA), value)

	// Change first 4 bits to output, this will make the value in ORA's first 4 bits be
	// put in Port A
	writeToVia(via, circuit, regDDRA, 0xF0, &context)

	// Value is expected to be FA (F from the previous write of FF to ORA) and A that remains up in the
	// input bus
	assert.Equal(t, uint8(0xFA), circuit.portA.Read())
}

func TestInputFromPortALatching(t *testing.T) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	// Set latching enabled on Port A
	writeToVia(via, circuit, regACR, 0x01, &context)

	// Set all pins on Port A to input
	writeToVia(via, circuit, regDDRA, 0x00, &context)

	// Set interrupt on CA1 transition enabled
	writeToVia(via, circuit, regIER, 0x82, &context)

	// Set interrupt on positive edge of CA1
	writeToVia(via, circuit, regPCR, 0x01, &context)

	// Set 0xAA on Port A
	circuit.portA.Write(0xAA)

	// Read IRA
	value := readFromVia(via, circuit, regORAIRA, &context)

	// Value is unaffected by the pin status
	assert.Equal(t, uint8(0x00), value)
	// At this point IRQ is clear (high)
	assert.Equal(t, true, circuit.irq.Status())

	// Raise CA1 to latch the record
	circuit.ca1.Set(true)

	// Read IRA
	value = readFromVia(via, circuit, regORAIRA, &context)

	// IRA must hold the latched value
	assert.Equal(t, uint8(0xAA), value)
	// IRQ must be triggering (Low)
	assert.Equal(t, false, circuit.irq.Status())

	// Change input on port A
	circuit.portA.Write(0xFF)

	// Read IRA
	value = readFromVia(via, circuit, regORAIRA, &context)

	// IRA must still hold the latched value
	assert.Equal(t, uint8(0xAA), value)

	// Change first 4 bits to output, this will make the value in ORA's first 4 bits be
	// put in Port A
	writeToVia(via, circuit, regDDRA, 0xF0, &context)

	// As we never wrote to ORA value should be 0 on all 4 pins. So output should be 0x0? with ? being F
	// set in the previous steps as input
	assert.Equal(t, uint8(0x0F), via.peripheralPortA.getConnector().Read())

	// Read IRA
	value = readFromVia(via, circuit, regORAIRA, &context)

	// If latching is enabled it always reads the latched value on IRA
	assert.Equal(t, uint8(0xAA), value)
}

func TestInputFromPortBNoLatching(t *testing.T) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	// Set all pins on Port B to input
	writeToVia(via, circuit, regDDRB, 0x00, &context)

	// Set 0xAA on Port B
	circuit.portB.Write(0xAA)

	// Read IRB
	value := readFromVia(via, circuit, regORBIRB, &context)

	// Value must reflect the current status of the pins
	assert.Equal(t, uint8(0xAA), value)

	// Write 0xFF to ORB
	writeToVia(via, circuit, regORBIRB, 0xFF, &context)

	// Port B must remain unchanged until DDR is changed
	assert.Equal(t, uint8(0xAA), circuit.portB.Read())

	// Read IRB
	value = readFromVia(via, circuit, regORBIRB, &context)

	// Value must remain unaffected by the write to ORB
	assert.Equal(t, uint8(0xAA), value)

	// Change first 4 bits to output, this will make the value in ORB's first 4 bits be
	// put in Port B
	writeToVia(via, circuit, regDDRB, 0xF0, &context)

	// Value is expected to be FA (F from the previous write of FF to ORA) and A that remains up in the
	// input bus
	assert.Equal(t, uint8(0xFA), circuit.portB.Read())
}

func TestInputFromPortBLatching(t *testing.T) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	// Set latching enabled on Port B
	writeToVia(via, circuit, regACR, 0x02, &context)

	// Set all pins on Port B to input
	writeToVia(via, circuit, regDDRB, 0x00, &context)

	// Set interrupt on CB1 transition enabled
	writeToVia(via, circuit, regIER, 0x90, &context)

	// Set interrupt on positive edge of CB1
	writeToVia(via, circuit, regPCR, 0x10, &context)

	// Set 0xAA on Port B
	circuit.portB.Write(0xAA)

	// Read IRA
	value := readFromVia(via, circuit, regORBIRB, &context)

	// Value is unaffected by the pin status
	assert.Equal(t, uint8(0x00), value)
	// At this point IRQ is clear (high)
	assert.Equal(t, true, circuit.irq.Status())

	// Raise CB1 to latch the record
	circuit.cb1.Set(true)

	// Read IRB
	value = readFromVia(via, circuit, regORBIRB, &context)

	// IRA must hold the latched value
	assert.Equal(t, uint8(0xAA), value)
	// IRQ must be triggering (Low)
	assert.Equal(t, false, circuit.irq.Status())

	// Change input on port B
	circuit.portB.Write(0xFF)

	// Read IRB
	value = readFromVia(via, circuit, regORBIRB, &context)

	// IRA must still hold the latched value
	assert.Equal(t, uint8(0xAA), value)
	// IRQ must be cleared by the read
	assert.Equal(t, true, circuit.irq.Status())

	// Change first 4 bits to output, this will make the value in ORB's first 4 bits be
	// put in Port B
	writeToVia(via, circuit, regDDRB, 0xF0, &context)

	// As we never wrote to ORB value should be 0 on all 4 pins. So output should be 0x0? with ? being F
	// set in the previous steps as input
	assert.Equal(t, uint8(0x0F), via.peripheralPortB.getConnector().Read())

	// Write 0x5A on the output register
	writeToVia(via, circuit, regORBIRB, 0x5A, &context)

	// Force port B output to 0xFF again
	circuit.portB.Write(0xFF)

	// Read IRA
	value = readFromVia(via, circuit, regORBIRB, &context)

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
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	// Set all pins on Port A to input
	writeToVia(via, circuit, regDDRA, 0x00, &context)

	// Set interrupt on CA1 transition enabled
	writeToVia(via, circuit, regIER, 0x82, &context)

	// Set interrupt on positive edge of CA1 (0x01) and CA2 in handshake desired handshake mode
	writeToVia(via, circuit, regPCR, mode|0x01, &context)

	// In handshake mode CA2 is default high
	assert.Equal(t, true, circuit.ca2.Status())
	// At this point IRQ is clear (high)
	assert.Equal(t, true, circuit.irq.Status())

	// Signal Data Ready on CA1
	circuit.ca1.Set(true)

	// Step time and check that IRQ is now active (low)
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, false, circuit.irq.Status())

	// Simulate some more steps and check
	disableChipAndStepTime(via, circuit, &context)
	disableChipAndStepTime(via, circuit, &context)
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, false, circuit.irq.Status())

	// Clear the data ready signal in CA 1
	circuit.ca1.Set(false)
	disableChipAndStepTime(via, circuit, &context)
	disableChipAndStepTime(via, circuit, &context)
	// IRQ stays triggered
	assert.Equal(t, false, circuit.irq.Status())

	// Re-enable the chip and read IRA
	enableChip(circuit)
	readFromVia(via, circuit, regORAIRA, &context)

	// CA2 should have dropped to signal "data taken"
	assert.Equal(t, false, circuit.ca2.Status())
	// IRQ must be cleared
	assert.Equal(t, true, circuit.irq.Status())

	if mode == 0x08 {
		// Simulate some more steps and check, in this mode CA2 will stay
		// low until transition of CB1 happens
		disableChipAndStepTime(via, circuit, &context)
		disableChipAndStepTime(via, circuit, &context)
		disableChipAndStepTime(via, circuit, &context)
		assert.Equal(t, false, circuit.ca2.Status())
		assert.Equal(t, true, circuit.irq.Status())
	} else {
		// In this mode CA2 will stay low for only 1 cycle after read IRA
		// and return to high
		disableChipAndStepTime(via, circuit, &context)
		assert.Equal(t, false, circuit.ca2.Status())
		assert.Equal(t, true, circuit.irq.Status())

		disableChipAndStepTime(via, circuit, &context)
		assert.Equal(t, true, circuit.ca2.Status())
		assert.Equal(t, true, circuit.irq.Status())
		disableChipAndStepTime(via, circuit, &context)
		assert.Equal(t, true, circuit.ca2.Status())
		assert.Equal(t, true, circuit.irq.Status())
	}

	// Signaling data ready on CA1 should make CA2 reset to high
	circuit.ca1.Set(true)
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, true, circuit.ca2.Status())
	assert.Equal(t, false, circuit.irq.Status())
}

func writeHandshakeOnPortA(t *testing.T, mode uint8) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	// Set all pins on Port A to output
	writeToVia(via, circuit, regDDRA, 0xFF, &context)

	// Set interrupt on CA1 transition enabled
	writeToVia(via, circuit, regIER, 0x82, &context)

	// Set interrupt on positive edge of CA1 (0x01) and CA2 in handshake desired handshake mode
	writeToVia(via, circuit, regPCR, mode|0x01, &context)

	// Data taken is low
	assert.Equal(t, false, circuit.ca1.Status())
	// In handshake mode CA2 is default high
	assert.Equal(t, true, circuit.ca2.Status())
	// At this point IRQ is clear (high)
	assert.Equal(t, true, circuit.irq.Status())

	// Write to ORA
	writeToVia(via, circuit, regORAIRA, 0xFF, &context)

	// CA2 will drop to signal "data ready"
	if mode == 0x08 {
		// Simulate some more steps and check, in this mode CA2 will stay
		// low until transition of CB1 happens
		disableChipAndStepTime(via, circuit, &context)
		disableChipAndStepTime(via, circuit, &context)
		disableChipAndStepTime(via, circuit, &context)
		assert.Equal(t, false, circuit.ca2.Status())
	} else {
		// In this mode CA2 will stay low for only 1 cycle
		// and return to high
		disableChipAndStepTime(via, circuit, &context)
		assert.Equal(t, false, circuit.ca2.Status())

		disableChipAndStepTime(via, circuit, &context)
		assert.Equal(t, true, circuit.ca2.Status())
		disableChipAndStepTime(via, circuit, &context)
		assert.Equal(t, true, circuit.ca2.Status())
	}

	// Signal Data Taken on CA1
	circuit.ca1.Set(true)

	// Step time and check that IRQ is now active (low)
	// And CA2 has been returned to high
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, false, circuit.irq.Status())
	assert.Equal(t, true, circuit.ca2.Status())

	// Do some steps and renable the Data Taken flag,
	// IRQ must stay triggered (low)
	disableChipAndStepTime(via, circuit, &context)
	disableChipAndStepTime(via, circuit, &context)
	circuit.ca1.Set(false)
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, false, circuit.irq.Status())

	// Re-enable the chip and write to ORA
	enableChip(circuit)
	writeToVia(via, circuit, regORAIRA, 0xFE, &context)

	// IRQ must be reset and CA2 goes low again
	assert.Equal(t, true, circuit.irq.Status())
	assert.Equal(t, false, circuit.ca2.Status())
}

func writeHandshakeOnPortB(t *testing.T, mode uint8) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	// Set all pins on Port B to output
	writeToVia(via, circuit, regDDRB, 0xFF, &context)

	// Set interrupt on CB1 transition enabled
	writeToVia(via, circuit, regIER, 0x90, &context)

	// Set interrupt on positive edge of CB1 (0x10) and CB2 in handshake desired handshake mode
	writeToVia(via, circuit, regPCR, mode|0x10, &context)

	// Data taken is low
	assert.Equal(t, false, circuit.cb1.Status())
	// In handshake mode CB2 is default high
	assert.Equal(t, true, circuit.cb2.Status())
	// At this point IRQ is clear (high)
	assert.Equal(t, true, circuit.irq.Status())

	// Write to ORB
	writeToVia(via, circuit, regORBIRB, 0xFF, &context)

	// CB2 will drop to signal "data ready"
	if mode == 0x80 {
		// Simulate some more steps and check, in this mode CB2 will stay
		// low until transition of CB1 happens
		disableChipAndStepTime(via, circuit, &context)
		disableChipAndStepTime(via, circuit, &context)
		disableChipAndStepTime(via, circuit, &context)
		assert.Equal(t, false, circuit.cb2.Status())
	} else {
		// In this mode CB2 will stay low for only 1 cycle
		// and return to high
		disableChipAndStepTime(via, circuit, &context)
		assert.Equal(t, false, circuit.cb2.Status())

		disableChipAndStepTime(via, circuit, &context)
		assert.Equal(t, true, circuit.cb2.Status())
		disableChipAndStepTime(via, circuit, &context)
		assert.Equal(t, true, circuit.cb2.Status())
	}

	// Signal Data Taken on CB1
	circuit.cb1.Set(true)

	// Step time and check that IRQ is now active (low)
	// And CB2 has been returned to high
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, false, circuit.irq.Status())
	assert.Equal(t, true, circuit.cb2.Status())

	// Do some steps and renable the Data Taken flag,
	// IRQ must stay triggered (low)
	disableChipAndStepTime(via, circuit, &context)
	disableChipAndStepTime(via, circuit, &context)
	circuit.cb1.Set(false)
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, false, circuit.irq.Status())

	// Re-enable the chip and write to ORB
	enableChip(circuit)
	writeToVia(via, circuit, regORBIRB, 0xFE, &context)

	// IRQ must be reset and CB2 goes low again
	assert.Equal(t, true, circuit.irq.Status())
	assert.Equal(t, false, circuit.cb2.Status())
}

func TestFixedModeOnPortA(t *testing.T) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	// Set CA2 in fixed mode low
	writeToVia(via, circuit, regPCR, 0x0C, &context)

	// CA2 is fixed low
	assert.Equal(t, false, circuit.ca2.Status())

	// Write to ORA
	writeToVia(via, circuit, regORAIRA, 0xFF, &context)

	// Make the time pass and check that CA2 is still high
	disableChipAndStepTime(via, circuit, &context)
	// CA2 is fixed low
	assert.Equal(t, false, circuit.ca2.Status())

	// Changing CA1 should not affect CA2
	circuit.ca1.Set(true)

	disableChipAndStepTime(via, circuit, &context)
	// CA2 is fixed low
	assert.Equal(t, false, circuit.ca2.Status())

	// Set CA2 in fixed mode high
	enableChip(circuit)
	writeToVia(via, circuit, regPCR, 0x0E, &context)

	disableChipAndStepTime(via, circuit, &context)
	// CA2 is fixed high
	assert.Equal(t, true, circuit.ca2.Status())
}

func TestFixedModeOnPortB(t *testing.T) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	// Set CB2 in fixed mode low
	writeToVia(via, circuit, regPCR, 0xC0, &context)

	// CB2 is fixed low
	assert.Equal(t, false, circuit.cb2.Status())

	// Write to ORB
	writeToVia(via, circuit, regORBIRB, 0xFF, &context)

	// Make the time pass and check that CB2 is still high
	disableChipAndStepTime(via, circuit, &context)
	// CB2 is fixed low
	assert.Equal(t, false, circuit.cb2.Status())

	// Changing CB1 should not affect CB2
	circuit.cb1.Set(true)

	disableChipAndStepTime(via, circuit, &context)
	// CB2 is fixed low
	assert.Equal(t, false, circuit.cb2.Status())

	// Set CB2 in fixed mode high
	enableChip(circuit)
	writeToVia(via, circuit, regPCR, 0xE0, &context)

	disableChipAndStepTime(via, circuit, &context)
	// CB2 is fixed high
	assert.Equal(t, true, circuit.cb2.Status())
}

/****************************************************************************************************************
* Timer tests
****************************************************************************************************************/

func TestTimer1OneShotMode(t *testing.T) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	config := coutingTestConfiguration{
		via:                           via,
		circuit:                       circuit,
		lcRegister:                    regT1CL,
		hcRegister:                    regT1CH,
		counterLSB:                    10,
		counterMSB:                    0,
		cyclesToExecute:               11,
		assertPB7:                     false,
		pB7expectedStatus:             false,
		expectedIRQStatusWhenCounting: true,
		expectedInitialIRQStatus:      true,
	}

	// Set ACR to 0x00, for this test is important bit 6 and 7 = 00 -> Timer 1 single shot PB7 disabled
	writeToVia(via, circuit, regACR, 0x00, &context)

	// Enable interrupts for T1 timeout (bit 7 -> enable, bit 6 -> T1)
	writeToVia(via, circuit, regIER, 0xC0, &context)

	// Counts down from 10, it takes N+1 cycles to count down
	// While counting PB7 is not driven and IRQ stays high
	setupAndCountFrom(t, &config, &context)

	// After counting to 0 requires extra 0.5 step
	// to trigger IRQ
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, false, circuit.irq.Status())

	// Counter keeps counting down.
	// When IRQ triggered counter was in the end of FFFF / beginning of FFFE
	// 2 extra cycles will move that to FFFC
	disableChipAndStepTime(via, circuit, &context)
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, uint16(0xFFFC), via.registers.counter1)

	// Reenable the chip
	enableChip(circuit)

	// Clear the interrupt flag by reading T1 low order counter
	counter := readFromVia(via, circuit, regT1CL, &context)
	assert.Equal(t, uint8(0xFB), counter)
	assert.Equal(t, true, circuit.irq.Status())

	// Repeats the couting from 10
	setupAndCountFrom(t, &config, &context)
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, false, circuit.irq.Status())

	// Reenable the chip
	enableChip(circuit)

	// Set ACR to 0x80, for this test is important bit 6 and 7 = 10 -> Timer 1 single shot PB7 enabled
	// When ACR sets PB7 as output, line goes high
	writeToVia(via, circuit, regACR, 0x80, &context)
	assert.Equal(t, true, circuit.portB.GetBusLine(7).Status())

	// Will now test PB7 behaviour
	config.assertPB7 = true
	config.pB7expectedStatus = false

	// Repeats the couting from 10, now evaluating the
	// PB7 flag to stay low while counting
	setupAndCountFrom(t, &config, &context)
	disableChipAndStepTime(via, circuit, &context)

	// At this point IRQ is set (low) and PB7 goes back to high
	assert.Equal(t, false, circuit.irq.Status())
	assert.Equal(t, true, circuit.portB.GetBusLine(7).Status())

	// Counter kept decreasing and rolled over to 0xFFFE
	assert.Equal(t, uint16(0xFFFE), via.registers.counter1)
	enableChip(circuit)
	t1HighCounter := readFromVia(via, circuit, regT1CH, &context)
	assert.Equal(t, uint8(0xFF), t1HighCounter)
}

func TestTimer1FreeRunMode(t *testing.T) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	config := coutingTestConfiguration{
		via:                           via,
		circuit:                       circuit,
		lcRegister:                    regT1CL,
		hcRegister:                    regT1CH,
		counterLSB:                    10,
		counterMSB:                    0,
		cyclesToExecute:               11,
		assertPB7:                     true,
		pB7expectedStatus:             false,
		expectedIRQStatusWhenCounting: true,
		expectedInitialIRQStatus:      true,
	}

	// Set ACR to 0x11, for this test is important bit 6 and 7 = 11 -> Timer 1 free run and PB7 enabled
	// Line 7 goes high when ACR is set to output
	writeToVia(via, circuit, regACR, 0xC0, &context)
	assert.Equal(t, true, circuit.portB.GetBusLine(7).Status())

	// Enable interrupts for T1 timeout (bit 7 -> enable, bit 6 -> T1)
	writeToVia(via, circuit, regIER, 0xC0, &context)

	// Counts down from 10, it takes N+1 cycles to count down
	// While counting PB7 is driven low and IRQ stays high
	setupAndCountFrom(t, &config, &context)

	// After counting to 0 requires extra 0.5 step
	// to trigger IRQ and port B toggles high.
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, false, circuit.irq.Status())
	assert.Equal(t, true, circuit.portB.GetBusLine(7).Status())

	// We'll keep counting. On Free-Run mode the PB7 is expected to toggle
	// between states, since last countdown was low, now is expected high
	// until timer reaches zero. Since we won't reset the interrupt, we
	// expect that one to remain low.
	config.pB7expectedStatus = true
	config.expectedIRQStatusWhenCounting = false
	countToTarget(t, &config, &context)

	// After couting to zero, we need one more step to transition.
	// PB7 will go low, IRQ remains low (unchanged)
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, false, circuit.irq.Status())
	assert.Equal(t, false, circuit.portB.GetBusLine(7).Status())

	// Clear the interrupt flag by reading T1 low order counter
	// This spent one cycle from the new counting so counter will be 9.
	enableChip(circuit)
	counter := readFromVia(via, circuit, regT1CL, &context)
	assert.Equal(t, uint8(0x09), counter)
	assert.Equal(t, true, circuit.irq.Status())

	// Since we consumed 1 cycle to read the counter and reset the IRQ
	// we will now count 10 cycles, also now IRQ is expected high now.
	// This cycle PB7 is expected low.
	config.cyclesToExecute = 10
	config.expectedIRQStatusWhenCounting = true
	config.pB7expectedStatus = false
	countToTarget(t, &config, &context)

	// One extra step to update status lines,
	// IRQ will trigger (go low) and PB7 will toggle high
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, false, circuit.irq.Status())
	assert.Equal(t, true, circuit.portB.GetBusLine(7).Status())

	// Clear the interrupt flag by writing zero to the high latch
	enableChip(circuit)
	writeToVia(via, circuit, regT1HL, 0x01, &context)
	assert.Equal(t, uint8(0x01), via.registers.highLatches1)
	assert.Equal(t, true, circuit.irq.Status())
}

func TestTimer2OneShotMode(t *testing.T) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	config := coutingTestConfiguration{
		via:                           via,
		circuit:                       circuit,
		lcRegister:                    regT2CL,
		hcRegister:                    regT2CH,
		counterLSB:                    10,
		counterMSB:                    0,
		cyclesToExecute:               11,
		assertPB7:                     false,
		pB7expectedStatus:             false,
		expectedIRQStatusWhenCounting: true,
		expectedInitialIRQStatus:      true,
	}

	// Set ACR to 0x00, for this test is important bit 5 = 00 -> Timer 2 single shot
	writeToVia(via, circuit, regACR, 0x00, &context)

	// Enable interrupts for T2 timeout (bit 7 -> enable, bit 5 -> T2)
	writeToVia(via, circuit, regIER, 0xA0, &context)

	// Counts down from 10, it takes N+1 cycles to count down
	// While counting PB7 is not driven and IRQ stays high
	setupAndCountFrom(t, &config, &context)

	// After counting to 0 requires extra 0.5 step
	// to trigger IRQ
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, false, circuit.irq.Status())

	// Counter keeps counting down.
	// When IRQ triggered counter was in the end of FFFF / beginning of FFFE
	// 2 extra cycles will move that to FFFC
	disableChipAndStepTime(via, circuit, &context)
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, uint16(0xFFFC), via.registers.counter2)

	// Reenable the chip
	enableChip(circuit)

	// Clear the interrupt flag by reading T2 low order counter
	counter := readFromVia(via, circuit, regT2CL, &context)
	assert.Equal(t, uint8(0xFB), counter)
	assert.Equal(t, true, circuit.irq.Status())

	// Repeats the couting from 10
	setupAndCountFrom(t, &config, &context)
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, false, circuit.irq.Status())

	// Counter kept going down and rolled over to 0xFFFE
	assert.Equal(t, uint16(0xFFFE), via.registers.counter2)
	enableChip(circuit)
	t2HighCounter := readFromVia(via, circuit, regT2CH, &context)
	assert.Equal(t, uint8(0xFF), t2HighCounter)

}

func TestTimer2PulseCountingMode(t *testing.T) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	// Set ACR to 0x00, for this test is important bit 5 = 1 -> Timer 2 pulse counting
	writeToVia(via, circuit, regACR, 0x20, &context)

	// Enable interrupts for T2 timeout (bit 7 -> enable, bit 5 -> T2)
	writeToVia(via, circuit, regIER, 0xA0, &context)

	// Set PB6 high so it doesn't count down
	circuit.portB.GetBusLine(6).Set(true)

	// Set counter to 10
	writeToVia(via, circuit, regT2CL, 10, &context)
	writeToVia(via, circuit, regT2CH, 0x00, &context)

	// Pass 2 cycles, counter should still be in 10
	disableChipAndStepTime(via, circuit, &context)
	disableChipAndStepTime(via, circuit, &context)

	assert.Equal(t, uint16(10), via.registers.counter2)

	// According to https://web.archive.org/web/20220708103848if_/http://archive.6502.org/datasheets/synertek_sy6522.pdf
	// IRQ is set when counter rolls over to FFFF
	// TODO: I cannot find documentation online about this behaviour and official manual states that this happens in the
	// beggining of cycle with 0 in the counter. Might need to test in real hardware
	for n := 11; n > 0; n-- {
		circuit.portB.GetBusLine(6).Set(false)
		disableChipAndStepTime(via, circuit, &context)
		assert.Equal(t, uint16(n-2), via.registers.counter2)

		circuit.portB.GetBusLine(6).Set(true)
		disableChipAndStepTime(via, circuit, &context)
		assert.Equal(t, uint16(n-2), via.registers.counter2)
	}

	// After counting to 0 requires extra 0.5 step
	// to trigger IRQ
	circuit.portB.GetBusLine(6).Set(false)
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, false, circuit.irq.Status())
}

/****************************************************************************************************************
* Shift register tests
****************************************************************************************************************/

type shiftingTestConfiguration struct {
	via              *Via65C22S
	circuit          *testCircuit
	automatic        bool  // Shifting is manual or automatic
	numberOfCycles   uint8 // Total of shifing cycles to execute
	bitValue         bool  // Value to shift in (this will be set in CB2)
	dataChangeCyle   uint8 // Cycle count when CB2 will be switched to the value to shift in
	manualShiftStart uint8 // Cycle when CB1 will be manually drop to start shifting in (automatic must be false)
	manualShiftStop  uint8 // Cycle when CB1 will be manually raised to stop shifting in (automatic must be false)
	outputMode       bool
}

func executeShiftingCycle(t *testing.T, config *shiftingTestConfiguration, context *common.StepContext) {
	for i := range config.numberOfCycles {
		if !config.automatic {
			if i == config.manualShiftStart {
				config.circuit.cb1.Set(false)
			}

			if i == config.manualShiftStop {
				config.circuit.cb1.Set(true)
			}
		}

		if i == config.dataChangeCyle && !config.outputMode {
			config.circuit.cb2.Set(config.bitValue)
		}

		disableChipAndStepTime(config.via, config.circuit, context)
		assert.Equal(t, true, config.circuit.irq.Status())

		mustEvaluateOutput := config.automatic || (!config.automatic && i > config.manualShiftStart && i <= config.manualShiftStop)

		if config.outputMode && mustEvaluateOutput {
			assert.Equal(t, !config.bitValue, config.circuit.cb2.Status())
		}

		if config.automatic {
			assert.Equal(t, false, config.circuit.cb1.Status())
		}
	}
}

func executeNonShiftingCycle(t *testing.T, config *shiftingTestConfiguration, context *common.StepContext) {
	if !config.automatic {
		config.circuit.cb1.Set(true)
	}

	for range config.numberOfCycles {
		disableChipAndStepTime(config.via, config.circuit, context)
		assert.Equal(t, true, config.circuit.irq.Status())

		if config.automatic {
			assert.Equal(t, true, config.circuit.cb1.Status())
		}
	}
}

func TestShiftInAtT2Rate(t *testing.T) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	// Set ACR to 0x04, for this test is important bit 4, 3 and 2 = 001 (Shift under T2 control)
	writeToVia(via, circuit, regACR, 0x04, &context)

	// Enable interrupts for SR completion (Bit 2)
	writeToVia(via, circuit, regIER, 0x84, &context)

	// Trigger shifting by writing 0 to SR
	writeToVia(via, circuit, regSR, 0x00, &context)

	// Write T2 low latch to set shifting every 5 cycles
	writeToVia(via, circuit, regT2CL, 0x05, &context)

	// TODO: Due to the above set of timer, isn't it waiting one extra cycle?

	// IRQ is not triggered and CB1 is raised high for 1.5 cycles (see WDC data sheet for 65C22S page 22)
	assert.Equal(t, true, circuit.irq.Status())
	assert.Equal(t, true, circuit.cb1.Status())
	// SR is 0
	assert.Equal(t, uint8(0x00), via.registers.shiftRegister)

	// Step will just decrement timer, no expected change
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, true, circuit.irq.Status())
	assert.Equal(t, true, circuit.cb1.Status())

	// Step will just decrement timer, no expected change
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, true, circuit.irq.Status())
	assert.Equal(t, true, circuit.cb1.Status())

	config := shiftingTestConfiguration{
		via:            via,
		circuit:        circuit,
		automatic:      true,
		numberOfCycles: 7,
		bitValue:       true,
		dataChangeCyle: 3,
	}

	// Execute shifting cycle, a total of N+2 cycles will be needed
	// to complete the shifting. CB1 will be down until cycle completion
	executeShiftingCycle(t, &config, &context)
	assert.Equal(t, uint8(0x01), via.registers.shiftRegister)
	executeNonShiftingCycle(t, &config, &context)

	// Shift a 0 on second cycle
	config.dataChangeCyle = 1
	config.bitValue = false
	executeShiftingCycle(t, &config, &context)
	assert.Equal(t, uint8(0x02), via.registers.shiftRegister)
	executeNonShiftingCycle(t, &config, &context)

	// Shift a 1 on last cycle
	config.dataChangeCyle = 6
	config.bitValue = true
	executeShiftingCycle(t, &config, &context)
	assert.Equal(t, uint8(0x05), via.registers.shiftRegister)
	executeNonShiftingCycle(t, &config, &context)

	// Shift a 0 on first cycle
	config.dataChangeCyle = 0
	config.bitValue = false
	executeShiftingCycle(t, &config, &context)
	assert.Equal(t, uint8(0x0a), via.registers.shiftRegister)
	executeNonShiftingCycle(t, &config, &context)

	// Shift 3 bits in one in a row
	config.dataChangeCyle = 0
	config.bitValue = true
	for range 3 {
		executeShiftingCycle(t, &config, &context)
		executeNonShiftingCycle(t, &config, &context)
	}

	assert.Equal(t, uint8(0x57), via.registers.shiftRegister)

	// Shift 1 more bit but stop 1 cycle short of shift completion
	config.dataChangeCyle = 0
	config.bitValue = true
	config.numberOfCycles = 6
	executeShiftingCycle(t, &config, &context)

	// Execute last shift for a full byte shifted
	disableChipAndStepTime(via, circuit, &context)

	// 8 bits were shifted, IRQ is triggered shifting stops.
	assert.Equal(t, uint8(0xaf), via.registers.shiftRegister)
	assert.Equal(t, false, circuit.irq.Status())
	assert.Equal(t, true, circuit.cb1.Status())
}

func TestShiftInAtClockRate(t *testing.T) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	// Set ACR to 0x04, for this test is important bit 4, 3 and 2 = 010 (Shift under clock control)
	writeToVia(via, circuit, regACR, 0x08, &context)

	// Enable interrupts for SR completion (Bit 2)
	writeToVia(via, circuit, regIER, 0x84, &context)

	// Trigger shifting by writing 0 to SR
	writeToVia(via, circuit, regSR, 0x00, &context)

	// IRQ is not triggered and CB1 is raised high for 1.5 cycles (see WDC data sheet for 65C22S page 22)
	// SR is 0

	assert.Equal(t, true, circuit.irq.Status())
	assert.Equal(t, true, circuit.cb1.Status())
	assert.Equal(t, uint8(0x00), via.registers.shiftRegister)

	config := shiftingTestConfiguration{
		via:            via,
		circuit:        circuit,
		automatic:      true,
		numberOfCycles: 1,
		bitValue:       true,
		dataChangeCyle: 0,
	}

	// Shift a 1
	executeShiftingCycle(t, &config, &context)
	assert.Equal(t, uint8(0x01), via.registers.shiftRegister)
	executeNonShiftingCycle(t, &config, &context)

	// Shift a 0
	config.bitValue = false
	executeShiftingCycle(t, &config, &context)
	assert.Equal(t, uint8(0x02), via.registers.shiftRegister)
	executeNonShiftingCycle(t, &config, &context)

	// Shift a 1
	config.bitValue = true
	executeShiftingCycle(t, &config, &context)
	assert.Equal(t, uint8(0x05), via.registers.shiftRegister)
	executeNonShiftingCycle(t, &config, &context)

	// Shift a 0
	config.bitValue = false
	executeShiftingCycle(t, &config, &context)
	assert.Equal(t, uint8(0x0a), via.registers.shiftRegister)
	executeNonShiftingCycle(t, &config, &context)

	// Shift 3 bits in one in a row
	config.bitValue = true
	for range 3 {
		executeShiftingCycle(t, &config, &context)
		executeNonShiftingCycle(t, &config, &context)
	}

	assert.Equal(t, uint8(0x57), via.registers.shiftRegister)

	// Last shifting cycle
	disableChipAndStepTime(via, circuit, &context)

	// 8 bits were shifted, IRQ is triggered shifting stops.
	assert.Equal(t, uint8(0xaf), via.registers.shiftRegister)
	assert.Equal(t, false, circuit.irq.Status())
	assert.Equal(t, true, circuit.cb1.Status())
}

func TestShiftInAtExternalRate(t *testing.T) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	// Set ACR to 0x0C, for this test is important bit 4, 3 and 2 = 011 (Shift under clock control)
	writeToVia(via, circuit, regACR, 0x0C, &context)

	// Shifting is controlled externally when CB1 is dropped
	// Let's make it high for now
	circuit.cb1.Set(true)

	// Enable interrupts for SR completion (Bit 2)
	writeToVia(via, circuit, regIER, 0x84, &context)

	// Trigger shifting by writing 0 to SR
	writeToVia(via, circuit, regSR, 0x00, &context)

	// IRQ is not triggered, SR is 0
	assert.Equal(t, true, circuit.irq.Status())
	assert.Equal(t, uint8(0x00), via.registers.shiftRegister)

	config := shiftingTestConfiguration{
		via:              via,
		circuit:          circuit,
		automatic:        false,
		numberOfCycles:   10,
		bitValue:         true,
		dataChangeCyle:   6,
		manualShiftStart: 5,
		manualShiftStop:  7,
	}

	// Shift a 1
	executeShiftingCycle(t, &config, &context)
	assert.Equal(t, uint8(0x01), via.registers.shiftRegister)
	executeNonShiftingCycle(t, &config, &context)

	// Shift a 0, but because data in CB2 changes too late
	circuit.cb2.Set(false)
	config.bitValue = true
	config.dataChangeCyle = 9
	executeShiftingCycle(t, &config, &context)
	assert.Equal(t, uint8(0x02), via.registers.shiftRegister)
	executeNonShiftingCycle(t, &config, &context)

	// Shift a 1
	config.bitValue = true
	config.dataChangeCyle = 7
	executeShiftingCycle(t, &config, &context)
	assert.Equal(t, uint8(0x05), via.registers.shiftRegister)
	executeNonShiftingCycle(t, &config, &context)

	// Shift a 0, data changing early
	config.bitValue = false
	config.dataChangeCyle = 1
	executeShiftingCycle(t, &config, &context)
	assert.Equal(t, uint8(0x0a), via.registers.shiftRegister)
	executeNonShiftingCycle(t, &config, &context)

	// Shift 3 bits in one in a row
	// Shift a 1
	config.bitValue = true
	config.dataChangeCyle = 6
	for range 3 {
		executeShiftingCycle(t, &config, &context)
		executeNonShiftingCycle(t, &config, &context)
	}

	assert.Equal(t, uint8(0x57), via.registers.shiftRegister)

	// Shift 1 more bit but stop 1 cycle short of shift completion
	config.dataChangeCyle = 0
	config.bitValue = true
	config.numberOfCycles = 6
	config.manualShiftStart = 0
	config.manualShiftStop = 10
	executeShiftingCycle(t, &config, &context)

	// Raise CB1 and execute 1 step to complete shifting last bit
	// of a full byte
	circuit.cb1.Set(true)
	disableChipAndStepTime(via, circuit, &context)

	// 8 bits were shifted, IRQ is triggered shifting stops.
	assert.Equal(t, uint8(0xaf), via.registers.shiftRegister)
	assert.Equal(t, false, circuit.irq.Status())
	assert.Equal(t, true, circuit.cb1.Status())
}

func TestShiftOutAtT2Rate(t *testing.T) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	// Set ACR to 0x14, for this test is important bit 4, 3 and 2 = 101 (Shift under T2 control)
	writeToVia(via, circuit, regACR, 0x14, &context)

	// Enable interrupts for SR completion (Bit 2)
	writeToVia(via, circuit, regIER, 0x84, &context)

	// Trigger shifting by writing 0 to SR
	writeToVia(via, circuit, regSR, 0xAA, &context)

	// Write T2 low latch to set shifting every 5 cycles
	writeToVia(via, circuit, regT2CL, 0x05, &context)

	// IRQ is not triggered and CB1 is raised high for 1.5 cycles (see WDC data sheet for 65C22S page 22)
	assert.Equal(t, true, circuit.irq.Status())
	assert.Equal(t, true, circuit.cb1.Status())
	assert.Equal(t, true, circuit.cb2.Status())
	assert.Equal(t, uint8(0xAA), via.registers.shiftRegister)

	// Step will just decrement timer, no expected change
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, true, circuit.irq.Status())
	assert.Equal(t, true, circuit.cb1.Status())
	assert.Equal(t, true, circuit.cb2.Status())

	// Step will just decrement timer, no expected change
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, true, circuit.irq.Status())
	assert.Equal(t, true, circuit.cb1.Status())
	assert.Equal(t, true, circuit.cb2.Status())

	config := shiftingTestConfiguration{
		via:            via,
		circuit:        circuit,
		automatic:      true,
		numberOfCycles: 7,
		bitValue:       true,
		outputMode:     true,
	}

	// Shift 7 bytes, AA on the SR will produce alternating 1 and 0 as CB2 output.
	// SR is shifter right and bit is pulled from bit 7 and reintroduced to bit 0.
	// This means that value will alternate between 0xAA and 0x55.
	// This sequence stops one bit short of a full byte
	for range 7 {
		executeShiftingCycle(t, &config, &context)

		if config.bitValue {
			assert.Equal(t, uint8(0x55), via.registers.shiftRegister)
		} else {
			assert.Equal(t, uint8(0xaa), via.registers.shiftRegister)
		}

		executeNonShiftingCycle(t, &config, &context)

		config.bitValue = !config.bitValue
	}

	// Shift 1 more bit but stop 1 cycle short of shift completion
	config.bitValue = false
	config.numberOfCycles = 6
	executeShiftingCycle(t, &config, &context)

	// Execute last shift for a full byte shifted
	disableChipAndStepTime(via, circuit, &context)

	// 8 bits were shifted out, IRQ is triggered shifting stops.
	assert.Equal(t, uint8(0xaa), via.registers.shiftRegister)
	assert.Equal(t, false, circuit.irq.Status())
	assert.Equal(t, true, circuit.cb1.Status())
	assert.Equal(t, true, circuit.cb2.Status())

	// Reads the SR
	enableChip(circuit)
	value := readFromVia(via, circuit, regSR, &context)
	assert.Equal(t, uint8(0xaa), value)
}

func TestShiftOutAtClockRate(t *testing.T) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	// Set ACR to 0x18, for this test is important bit 4, 3 and 2 = 110 (Shift under T2 control)
	writeToVia(via, circuit, regACR, 0x18, &context)

	// Enable interrupts for SR completion (Bit 2)
	writeToVia(via, circuit, regIER, 0x84, &context)

	// Trigger shifting by writing 0 to SR
	writeToVia(via, circuit, regSR, 0xAA, &context)

	// TODO: Not waiting 2 cycles to start ?

	assert.Equal(t, true, circuit.irq.Status())
	assert.Equal(t, true, circuit.cb1.Status())
	assert.Equal(t, true, circuit.cb2.Status())
	assert.Equal(t, uint8(0xAA), via.registers.shiftRegister)

	config := shiftingTestConfiguration{
		via:            via,
		circuit:        circuit,
		automatic:      true,
		numberOfCycles: 1,
		bitValue:       true,
		outputMode:     true,
	}

	// Shift 7 bytes, AA on the SR will produce alternating 1 and 0 as CB2 output.
	// SR is shifter right and bit is pulled from bit 7 and reintroduced to bit 0.
	// This means that value will alternate between 0xAA and 0x55.
	// This sequence stops one bit short of a full byte
	for range 7 {
		executeShiftingCycle(t, &config, &context)

		if config.bitValue {
			assert.Equal(t, uint8(0x55), via.registers.shiftRegister)
		} else {
			assert.Equal(t, uint8(0xaa), via.registers.shiftRegister)
		}

		executeNonShiftingCycle(t, &config, &context)

		config.bitValue = !config.bitValue
	}

	// Execute last shift for a full byte shifted
	disableChipAndStepTime(via, circuit, &context)

	// 8 bits were shifted out, IRQ is triggered shifting stops.
	assert.Equal(t, uint8(0xaa), via.registers.shiftRegister)
	assert.Equal(t, false, circuit.irq.Status())
	assert.Equal(t, true, circuit.cb1.Status())
	assert.Equal(t, true, circuit.cb2.Status())

}

func TestShiftOutAtExternalRate(t *testing.T) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	// Set ACR to 0x1C, for this test is important bit 4, 3 and 2 = 111 (Shift under clock control)
	writeToVia(via, circuit, regACR, 0x1C, &context)

	// Shifting is controlled externally when CB1 is dropped
	// Let's make it high for now
	circuit.cb1.Set(true)

	// Enable interrupts for SR completion (Bit 2)
	writeToVia(via, circuit, regIER, 0x84, &context)

	// Trigger shifting by writing 0xAA to SR
	writeToVia(via, circuit, regSR, 0xAA, &context)

	// IRQ is not triggered, SR is 0
	assert.Equal(t, true, circuit.irq.Status())
	assert.Equal(t, uint8(0xAA), via.registers.shiftRegister)

	config := shiftingTestConfiguration{
		via:              via,
		circuit:          circuit,
		automatic:        false,
		numberOfCycles:   10,
		bitValue:         true,
		manualShiftStart: 5,
		manualShiftStop:  7,
		outputMode:       true,
	}

	// Shift a 1
	executeShiftingCycle(t, &config, &context)
	assert.Equal(t, uint8(0x55), via.registers.shiftRegister)
	executeNonShiftingCycle(t, &config, &context)

	// Shift a 0
	config.bitValue = false
	config.manualShiftStart = 0
	executeShiftingCycle(t, &config, &context)
	assert.Equal(t, uint8(0xAA), via.registers.shiftRegister)
	executeNonShiftingCycle(t, &config, &context)

	// Shift a 1
	config.bitValue = true
	config.manualShiftStart = 1
	executeShiftingCycle(t, &config, &context)
	assert.Equal(t, uint8(0x55), via.registers.shiftRegister)
	executeNonShiftingCycle(t, &config, &context)

	// Shift a 0
	config.bitValue = false
	config.manualShiftStart = 4
	executeShiftingCycle(t, &config, &context)
	assert.Equal(t, uint8(0xAA), via.registers.shiftRegister)
	executeNonShiftingCycle(t, &config, &context)

	// Shift a 1
	config.bitValue = true
	config.manualShiftStart = 6
	executeShiftingCycle(t, &config, &context)
	assert.Equal(t, uint8(0x55), via.registers.shiftRegister)
	executeNonShiftingCycle(t, &config, &context)

	// Shift a 0
	config.bitValue = false
	config.manualShiftStart = 2
	executeShiftingCycle(t, &config, &context)
	assert.Equal(t, uint8(0xAA), via.registers.shiftRegister)
	executeNonShiftingCycle(t, &config, &context)

	// Shift a 1
	config.bitValue = true
	config.manualShiftStart = 3
	executeShiftingCycle(t, &config, &context)
	assert.Equal(t, uint8(0x55), via.registers.shiftRegister)
	executeNonShiftingCycle(t, &config, &context)

	// Shift 1 more bit but stop 1 cycle short of shift completion
	config.bitValue = false
	config.numberOfCycles = 6
	config.manualShiftStart = 0
	config.manualShiftStop = 10
	executeShiftingCycle(t, &config, &context)

	// Raise CB1 and execute 1 step to complete shifting last bit
	// of a full byte
	circuit.cb1.Set(true)
	disableChipAndStepTime(via, circuit, &context)

	// 8 bits were shifted, IRQ is triggered shifting stops.
	assert.Equal(t, uint8(0xaa), via.registers.shiftRegister)
	assert.Equal(t, false, circuit.irq.Status())
	assert.Equal(t, true, circuit.cb1.Status())
	assert.Equal(t, true, circuit.cb2.Status())
}

func TestShiftOutAtFreeRate(t *testing.T) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	// Set ACR to 0x10, for this test is important bit 4, 3 and 2 = 100 (Shift free under T2 control)
	writeToVia(via, circuit, regACR, 0x10, &context)

	// Enable interrupts for SR completion (Bit 2)
	writeToVia(via, circuit, regIER, 0x84, &context)

	// Trigger shifting by writing 0 to SR
	writeToVia(via, circuit, regSR, 0xAA, &context)

	// Write T2 low latch to set shifting every 5 cycles
	writeToVia(via, circuit, regT2CL, 0x05, &context)

	// IRQ is not triggered and CB1 is raised high for 1.5 cycles (see WDC data sheet for 65C22S page 22)
	assert.Equal(t, true, circuit.irq.Status())
	assert.Equal(t, true, circuit.cb1.Status())
	assert.Equal(t, true, circuit.cb2.Status())
	assert.Equal(t, uint8(0xAA), via.registers.shiftRegister)

	// Step will just decrement timer, no expected change
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, true, circuit.irq.Status())
	assert.Equal(t, true, circuit.cb1.Status())
	assert.Equal(t, true, circuit.cb2.Status())

	// Step will just decrement timer, no expected change
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, true, circuit.irq.Status())
	assert.Equal(t, true, circuit.cb1.Status())
	assert.Equal(t, true, circuit.cb2.Status())

	config := shiftingTestConfiguration{
		via:            via,
		circuit:        circuit,
		automatic:      true,
		numberOfCycles: 7,
		bitValue:       true,
		outputMode:     true,
	}

	// Shift 2 full sequences of bytes, AA on the SR will produce alternating 1 and 0 as CB2 output.
	// SR is shifter right and bit is pulled from bit 7 and reintroduced to bit 0.
	// This means that value will alternate between 0xAA and 0x55.
	// Since the Shift Register bit 7 (SR7) is recirculated back into bit 0, the 8 bits loaded into the
	// shift register will be clocked onto CB2 repetitively. In this mode the shift register counter is disabled.
	for range 16 {
		executeShiftingCycle(t, &config, &context)

		if config.bitValue {
			assert.Equal(t, uint8(0x55), via.registers.shiftRegister)
		} else {
			assert.Equal(t, uint8(0xaa), via.registers.shiftRegister)
		}

		executeNonShiftingCycle(t, &config, &context)

		config.bitValue = !config.bitValue
	}
}

/****************************************************************************************************************
* Interrupt flag R/W tests
****************************************************************************************************************/

func TestCausingAnInterruptInT1AndT2andClearByWritingToIFR(t *testing.T) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	configT1 := coutingTestConfiguration{
		via:                           via,
		circuit:                       circuit,
		lcRegister:                    regT1CL,
		hcRegister:                    regT1CH,
		counterLSB:                    10,
		counterMSB:                    0,
		cyclesToExecute:               11,
		assertPB7:                     false,
		pB7expectedStatus:             false,
		expectedIRQStatusWhenCounting: true,
		expectedInitialIRQStatus:      true,
	}

	configT2 := coutingTestConfiguration{
		via:                           via,
		circuit:                       circuit,
		lcRegister:                    regT2CL,
		hcRegister:                    regT2CH,
		counterLSB:                    10,
		counterMSB:                    0,
		cyclesToExecute:               11,
		assertPB7:                     false,
		pB7expectedStatus:             false,
		expectedIRQStatusWhenCounting: false,
		expectedInitialIRQStatus:      false,
	}

	// Set ACR to 0x00, for this test is important bit 6 and 7 = 00 -> Timer 1 single shot PB7 disabled
	writeToVia(via, circuit, regACR, 0x00, &context)

	// Enable interrupts for T1 and T2 timeout (bit 7 -> enable, bit 6 -> T1, bit 5 -> T2)
	writeToVia(via, circuit, regIER, 0xE0, &context)

	// Counts down from 10 on T1, it takes N+1 cycles to count down
	setupAndCountFrom(t, &configT1, &context)
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, false, circuit.irq.Status())

	enableChip(circuit)
	// Counts down from 10 on T2, it takes N+1 cycles to count down
	setupAndCountFrom(t, &configT2, &context)
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, false, circuit.irq.Status())

	// Reenable the chip and read IFR, value should be IRQ Enabled (Bit 7) | Timout T1 (Bit 6) | Timeout T2 (Bit 5)
	enableChip(circuit)
	ifr := readFromVia(via, circuit, regIFR, &context)
	assert.Equal(t, uint8(0xE0), ifr)

	// Clear T2 only by writing to IFR
	writeToVia(via, circuit, regIFR, 0x20, &context)

	// Assert that IRQ nad T1 are still set
	ifr = readFromVia(via, circuit, regIFR, &context)
	assert.Equal(t, uint8(0xC0), ifr)

	// Clear T1 only by writing to IFR
	writeToVia(via, circuit, regIFR, 0x40, &context)

	// Since now T1 is cleared and there are not active flags IRQ flag (bit 7) also clears
	// and IRQ line goes high
	ifr = readFromVia(via, circuit, regIFR, &context)
	assert.Equal(t, uint8(0x00), ifr)
	assert.Equal(t, true, circuit.irq.Status())

	// IER is still enabled for T1 and T2 (bit 7 is always returned as 0 when reading)
	ier := readFromVia(via, circuit, regIER, &context)
	assert.Equal(t, uint8(0x60), ier)

	// Writing to IER with bit 7 in 0 clears the corresponding bits.
	// Disable interrupts for T2
	writeToVia(via, circuit, regIER, 0x20, &context)
	ier = readFromVia(via, circuit, regIER, &context)
	assert.Equal(t, uint8(0x40), ier)
}

/****************************************************************************************************************
* R/W with no handshake when R/W from 0x0F
****************************************************************************************************************/

func TestNoHandshakeOnPortAWhenReadingOnRSEqualF(t *testing.T) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	// Set all pins on Port A to input
	writeToVia(via, circuit, regDDRA, 0x00, &context)

	// Set interrupt on CA1 transition enabled
	writeToVia(via, circuit, regIER, 0x82, &context)

	// Set interrupt on positive edge of CA1 (0x01) and CA2 in handshake desired handshake mode
	writeToVia(via, circuit, regPCR, 0x08|0x01, &context)

	// In handshake mode CA2 is default high
	assert.Equal(t, true, circuit.ca2.Status())
	// At this point IRQ is clear (high)
	assert.Equal(t, true, circuit.irq.Status())

	// Signal Data Ready on CA1
	circuit.ca1.Set(true)

	// Step time and check that IRQ is now active (low)
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, false, circuit.irq.Status())

	// Simulate some more steps and check
	disableChipAndStepTime(via, circuit, &context)
	disableChipAndStepTime(via, circuit, &context)
	disableChipAndStepTime(via, circuit, &context)
	assert.Equal(t, false, circuit.irq.Status())

	// Clear the data ready signal in CA 1
	circuit.ca1.Set(false)
	disableChipAndStepTime(via, circuit, &context)
	disableChipAndStepTime(via, circuit, &context)
	// IRQ stays triggered
	assert.Equal(t, false, circuit.irq.Status())

	// Re-enable the chip and read IRA
	enableChip(circuit)
	readFromVia(via, circuit, regORAIRANoHandshake, &context)

	// CA2 will not dropp to signal "data taken" as no handshake is triggered
	assert.Equal(t, true, circuit.ca2.Status())
	// IRQ must be cleared
	assert.Equal(t, true, circuit.irq.Status())

	// Set latching enabled on Port A
	writeToVia(via, circuit, regACR, 0x01, &context)

	// With latching enabled the value of CA2 is also not changed
	readFromVia(via, circuit, regORAIRANoHandshake, &context)

	// CA2 will not dropp to signal "data taken" as no handshake is triggered
	assert.Equal(t, true, circuit.ca2.Status())
	// IRQ must be cleared
	assert.Equal(t, true, circuit.irq.Status())

}

func TestNoHandshakeOnPortAWhenWritingOnRSEqualF(t *testing.T) {
	context := common.NewStepContext()

	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	// Set all pins on Port A to output
	writeToVia(via, circuit, regDDRA, 0xFF, &context)

	// Set interrupt on CA1 transition enabled
	writeToVia(via, circuit, regIER, 0x82, &context)

	// Set interrupt on positive edge of CA1 (0x01) and CA2 in handshake desired handshake mode
	writeToVia(via, circuit, regPCR, 0x08|0x01, &context)

	// Data taken is low
	assert.Equal(t, false, circuit.ca1.Status())
	// In handshake mode CA2 is default high
	assert.Equal(t, true, circuit.ca2.Status())
	// At this point IRQ is clear (high)
	assert.Equal(t, true, circuit.irq.Status())

	// Write to ORA
	writeToVia(via, circuit, regORAIRANoHandshake, 0xFF, &context)

	// CA2 will stay high as no signal "data ready" will be done when writing to 0x0F
	assert.Equal(t, true, circuit.ca2.Status())
}

/****************************************************************************************************************
* Getters Test
****************************************************************************************************************/

func TestVia65C22SGetters(t *testing.T) {
	// Setup using constructor
	via := NewVia65C22()

	// Directly assign test values to registers
	via.registers.outputRegisterA = 0xAA
	via.registers.outputRegisterB = 0xBB
	via.registers.inputRegisterA = 0xCC
	via.registers.inputRegisterB = 0xDD
	via.registers.dataDirectionRegisterA = 0xEE
	via.registers.dataDirectionRegisterB = 0xFF
	via.registers.lowLatches2 = 0x11
	via.registers.lowLatches1 = 0x22
	via.registers.highLatches2 = 0x33
	via.registers.highLatches1 = 0x44
	via.registers.counter2 = 0x5555
	via.registers.counter1 = 0x6666
	via.registers.shiftRegister = 0x77
	via.registers.auxiliaryControl = 0x88
	via.registers.peripheralControl = 0x99

	// Set interrupt-related registers
	via.registers.interrupts.setInterruptFlagValue(0x55)
	via.registers.interrupts.setInterruptEnabledFlag(0xAA)

	// Test cases
	t.Run("GetOutputRegisterA", func(t *testing.T) {
		if got := via.GetOutputRegisterA(); got != 0xAA {
			t.Errorf("GetOutputRegisterA() = %#02x; want %#02x", got, 0xAA)
		}
	})

	t.Run("GetOutputRegisterB", func(t *testing.T) {
		if got := via.GetOutputRegisterB(); got != 0xBB {
			t.Errorf("GetOutputRegisterB() = %#02x; want %#02x", got, 0xBB)
		}
	})

	t.Run("GetInputRegisterA", func(t *testing.T) {
		if got := via.GetInputRegisterA(); got != 0xCC {
			t.Errorf("GetInputRegisterA() = %#02x; want %#02x", got, 0xCC)
		}
	})

	t.Run("GetInputRegisterB", func(t *testing.T) {
		if got := via.GetInputRegisterB(); got != 0xDD {
			t.Errorf("GetInputRegisterB() = %#02x; want %#02x", got, 0xDD)
		}
	})

	t.Run("GetDataDirectionRegisterA", func(t *testing.T) {
		if got := via.GetDataDirectionRegisterA(); got != 0xEE {
			t.Errorf("GetDataDirectionRegisterA() = %#02x; want %#02x", got, 0xEE)
		}
	})

	t.Run("GetDataDirectionRegisterB", func(t *testing.T) {
		if got := via.GetDataDirectionRegisterB(); got != 0xFF {
			t.Errorf("GetDataDirectionRegisterB() = %#02x; want %#02x", got, 0xFF)
		}
	})

	t.Run("GetLowLatches", func(t *testing.T) {
		if got := via.GetLowLatches2(); got != 0x11 {
			t.Errorf("GetLowLatches2() = %#02x; want %#02x", got, 0x11)
		}
		if got := via.GetLowLatches1(); got != 0x22 {
			t.Errorf("GetLowLatches1() = %#02x; want %#02x", got, 0x22)
		}
	})

	t.Run("GetHighLatches", func(t *testing.T) {
		if got := via.GetHighLatches2(); got != 0x33 {
			t.Errorf("GetHighLatches2() = %#02x; want %#02x", got, 0x33)
		}
		if got := via.GetHighLatches1(); got != 0x44 {
			t.Errorf("GetHighLatches1() = %#02x; want %#02x", got, 0x44)
		}
	})

	t.Run("GetCounters", func(t *testing.T) {
		if got := via.GetCounter2(); got != 0x5555 {
			t.Errorf("GetCounter2() = %#04x; want %#04x", got, 0x5555)
		}
		if got := via.GetCounter1(); got != 0x6666 {
			t.Errorf("GetCounter1() = %#04x; want %#04x", got, 0x6666)
		}
	})

	t.Run("GetShiftRegister", func(t *testing.T) {
		if got := via.GetShiftRegister(); got != 0x77 {
			t.Errorf("GetShiftRegister() = %#02x; want %#02x", got, 0x77)
		}
	})

	t.Run("GetControls", func(t *testing.T) {
		if got := via.GetAuxiliaryControl(); got != 0x88 {
			t.Errorf("GetAuxiliaryControl() = %#02x; want %#02x", got, 0x88)
		}
		if got := via.GetPeripheralControl(); got != 0x99 {
			t.Errorf("GetPeripheralControl() = %#02x; want %#02x", got, 0x99)
		}
	})

	t.Run("GetInterrupts", func(t *testing.T) {
		if got := via.GetInterruptFlagValue(); got != 0x55 {
			t.Errorf("GetInterruptFlagValue() = %#02x; want %#02x", got, 0x55)
		}
		if got := via.GetInterruptEnabledFlag(); got != 0x2A {
			t.Errorf("GetInterruptEnabledFlag() = %#02x; want %#02x", got, 0x2A)
		}
	})
}

func TestViaRegisterSelect(t *testing.T) {
	tests := []struct {
		name      string
		lineNum   uint8
		wantPanic bool
	}{
		{
			name:      "Valid RS0 line",
			lineNum:   0,
			wantPanic: false,
		},
		{
			name:      "Valid RS1 line",
			lineNum:   1,
			wantPanic: false,
		},
		{
			name:      "Valid RS2 line",
			lineNum:   2,
			wantPanic: false,
		},
		{
			name:      "Valid RS3 line",
			lineNum:   3,
			wantPanic: false,
		},
		{
			name:      "Invalid line number",
			lineNum:   4,
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			via := NewVia65C22()

			if tt.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("RegisterSelect(%d) should have panicked", tt.lineNum)
					}
				}()
			}

			result := via.RegisterSelect(tt.lineNum)

			if !tt.wantPanic {
				if result == nil {
					t.Errorf("RegisterSelect(%d) returned nil, expected valid connector", tt.lineNum)
				}

				// Verify we got the correct connector from the array
				if result != via.registerSelect[tt.lineNum] {
					t.Errorf("RegisterSelect(%d) returned wrong connector", tt.lineNum)
				}
			}
		})
	}
}

func TestReadingInvalidControlLinesReturnsNil(t *testing.T) {
	via := NewVia65C22()
	circuit := newTestCircuit()

	circuit.wire(via)

	// Attempt to read from an invalid line number
	result := via.PeripheralAControlLines(4)

	if result != nil {
		t.Errorf("RegisterSelect(4) returned a connector, expected nil")
	}
}
