package cpu

import (
	"slices"

	"github.com/fran150/clementina6502/buses"
)

// Represents the WDC 65C02S processor. See https://www.westerndesigncenter.com/wdc/documentation/w65c02s.pdf
// for details.
// There is another document for the rockwell processor that has better data about cycle timing here:
// https://web.archive.org/web/20221112220234if_/http://archive.6502.org/datasheets/rockwell_r65c00_microprocessors.pdf
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
	memoryLock           *buses.ConnectorEnabledLow
	nonMaskableInterrupt *buses.ConnectorEnabledLow
	readWrite            *buses.ConnectorEnabledLow
	ready                *buses.ConnectorEnabledHigh
	reset                *buses.ConnectorEnabledLow
	setOverflow          *buses.ConnectorEnabledLow
	sync                 *buses.ConnectorEnabledHigh
	vectorPull           *buses.ConnectorEnabledLow

	addressModeSet *AddressModeSet
	instructionSet *CpuInstructionSet

	accumulatorRegister     uint8
	xRegister               uint8
	yRegister               uint8
	stackPointer            uint8
	programCounter          uint16
	processorStatusRegister StatusRegister

	currentInstruction *CpuInstructionData
	currentAddressMode *AddressModeData
	currentCycleIndex  int
	currentCycle       cycleActions

	instructionRegisterCarry bool
	branchTaken              bool
	currentOpCode            OpCode
	instructionRegister      uint16
	dataRegister             uint8

	irqRequested      bool
	previousNMIStatus bool
	nmiRequested      bool
	cyclesWithReset   uint8

	processorPaused  bool
	processorStopped bool
}

// Creates a CPU with typical values for all registers, address and data bus are not connected
func CreateCPU() *Cpu65C02S {
	return &Cpu65C02S{
		busEnable:            buses.CreateConnectorEnabledHigh(),
		interruptRequest:     buses.CreateConnectorEnabledLow(),
		memoryLock:           buses.CreateConnectorEnabledLow(),
		nonMaskableInterrupt: buses.CreateConnectorEnabledLow(),
		reset:                buses.CreateConnectorEnabledLow(),
		setOverflow:          buses.CreateConnectorEnabledLow(),
		readWrite:            buses.CreateConnectorEnabledLow(),
		ready:                buses.CreateConnectorEnabledHigh(),
		sync:                 buses.CreateConnectorEnabledHigh(),
		vectorPull:           buses.CreateConnectorEnabledLow(),

		instructionSet: CreateInstructionSet(),
		addressModeSet: CreateAddressModesSet(),

		accumulatorRegister: 0x00,
		xRegister:           0x00,
		yRegister:           0x00,
		stackPointer:        0xFD,
		programCounter:      0xFFFC,

		// Set default value for flags B and I   (NV-BDIZC) = 0x34
		processorStatusRegister: StatusRegister(0b00110100),

		currentCycleIndex:        0,
		currentCycle:             readOpCode,
		instructionRegisterCarry: false,
		branchTaken:              false,
		currentOpCode:            0x00,

		instructionRegister: 0x00,
		dataRegister:        0x00,

		irqRequested:      false,
		previousNMIStatus: false,
		nmiRequested:      false,

		cyclesWithReset: 0,

		processorPaused:  false,
		processorStopped: false,
	}
}

/*
 ****************************************************
 * Buses
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

func (cpu *Cpu65C02S) MemoryLock() *buses.ConnectorEnabledLow {
	return cpu.memoryLock
}

func (cpu *Cpu65C02S) NonMaskableInterrupt() *buses.ConnectorEnabledLow {
	return cpu.nonMaskableInterrupt
}

func (cpu *Cpu65C02S) Reset() *buses.ConnectorEnabledLow {
	return cpu.reset
}

func (cpu *Cpu65C02S) SetOverflow() *buses.ConnectorEnabledLow {
	return cpu.setOverflow
}

func (cpu *Cpu65C02S) ReadWrite() *buses.ConnectorEnabledLow {
	return cpu.readWrite
}

func (cpu *Cpu65C02S) Ready() *buses.ConnectorEnabledHigh {
	return cpu.ready
}

func (cpu *Cpu65C02S) Sync() *buses.ConnectorEnabledHigh {
	return cpu.sync
}

func (cpu *Cpu65C02S) VectorPull() *buses.ConnectorEnabledLow {
	return cpu.vectorPull
}

/*
 ****************************************************
 * Timer Tick
 ****************************************************
 */

