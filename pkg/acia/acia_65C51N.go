package acia

import (
	"math"
	"sync"

	"github.com/fran150/clementina6502/pkg/buses"
	"github.com/fran150/clementina6502/pkg/common"
	"go.bug.st/serial"
)

const NUM_OF_RS_LINES uint8 = 2

const (
	statusIRQ          uint8 = 0x80
	statusDSR          uint8 = 0x40
	statusDCD          uint8 = 0x20
	statusTDRE         uint8 = 0x10
	statusRDRF         uint8 = 0x08
	statusOverrun      uint8 = 0x04
	statusFramingError uint8 = 0x02
	statusParityError  uint8 = 0x01
)

const (
	controlStopBitNumberMask uint8 = 0x80
	controlWordLengthMask    uint8 = 0x60
	controlBaudMask          uint8 = 0x0F
)

const (
	commandDTRMask = 0x01
	commandRIDMask = 0x02
	commandTICMask = 0x0C
)

var baudRate = [...]int{
	115200, // 0x00
	50,     // 0x01
	75,     // 0x02
	110,    // 0x03
	135,    // 0x04
	150,    // 0x05
	300,    // 0x06
	600,    // 0x07
	1200,   // 0x08
	1800,   // 0x09
	2400,   // 0x0A
	3600,   // 0x0B
	4800,   // 0x0C
	7200,   // 0x0D
	9600,   // 0x0E
	19200,  // 0x0F
}

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

	rxMutex *sync.Mutex
	txMutex *sync.Mutex

	running bool
}

func CreateAcia65C51N() *Acia65C51N {
	acia := &Acia65C51N{
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
		statusRegister:  0x10,

		txRegisterEmpty: true,
		rxRegisterEmpty: true,
		txRegister:      0x00,
		rxRegister:      0x00,

		rxMutex: &sync.Mutex{},
		txMutex: &sync.Mutex{},

		running: true,
	}

	go acia.writeBytes()
	go acia.readBytes()

	return acia
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

func (acia *Acia65C51N) ConnectToPort(port serial.Port) {
	acia.port = port

	mode := acia.getMode()
	err := acia.port.SetMode(mode)
	if err != nil {
		panic(err)
	}

	acia.setModemLines()
}

func (acia *Acia65C51N) Close() {
	acia.running = false
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

func (acia *Acia65C51N) Tick(stepContext common.StepContext) {
	if acia.chipSelect1.Enabled() && acia.chipSelect2.Enabled() {
		selectedRegisterValue := acia.getRegisterSelectValue()

		if !acia.readWrite.Enabled() {
			registerReadHandlers[uint8(selectedRegisterValue)](acia)
		} else {
			registerWriteHandlers[uint8(selectedRegisterValue)](acia)
		}
	}

	acia.setStatusRegister()

	if acia.reset.Enabled() {
		acia.hardwareReset()
	}
}

func (acia *Acia65C51N) setModemLines() {
	if acia.commandRegister&commandDTRMask == 0x00 {
		acia.port.SetDTR(false)
	} else {
		acia.port.SetDTR(true)
	}

	if acia.commandRegister&commandTICMask == 0x00 {
		acia.port.SetRTS(false)
	} else {
		acia.port.SetRTS(true)
	}
}

func (acia *Acia65C51N) setModemStatusBit(value bool, statusBit uint8) {
	if value {
		if acia.statusRegister&statusBit == 0x00 {
			acia.statusRegister |= (statusBit | statusIRQ)
		}
	} else {
		if acia.statusRegister&statusBit == statusBit {
			acia.statusRegister &= ^statusBit
			acia.statusRegister |= statusIRQ
		}
	}
}

func (acia *Acia65C51N) setStatusRegister() {
	if acia.port != nil {
		status, err := acia.port.GetModemStatusBits()

		if err == nil {
			acia.setModemStatusBit(status.DSR, statusDSR)
			acia.setModemStatusBit(status.DCD, statusDSR)
		}
	}

	if !acia.rxRegisterEmpty {
		if acia.statusRegister&statusRDRF == 0x00 {
			acia.statusRegister |= statusRDRF

			// Receiver Interrupt Disabled (Bit 1)
			// This bit disables the Receiver from generating an interrupt when set to a 1. The Receiver interrupt is
			// enabled when this bit is set to a 0 and Bit 0 is set to a 1.
			if acia.commandRegister&(commandRIDMask|commandDTRMask) == commandRIDMask {
				acia.statusRegister |= statusIRQ
			}
		}
	}

	if acia.statusRegister&statusIRQ == statusIRQ {
		acia.irqRequest.SetEnable(true)
	} else {
		acia.irqRequest.SetEnable(false)
	}
}

func (acia *Acia65C51N) getMode() *serial.Mode {
	mode := &serial.Mode{
		BaudRate: baudRate[(acia.controlRegister & controlBaudMask)],
		DataBits: int(8 - ((acia.controlRegister & controlStopBitNumberMask) >> 5)),
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

	return mode
}

func (acia *Acia65C51N) hardwareReset() {
	acia.statusRegister &= 0x70
	acia.statusRegister |= 0x10
	acia.controlRegister &= 0x00
	acia.commandRegister &= 0x00
}
