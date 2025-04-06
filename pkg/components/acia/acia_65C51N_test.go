package acia

import (
	"math"
	"testing"
	"time"

	"github.com/fran150/clementina-6502/internal/testutils"
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/stretchr/testify/assert"
	"go.bug.st/serial"
)

// Represents the circuit board, it is used to wire the
// ACIA chip to all the required lines.
type testCircuit struct {
	dataBus buses.Bus[uint8]
	irq     buses.Line
	rw      buses.Line
	cs0     buses.Line
	cs1     buses.Line
	rs      [numOfRSLines]buses.Line
	reset   buses.Line
}

// Creates and returns the test circuit including the ACIA chip
// the circuit and a mock serial port implementation to interface with.
func newTestCircuit() (*Acia65C51N, *testCircuit, *testutils.SerialPortMock) {
	var rsLines [numOfRSLines]buses.Line

	for i := range numOfRSLines {
		rsLines[i] = buses.NewStandaloneLine(false)
	}

	acia := NewAcia65C51N(true)

	circuit := testCircuit{
		dataBus: buses.New8BitStandaloneBus(),
		irq:     buses.NewStandaloneLine(true),
		rw:      buses.NewStandaloneLine(true),
		cs0:     buses.NewStandaloneLine(true),
		cs1:     buses.NewStandaloneLine(false),
		rs:      rsLines,
		reset:   buses.NewStandaloneLine(true),
	}

	mock := testutils.NewPortMock(&serial.Mode{})

	// Start a go soubrouting used to write or read bytes
	// to and from the serial port.
	go mock.Tick()

	return acia, &circuit, mock
}

// Wire all components together in the circuit board. It allows to change
// the input value and assert the output lines.
func (circuit *testCircuit) wire(acia *Acia65C51N, mock *testutils.SerialPortMock) error {
	if err := acia.ConnectToPort(mock); err != nil {
		return err
	}

	acia.DataBus().Connect(circuit.dataBus)

	acia.IrqRequest().Connect(circuit.irq)

	acia.ReadWrite().Connect(circuit.rw)

	acia.ChipSelect0().Connect(circuit.cs0)
	acia.ChipSelect1().Connect(circuit.cs1)

	acia.ConnectRegisterSelectLines(circuit.rs)

	acia.Reset().Connect(circuit.reset)

	return nil
}

// Calls the executor function at the specified baud rate, once per byte (not bit)
// The executor function receives a parameter with the execution number and must return
// if the cycle must continue.
// The first execution is immediate, the others follows specified baud rate.
// For example a baud rate of 8 will make executor to be called once every second
// Some tests require to wait some extra cycles to ensure the buffers are emptied,
// the extra cycle parameter will make the function to run without calling executor function
// for the specified extra cycles.
func processAtBaudRates(baudRate int, extraCycles int, executor func(int) bool) {
	bytesPerSecond := float64(baudRate) / 8.0
	interval := 1.0 / bytesPerSecond
	duration := time.Duration(interval * float64(time.Second))

	execute := true

	i := 0
	var t time.Time
	for execute || extraCycles > 0 {
		passed := time.Since(t).Seconds()
		if t.IsZero() || passed >= interval {
			if t.IsZero() {
				t = time.Now()
			} else {
				t = t.Add(duration)
			}

			if execute && !executor(i) {
				execute = false
			}

			if !execute {
				extraCycles--
			}

			i++
		}
	}
}

/****************************************************************************************************************
* Test utilities
****************************************************************************************************************/

// Sets the register value. The register value is composed of 2 lines (RS0, RS1), we use the first 2 bits of
// the specified value to set this 2 lines.
// The two register select lines are normally connected to the processor address lines to allow the processor
// to select the various ACIA internal registers
// 0x00 - Tx / Rx register
// 0x01 - Status register
// 0x02 - Command register
// 0x03 - Control Register
func (circuit *testCircuit) setRegisterSelectValue(value uint8) {
	// Mask to use only the first 2 bits
	const rsMask = 0x03
	value &= rsMask

	// Set the value of the lines based on the specified value
	for i := range numOfRSLines {
		if uint8(value)&uint8(math.Pow(2, float64(i))) > 0 {
			circuit.rs[i].Set(true)
		} else {
			circuit.rs[i].Set(false)
		}
	}
}