// As part of the emulation for every cycle we will execute 2 functions:
// First Tick for all emulated components and then PostTick.
// The parameter T represents the elapsed time between executions
func (cpu *Cpu65C02S) Tick(t uint64) {
	if cpu.processorPaused || cpu.processorStopped {
		cpu.Ready().SetEnable(false)
	}

	if !cpu.processorStopped {
		cpu.checkInterrupts()

		// If the processor was paused by a WAI instruction and an interrupt is requested
		// then it must be unpaused
		if cpu.processorPaused {
			if cpu.NonMaskableInterrupt().Enabled() || cpu.InterruptRequest().Enabled() {
				cpu.processorPaused = false
				cpu.Ready().SetEnable(true)
			}
		}

		if cpu.Ready().Enabled() {
			cpu.checkOverflowSet()
			cpu.setCycleSignaling()

			cpu.executeCycle(t)
		}
	}
}

// As part of the emulation for every cycle we will execute 2 functions:
// First Tick for all emulated components and then PostTick.
// The parameter T represents the elapsed time between executions
func (cpu *Cpu65C02S) PostTick(t uint64) {
	// Execute post action if CPU is not paused or stopped
	if cpu.Ready().Enabled() {
		cpu.currentCycle.postCycle(cpu)
		cpu.moveToNextCycle()
	}

	// If cpu is stopped only reset can remove it from that state
	cpu.checkReset()
}

// Sets the signals according to the current cycle being executed.
// Memory lock is enabled on last 3 cycles on RMW instructions and can be used to lock
// memory updates and guarantee consistency.
// Sync is enabled when processor is reading opcode
// Vector pull is enabled when the processor is pulling the interrupt vector (FFFA - FFFF)
// during an interrupt. It allows to change the response and have different handlers
// depending on what triggered the interrupt
func (cpu *Cpu65C02S) setCycleSignaling() {
	signaling := cpu.currentCycle.signaling

	cpu.memoryLock.SetEnable(signaling.memoryLock)
	cpu.sync.SetEnable(signaling.sync)
	cpu.vectorPull.SetEnable(signaling.vectorPull)
}

// If the set overflow flag pin is enabled then V flag is set.
// SOB was originally intended for fast input recognition because it can be tested with a branch instruction;
// however, it is not recommended in new system design
// and was seldom used in the past.
func (cpu *Cpu65C02S) checkOverflowSet() {
	if cpu.setOverflow.Enabled() {
		cpu.processorStatusRegister.SetFlag(OverflowFlagBit, true)
	}
}

// Check the interrupt lines and marks if an interrupt was requested. The interrupts are served once the current
// instruction completes.
func (cpu *Cpu65C02S) checkInterrupts() {
	if !cpu.irqRequested && cpu.InterruptRequest().Enabled() && !cpu.processorStatusRegister.Flag(IrqDisableFlagBit) {
		cpu.irqRequested = true
	}

	// NMI is edge enabled, only will trigger interrupt when it transitions from high to low.
	// If it's held low no further interrupts will be triggered
	if !cpu.previousNMIStatus && cpu.NonMaskableInterrupt().Enabled() {
		cpu.nmiRequested = true
	}
	cpu.previousNMIStatus = cpu.NonMaskableInterrupt().Enabled()
}

func (cpu *Cpu65C02S) checkReset() {
	if cpu.reset.Enabled() {
		cpu.cyclesWithReset++

		if cpu.cyclesWithReset >= 2 {
			cpu.currentAddressMode = cpu.addressModeSet.GetByName(AddressModeReset)
			cpu.currentCycle = interruptCycle

			cpu.setDefaultValues()
		}
	} else {
		cpu.cyclesWithReset = 0
	}
}

func (cpu *Cpu65C02S) setDefaultValues() {
	cpu.stackPointer = 0xFD

	// Set default value for flags B and I   (NV-BDIZC) = 0x34
	cpu.processorStatusRegister = StatusRegister(0b00110100)

	cpu.currentCycleIndex = 0
	cpu.instructionRegisterCarry = false
	cpu.branchTaken = false
	cpu.currentOpCode = 0x00

	cpu.instructionRegister = 0x00
	cpu.dataRegister = 0x00

	cpu.irqRequested = false
	cpu.previousNMIStatus = false
	cpu.nmiRequested = false

	cpu.cyclesWithReset = 0

	cpu.processorPaused = false
	cpu.processorStopped = false
}

func (cpu *Cpu65C02S) executeCycle(t uint64) {
	continueCycle := cpu.currentCycle.cycle(cpu)

	if !continueCycle {
		cpu.moveToNextCycle()
		cpu.Tick(t)
	}
}

