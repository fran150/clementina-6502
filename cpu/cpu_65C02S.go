package cpu

import "github.com/fran150/clementina6502/buses"

// Represents the WDC 65C02S processor. See https://www.westerndesigncenter.com/wdc/documentation/w65c02s.pdf
// for details.
//
//   - Address bus are pins A0-A15
//   - The Bus Enable (BE) input signal provides external control of the Address, Data and the RWB buffers. When
//     Bus Enable is high, the Address, Data and RWB buffers are active.
//   - Data bus are pins D0-D7
//   - The Interrupt Request (IRQB) input signal is used to request that an interrupt sequence be initiated. The
//     program counter (PC) and Processor Status Register (P) are pushed onto the stack and the IRQB disable
//     (I) flag is set to a “1” disabling further interrupts before jumping to the interrupt handler. These values are
//     used to return the processor to its original state prior to the IRQB interrupt. The IRQB low level should be
//     held until the interrupt handler clears the interrupt request source. When Return from Interrupt (RTI) is
//     executed the (I) flag is restored and a new interrupt can be handled. If the (I) flag is cleared in an interrupt
//     handler, nested interrupts can occur.
//   - A negative transition on the Non-Maskable Interrupt (NMIB) input initiates an interrupt sequence after the
//     current instruction is completed. Since NMIB is an edge-sensitive input, an interrupt will occur if there is a
//     negative transition while servicing a previous interrupt. Also, after the edge interrupt occurs no further
//     interrupts will occur if NMIB remains low. The NMIB signal going low causes the Program Counter (PC) and
//     Processor Status Register information to be pushed onto the stack before jumping to the interrupt handler.
//     These values are used to return the processor to its original state prior to the NMIB interrupt.
//   - The Read/Write (RWB) output signal is used to control data transfer. When in the high state, the
//     microprocessor is reading data from memory or I/O. When in the low state, the Data Bus contains valid data
//     to be written from the microprocessor and stored at the addressed memory or I/O location. The RWB signal
//     is set to the high impedance state when Bus Enable (BE) is low
//   - The Reset (RESB) input is used to initialize the microprocessor and start program execution. The RESB
//     signal must be held low for at least two clock cycles after VDD reaches operating voltage.
//     All Registers are initialized by software except the Decimal and Interrupt disable mode select bits of
//     the Processor Status Register (P) are initialized by hardware.
//     When a positive edge is detected, there will be a reset sequence lasting seven clock cycles. The program
//     counter is loaded with the reset vector from locations FFFC (low byte) and FFFD (high byte). This is the
//     start location for program control. RESB should be held high after reset for normal operation

type Cpu65C02S struct {
	addressBus           *buses.Bus[uint16]
	busEnable            *buses.ConnectorEnabledHigh
	dataBus              *buses.Bus[uint8]
	interruptRequest     *buses.ConnectorEnabledLow
	nonMaskableInterrupt *buses.ConnectorEnabledLow
	reset                *buses.ConnectorEnabledLow
	readWrite            *buses.ConnectorEnabledLow

	accumulatorRegister     uint8
	xRegister               uint8
	yRegister               uint8
	stackPointer            uint8
	programCounter          uint16
	processorStatusRegister ProcessorStatusRegister

	currentCycleType  CycleType
	currentCycleIndex uint8
	currentOpCode     uint8
}

// Creates a CPU with typical values for all lines, address and data bus are not connected
func CreateCPU() *Cpu65C02S {
	return &Cpu65C02S{
		busEnable:               buses.CreateConnectorEnabledHigh(),
		interruptRequest:        buses.CreateConnectorEnabledLow(),
		nonMaskableInterrupt:    buses.CreateConnectorEnabledLow(),
		reset:                   buses.CreateConnectorEnabledLow(),
		readWrite:               buses.CreateConnectorEnabledLow(),
		accumulatorRegister:     0x00,
		xRegister:               0x00,
		yRegister:               0x00,
		stackPointer:            0xFF,
		programCounter:          0xFFFC,
		processorStatusRegister: 0x00,

		currentCycleType:  CycleReadOpCode,
		currentCycleIndex: 0,
		currentOpCode:     0x00,
	}
}

/*
 ****************************************************
 * Connections
 ****************************************************
 */

// Connects the CPU to an address bus, must be 16 bits long
func (cpu *Cpu65C02S) ConnectAddressBus(addressBus *buses.Bus[uint16]) {
	cpu.addressBus = addressBus
}

// Connects the CPU to a data bus, must be 8 bits long
func (cpu *Cpu65C02S) ConnectDataBus(dataBus *buses.Bus[uint8]) {
	cpu.dataBus = dataBus
}

/*
 ****************************************************
 * Control Lines
 ****************************************************
 */

func (cpu *Cpu65C02S) BusEnable() *buses.ConnectorEnabledHigh {
	return cpu.busEnable
}

func (cpu *Cpu65C02S) InterruptRequest() *buses.ConnectorEnabledLow {
	return cpu.interruptRequest
}

func (cpu *Cpu65C02S) NonMaskableInterrupt() *buses.ConnectorEnabledLow {
	return cpu.nonMaskableInterrupt
}

func (cpu *Cpu65C02S) Reset() *buses.ConnectorEnabledLow {
	return cpu.reset
}

func (cpu *Cpu65C02S) ReadWrite() *buses.ConnectorEnabledLow {
	return cpu.readWrite
}

/*
 ****************************************************
 * Timer Tick
 ****************************************************
 */

func (cpu *Cpu65C02S) Tick(t uint64) {
	switch cpu.currentCycleType {
	case CycleReadOpCode:
		cpu.setReadBus()
	case CycleAction:
		cpu.setReadBus()
	}
}

func (cpu *Cpu65C02S) PostTick(t uint64) {
	switch cpu.currentCycleType {
	case CycleReadOpCode:
		cpu.currentOpCode = cpu.dataBus.Read()
	case CycleAction:
		cpu.accumulatorRegister = cpu.dataBus.Read()
	}

	cpu.programCounter++

	cpu.currentCycleIndex++

	if int(cpu.currentCycleIndex) >= len(cpu.getCurrentAddressMode().Cycles) {
		cpu.currentCycleIndex = 0
		cpu.currentCycleType = CycleReadOpCode
	} else {
		cpu.currentCycleType = cpu.getCurrentAddressMode().Cycles[cpu.currentCycleIndex]
	}
}

/*
 ****************************************************
 * Internal Bus Handling
 ****************************************************
 */

// TODO: Handle disconnected lines, Handle bus conflict
func (cpu *Cpu65C02S) setReadBus() {
	if cpu.busEnable.Enabled() {
		cpu.readWrite.SetEnable(false)
		cpu.addressBus.Write(cpu.programCounter)
	}
}

func (cpu *Cpu65C02S) setWriteBus(address uint16, data uint8) {
	if cpu.busEnable.Enabled() {
		cpu.readWrite.SetEnable(true)
		cpu.addressBus.Write(address)
		cpu.dataBus.Write(data)
	}
}

func (cpu *Cpu65C02S) getCurrentOpCodeData() OpCodeData {
	return OpCodes[cpu.currentOpCode]
}

func (cpu *Cpu65C02S) getCurrentAddressMode() AddressModeData {
	return AddressModes[OpCodes[cpu.currentOpCode].AddressMode]
}
