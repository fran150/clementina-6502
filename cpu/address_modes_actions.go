package cpu

type cycleAction func(cpu *Cpu65C02S) bool
type cyclePostAction func(cpu *Cpu65C02S)
type syncSignaling struct {
	memoryLock bool
	sync       bool
	vectorPull bool
}

type cycleActions struct {
	cycle     cycleAction
	postCycle cyclePostAction
	signaling syncSignaling
}

type sumOrigin uint8

const (
	fromXRegister sumOrigin = 0
	fromYRegister sumOrigin = 1
)

const NMI_VECTOR_LSB uint16 = 0xFFFA
const NMI_VECTOR_MSB uint16 = 0xFFFB

const RESET_VECTOR_LSB uint16 = 0xFFFC
const RESET_VECTOR_MSB uint16 = 0xFFFD

const IRQ_VECTOR_LSB uint16 = 0xFFFE
const IRQ_VECTOR_MSB uint16 = 0xFFFF

/**********************************************************************************************************
* Signaling status
*
* These values control the status for the sync signals for memory lock, opcode reading sync and vector pull
***********************************************************************************************************/

// This is the default status for most of the cycles
var defaultSignaling = syncSignaling{
	memoryLock: false,
	sync:       false,
	vectorPull: false,
}

// This signal indicates that the processor is reading an opcode
var opCodeSyncSignaling = syncSignaling{
	memoryLock: false,
	sync:       true,
	vectorPull: false,
}

// This is used to signal that memory updates must be locked to avoid inconsistencies
// in RMW operations. It is enabled when the processor is performing the read and
// write cycles
var memoryLockRMWSignaling = syncSignaling{
	memoryLock: true,
	sync:       false,
	vectorPull: false,
}

// This is used to signal that the processor is reading the interrupt vector
var vectorPullingSignaling = syncSignaling{
	memoryLock: false,
	sync:       false,
	vectorPull: true,
}

/**********************************************************************************************************
* Cycle Actions
***********************************************************************************************************/

// Sets the program counter value on the bus for reading. If incrementProgramCounter parameter is true,
// program counter is automatically increased to the next address.
func readFromProgramCounter(incrementProgramCounter bool) cycleAction {
	return func(cpu *Cpu65C02S) bool {
		cpu.setReadBus(cpu.programCounter)
		if incrementProgramCounter {
			cpu.programCounter++
		}

		return true
	}
}

// Sets the current value of the intruction register on the bus for reading.
func readFromInstructionRegister() cycleAction {
	return func(cpu *Cpu65C02S) bool {
		cpu.setReadBus(cpu.instructionRegister)

		return true
	}
}

// Reads from the current value in the bus. This does basically leave the address in the bus untouched.
// Just sets the R/W flag to read.
func readFromAddressInBus(performAction bool) cycleAction {
	return func(cpu *Cpu65C02S) bool {
		cpu.setReadBus(cpu.addressBus.Read())
		if performAction {
			cpu.performAction()
		}

		return true
	}
}

// Increment the current value in the bus by one and sets is to read. This is commonly used to read
// 2 bytes address from memory, for example in indirect address modes.
func readFromNextAddressInBus() cycleAction {
	return func(cpu *Cpu65C02S) bool {
		cpu.setReadBus(uint16(cpu.addressBus.Read() + 1))

		return true
	}
}

// Sets a specific address on the bus for reading. This is commonly used to read from IRQ, NMI or reset
// vectors.
func readFromAddress(address uint16) cycleAction {
	return func(cpu *Cpu65C02S) bool {
		cpu.setReadBus(address)

		return true
	}
}

// Sets the current value of the stack pointer on the bus for reading. Basically, the SP is in range
// 0x100 to 0x1FF, so effective address will be 0x100 + stack pointer value. Typically the stack pointer
// is moved up, but some cycles requires a repeated read. If increasedStackPointer parameter is true
// the stack pointer value is automatically incremented.
func readFromStackPointer(increaseStackPointer bool) cycleAction {
	return func(cpu *Cpu65C02S) bool {
		cpu.readFromStack()

		if increaseStackPointer {
			cpu.stackPointer++
		}

		return true
	}
}

