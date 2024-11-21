package acia

import (
	"math"
	"testing"
	"time"

	"github.com/fran150/clementina6502/pkg/buses"
	"github.com/fran150/clementina6502/pkg/common"
	"github.com/stretchr/testify/assert"
	"go.bug.st/serial"
)

// Represents the circuit board, it is used to wire the
// ACIA chip to all the required lines.
type testCircuit struct {
	dataBus *buses.Bus[uint8]
	irq     buses.Line
	rw      buses.Line
	cs0     buses.Line
	cs1     buses.Line
	rs      [numOfRSLines]buses.Line
	reset   buses.Line
}

// Creates and returns the test circuit including the ACIA chip
// the circuit and a mock serial port implementation to interface with.
func createTestCircuit() (*Acia65C51N, *testCircuit, *portMock) {
	var rsLines [numOfRSLines]buses.Line

	for i := range numOfRSLines {
		rsLines[i] = buses.CreateStandaloneLine(false)
	}

	acia := CreateAcia65C51N()

	circuit := testCircuit{
		dataBus: buses.CreateBus[uint8](),
		irq:     buses.CreateStandaloneLine(true),
		rw:      buses.CreateStandaloneLine(true),
		cs0:     buses.CreateStandaloneLine(true),
		cs1:     buses.CreateStandaloneLine(false),
		rs:      rsLines,
		reset:   buses.CreateStandaloneLine(true),
	}

	mock := createPortMock(&serial.Mode{})

	// Connects the acia chip to the serial port and wires
	// the ACIA to the circuit
	acia.ConnectToPort(mock)
	circuit.wire(acia)

	// Start a go soubrouting used to write or read bytes
	// to and from the serial port.
	go mock.Tick()

	return acia, &circuit, mock
}

// Wire all components together in the circuit board. It allows to change
// the input value and assert the output lines.
func (circuit *testCircuit) wire(acia *Acia65C51N) {
	acia.DataBus().Connect(circuit.dataBus)

	acia.IrqRequest().Connect(circuit.irq)

	acia.ReadWrite().Connect(circuit.rw)

	acia.ChipSelect0().Connect(circuit.cs0)
	acia.ChipSelect1().Connect(circuit.cs1)

	acia.ConnectRegisterSelectLines(circuit.rs)

	acia.Reset().Connect(circuit.reset)
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

	step.Cycle++
	step.T = time.Now()

	acia.Tick(*step)
}

// Reads and returns the value from the specified register in the ACIA chip.
func readFromAcia(acia *Acia65C51N, circuit *testCircuit, register uint8, step *common.StepContext) uint8 {
	circuit.setRegisterSelectValue(register)
	circuit.rw.Set(true)

	step.Cycle++
	step.T = time.Now()

	acia.Tick(*step)

	return circuit.dataBus.Read()
}

/****************************************************************************************************************
* Write to registers
****************************************************************************************************************/

// Writes data to the TX register and checks if it's sent to the serial port
func TestWriteToTX(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := createTestCircuit()
	defer acia.Close()
	defer mock.Close()

	circuit.wire(acia)

	const data string = "Hello World!!!"

	// Writes to acia at 1000 bauds, default speed for acia is 115200, so this will be well within
	// this speed to avoid overruns. Run 2 extra cycles to allow last written byte to be transmitted
	processAtBaudRates(1000, 2, func(i int) bool {
		writeToAcia(acia, circuit, 0x00, uint8(data[i]), &step)
		return i < len(data)-1
	})

	assert.Equal(t, data, string(mock.terminalReceive()))
}

// Test that the records are changed back to the expected value during a programmed reset.
// A programmed reset is caused by writing any value to status register RS = 0x01 (RS1 = L, RS0 = H)
func TestProgrammedReset(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := createTestCircuit()
	defer acia.Close()
	defer mock.Close()

	circuit.wire(acia)

	// Write 0xFF in both registers
	writeToAcia(acia, circuit, 0x02, 0xFF, &step)
	writeToAcia(acia, circuit, 0x03, 0xFF, &step)

	// Assert the new status (status register cannot be written and has default value)
	assert.Equal(t, uint8(0x10), acia.statusRegister)
	assert.Equal(t, uint8(0xFF), acia.commandRegister)
	assert.Equal(t, uint8(0xFF), acia.controlRegister)

	// Write to status register causes a programmed reset
	writeToAcia(acia, circuit, 0x01, 0xFF, &step)

	// Check values are reset (control register remains untouched by programmed reset)
	assert.Equal(t, uint8(0x10), acia.statusRegister)
	assert.Equal(t, uint8(0xFF), acia.controlRegister)
	assert.Equal(t, uint8(0xE0), acia.commandRegister)
}

