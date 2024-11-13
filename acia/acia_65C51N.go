package acia

import (
	"math"
	"time"

	"github.com/fran150/clementina6502/buses"
	"go.bug.st/serial"
)

const NUM_OF_RS_LINES uint8 = 2

const (
	StatusIRQ          uint8 = 0x80
	StatusDSR          uint8 = 0x40
	StatusDCD          uint8 = 0x20
	StatusTDRE         uint8 = 0x10
	StatusRDRF         uint8 = 0x08
	StatusOverrun      uint8 = 0x04
	StatusFramingError uint8 = 0x02
	StatusParityError  uint8 = 0x01
)

type Acia65C51N struct {
	dataBus        *buses.BusConnector[uint8]
	irqRequest     *buses.ConnectorEnabledLow
	readWrite      *buses.ConnectorEnabledLow
	chipSelect1    *buses.ConnectorEnabledHigh
	chipSelect2    *buses.ConnectorEnabledLow
	registerSelect [2]*buses.ConnectorEnabledHigh
	reset          *buses.ConnectorEnabledLow

	statusRegister  uint8
	controlRegister uint8
	commandRegister uint8

	txRegisterEmpty bool
	rxRegisterEmpty bool

	txRegister uint8
	rxRegister uint8

	port serial.Port
}

func CreateAcia65C51N(portName string) *Acia65C51N {
	acia := createAcia65C51N()

	acia.openPort(portName)

	go acia.writeBytes()
	go acia.readBytes()

	return acia
}

func InitializeAcia65C51NWithPort(port serial.Port) *Acia65C51N {
	acia := createAcia65C51N()

	mode := acia.getMode()
	port.SetMode(mode)
	acia.port = port

	go acia.writeBytes()
	go acia.readBytes()

	return acia
}

func createAcia65C51N() *Acia65C51N {
	return &Acia65C51N{
		dataBus:     buses.CreateBusConnector[uint8](),
		irqRequest:  buses.CreateConnectorEnabledLow(),
		readWrite:   buses.CreateConnectorEnabledLow(),
		chipSelect1: buses.CreateConnectorEnabledHigh(),
		chipSelect2: buses.CreateConnectorEnabledLow(),
		registerSelect: [NUM_OF_RS_LINES]*buses.ConnectorEnabledHigh{
			buses.CreateConnectorEnabledHigh(),
			buses.CreateConnectorEnabledHigh(),
		},
		reset: buses.CreateConnectorEnabledLow(),

		commandRegister: 0x00,
		controlRegister: 0x00,
		statusRegister:  0x00,

		txRegisterEmpty: true,
		rxRegisterEmpty: true,
		txRegister:      0x00,
		rxRegister:      0x00,
	}
}

var registerWriteHandlers = []func(*Acia65C51N){
	writeTransmitData,
	programmedReset,
	writeCommand,
	writeControl,
}

var registerReadHandlers = []func(*Acia65C51N){
	readReceiverData,
	readStatus,
	readCommand,
	readControl,
}

func (via *Acia65C51N) DataBus() *buses.BusConnector[uint8] {
	return via.dataBus
}

func (via *Acia65C51N) IrqRequest() *buses.ConnectorEnabledLow {
	return via.irqRequest
}

func (via *Acia65C51N) ReadWrite() *buses.ConnectorEnabledLow {
	return via.readWrite
}

func (via *Acia65C51N) ChipSelect0() *buses.ConnectorEnabledHigh {
	return via.chipSelect1
}

func (via *Acia65C51N) ChipSelect1() *buses.ConnectorEnabledLow {
	return via.chipSelect2
}

func (via *Acia65C51N) RegisterSelect(num uint8) *buses.ConnectorEnabledHigh {
	return via.registerSelect[num]
}

func (via *Acia65C51N) Reset() *buses.ConnectorEnabledLow {
	return via.reset
}

func (via *Acia65C51N) ConnectRegisterSelectLines(lines [NUM_OF_RS_LINES]buses.Line) {
	for i := range NUM_OF_RS_LINES {
		via.registerSelect[i].Connect(lines[i])
	}
}