// If the previous add to the instruction registers caused a carry it means that a page boundary was
// reached. In these cases the processor needs an extra cycle, on the 65C02S this is an extra read
// on the current bus value.
func extraCycleIfCarryInstructionRegister() cycleAction {
	return func(cpu *Cpu65C02S) bool {
		cpu.setReadBus(cpu.addressBus.Read())

		if cpu.instructionRegisterCarry {
			cpu.instructionRegisterCarry = false
			return true
		} else {
			return false
		}
	}
}

// On relative address modes, if the branch is taken the CPU requires an extra cycle to do the jump.
// This causes a read on the current program counter and it is used to update the program counter
// to branch value
func extraCycleIfBranchTaken() cycleAction {
	return func(cpu *Cpu65C02S) bool {
		if cpu.branchTaken {
			cpu.branchTaken = false

			cpu.setReadBus(cpu.programCounter)

			cpu.instructionRegister = cpu.programCounter
			cpu.addToInstructionRegister(uint16(cpu.dataRegister))

			return true
		} else {
			return false
		}
	}
}

// This is used to push the program counter MSB to the stack. It configures the bus to write the MSB of the PC
// into the stack pointer address and updates the stack pointer value accordingly
func writeProgramCounterMSBToStack() cycleAction {
	return func(cpu *Cpu65C02S) bool {
		counterMSB := cpu.programCounter & 0xFF00
		counterMSB = counterMSB >> 8
		cpu.writeToStack(uint8(counterMSB))
		cpu.stackPointer--
		return true
	}
}

// This is used to push the program counter LSB to the stack. It configures the bus to write the LSB of the PC
// into the stack pointer address and updates the stack pointer value accordingly
func writeProgramCounterLSBToStack() cycleAction {
	return func(cpu *Cpu65C02S) bool {
		counterLSB := cpu.programCounter & 0x00FF
		cpu.writeToStack(uint8(counterLSB))
		cpu.stackPointer--
		return true
	}
}

// This is used to push the processor status to the stack. It configures the bus to write the processor status
// into the stack pointer address and updates the stack pointer value accordingly
// The B flag is always set, but it's written in 0 to the stack when the processor stauts is persisted to the
// stack as part of a HW interrupt.
func writeProcessorStatusRegisterToStack(hardwareInterrupt bool) cycleAction {
	return func(cpu *Cpu65C02S) bool {
		value := uint8(cpu.processorStatusRegister)

		if hardwareInterrupt {
			// If it's a hardware interrupt disable B flag when pushing to the stack
			value &= 0xEF
		}

		cpu.writeToStack(value)
		cpu.stackPointer--
		return true
	}
}

/**********************************************************************************************************
* Cycle Post Actions
***********************************************************************************************************/

// Copies the value in the data bus as the current opcode. This is the instruction
// being processed
func intoOpCode() cyclePostAction {
	return func(cpu *Cpu65C02S) {
		cpu.currentOpCode = OpCode(cpu.dataBus.Read())
	}
}

// Copies the value in the data bus in the data register. The instruction action functions
// will pick the value from here to perfrom their operations.
// If the `performAction` parameter is true, the instruction action will be executed
func intoDataRegister(performAction bool) cyclePostAction {
	return func(cpu *Cpu65C02S) {
		cpu.dataRegister = cpu.dataBus.Read()
		if performAction {
			cpu.performAction()
		}
	}
}

// Copies the value in the data bus to the LSB of the instruction register.
// The instruction register is used as temporary buffer to store the address of
// the operand for certain instructions.
func intoInstructionRegisterLSB() cyclePostAction {
	return func(cpu *Cpu65C02S) {
		cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
	}
}

