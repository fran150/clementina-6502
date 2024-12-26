package acia

import (
	"math"
	"sync"

	"github.com/fran150/clementina6502/pkg/components/buses"
	"github.com/fran150/clementina6502/pkg/components/common"
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
	controlStopBitNumberBit uint8 = 0x80 // Bit 7 is the number of stop bits
	controlWordLengthMask   uint8 = 0x60 // Bit 6 - 5 is the word length
	controlBaudMask         uint8 = 0x0F // Bit 3 - 0 is the selected baud rate
)

// Command register bit masks. Bits 7 to 5 control parity and are not emulated
const (
	commandPMCMask   uint8 = 0xC0 // Bit 7 - 6 is parity control
	commandPMEBit    uint8 = 0x20 // Bit 5 is Parity Mode Enabled (must be always disabled)
	commandREMBit    uint8 = 0x10 // Bit 4 is Receiver Echo Mode
	commandTICRTSBit uint8 = 0x08 // Bit 3 controls if RTS enabled or not
	commandTICTXBit  uint8 = 0x04 // Bit 2 controls if TX IRQ is enabled
	commandRIDBit    uint8 = 0x02 // Bit 1 is Receiver interrupt disabled
	commandDTRBit    uint8 = 0x01 // Bit 0 is Data terminal ready

)

const (
	softResetStatusRegMask  uint8 = 0xFB
	softResetCommandRegMask uint8 = 0xE0

	hardResetStatusRegMask   uint8 = 0x60
	hardResetControlRegValue uint8 = 0x00
	hardResetCommandRegValue uint8 = 0x00
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
	wg      *sync.WaitGroup

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
		statusRegister:  statusTDRE,

		txRegisterEmpty: true,
		rxRegisterEmpty: true,
		txRegister:      0x00,
		rxRegister:      0x00,

		rxMutex: &sync.Mutex{},
		txMutex: &sync.Mutex{},
		wg:      &sync.WaitGroup{},

		running: true,
	}

	// Start background pollers to read and write from the serial
	// port in a non blocking way
	acia.wg.Add(1)
	go acia.writeBytes()
	acia.wg.Add(1)
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

	// Sets the values of the modem lines in the port according to the ACIA registers
	acia.setModemLines()
}