func (acia *Acia65C51N) getRegisterSelectValue() uint8 {
	var value uint8

	for i := range NUM_OF_RS_LINES {
		if acia.registerSelect[i].Enabled() {
			value += uint8(math.Pow(2, float64(i)))
		}
	}

	return value
}

func (acia *Acia65C51N) Tick(cycles uint64, t time.Time) {
	if acia.chipSelect1.Enabled() && acia.chipSelect2.Enabled() {
		selectedRegisterValue := acia.getRegisterSelectValue()

		if !acia.readWrite.Enabled() {
			registerReadHandlers[uint8(selectedRegisterValue)](acia)
		} else {
			registerWriteHandlers[uint8(selectedRegisterValue)](acia)
		}
	}

	acia.setStatusRegister()
}

func (acia *Acia65C51N) setStatusRegister() {

	status, err := acia.port.GetModemStatusBits()

	if err == nil {
		if status.DSR {
			if acia.statusRegister&StatusDSR == 0x00 {
				acia.statusRegister |= (StatusDSR | StatusIRQ)
			}
		} else {
			if acia.statusRegister&StatusDSR == 0x00 {
				acia.statusRegister &= ^StatusDSR
				acia.statusRegister |= StatusIRQ
			}
		}

		if status.DCD {
			if acia.statusRegister&StatusDCD == 0x00 {
				acia.statusRegister |= (StatusDCD | StatusIRQ)
			}
		} else {
			if acia.statusRegister&StatusDCD == 0x00 {
				acia.statusRegister &= ^StatusDCD
				acia.statusRegister |= StatusIRQ
			}
		}
	}

	if !acia.rxRegisterEmpty {
		if acia.statusRegister&StatusRDRF == 0x00 {
			acia.statusRegister |= StatusRDRF

			// Receiver Interrupt Control (Bit 1)
			// This bit disables the Receiver from generating an interrupt when set to a 1. The Receiver interrupt is
			// enabled when this bit is set to a 0 and Bit 0 is set to a 1.
			if acia.commandRegister&0x02 == 0x00 && acia.commandRegister&0x01 == 0x01 {
				acia.statusRegister |= StatusIRQ
			}
		}
	}

	if acia.statusRegister&StatusIRQ == StatusIRQ {
		acia.irqRequest.SetEnable(true)
	} else {
		acia.irqRequest.SetEnable(false)
	}
}

func (acia *Acia65C51N) openPort(portName string) {
	var err error

	mode := acia.getMode()

	acia.port, err = serial.Open(portName, mode)

	if err != nil {
		panic(err)
	}
}

func (acia *Acia65C51N) getMode() *serial.Mode {
	mode := &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	if acia.controlRegister&0x80 == 0x80 {
		if acia.controlRegister&0x60 == 0x60 {
			mode.StopBits = serial.OnePointFiveStopBits
		} else {
			mode.StopBits = serial.TwoStopBits
		}
	} else {
		mode.StopBits = serial.OneStopBit
	}

	switch acia.controlRegister & 0x60 {
	case 0x00:
		mode.DataBits = 8
	case 0x20:
		mode.DataBits = 7
	case 0x40:
		mode.DataBits = 6
	case 0x60:
		mode.DataBits = 5
	}

	switch acia.controlRegister & 0x0F {
	case 0x00:
		mode.BaudRate = 115200
	case 0x01:
		mode.BaudRate = 50
	case 0x02:
		mode.BaudRate = 75
	case 0x03:
		mode.BaudRate = 109920
	case 0x04:
		mode.BaudRate = 134580
	case 0x05:
		mode.BaudRate = 150
	case 0x06:
		mode.BaudRate = 300
	case 0x07:
		mode.BaudRate = 600
	case 0x08:
		mode.BaudRate = 1200
	case 0x09:
		mode.BaudRate = 1800
	case 0x0A:
		mode.BaudRate = 2400
	case 0x0B:
		mode.BaudRate = 3600
	case 0x0C:
		mode.BaudRate = 4800
	case 0x0D:
		mode.BaudRate = 7200
	case 0x0E:
		mode.BaudRate = 9600
	case 0x0F:
		mode.BaudRate = 19200
	}

	return mode
}