// Writes the specified value to the selected register in the ACIA chip.
func writeToAcia(acia *Acia65C51N, circuit *testCircuit, register uint8, value uint8, step *common.StepContext) {
	circuit.setRegisterSelectValue(register)
	circuit.rw.Set(false)
	circuit.dataBus.Write(value)

	step.NextCycle()

	acia.Tick(step)
}

// Reads and returns the value from the specified register in the ACIA chip.
func readFromAcia(acia *Acia65C51N, circuit *testCircuit, register uint8, step *common.StepContext) uint8 {
	circuit.setRegisterSelectValue(register)
	circuit.rw.Set(true)

	step.NextCycle()

	acia.Tick(step)

	return circuit.dataBus.Read()
}

// Disables chip and steps time
func disableChipAndStepTime(acia *Acia65C51N, circuit *testCircuit, step *common.StepContext) {
	circuit.cs0.Set(false)
	circuit.cs1.Set(true)

	step.NextCycle()

	acia.Tick(step)
}

// Re-enables chip
func enableChip(circuit *testCircuit) {
	circuit.cs0.Set(true)
	circuit.cs1.Set(false)
}

// Tests that the modem lines updates the status registers accordingly and the
// IRQ behaviour of these lines. This function is used to perform the same batch of tests on the DCD and DSR lines
func testModemStatusLine(t *testing.T, acia *Acia65C51N, circuit *testCircuit, modemLine *bool, flag uint8, step *common.StepContext) {
	// Modem enables line, this should trigger an interrupt
	*modemLine = true
	disableChipAndStepTime(acia, circuit, step)
	assert.Equal(t, false, circuit.irq.Status())

	// Reading from the status register should get the updated values, and clear the interrupt
	enableChip(circuit)
	status := readFromAcia(acia, circuit, 0x01, step)
	assert.Equal(t, uint8(statusIRQ|flag), (status & (statusIRQ | statusDCD | statusDSR)))
	assert.Equal(t, true, circuit.irq.Status())

	// Modem disables line, this should trigger an interrupt
	*modemLine = false
	disableChipAndStepTime(acia, circuit, step)
	assert.Equal(t, false, circuit.irq.Status())

	// Reading from the status register should get the updated values, and clear the interrupt
	enableChip(circuit)
	status = readFromAcia(acia, circuit, 0x01, step)
	assert.Equal(t, uint8(statusIRQ), (status & (statusIRQ | statusDCD | statusDSR)))
	assert.Equal(t, true, circuit.irq.Status())

	// From manual: Subsequent level changes will not affect the status bits until the Status
	// Register is interrogated by the processor.

	// Modem re-enables line again, this should trigger an interrupt
	*modemLine = true
	disableChipAndStepTime(acia, circuit, step)
	assert.Equal(t, false, circuit.irq.Status())

	// Modem disables line, interrupt was not yet handled, this should not change the status register
	*modemLine = false
	enableChip(circuit)
	status = readFromAcia(acia, circuit, 0x01, step)
	// Status register reads DCD flag high and interrupt flag, IRQ line is cleared after reading
	assert.Equal(t, uint8(statusIRQ|flag), (status & (statusIRQ | statusDCD | statusDSR)))
	assert.Equal(t, true, circuit.irq.Status())

	// From manual: At that time, another interrupt will immediately occur and the
	// status bits reflect the new input levels.
	disableChipAndStepTime(acia, circuit, step)
	assert.Equal(t, false, circuit.irq.Status())

	// Read new levels clearing IRQ
	enableChip(circuit)
	status = readFromAcia(acia, circuit, 0x01, step)
	assert.Equal(t, uint8(statusIRQ), (status & (statusIRQ | statusDCD | statusDSR)))
	assert.Equal(t, true, circuit.irq.Status())
}

/****************************************************************************************************************
* Write to registers
****************************************************************************************************************/

// Writes data to the TX register and checks if it's sent to the serial port
func TestWriteToTX(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	const data string = "Hello World!!!"

	// Writes to acia at 1000 bauds, default speed for acia is 115200, so this will be well within
	// this speed to avoid overruns. Run 2 extra cycles to allow last written byte to be transmitted
	processAtBaudRates(1000, 2, func(i int) bool {
		writeToAcia(acia, circuit, 0x00, uint8(data[i]), &step)
		return i < len(data)-1
	})

	assert.Equal(t, data, string(mock.TerminalReceive()))
}