// Free resources used by the emulation. In particular it will stop the R/W pollers
func (acia *Acia65C51N) Close() {
	acia.running = false
	acia.wg.Wait()
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

// Chip select line (CS0). The ACIA is selected when CS0 is high and CS1B is low. When the ACIA is selected, the
// internal registers are addressed in accordance with the register select lines (RS0, RS1).
func (via *Acia65C51N) ChipSelect0() *buses.ConnectorEnabledHigh {
	return via.chipSelect1
}

// Chip select line (CS1B). The ACIA is selected when CS0 is high and CS1B is low. When the ACIA is selected, the
// internal registers are addressed in accordance with the register select lines (RS0, RS1).
func (via *Acia65C51N) ChipSelect1() *buses.ConnectorEnabledLow {
	return via.chipSelect2
}

// The two register select lines are normally connected to the processor address lines to allow the processor
// to select the various ACIA internal registers.
// Considering the values of RS0 as bit 0 and RS1 as bit 1 and the R/W line status:
// 0x00 - W: Transmit Data/Shift Register / R: Read Receiver Data Register
// 0x01 - W: Programmed Reset (Data is “Don’t Care”) / R: Read Status Register
// 0x02 - W: Write Command Register / R: Read Command Register
// 0x03 - W:  Write Control Register / R: Read Control Register
func (via *Acia65C51N) RegisterSelect(num uint8) *buses.ConnectorEnabledHigh {
	return via.registerSelect[num]
}

// Resets the ACIA chip when low.
func (via *Acia65C51N) Reset() *buses.ConnectorEnabledLow {
	return via.reset
}

// Connects the specified lines to the register select (RS) lines
func (via *Acia65C51N) ConnectRegisterSelectLines(lines [numOfRSLines]buses.Line) {
	for i := range numOfRSLines {
		via.registerSelect[i].Connect(lines[i])
	}
}

// Returns a byte that respresents the status of the RS lines in where
// RS0 is bit 0 and RS1 is bit 1
func (acia *Acia65C51N) getRegisterSelectValue() uint8 {
	var value uint8

	for i := range numOfRSLines {
		if acia.registerSelect[i].Enabled() {
			value += uint8(math.Pow(2, float64(i)))
		}
	}

	return value
}

// Executes one emulation step
func (acia *Acia65C51N) Tick(stepContext common.StepContext) {
	// Sets the status flag based on modem lines (these are controlled by the modem)
	acia.evaluateModemStatus()
	// Evaluates if the rx record is full and sets the status register accordingly
	acia.evaluateRxRegisterStatus()

	if acia.chipSelect1.Enabled() && acia.chipSelect2.Enabled() {
		// If the chip is enabled trigger the handler function for the
		// seleted register and R/W values
		selectedRegisterValue := acia.getRegisterSelectValue()

		if !acia.readWrite.Enabled() {
			registerReadHandlers[uint8(selectedRegisterValue)](acia)
		} else {
			registerWriteHandlers[uint8(selectedRegisterValue)](acia)
		}
	}

	// Sets the DTR and RTS modem lines (these are controlled by the ACIA)
	acia.setModemLines()

	// Drives the IRQ line based on the status register
	acia.setIRQLine()

	// If the reset line is enabled do a hardware reset
	if acia.reset.Enabled() {
		acia.hardwareReset()
	}
}

// Sets the Data Terminal Ready (DTR) and Ready to Receive (RTS)
// pins in the serial port according to the command register values
func (acia *Acia65C51N) setModemLines() {
	if acia.port != nil {
		dtr := isBitSet(acia.commandRegister, commandDTRBit)
		rts := isBitSet(acia.commandRegister, commandTICRTSBit)

		acia.port.SetDTR(dtr)
		acia.port.SetRTS(rts)
	}
}

// If the modem changes the values of the DSR and DCD values this function updates
// the status accordingly and attempts to triggering an interrupt by setting the IRQ flag
func (acia *Acia65C51N) evaluateModemStatus() {
	isIRQTriggered := isBitSet(acia.statusRegister, statusIRQ)

	if acia.port != nil && !isIRQTriggered {
		status, err := acia.port.GetModemStatusBits()
		if err != nil {
			status = &serial.ModemStatusBits{
				DSR: false,
				DCD: false,
			}
		}

		dsr := isBitSet(acia.statusRegister, statusDSR)
		dcd := isBitSet(acia.statusRegister, statusDCD)

		// If DSR status has changed, update the status register and set interrupt
		if dsr != status.DSR {
			setRegisterBit(&acia.statusRegister, statusDSR, status.DSR)
			setRegisterBit(&acia.statusRegister, statusIRQ, true)
		}

		// If DCD status has changed, update the status register and set interrupt
		if dcd != status.DCD {
			setRegisterBit(&acia.statusRegister, statusDCD, status.DCD)
			setRegisterBit(&acia.statusRegister, statusIRQ, true)
		}
	}
}

// Sets the RDRF status accordingly, if is set to true, it attempts to trigger an
// interrupt by setting the IRQ flag
func (acia *Acia65C51N) evaluateRxRegisterStatus() {
	if !acia.rxRegisterEmpty {
		if !isBitSet(acia.statusRegister, statusRDRF) {
			setRegisterBit(&acia.statusRegister, (statusRDRF | statusIRQ), true)
		}
	}
}

// Drives the IRQ line, if the IRQ status flag is set, the DTR is enabled and
// the IRD is not set, then it enables the exception generating line
func (acia *Acia65C51N) setIRQLine() {
	isIRQDisabled := isBitSet(acia.commandRegister, commandRIDBit)
	isDTREnabled := isBitSet(acia.commandRegister, commandDTRBit)
	isIRQTriggered := isBitSet(acia.statusRegister, statusIRQ)

	if isDTREnabled && !isIRQDisabled && isIRQTriggered {
		acia.irqRequest.SetEnable(true)
	} else {
		acia.irqRequest.SetEnable(false)
	}
}

// Return the current baud rate based on the chip configuration
func (acia *Acia65C51N) getBaudRate() int {
	return baudRate[(acia.controlRegister & controlBaudMask)]
}

// Returns the the word lenght (or number of data bits) based on the chip configuration
func (acia *Acia65C51N) getWordLength() int {
	return int(8 - ((acia.controlRegister & controlWordLengthMask) >> 5))
}

// Returns the number of stop bits
func (acia *Acia65C51N) getStopBits() serial.StopBits {
	// If stop bit is set then it can be 2 or 1.5 depending on the word length
	// if it's unset then its 1
	if isBitSet(acia.controlRegister, controlStopBitNumberBit) {
		dataBits := acia.getWordLength()

		if dataBits == 5 {
			return serial.OnePointFiveStopBits
		} else {
			return serial.TwoStopBits
		}
	} else {
		return serial.OneStopBit
	}
}

// Gets the serial port mode based on the ACIA chip configuration.
// This sets the baud rate, number of data bits, parity and stop bits configuration
func (acia *Acia65C51N) getMode() *serial.Mode {
	mode := &serial.Mode{
		BaudRate: acia.getBaudRate(),
		DataBits: acia.getWordLength(),
		Parity:   serial.NoParity,
		StopBits: acia.getStopBits(),
	}

	return mode
}

// Returns if the CTS line is enabled. If serial port is not connected it will return false
// If an error occurs reading the line it will assume that the value is true. This is to
// allow the emulation to work when this line is not supported (for example when using SOCAT command)
func (acia *Acia65C51N) isCTSEnabled() bool {
	var cts bool

	if acia.port != nil {
		status, err := acia.port.GetModemStatusBits()
		if err == nil {
			cts = status.CTS
		} else {
			cts = true
		}
	} else {
		cts = false
	}

	return cts
}

func (acia *Acia65C51N) isReceiverEchoModeEnabled() bool {
	return (acia.commandRegister & (commandREMBit | commandTICRTSBit | commandTICTXBit)) == commandREMBit
}

// Performs a hardware reset
func (acia *Acia65C51N) hardwareReset() {
	acia.statusRegister &= hardResetStatusRegMask
	acia.statusRegister |= statusTDRE
	acia.controlRegister &= hardResetControlRegValue
	acia.commandRegister &= hardResetCommandRegValue
}
