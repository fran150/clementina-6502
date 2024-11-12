package acia

import (
	"time"

	"go.bug.st/serial"
)

type PortMock struct {
	mode *serial.Mode
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
	return 0, nil
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
	return nil
}

// SetRTS sets the modem status bit RequestToSend
func (port *PortMock) SetRTS(rts bool) error {
	return nil
}

// GetModemStatusBits returns a ModemStatusBits structure containing the
// modem status bits for the serial port (CTS, DSR, etc...)
func (port *PortMock) GetModemStatusBits() (*serial.ModemStatusBits, error) {
	return &serial.ModemStatusBits{
		CTS: true,
		DSR: true,
		RI:  false,
		DCD: false,
	}, nil
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