// Writes data to the TX register with CTS disabled. This means that the other side (modem or computer)
// is not ready. According to documentation: "The CTSB input pin controls the transmitter operation. The
// enable state is with CTSB low. The transmitter is automatically disabled if CTSB is high."
func TestWriteToTXWithCTSDisabled(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	const data string = "Hello World!!!"

	// Disables CTS from the serial port, this means that the other side is not ready to receive and the
	// transmitter is automatically disabled.
	mock.Status.CTS = false

	// Writes to acia at 1000 bauds, default speed for acia is 115200, so this will be well within
	// this speed to avoid overruns. Run 2 extra cycles to allow last written byte to be transmitted
	processAtBaudRates(1000, 2, func(i int) bool {
		writeToAcia(acia, circuit, 0x00, uint8(data[i]), &step)
		return i < len(data)-1
	})

	// No data will be sent with CTS disabled (high)
	assert.Equal(t, "", string(mock.TerminalReceive()))
}

// Test that the records are changed back to the expected value during a programmed reset.
// A programmed reset is caused by writing any value to status register RS = 0x01 (RS1 = L, RS0 = H)
func TestProgrammedReset(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	// Set all 1s (where possible) in the control and command registers
	writeToAcia(acia, circuit, 0x02, 0xDF, &step)
	writeToAcia(acia, circuit, 0x03, 0xFF, &step)

	// Assert the new status (status register cannot be written and has default value)
	assert.Equal(t, uint8(0x10), acia.statusRegister)
	assert.Equal(t, uint8(0xFF), acia.controlRegister)
	assert.Equal(t, uint8(0xDF), acia.commandRegister)

	// Write to status register causes a programmed reset
	writeToAcia(acia, circuit, 0x01, 0xFF, &step)

	// Check values are reset (control register remains untouched by programmed reset)
	assert.Equal(t, uint8(0x10), acia.statusRegister)
	assert.Equal(t, uint8(0xFF), acia.controlRegister)
	assert.Equal(t, uint8(0xC0), acia.commandRegister)
}

// Test writing a value to the command register RS = 0x02 (RS1 = H, RS0 = L)
func TestWriteToCommandRegister(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	writeToAcia(acia, circuit, 0x02, 0xDF, &step)

	assert.Equal(t, uint8(0xDF), acia.commandRegister)
}

// Test writing a value to the control register RS = 0x03 (RS1 = H, RS0 = H)
func TestWriteToControlRegister(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	writeToAcia(acia, circuit, 0x03, 0xFF, &step)

	assert.Equal(t, uint8(0xFF), acia.controlRegister)
}

// Selected stop bits depends on configuraiton of the stop bits and word length
func TestWriteToControlConfiguresCorrectStopBit(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	type testConfig struct {
		value            uint8
		expectedDataBits int
		expectedStopBits serial.StopBits
	}

	tests := []testConfig{
		{0x00, 8, serial.OneStopBit},
		{0x20, 7, serial.OneStopBit},
		{0x40, 6, serial.OneStopBit},
		{0x60, 5, serial.OneStopBit},

		{0x80, 8, serial.TwoStopBits},
		{0xA0, 7, serial.TwoStopBits},
		{0xC0, 6, serial.TwoStopBits},
		{0xE0, 5, serial.OnePointFiveStopBits},
	}

	for _, test := range tests {
		writeToAcia(acia, circuit, 0x03, test.value, &step)
		assert.Equal(t, test.expectedDataBits, mock.Mode.DataBits)
		assert.Equal(t, test.expectedStopBits, mock.Mode.StopBits)
	}
}

// The 65C51N model of the acia chip does not support bit parity nor
// TX interrupt handling. Enabling TRDE will be ignore or cause constant
// IRQs as the flag in the status register is always 1.
func TestPanicForInvalidModesFor65C51N(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	// Enabling TRDE will result in panic
	assert.Panics(t, func() {
		writeToAcia(acia, circuit, 0x02, 0x04, &step)
	})

	// Enabling bit parity will result in panic
	assert.Panics(t, func() {
		writeToAcia(acia, circuit, 0x02, 0x20, &step)
	})
}

/****************************************************************************************************************
* Read from registers
****************************************************************************************************************/