// Copies the value in the data bus to the MSB of the instruction register.
// The instruction register is used as temporary buffer to store the address of
// the operand for certain instructions.
// If the `performAction` parameter is true, the instruction action will be executed
func intoInstructionRegisterMSB(performAction bool) cyclePostAction {
	return func(cpu *Cpu65C02S) {
		cpu.setInstructionRegisterMSB(cpu.dataBus.Read())

		if performAction {
			cpu.performAction()
		}
	}
}

// Copies the value in the data bus to the processor status register.
// This is typically used when restoring the status from the stack after
// an interruption but it can be also triggered manual for exmaple with PLP
func intoStatusRegister() cyclePostAction {
	return func(cpu *Cpu65C02S) {
		cpu.processorStatusRegister = StatusRegister(cpu.dataBus.Read())
	}
}

// Adds the X or Y register to the instruction register LSB.
// Any carry is ignored. This is used mostly in the zero page indexed
// address modes in where if the page boundary is reached it just
// "wraps around"
func addToInstructionRegisterLSB(origin sumOrigin) cyclePostAction {
	return func(cpu *Cpu65C02S) {
		switch origin {
		case fromXRegister:
			cpu.addToInstructionRegisterLSB(cpu.xRegister)
		case fromYRegister:
			cpu.addToInstructionRegisterLSB(cpu.yRegister)
		}
	}
}

// Adds the X or Y register to the instruction address.
// Any carry in the addition will be added to the MSB of the instruction register.
// This will also cause the instructionRegisterCarry value to be set to true.
// In most cases this means that the CPU will require an extra cycle to update the
// value of the MSB. In the emulation this internally already happens in this cycle,
// the bus is set to read from the unchanged value as the extra cycle causes a read
// from this value.
func addToInstructionRegister(origin sumOrigin) cyclePostAction {
	return func(cpu *Cpu65C02S) {
		cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
		cpu.setReadBus(cpu.instructionRegister)

		switch origin {
		case fromXRegister:
			cpu.addToInstructionRegister(uint16(cpu.xRegister))
		case fromYRegister:
			cpu.addToInstructionRegister(uint16(cpu.yRegister))
		}
	}
}

// Moves the instruction register to the program counter.
// This will cause the execution to jump to the instruction register address.
// This is commonly used for address modes that jump to subroutines,
// branches (for example BCC, JSR, BRK) or handle interrupts (RTI)
// Because the same function is used for 1 byte or 2 byte operands the parameter
// "setInstructionRegisterMSB" can be used to read the MSB of the 2nd byte from
// the bus.
func moveInstructionRegisterToProgramCounter(setInstructionRegisterMSB bool) cyclePostAction {
	return func(cpu *Cpu65C02S) {
		if setInstructionRegisterMSB {
			cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
		}
		cpu.programCounter = cpu.instructionRegister
		cpu.setReadBus(cpu.programCounter)
	}
}

func moveInstructionRegisterToProgramCounterIfNotCarry() cyclePostAction {
	return func(cpu *Cpu65C02S) {
		if !cpu.instructionRegisterCarry {
			cpu.programCounter = cpu.instructionRegister
			cpu.setReadBus(cpu.programCounter)
		}
	}
}

// Used when we don´t need to do anything in the post cycle phase.
func doNothing() cyclePostAction {
	return func(cpu *Cpu65C02S) {
	}
}

/**********************************************************************************************************
* Address Modes Cycles
***********************************************************************************************************/

// This is always the first cycle after an opcode execution is completed.
var readOpCode cycleActions = cycleActions{
	cycle:     readFromProgramCounter(true),
	postCycle: intoOpCode(),
	signaling: opCodeSyncSignaling,
}

/**********************************
* Implied / Accumulator / Immediate
***********************************/

var addressModeImplicitOrAccumulatorActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(false),
		postCycle: intoDataRegister(true),
		signaling: defaultSignaling,
	},
}

var addressModeImmediateActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoDataRegister(true),
		signaling: defaultSignaling,
	},
}

/**********************************
* Absolute
***********************************/

var addressModeAbsoluteJumpActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterMSB(true),
		signaling: defaultSignaling,
	},
}

var addressModeAbsoluteActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterMSB(false),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
		signaling: defaultSignaling,
	},
}

var addressModeAbsoluteRMWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterMSB(false),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(false),
		signaling: memoryLockRMWSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(false),
		signaling: memoryLockRMWSignaling,
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
		signaling: memoryLockRMWSignaling,
	},
}

var addressModeAbsoluteWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterMSB(false),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
}

/**********************************
* Zero Page
***********************************/

var addressModeZeroPageActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
		signaling: defaultSignaling,
	},
}

var addressModeZeroPageRMWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(false),
		signaling: memoryLockRMWSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(false),
		signaling: memoryLockRMWSignaling,
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
		signaling: memoryLockRMWSignaling,
	},
}

var addressModeZeroPageWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
}

/**********************************
* Zero Page Indexed
***********************************/

var addressModeZeroPageXActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: addToInstructionRegisterLSB(fromXRegister),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
		signaling: defaultSignaling,
	},
}

var addressModeZeroPageXRMWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: addToInstructionRegisterLSB(fromXRegister),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(false),
		signaling: memoryLockRMWSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(false),
		signaling: memoryLockRMWSignaling,
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
		signaling: memoryLockRMWSignaling,
	},
}

var addressModeZeroPageXWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: addToInstructionRegisterLSB(fromXRegister),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
}

var addressModeZeroPageYActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: addToInstructionRegisterLSB(fromYRegister),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
		signaling: defaultSignaling,
	},
}

var addressModeZeroPageYWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: addToInstructionRegisterLSB(fromYRegister),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
}

/**********************************
* Absolute Indexed Addressing
***********************************/
var addressModeAbsoluteXActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: addToInstructionRegister(fromXRegister),
		signaling: defaultSignaling,
	},
	{
		cycle:     extraCycleIfCarryInstructionRegister(),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
		signaling: defaultSignaling,
	},
}

var addressModeAbsoluteXRMWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: addToInstructionRegister(fromXRegister),
		signaling: defaultSignaling,
	},
	{
		cycle:     extraCycleIfCarryInstructionRegister(),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(false),
		signaling: memoryLockRMWSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(false),
		signaling: memoryLockRMWSignaling,
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
		signaling: memoryLockRMWSignaling,
	},
}

var addressModeAbsoluteXWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: addToInstructionRegister(fromXRegister),
		signaling: defaultSignaling,
	},
	{
		cycle:     extraCycleIfCarryInstructionRegister(),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
}

var addressModeAbsoluteYActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: addToInstructionRegister(fromYRegister),
		signaling: defaultSignaling,
	},
	{
		cycle:     extraCycleIfCarryInstructionRegister(),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
		signaling: defaultSignaling,
	},
}

var addressModeAbsoluteYWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: addToInstructionRegister(fromYRegister),
		signaling: defaultSignaling,
	},
	{
		cycle:     extraCycleIfCarryInstructionRegister(),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
}

/**********************************
* Relative
***********************************/
var addressModeRelativeActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoDataRegister(true),
		signaling: defaultSignaling,
	},
	{
		cycle:     extraCycleIfBranchTaken(),
		postCycle: moveInstructionRegisterToProgramCounter(false),
		signaling: defaultSignaling,
	},
	{
		cycle:     extraCycleIfCarryInstructionRegister(),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
}

var addressModeRelativeExtendedActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoDataRegister(false),
		signaling: defaultSignaling,
	},
	{
		cycle:     extraCycleIfBranchTaken(),
		postCycle: moveInstructionRegisterToProgramCounterIfNotCarry(),
		signaling: defaultSignaling,
	},
	{
		cycle:     extraCycleIfCarryInstructionRegister(),
		postCycle: moveInstructionRegisterToProgramCounter(false),
		signaling: defaultSignaling,
	},
}

/**********************************
* Indexed Indirect X
***********************************/
var addressModeZeroPageIndexedIndirectXActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: addToInstructionRegisterLSB(fromXRegister),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromNextAddressInBus(),
		postCycle: intoInstructionRegisterMSB(false),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
		signaling: defaultSignaling,
	},
}

var addressModeZeroPageIndexedIndirectXWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: addToInstructionRegisterLSB(fromXRegister),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromNextAddressInBus(),
		postCycle: intoInstructionRegisterMSB(false),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
}

/**********************************
* Indirect Indexed
***********************************/

var addressModeZeroPageIndirectIndexedYActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{ // TODO: If page boundary crossed might require extra cycle (couldn't find documentation might need to check with real hardware)
		cycle:     readFromNextAddressInBus(),
		postCycle: addToInstructionRegister(fromYRegister),
		signaling: defaultSignaling,
	},
	{
		cycle:     extraCycleIfCarryInstructionRegister(),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
		signaling: defaultSignaling,
	},
}

var addressModeZeroPageIndirectIndexedYWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromNextAddressInBus(),
		postCycle: addToInstructionRegister(fromYRegister),
		signaling: defaultSignaling,
	},
	{
		cycle:     extraCycleIfCarryInstructionRegister(),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
}

/**********************************
* Indirect
***********************************/

var addressModeIndirectActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterMSB(false),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromNextAddressInBus(),
		postCycle: intoInstructionRegisterMSB(false),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
		signaling: defaultSignaling,
	},
}

/**********************************
* Zero Page Indirect
***********************************/

var addressModeIndirectZeroPageActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromNextAddressInBus(),
		postCycle: intoInstructionRegisterMSB(false),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
		signaling: defaultSignaling,
	},
}

var addressModeIndirectZeroPageWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromNextAddressInBus(),
		postCycle: intoInstructionRegisterMSB(false),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
}

/**********************************
* Absolute Indexed Indirect
***********************************/

var addressModeAbsoluteIndexedIndirectActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: addToInstructionRegister(fromXRegister),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromNextAddressInBus(),
		postCycle: intoInstructionRegisterMSB(false),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
		signaling: defaultSignaling,
	},
}

/**********************************
* Stack pointer instructions
***********************************/
var addressModePushStackActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(false),
		postCycle: intoDataRegister(false),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
}

var addressModePullStackActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(false),
		postCycle: intoDataRegister(false),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromStackPointer(true),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromStackPointer(false),
		postCycle: intoDataRegister(true),
		signaling: defaultSignaling,
	},
}

var addressModeBreakActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoDataRegister(true),
		signaling: defaultSignaling,
	},
	{
		cycle:     writeProgramCounterMSBToStack(),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
	{
		cycle:     writeProgramCounterLSBToStack(),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
	{
		cycle:     writeProcessorStatusRegisterToStack(false),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromAddress(IRQ_VECTOR_LSB),
		postCycle: intoInstructionRegisterLSB(),
		signaling: vectorPullingSignaling,
	},
	{
		cycle:     readFromAddress(IRQ_VECTOR_MSB),
		postCycle: moveInstructionRegisterToProgramCounter(true),
		signaling: vectorPullingSignaling,
	},
}

var addressModeReturnFromInterruptActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(false),
		postCycle: intoDataRegister(false),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromStackPointer(true),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromStackPointer(true),
		postCycle: intoStatusRegister(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromStackPointer(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromStackPointer(true),
		postCycle: moveInstructionRegisterToProgramCounter(true),
		signaling: defaultSignaling,
	},
}

var addressModeJumpToSubroutineActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromStackPointer(false),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
	{
		cycle:     writeProgramCounterMSBToStack(),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
	{
		cycle:     writeProgramCounterLSBToStack(),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: moveInstructionRegisterToProgramCounter(true),
		signaling: defaultSignaling,
	},
}

var addressModeReturnFromSubroutineActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(false),
		postCycle: intoDataRegister(false),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromStackPointer(true),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromStackPointer(true),
		postCycle: intoInstructionRegisterLSB(),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromStackPointer(true),
		postCycle: moveInstructionRegisterToProgramCounter(true),
		signaling: defaultSignaling,
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: doNothing(),
		signaling: defaultSignaling,
	},
}