// Test writing a value to the command register RS = 0x02 (RS1 = H, RS0 = L)
func TestWriteToCommandRegister(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := createTestCircuit()
	defer acia.Close()
	defer mock.Close()

	circuit.wire(acia)

	writeToAcia(acia, circuit, 0x02, 0xFF, &step)

	assert.Equal(t, uint8(0xFF), acia.commandRegister)
}

// Test writing a value to the control register RS = 0x03 (RS1 = H, RS0 = H)
func TestWriteToControlRegister(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := createTestCircuit()
	defer acia.Close()
	defer mock.Close()

	circuit.wire(acia)

	writeToAcia(acia, circuit, 0x03, 0xFF, &step)

	assert.Equal(t, uint8(0xFF), acia.controlRegister)
}

/****************************************************************************************************************
* Read from registers
****************************************************************************************************************/

// Simulates a terminal sending a string through serial port and receiving the values through
// the ACIA chip.
func TestReadFromRx(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := createTestCircuit()
	defer acia.Close()
	defer mock.Close()

	circuit.wire(acia)

	const data string = "Hello World!!!"
	var read []uint8

	// Used to indicate if the fake terminal has started sending the bytes
	startedSending := false

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
		// in this case 115200
		if !startedSending {
			go mock.terminalSend([]byte(data))
			startedSending = true
		}
	}

	assert.Equal(t, data, string(read))
}

// Simulates a terminal sending a string through serial port too fast and causing
// a buffer overrun in the ACIA chip
func TestReadFromRxOverrunning(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := createTestCircuit()
	defer acia.Close()
	defer mock.Close()

	circuit.wire(acia)

	const data string = "Hello World!!!"
	var read []uint8

	// Start sending bytes at the configured rate (115200 bauds)
	mock.terminalSend([]byte(data))

	// Read bytes a 10 bauds
	processAtBaudRates(10, 2, func(i int) bool {
		// First execution happens immediately so it will not overrun.
		if i > 0 {
			// Validates that the overrun flag is high
			assert.Equal(t, statusOverrun, acia.statusRegister&statusOverrun)
		}

		// Read from ACIA
		read = append(read, readFromAcia(acia, circuit, 0x00, &step))

		// Stop when all bytes are sent
		return !mock.terminalTxBuffer.IsEmpty()
	})
}

// Test reading from status register RS = 0x01 (RS1 = L, RS1 = H)
func TestReadFromStatusRegister(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := createTestCircuit()
	defer acia.Close()
	defer mock.Close()

	circuit.wire(acia)

	// Internally force the register to 0xFF and read
	acia.statusRegister = 0xFF
	status := readFromAcia(acia, circuit, 0x01, &step)

	assert.Equal(t, uint8(0xFF), status)
}

// Test reading from command register RS = 0x02 (RS1 = H, RS1 = L)
func TestReadFromCommandRegister(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := createTestCircuit()
	defer acia.Close()
	defer mock.Close()

	circuit.wire(acia)

	// Internally force the register to 0xFF and read
	acia.commandRegister = 0xFF
	command := readFromAcia(acia, circuit, 0x02, &step)

	assert.Equal(t, uint8(0xFF), command)
}

// Test reading from control register RS = 0x03 (RS1 = H, RS1 = H)
func TestReadFromControlRegister(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := createTestCircuit()
	defer acia.Close()
	defer mock.Close()

	circuit.wire(acia)

	// Internally force the register to 0xFF and read
	acia.controlRegister = 0xFF
	control := readFromAcia(acia, circuit, 0x03, &step)

	assert.Equal(t, uint8(0xFF), control)
}