// Simulates a terminal sending a string through serial port and receiving the values through
// the ACIA chip.
func TestReadFromRxPollingStatusRegister(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	const data string = "Hello World!!!"
	var read []uint8

	// Used to indicate if the fake terminal has started sending the bytes
	startedSending := false

	// Enable DTR and interrupts
	writeToAcia(acia, circuit, 0x02, 0x01, &step)
	assert.Equal(t, true, mock.DTR)
	assert.Equal(t, true, circuit.irq.Status())

	// Set the bauds to 4800
	writeToAcia(acia, circuit, 0x03, 0x0C, &step)

	for {
		// Reads the status record
		status := readFromAcia(acia, circuit, 0x01, &step)

		// Checks if an overrun happened, this can only happen if a new value arrives before
		// we read the previous one.
		if status&statusOverrun == statusOverrun {
			t.Fatalf("Overrun occurred")
		}

		// Polls the Receiver Data Register Full flag, this means that a byte was received
		// from the serial port.
		if status&statusRDRF == statusRDRF {
			read = append(read, readFromAcia(acia, circuit, 0x00, &step))
		}

		// Stop when we read the entire message.
		if len(read) == len(data) {
			break
		}

		// Once we started polling start sending in the background.
		// The terminal will start putting bytes in the serial port at the configured byte rate
		// in this case 4800
		if !startedSending {
			go mock.TerminalSend([]byte(data))
			startedSending = true
		}
	}

	assert.Equal(t, data, string(read))
}

// Simulates a terminal sending a string through serial port and receiving the values through
// the ACIA chip.
func TestReadFromRxUsingIRQ(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	const data string = "Hello World!!!"
	var read []uint8

	// Used to indicate if the fake terminal has started sending the bytes
	startedSending := false

	// Enable DTR and interrupts
	writeToAcia(acia, circuit, 0x02, 0x01, &step)
	assert.Equal(t, true, mock.DTR)
	assert.Equal(t, true, circuit.irq.Status())

	// Set the bauds to 4800
	writeToAcia(acia, circuit, 0x03, 0x0C, &step)

	for {
		// If IRQ is triggered
		if !circuit.irq.Status() {
			enableChip(circuit)

			// Reads the status record
			status := readFromAcia(acia, circuit, 0x01, &step)
			read = append(read, readFromAcia(acia, circuit, 0x00, &step))

			// Checks if an overrun happened, this can only happen if a new value arrives before
			// we read the previous one.
			if status&statusOverrun == statusOverrun {
				t.Fatalf("Overrun occurred")
			}
		} else {
			// Wait for IRQ to happen
			disableChipAndStepTime(acia, circuit, &step)
		}

		// Stop when we read the entire message.
		if len(read) == len(data) {
			break
		}

		// Once we started polling start sending in the background.
		// The terminal will start putting bytes in the serial port at the configured byte rate
		// in this case 4800
		if !startedSending {
			go mock.TerminalSend([]byte(data))
			startedSending = true
		}
	}

	assert.Equal(t, data, string(read))
}

// Simulates a terminal sending a string through serial port and receiving the values through
// the ACIA chip.

func TestReadFromRxUsingIRQAndReceiverEchoModeEnabled(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	const data string = "Hello World!!!"
	var read []uint8

	// Used to indicate if the fake terminal has started sending the bytes
	startedSending := false

	// Enable DTR (0x01), interrupts and Receiver Echo Mode (0x10)
	writeToAcia(acia, circuit, 0x02, 0x11, &step)
	assert.Equal(t, true, mock.DTR)
	assert.Equal(t, true, circuit.irq.Status())

	// Set the bauds to 4800
	writeToAcia(acia, circuit, 0x03, 0x0C, &step)

	for {
		// If IRQ is triggered
		if !circuit.irq.Status() {
			enableChip(circuit)

			// Reads the status record
			status := readFromAcia(acia, circuit, 0x01, &step)
			read = append(read, readFromAcia(acia, circuit, 0x00, &step))

			// Checks if an overrun happened, this can only happen if a new value arrives before
			// we read the previous one.
			if status&statusOverrun == statusOverrun {
				t.Fatalf("Overrun occurred")
			}
		} else {
			// Wait for IRQ to happen
			disableChipAndStepTime(acia, circuit, &step)
		}

		// Stop when we write the entire message.
		if mock.TerminalRxBuffer.Size() == len(data) {
			break
		}

		// Once we started polling start sending in the background.
		// The terminal will start putting bytes in the serial port at the configured byte rate
		// in this case 4800
		if !startedSending {
			go mock.TerminalSend([]byte(data))
			startedSending = true
		}
	}

	assert.Equal(t, data, string(read))
	assert.Equal(t, data, string(mock.TerminalReceive()))
}

