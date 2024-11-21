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

func createTestCircuit() (*Acia65C51N, *testCircuit, *portMock) {
	var rsLines [NUM_OF_RS_LINES]buses.Line

	for i := range NUM_OF_RS_LINES {
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

	acia.ConnectToPort(mock)
	circuit.wire(acia)

	go mock.Tick()

	return acia, &circuit, mock
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

func writeToAcia(acia *Acia65C51N, circuit *testCircuit, register uint8, value uint8, step *common.StepContext) {
	circuit.setRegisterSelectValue(register)
	circuit.rw.Set(false)
	circuit.dataBus.Write(value)

	step.Cycle++
	step.T = time.Now()

	acia.Tick(*step)
}

func readFromAcia(acia *Acia65C51N, circuit *testCircuit, register uint8, step *common.StepContext) uint8 {
	circuit.setRegisterSelectValue(register)
	circuit.rw.Set(true)

	step.Cycle++
	step.T = time.Now()

	acia.Tick(*step)

	return circuit.dataBus.Read()
}

func TestWriteToTX(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := createTestCircuit()
	defer acia.Close()
	defer mock.Close()

	circuit.wire(acia)

	var data string = "Hello World!!!"

	start := time.Now()

	processAtBaudRates(1000, func(i int) bool {
		if i < len(data) {
			writeToAcia(acia, circuit, 0x00, uint8(data[i]), &step)
		}
		return i <= len(data)
	})

	baud := float64(len(data)*8) / time.Since(start).Seconds()

	t.Logf("BAUD: %v", baud)

	assert.Equal(t, data, string(mock.terminalReceive()))
}

func TestProgrammedReset(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := createTestCircuit()
	defer acia.Close()
	defer mock.Close()

	circuit.wire(acia)

	writeToAcia(acia, circuit, 0x02, 0xFF, &step)
	writeToAcia(acia, circuit, 0x03, 0xFF, &step)

	assert.Equal(t, uint8(0x10), acia.statusRegister)
	assert.Equal(t, uint8(0xFF), acia.commandRegister)
	assert.Equal(t, uint8(0xFF), acia.controlRegister)

	// Write to status register causes a programmed reset
	writeToAcia(acia, circuit, 0x01, 0xFF, &step)

	assert.Equal(t, uint8(0x10), acia.statusRegister)
	assert.Equal(t, uint8(0xFF), acia.controlRegister)
	assert.Equal(t, uint8(0xE0), acia.commandRegister)
}

func TestWriteToCommandRegister(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := createTestCircuit()
	defer acia.Close()
	defer mock.Close()

	circuit.wire(acia)

	writeToAcia(acia, circuit, 0x02, 0xFF, &step)

	assert.Equal(t, uint8(0xFF), acia.commandRegister)
}

func TestWriteToControlRegister(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := createTestCircuit()
	defer acia.Close()
	defer mock.Close()

	circuit.wire(acia)

	writeToAcia(acia, circuit, 0x03, 0xFF, &step)

	assert.Equal(t, uint8(0xFF), acia.controlRegister)
}

func TestReadFromRxOverrunning(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := createTestCircuit()
	defer acia.Close()
	defer mock.Close()

	circuit.wire(acia)

	var data string = "Hello World!!!"
	var read []uint8

	mock.terminalSend([]byte(data))

	processAtBaudRates(10, func(i int) bool {
		if i > 0 {
			assert.Equal(t, statusOverrun, acia.statusRegister&statusOverrun)
		}

		read = append(read, readFromAcia(acia, circuit, 0x00, &step))

		return !mock.terminalTxBuffer.isEmpty()
	})
}

func TestReadFromStatusRegister(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := createTestCircuit()
	defer acia.Close()
	defer mock.Close()

	circuit.wire(acia)

	acia.statusRegister = 0xFF
	status := readFromAcia(acia, circuit, 0x01, &step)

	assert.Equal(t, uint8(0xFF), status)
}

func TestReadFromCommandRegister(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := createTestCircuit()
	defer acia.Close()
	defer mock.Close()

	circuit.wire(acia)

	acia.commandRegister = 0xFF
	command := readFromAcia(acia, circuit, 0x02, &step)

	assert.Equal(t, uint8(0xFF), command)
}

func TestReadFromControlRegister(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := createTestCircuit()
	defer acia.Close()
	defer mock.Close()

	circuit.wire(acia)

	acia.controlRegister = 0xFF
	control := readFromAcia(acia, circuit, 0x03, &step)

	assert.Equal(t, uint8(0xFF), control)
}

func TestReadFromRx(t *testing.T) {
	var step common.StepContext

	acia, circuit, mock := createTestCircuit()
	defer acia.Close()
	defer mock.Close()

	circuit.wire(acia)

	var data string = "Hello World!!!"
	var read []uint8

	sent := false

	for {
		status := readFromAcia(acia, circuit, 0x01, &step)

		if acia.statusRegister&statusOverrun == statusOverrun {
			t.Errorf("Overrun occurred")
			t.Fail()
		}

		if status&0x08 == 0x08 {
			read = append(read, readFromAcia(acia, circuit, 0x00, &step))
		}

		if len(read) == len(data) {
			break
		}

		if !sent {
			go mock.terminalSend([]byte(data))
			sent = true
		}
	}

	assert.Equal(t, data, string(read))
}

func processAtBaudRates(baudRate int, executor func(int) bool) {
	bytesPerSecond := float64(baudRate) / 8.0
	interval := 1.0 / bytesPerSecond
	duration := time.Duration(interval * float64(time.Second))

	i := 0
	var t time.Time
	for {
		passed := time.Since(t).Seconds()
		if t.IsZero() || passed >= interval {
			if t.IsZero() {
				t = time.Now()
			} else {
				t = t.Add(duration)
			}

			if !executor(i) {
				break
			}
			i++
		}
	}
}
