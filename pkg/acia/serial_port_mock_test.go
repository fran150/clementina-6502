package acia

import (
	"time"

	"github.com/fran150/clementina6502/internal/queue"
	"go.bug.st/serial"
)

type failInFunction int

const (
	failInNone failInFunction = iota
	failInSetMode
	failInRead
	failInWrite
	failInSetDTR
	failInSetRTS
	failInGetModemStatusBits
	failInOther
)

// Struct that mocks the serial port interface for testing
type portMock struct {
	mode   *serial.Mode
	status serial.ModemStatusBits
	dtr    bool
	rts    bool

	portTxBuffer *queue.SimpleQueue
	portRxBuffer *queue.SimpleQueue

	terminalTxBuffer *queue.SimpleQueue
	terminalRxBuffer *queue.SimpleQueue

	previousTick time.Time

	stop bool

	makeCallsFailFrom failInFunction
}

// Creates a new mock of the serial port
func createPortMock(mode *serial.Mode) *portMock {
	return &portMock{
		mode: mode,
		status: serial.ModemStatusBits{
			CTS: true,
			DSR: false,
			RI:  false,
			DCD: false,
		},
		dtr:              false,
		rts:              false,
		portTxBuffer:     queue.CreateQueue(),
		terminalTxBuffer: queue.CreateQueue(),
		portRxBuffer:     queue.CreateQueue(),
		terminalRxBuffer: queue.CreateQueue(),

		makeCallsFailFrom: failInNone,
	}
}

// Returns error if the mock is configured to fail
func (port *portMock) checkError(calledFrom failInFunction) error {
	if port.makeCallsFailFrom == calledFrom {
		return serial.PortError{}
	} else {
		return nil
	}
}

// SetMode sets all parameters of the serial port
func (port *portMock) SetMode(mode *serial.Mode) error {
	port.mode = mode

	return port.checkError(failInSetMode)
}

// Stores data received from the serial port into the provided byte array
// buffer. The function returns the number of bytes read.
//
// The Read function blocks until (at least) one byte is received from
// the serial port or an error occurs.
func (port *portMock) Read(p []byte) (n int, err error) {
	for port.portRxBuffer.IsEmpty() && !port.stop {
	}

	i := 0
	for !port.portRxBuffer.IsEmpty() && i < len(p) {
		p[i] = port.portRxBuffer.DeQueue()
		i++
	}

	return len(p), port.checkError(failInRead)
}

// Send the content of the data byte array to the serial port.
// Returns the number of bytes written.
func (port *portMock) Write(p []byte) (n int, err error) {
	for _, v := range p {
		port.portTxBuffer.Queue(v)
	}

	return port.portTxBuffer.Size(), port.checkError(failInWrite)
}

// Wait until all data in the buffer are sent
func (port *portMock) Drain() error {
	return port.checkError(failInOther)
}

// ResetInputBuffer Purges port read buffer
func (port *portMock) ResetInputBuffer() error {
	return port.checkError(failInOther)
}

// ResetOutputBuffer Purges port write buffer
func (port *portMock) ResetOutputBuffer() error {
	return port.checkError(failInOther)
}

// SetDTR sets the modem status bit DataTerminalReady
func (port *portMock) SetDTR(dtr bool) error {
	port.dtr = dtr
	return port.checkError(failInSetDTR)
}

// SetRTS sets the modem status bit RequestToSend
func (port *portMock) SetRTS(rts bool) error {
	port.rts = rts
	return port.checkError(failInSetRTS)
}

// GetModemStatusBits returns a ModemStatusBits structure containing the
// modem status bits for the serial port (CTS, DSR, etc...)
func (port *portMock) GetModemStatusBits() (*serial.ModemStatusBits, error) {
	return &port.status, port.checkError(failInGetModemStatusBits)
}

// SetReadTimeout sets the timeout for the Read operation or use serial.NoTimeout
// to disable read timeout.
func (port *portMock) SetReadTimeout(t time.Duration) error {
	return port.checkError(failInOther)
}

// Close the serial port
func (port *portMock) Close() error {
	port.stop = true
	return nil
}

// Break sends a break for a determined time
func (port *portMock) Break(time.Duration) error {
	return port.checkError(failInOther)
}

func (port *portMock) Tick() {
	for !port.stop {
		// Must be read every cycle to update in case of changes
		bytesPerSecond := float64(port.mode.BaudRate) / 8.0
		period := 1.0 / bytesPerSecond
		duration := time.Duration(period * float64(time.Second))

		seconds := time.Since(port.previousTick).Seconds()

		if port.previousTick.IsZero() || seconds >= period {
			if port.previousTick.IsZero() {
				port.previousTick = time.Now()
			} else {
				port.previousTick = port.previousTick.Add(duration)
			}

			if !port.portTxBuffer.IsEmpty() {
				port.terminalRxBuffer.Queue(port.portTxBuffer.DeQueue())
			}

			if !port.terminalTxBuffer.IsEmpty() {
				port.portRxBuffer.Queue(port.terminalTxBuffer.DeQueue())
			}
		}
	}
}

func (port *portMock) terminalReceive() []byte {
	var received []byte

	for !port.terminalRxBuffer.IsEmpty() {
		received = append(received, port.terminalRxBuffer.DeQueue())
	}

	return received
}

func (port *portMock) terminalSend(values []byte) {
	for _, value := range values {
		port.terminalTxBuffer.Queue(value)
	}
}