// Simulates a terminal sending a string through serial port too fast and causing
// a buffer overrun in the ACIA chip
func TestReadFromRxOverrunning(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	const data string = "Hello World!!!"
	var read []uint8

	// Start sending bytes at the configured rate (115200 bauds)
	mock.TerminalSend([]byte(data))

	// Read bytes a 10 bauds
	processAtBaudRates(100, 2, func(i int) bool {
		// First execution happens immediately so it will not overrun.
		if i > 0 {
			// Validates that the overrun flag is high
			assert.Equal(t, statusOverrun, acia.statusRegister&statusOverrun)
		}

		// Read from ACIA
		read = append(read, readFromAcia(acia, circuit, 0x00, &step))

		// Stop when all bytes are sent
		return !mock.TerminalTxBuffer.IsEmpty()
	})
}

// Test reading from status register RS = 0x01 (RS1 = L, RS1 = H)
func TestReadFromStatusRegister(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	// Internally force the register to 0xFF and read
	acia.statusRegister = 0xFF
	status := readFromAcia(acia, circuit, 0x01, &step)

	assert.Equal(t, uint8(0xFF), status)
}

// Test reading from command register RS = 0x02 (RS1 = H, RS1 = L)
func TestReadFromCommandRegister(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	// Internally force the register to 0xDF and read
	acia.commandRegister = 0xDF
	command := readFromAcia(acia, circuit, 0x02, &step)

	assert.Equal(t, uint8(0xDF), command)
}

// Test reading from control register RS = 0x03 (RS1 = H, RS1 = H)
func TestReadFromControlRegister(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	// Internally force the register to 0xFF and read
	acia.controlRegister = 0xFF
	control := readFromAcia(acia, circuit, 0x03, &step)

	assert.Equal(t, uint8(0xFF), control)
}

/****************************************************************************************************************
* Hardware Reset
****************************************************************************************************************/

// Test hardware reset moves flags to expected values
func TestHardwareReset(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	writeToAcia(acia, circuit, 0x02, 0xDF, &step)
	writeToAcia(acia, circuit, 0x03, 0xFF, &step)

	// Assert the new status (status register cannot be written and has default value)
	assert.Equal(t, uint8(0x10), acia.statusRegister)
	assert.Equal(t, uint8(0xFF), acia.controlRegister)
	assert.Equal(t, uint8(0xDF), acia.commandRegister)

	// Lower reset line causing a reset
	circuit.reset.Set(false)

	// Write to status register causes a programmed reset
	disableChipAndStepTime(acia, circuit, &step)

	// Check values are reset (control register remains untouched by programmed reset)
	assert.Equal(t, uint8(0x10), acia.statusRegister)
	assert.Equal(t, uint8(0x00), acia.controlRegister)
	assert.Equal(t, uint8(0x00), acia.commandRegister)
}

/****************************************************************************************************************
* Interrupt and flags
****************************************************************************************************************/

func TestInterruptFromModemLines(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	// Check that DSR and DCD status registers are 0 after initialization
	status := readFromAcia(acia, circuit, 0x01, &step)
	assert.Equal(t, uint8(0x00), (status & (statusDCD | statusDSR)))
	assert.Equal(t, true, circuit.irq.Status())

	// Enable DTR and interrupts
	writeToAcia(acia, circuit, 0x02, 0x01, &step)
	assert.Equal(t, true, mock.DTR)
	assert.Equal(t, true, circuit.irq.Status())

	// From manual: Whenever either of these inputs change state [DCD, DSR], an
	// immediate processor interrupt (IRQ) occurs, unless bit 1 of the Command Register (IRD) is set to a 1 to
	// disable IRQB. When the interrupt occurs, the status bits indicate the levels of the inputs immediately after
	// the change of state occurred. Subsequent level changes will not affect the status bits until the Status
	// Register is interrogated by the processor. At that time, another interrupt will immediately occur and the
	// status bits reflect the new input levels.

	testModemStatusLine(t, acia, circuit, &mock.Status.DCD, statusDCD, &step)
	testModemStatusLine(t, acia, circuit, &mock.Status.DSR, statusDSR, &step)
}

/****************************************************************************************************************
* External lines control
****************************************************************************************************************/
func TestCPUControlledLinesToModem(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, false, mock.RTS)
	assert.Equal(t, false, mock.DTR)

	// Enable RTS (0x08) and DTR (0x01)
	writeToAcia(acia, circuit, 0x02, 0x09, &step)

	assert.Equal(t, true, mock.RTS)
	assert.Equal(t, true, mock.DTR)
}

