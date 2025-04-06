package testutils

import (
	"time"

	"github.com/fran150/clementina6502/internal/queue"
	"go.bug.st/serial"
)

type failInFunction int

const (
	FailInNone failInFunction = iota
	FailInSetMode
	FailInRead
	FailInWrite
	FailInSetDTR
	FailInSetRTS
	FailInGetModemStatusBits
	FailInSetReadTimeout
	FailInOther
)

// Struct that mocks the serial port interface for testing
type SerialPortMock struct {
	Mode   *serial.Mode
	Status serial.ModemStatusBits
	DTR    bool
	RTS    bool

	PortTxBuffer *queue.SimpleQueue[byte]
	PortRxBuffer *queue.SimpleQueue[byte]

	TerminalTxBuffer *queue.SimpleQueue[byte]
	TerminalRxBuffer *queue.SimpleQueue[byte]

	previousTick time.Time

	stop bool

	MakeCallsFailFrom failInFunction
}

// Creates a new mock of the serial port
func NewPortMock(mode *serial.Mode) *SerialPortMock {
	return &SerialPortMock{
		Mode: mode,
		Status: serial.ModemStatusBits{
			CTS: true,
			DSR: false,
			RI:  false,
			DCD: false,
		},
		DTR:              false,
		RTS:              false,
		PortTxBuffer:     queue.NewQueue[byte](),
		TerminalTxBuffer: queue.NewQueue[byte](),
		PortRxBuffer:     queue.NewQueue[byte](),
		TerminalRxBuffer: queue.NewQueue[byte](),

		MakeCallsFailFrom: FailInNone,
	}
}

// Returns error if the mock is configured to fail
func (port *SerialPortMock) checkError(calledFrom failInFunction) error {
	if port.MakeCallsFailFrom == calledFrom {
		return serial.PortError{}
	} else {
		return nil
	}
}

// SetMode sets all parameters of the serial port
func (port *SerialPortMock) SetMode(mode *serial.Mode) error {
	port.Mode = mode

	return port.checkError(FailInSetMode)
}

// Stores data received from the serial port into the provided byte array
// buffer. The function returns the number of bytes read.
//
// The Read function blocks until (at least) one byte is received from
// the serial port or an error occurs.
func (port *SerialPortMock) Read(p []byte) (n int, err error) {
	for port.PortRxBuffer.IsEmpty() && !port.stop {
	}

	i := 0
	for !port.PortRxBuffer.IsEmpty() && i < len(p) {
		p[i] = port.PortRxBuffer.DeQueue()
		i++
	}

	return len(p), port.checkError(FailInRead)
}

// Send the content of the data byte array to the serial port.
// Returns the number of bytes written.
func (port *SerialPortMock) Write(p []byte) (n int, err error) {
	for _, v := range p {
		port.PortTxBuffer.Queue(v)
	}

	return port.PortTxBuffer.Size(), port.checkError(FailInWrite)
}

// Wait until all data in the buffer are sent
func (port *SerialPortMock) Drain() error {
	return port.checkError(FailInOther)
}

// ResetInputBuffer Purges port read buffer
func (port *SerialPortMock) ResetInputBuffer() error {
	return port.checkError(FailInOther)
}

// ResetOutputBuffer Purges port write buffer
func (port *SerialPortMock) ResetOutputBuffer() error {
	return port.checkError(FailInOther)
}

// SetDTR sets the modem status bit DataTerminalReady
func (port *SerialPortMock) SetDTR(dtr bool) error {
	port.DTR = dtr
	return port.checkError(FailInSetDTR)
}

// SetRTS sets the modem status bit RequestToSend
func (port *SerialPortMock) SetRTS(rts bool) error {
	port.RTS = rts
	return port.checkError(FailInSetRTS)
}

// GetModemStatusBits returns a ModemStatusBits structure containing the
// modem status bits for the serial port (CTS, DSR, etc...)
func (port *SerialPortMock) GetModemStatusBits() (*serial.ModemStatusBits, error) {
	return &port.Status, port.checkError(FailInGetModemStatusBits)
}

// SetReadTimeout sets the timeout for the Read operation or use serial.NoTimeout
// to disable read timeout.
func (port *SerialPortMock) SetReadTimeout(t time.Duration) error {
	return port.checkError(FailInSetReadTimeout)
}

// Close the serial port
func (port *SerialPortMock) Close() error {
	port.stop = true
	return nil
}

// Break sends a break for a determined time
func (port *SerialPortMock) Break(time.Duration) error {
	return port.checkError(FailInOther)
}

func (port *SerialPortMock) Tick() {
	for !port.stop {
		// Must be read every cycle to update in case of changes
		bytesPerSecond := float64(port.Mode.BaudRate) / 8.0
		period := 1.0 / bytesPerSecond
		duration := time.Duration(period * float64(time.Second))

		seconds := time.Since(port.previousTick).Seconds()

		if port.previousTick.IsZero() || seconds >= period {
			if port.previousTick.IsZero() {
				port.previousTick = time.Now()
			} else {
				port.previousTick = port.previousTick.Add(duration)
			}

			if !port.PortTxBuffer.IsEmpty() {
				port.TerminalRxBuffer.Queue(port.PortTxBuffer.DeQueue())
			}

			if !port.TerminalTxBuffer.IsEmpty() {
				port.PortRxBuffer.Queue(port.TerminalTxBuffer.DeQueue())
			}
		}
	}
}

func (port *SerialPortMock) TerminalReceive() []byte {
	var received []byte

	for !port.TerminalRxBuffer.IsEmpty() {
		received = append(received, port.TerminalRxBuffer.DeQueue())
	}

	return received
}

func (port *SerialPortMock) TerminalSend(values []byte) {
	for _, value := range values {
		port.TerminalTxBuffer.Queue(value)
	}
}