// Called after each cycle to move the processor to the next cycle.
// CurrentAddressCycle always starts in 0 and points to the readOpCode
// cycle.
// The currentCycleIndex is increased and currentCycle values are updated
// until the current instruction has no more cycles.
// At that point the current cycle is reset to reaOpCode and the instruction
// and data registers are set to 0 in preparation for a new instruction.
func (cpu *Cpu65C02S) moveToNextCycle() {
	cpu.currentCycleIndex++

	if int(cpu.currentCycleIndex) >= cpu.currentAddressMode.Cycles() {
		cpu.currentCycleIndex = 0
		cpu.instructionRegister = 0x0000
		cpu.dataRegister = 0x00

		switch {
		case cpu.nmiRequested:
			cpu.currentAddressMode = cpu.addressModeSet.GetByName(AddressModeNMI)
			cpu.currentCycle = interruptCycle
		case cpu.irqRequested:
			cpu.currentAddressMode = cpu.addressModeSet.GetByName(AddressModeIRQ)
			cpu.currentCycle = interruptCycle
		default:
			cpu.currentCycle = readOpCode
		}
	} else {
		cpu.currentCycle = cpu.currentAddressMode.Cycle(cpu.currentCycleIndex - 1)
	}
}

/*
 ****************************************************
 * Operations
 ****************************************************
 */

// The 65C02S has optimized RMW instructions and they will take 6 cycles when
// no page boundary is crossed vs the typical 7 cycles of the regular 6502.
// This was fixed on most instructions except the ones below.
var alwaysExtra []uint8 = []uint8{
	0xFE, // INC,x
	0xDE, // DEC,x
	0x9D, // STA,x
}

// Adds the specified value to the instruction register.
// The instruction register is used as a temporary buffer to hold the operand
// of the curent opcode.
// If a carry happens, normally the processor needs an extra cycle to udpate
// the instruction register MSB.
// As part of the emulation the carry addition is performed here and the
// instructionRegisterCarry field is set to true.
// The extra cycle will be executed or skipped by looking at this flag.
func (cpu *Cpu65C02S) addToInstructionRegister(value uint16) {
	original := cpu.instructionRegister
	data := (original & 0xff) + value
	cpu.instructionRegister += value

	if data > 0xFF || slices.Contains(alwaysExtra, uint8(cpu.currentOpCode)) {
		cpu.instructionRegisterCarry = true
	}
}

// Adds the specified value to the instruction register LSB.
// Any carry will be ignored. This is used mostly in the zero page indexed
// address modes in where if the page boundary is reached it just
// "wraps around"
func (cpu *Cpu65C02S) addToInstructionRegisterLSB(value uint8) {
	cpu.instructionRegister = uint16(uint8(cpu.instructionRegister) + value)
}

// Sets the specified value on the instruction register LSB
func (cpu *Cpu65C02S) setInstructionRegisterLSB(value uint8) {
	cpu.instructionRegister = (cpu.instructionRegister & 0xFF00) + uint16(value)
}

// Sets the specified value on the instruction register MSB
func (cpu *Cpu65C02S) setInstructionRegisterMSB(value uint8) {
	cpu.instructionRegister = (cpu.instructionRegister & 0x00FF) + uint16(value)*0x100
}

// Configures the bus to read from the current stack pointer.
// In the 6502 family the stack pointer is located from 0x100 to 0x1FF.
// The effective address of the stack pointer is then formed by adding
// 0x100 to the stack pointer value.
func (cpu *Cpu65C02S) readFromStack() {
	cpu.setReadBus(uint16(cpu.stackPointer) + 0x100)
}

// Configures the bus to read from the current stack pointer.
// In the 6502 family the stack pointer is located from 0x100 to 0x1FF.
// The effective address of the stack pointer is then formed by adding
// 0x100 to the stack pointer value.
func (cpu *Cpu65C02S) writeToStack(value uint8) {
	cpu.setWriteBus(uint16(cpu.stackPointer)+0x100, value)
}

// Executes the instruction action
func (cpu *Cpu65C02S) performAction() {
	cpu.currentInstruction.Execute(cpu)
}

/*
 ****************************************************
 * Internal Bus Handling
 ****************************************************
 */

// Configures the processor to read from the specified address
func (cpu *Cpu65C02S) setReadBus(address uint16) {
	// TODO: Handle disconnected lines, Handle bus conflict

	if cpu.busEnable.Enabled() {
		cpu.readWrite.SetEnable(false)
		cpu.addressBus.Write(address)
	}
}

// Configures the processor to write the data parameter into the
// specified address.
func (cpu *Cpu65C02S) setWriteBus(address uint16, data uint8) {
	if cpu.busEnable.Enabled() {
		cpu.readWrite.SetEnable(true)
		cpu.addressBus.Write(address)
		cpu.dataBus.Write(data)
	}
}

/*
 ****************************************************
 * Public Methods
 ****************************************************
 */

// Returns data about the current instruction being executed by the processor
func (cpu *Cpu65C02S) GetCurrentInstruction() *CpuInstructionData {
	return cpu.currentInstruction
}

// Returns data about the address mode of the current instruction being
// executed by the processor.
func (cpu *Cpu65C02S) GetCurrentAddressMode() *AddressModeData {
	return cpu.currentAddressMode
}