func TestCTSStatusWhenNotConnected(t *testing.T) {
	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	// CTS is considered ready when there is an error reading the status. This if to support tesing using SOCAT
	// or tool that don't handle the lines.
	mock.MakeCallsFailFrom = testutils.FailInGetModemStatusBits
	assert.Equal(t, true, acia.isCTSEnabled())

	// CTS is considered not ready when serial port is not connected, this will disable
	// transmitter
	acia.port = nil
	assert.Equal(t, false, acia.isCTSEnabled())
}

/****************************************************************************************************************
* Panics when using serial
****************************************************************************************************************/

func TestPanicsWhenFailsToSetModeWhenConnectingToSerial(t *testing.T) {
	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	mock.MakeCallsFailFrom = testutils.FailInSetMode

	err := circuit.wire(acia, mock)

	assert.Error(t, err)
}

func TestPanicsWhenFailsToSetDTRandRTS(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	mock.MakeCallsFailFrom = testutils.FailInSetDTR

	assert.Panics(t, func() {
		writeToAcia(acia, circuit, 0x03, 0x01, &step)
	})

	mock.MakeCallsFailFrom = testutils.FailInSetRTS

	assert.Panics(t, func() {
		writeToAcia(acia, circuit, 0x03, 0x08, &step)
	})
}

func TestReturnsFalseWhenFailsToGetModemStatusLines(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	// First set mock values to true
	mock.Status.DCD = true
	mock.Status.DSR = true

	status := readFromAcia(acia, circuit, 0x01, &step)
	assert.Equal(t, uint8(statusDSR|statusDCD), (status & (statusDSR | statusDCD)))

	// Make calls fails
	mock.MakeCallsFailFrom = testutils.FailInGetModemStatusBits

	status = readFromAcia(acia, circuit, 0x01, &step)
	assert.Equal(t, uint8(0x00), (status & (statusDSR | statusDCD)))
}

func TestPanicsWhenPollerFailsToRead(t *testing.T) {
	acia, circuit, mock := newTestCircuit()
	mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	// Calls close on the acia to make go functions
	// stop
	acia.Close()

	acia.running = true
	acia.rxRegisterEmpty = true
	mock.MakeCallsFailFrom = testutils.FailInRead

	// Manually call the poller to assert the panic
	assert.Panics(t, acia.readBytes)
}

func TestPanicsWhenPollerFailsToWrite(t *testing.T) {
	acia, circuit, mock := newTestCircuit()
	mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	// Calls close on the acia to make go functions
	// stop
	acia.Close()

	acia.running = true
	acia.txRegisterEmpty = false
	mock.MakeCallsFailFrom = testutils.FailInWrite

	// Manually call the poller to assert the panic
	assert.Panics(t, acia.writeBytes)
}

func TestPanicsWhenFailsToSetModeWhenChangingControlRegister(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	mock.MakeCallsFailFrom = testutils.FailInSetMode

	assert.Panics(t, func() {
		writeToAcia(acia, circuit, 0x03, 0xFF, &step)
	})
}

func TestPanicsWhenFailsToSetReadTimeout(t *testing.T) {
	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	mock.MakeCallsFailFrom = testutils.FailInSetReadTimeout

	err := circuit.wire(acia, mock)
	assert.Error(t, err)
}

func TestConnectToPortModemLinesError(t *testing.T) {
	acia, circuit, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	// First make the initial connection succeed
	if err := circuit.wire(acia, mock); err != nil {
		t.Fatal(err)
	}

	// Create a new ACIA with modem lines enabled
	newAcia := NewAcia65C51N(true)

	// Make SetDTR fail
	mock.MakeCallsFailFrom = testutils.FailInSetDTR

	// Attempt to connect - should fail due to modem lines error
	err := newAcia.ConnectToPort(mock)
	assert.Error(t, err, "Expected error when setting modem lines fails")

	// Make SetRTS fail
	mock.MakeCallsFailFrom = testutils.FailInSetRTS

	// Attempt to connect again - should fail due to modem lines error
	err = newAcia.ConnectToPort(mock)
	assert.Error(t, err, "Expected error when setting modem lines fails")
}

