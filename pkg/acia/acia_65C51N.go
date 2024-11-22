package acia

import (
	"math"
	"sync"

	"github.com/fran150/clementina6502/pkg/buses"
	"github.com/fran150/clementina6502/pkg/common"
	"go.bug.st/serial"
)

// Number of lines / pins used for register select
const numOfRSLines uint8 = 2

// Status register bits
const (
	statusIRQ          uint8 = 0x80 // Interrupt has ocurred (IRQ)
	statusDSR          uint8 = 0x40 // Data set ready
	statusDCD          uint8 = 0x20 // Data carrier detect
	statusTDRE         uint8 = 0x10 // Transmitter data register empty (always 1 in 65C51N)
	statusRDRF         uint8 = 0x08 // Receiver data register full
	statusOverrun      uint8 = 0x04 // Overrun has ocurred
	statusFramingError uint8 = 0x02 // Framing error has ocurrer
	statusParityError  uint8 = 0x01 // Parity error detected
)

// Control register bit masks (Bit 4 Receiver Clock Source not emulated)
const (
	controlStopBitNumberMask uint8 = 0x80 // Bit 7 is the number of stop bits
	controlWordLengthMask    uint8 = 0x60 // Bit 6 - 5 is the word length
	controlBaudMask          uint8 = 0x0F // Bit 3 - 0 is the selected baud rate
)

// Command register bit masks. Bits 7 to 5 control parity and are not emulated
const (
	commandDTRMask = 0x01 // Bit 0 is Data terminal ready
	commandRIDMask = 0x02 // Bit 1 is Receiver interrupt disabled
	commandTICMask = 0x0C // Bit 2 - 3 is Transmitter interrupt control
	commandREMMask = 0x10 // Bit 4 is Receiver echo mode
)

// Map of the baud rate the value of the last 4 bits in the control register
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

// The WDC CMOS W65C51N Asynchronous Communications Interface Adapter (ACIA) provides an easily
// implemented, program controlled interface between 8-bit microprocessor based systems and serial
// communication data sets and modems.
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

// Creates an ACIA chip in default initialization state
func CreateAcia65C51N() *Acia65C51N {
	acia := &Acia65C51N{
		dataBus:     buses.CreateBusConnector[uint8](),
		irqRequest:  buses.CreateConnectorEnabledLow(),
		readWrite:   buses.CreateConnectorEnabledLow(),
		chipSelect1: buses.CreateConnectorEnabledHigh(),
		chipSelect2: buses.CreateConnectorEnabledLow(),
		registerSelect: [numOfRSLines]*buses.ConnectorEnabledHigh{
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

	// Start background pollers to read and write from the serial
	// port in a non blocking way
	go acia.writeBytes()
	go acia.readBytes()

	return acia
}

// These are the actions executed according to each value of the register
// select lines and when the R/W pin is in write
var registerWriteHandlers = []func(*Acia65C51N){
	writeTransmitData,
	programmedReset,
	writeCommand,
	writeControl,
}

// These are the actions executed according to each value of the register
// select lines and when the R/W pin is in read
var registerReadHandlers = []func(*Acia65C51N){
	readReceiverData,
	readStatus,
	readCommand,
	readControl,
}

// Connects the ACIA chip to the specified serial port.
// The port must be open and it's mode will be reconfigured according with the register
// values withing the ACIA chip
func (acia *Acia65C51N) ConnectToPort(port serial.Port) {
	acia.port = port

	mode := acia.getMode()
	err := acia.port.SetMode(mode)
	if err != nil {
		panic(err)
	}

	acia.setModemLines()
}

// Free resources used by the emulation. In particular it will stop the R/W pollers
func (acia *Acia65C51N) Close() {
	acia.running = false
}

// The eight data line (D0-D7) pins transfer data between the processor and the ACIA. These lines are bi-
// directional and are normally high-impedance except during Read cycles when the ACIA is selected.
func (via *Acia65C51N) DataBus() *buses.BusConnector[uint8] {
	return via.dataBus
}

// The IRQB pin is an interrupt output from the interrupt control logic. Normally a high level, IRQB
// goes low when an interrupt occurs.
func (via *Acia65C51N) IrqRequest() *buses.ConnectorEnabledLow {
	return via.irqRequest
}

// The RWB input, generated by the microprocessor controls the
// direction of data transfers. A high on the RWB pin allows the processor to read the data supplied by the ACIA, a low allows a write to the ACIA.
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

func (via *Acia65C51N) ConnectRegisterSelectLines(lines [numOfRSLines]buses.Line) {
	for i := range numOfRSLines {
		via.registerSelect[i].Connect(lines[i])
	}
}

func (acia *Acia65C51N) getRegisterSelectValue() uint8 {
	var value uint8

	for i := range numOfRSLines {
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
