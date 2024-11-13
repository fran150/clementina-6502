package acia

import (
	"math"
	"time"

	"go.bug.st/serial"
)

type PortMock struct {
	mode   *serial.Mode
	status serial.ModemStatusBits
	dtr    bool
	rts    bool

	rxPointer int
	rxBuffer  [1000]byte
	txPointer int
	txBuffer  [1000]byte

	sent     []byte
	received []byte

	previousTick time.Time
}

// SetMode sets all parameters of the serial port
func (port *PortMock) SetMode(mode *serial.Mode) error {
	port.mode = mode

	return nil
}

// Stores data received from the serial port into the provided byte array
// buffer. The function returns the number of bytes read.
//
// The Read function blocks until (at least) one byte is received from
// the serial port or an error occurs.
func (port *PortMock) Read(p []byte) (n int, err error) {
	return 0, nil
}

// Send the content of the data byte array to the serial port.
// Returns the number of bytes written.
func (port *PortMock) Write(p []byte) (n int, err error) {
	for _, value := range p {
		port.rxBuffer[port.rxPointer] = value
		port.rxPointer = (port.rxPointer + 1) % 1000
	}

	return len(p), nil
}

// Wait until all data in the buffer are sent
func (port *PortMock) Drain() error {
	return nil
}

// ResetInputBuffer Purges port read buffer
func (port *PortMock) ResetInputBuffer() error {
	return nil
}

// ResetOutputBuffer Purges port write buffer
func (port *PortMock) ResetOutputBuffer() error {
	return nil
}

// SetDTR sets the modem status bit DataTerminalReady
func (port *PortMock) SetDTR(dtr bool) error {
	port.dtr = dtr
	return nil
}

// SetRTS sets the modem status bit RequestToSend
func (port *PortMock) SetRTS(rts bool) error {
	port.rts = rts
	return nil
}

// GetModemStatusBits returns a ModemStatusBits structure containing the
// modem status bits for the serial port (CTS, DSR, etc...)
func (port *PortMock) GetModemStatusBits() (*serial.ModemStatusBits, error) {
	return &port.status, nil
}

// SetReadTimeout sets the timeout for the Read operation or use serial.NoTimeout
// to disable read timeout.
func (port *PortMock) SetReadTimeout(t time.Duration) error {
	return nil
}

// Close the serial port
func (port *PortMock) Close() error {
	return nil
}

// Break sends a break for a determined time
func (port *PortMock) Break(time.Duration) error {
	return nil
}

func (port *PortMock) Tick() {
	for {
		if port.previousTick.IsZero() {
			port.previousTick = time.Now()
		} else {
			t := time.Now()
			dt := t.Sub(port.previousTick)
			seconds := dt.Seconds()

			bytesPerSecond := float64(port.mode.BaudRate) / 8.0
			bytesToProcess := math.Floor(bytesPerSecond * seconds)

			for bytesToProcess > 0 {
				// Receive
				if port.rxPointer > 0 {
					port.rxPointer--
					port.received = append(port.received, port.rxBuffer[port.rxPointer])
				} else {
					break
				}

				bytesToProcess--
				port.previousTick = t
			}
		}
	}
}