func TestConnectToPortSkipsModemLinesWhenDisabled(t *testing.T) {
	acia, _, mock := newTestCircuit()
	defer acia.Close()
	defer mock.Close()

	// Create a new ACIA with modem lines disabled
	newAcia := NewAcia65C51N(false)

	// Make modem line operations fail
	mock.MakeCallsFailFrom = testutils.FailInSetDTR

	// Should succeed because modem lines are disabled
	err := newAcia.ConnectToPort(mock)
	assert.NoError(t, err, "Expected success when modem lines are disabled")
}

/****************************************************************************************************************
* Amazon Q generated tests for getters
****************************************************************************************************************/

func TestAcia65C51N_GetRegisters(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Acia65C51N)
		want     uint8
		testFunc func(*Acia65C51N) uint8
	}{
		{
			name: "Get Status Register",
			setup: func(a *Acia65C51N) {
				a.statusRegister = 0x55
			},
			want:     0x55,
			testFunc: (*Acia65C51N).GetStatusRegister,
		},
		{
			name: "Get Control Register",
			setup: func(a *Acia65C51N) {
				a.controlRegister = 0xAA
			},
			want:     0xAA,
			testFunc: (*Acia65C51N).GetControlRegister,
		},
		{
			name: "Get Command Register",
			setup: func(a *Acia65C51N) {
				a.commandRegister = 0x33
			},
			want:     0x33,
			testFunc: (*Acia65C51N).GetCommandRegister,
		},
		{
			name: "Get TX Register",
			setup: func(a *Acia65C51N) {
				a.txRegister = 0xFF
			},
			want:     0xFF,
			testFunc: (*Acia65C51N).GetTXRegister,
		},
		{
			name: "Get RX Register",
			setup: func(a *Acia65C51N) {
				a.rxRegister = 0x12
			},
			want:     0x12,
			testFunc: (*Acia65C51N).GetRXRegister,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acia := &Acia65C51N{}
			tt.setup(acia)
			if got := tt.testFunc(acia); got != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestAcia65C51N_GetRegisterStates(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Acia65C51N)
		want     bool
		testFunc func(*Acia65C51N) bool
	}{
		{
			name: "Get TX Register Empty - True",
			setup: func(a *Acia65C51N) {
				a.txRegisterEmpty = true
			},
			want:     true,
			testFunc: (*Acia65C51N).GetTXRegisterEmpty,
		},
		{
			name: "Get TX Register Empty - False",
			setup: func(a *Acia65C51N) {
				a.txRegisterEmpty = false
			},
			want:     false,
			testFunc: (*Acia65C51N).GetTXRegisterEmpty,
		},
		{
			name: "Get RX Register Empty - True",
			setup: func(a *Acia65C51N) {
				a.rxRegisterEmpty = true
			},
			want:     true,
			testFunc: (*Acia65C51N).GetRXRegisterEmpty,
		},
		{
			name: "Get RX Register Empty - False",
			setup: func(a *Acia65C51N) {
				a.rxRegisterEmpty = false
			},
			want:     false,
			testFunc: (*Acia65C51N).GetRXRegisterEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acia := &Acia65C51N{}
			tt.setup(acia)
			if got := tt.testFunc(acia); got != tt.want {
				t.Errorf("%s = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestAcia65C51N_RegisterSelect(t *testing.T) {
	tests := []struct {
		name      string
		lineNum   uint8
		expectNil bool
	}{
		{
			name:      "Valid RS0 line",
			lineNum:   0,
			expectNil: false,
		},
		{
			name:      "Valid RS1 line",
			lineNum:   1,
			expectNil: false,
		},
		{
			name:      "Invalid line number",
			lineNum:   2,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acia := &Acia65C51N{
				registerSelect: [numOfRSLines]*buses.ConnectorEnabledHigh{
					buses.NewConnectorEnabledHigh(),
					buses.NewConnectorEnabledHigh(),
				},
			}

			result := acia.RegisterSelect(tt.lineNum)

			if !tt.expectNil {
				if result == nil {
					t.Errorf("RegisterSelect(%d) returned nil, expected valid connector", tt.lineNum)
				}

				// Verify we got the correct connector from the array
				if result != acia.registerSelect[tt.lineNum] {
					t.Errorf("RegisterSelect(%d) returned wrong connector", tt.lineNum)
				}
			} else {
				if result != nil {
					t.Errorf("RegisterSelect(%d) returned valid connector, expected nil", tt.lineNum)
				}
			}
		})
	}
}
