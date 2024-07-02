package cpu

type cycleAction func(cpu *Cpu65C02S) bool
type cyclePostAction func(cpu *Cpu65C02S)

type cycleActions struct {
	cycle     cycleAction
	postCycle cyclePostAction
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

// TODO: Review this documentation
// This is used to push the program counter MSB to the stack. It sets the MSB value of the PC into the
// current value of the stack pointer on the bus for write and updates the stack pointer value accordingly
func writeProgramCounterMSBToStack() cycleAction {
	return func(cpu *Cpu65C02S) bool {
		counterMSB := cpu.programCounter & 0xFF00
		counterMSB = counterMSB >> 8
		cpu.writeToStack(uint8(counterMSB))
		cpu.stackPointer--
		return true
	}
}

// This is used to push the program counter MSB to the stack. It sets the LSB value of the PC into the
// current value of the stack pointer on the bus for write and updates the stack pointer value accordingly
func writeProgramCounterLSBToStack() cycleAction {
	return func(cpu *Cpu65C02S) bool {
		counterLSB := cpu.programCounter & 0x00FF
		cpu.writeToStack(uint8(counterLSB))
		cpu.stackPointer--
		return true
	}
}

// This is used to push the processor status to the stack. It sets the processor status value into the
// current value of the stack pointer on the bus for write and updates the stack pointer value accordingly
func writeProcessorStatusRegisterToStack() cycleAction {
	return func(cpu *Cpu65C02S) bool {
		cpu.writeToStack(uint8(cpu.processorStatusRegister))
		cpu.stackPointer--
		return true
	}
}

/**********************************************************************************************************
* Cycle Post Actions
***********************************************************************************************************/

func intoOpCode() cyclePostAction {
	return func(cpu *Cpu65C02S) {
		cpu.currentOpCode = OpCode(cpu.dataBus.Read())
	}
}

func intoDataRegister(performAction bool) cyclePostAction {
	return func(cpu *Cpu65C02S) {
		cpu.dataRegister = cpu.dataBus.Read()
		if performAction {
			cpu.performAction()
		}
	}
}

func intoInstructionRegisterLSB() cyclePostAction {
	return func(cpu *Cpu65C02S) {
		cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
	}
}

func intoInstructionRegisterMSB(performAction bool) cyclePostAction {
	return func(cpu *Cpu65C02S) {
		cpu.setInstructionRegisterMSB(cpu.dataBus.Read())

		if performAction {
			cpu.performAction()
		}
	}
}

func intoStatusRegister() cyclePostAction {
	return func(cpu *Cpu65C02S) {
		cpu.processorStatusRegister = StatusRegister(cpu.dataBus.Read())
	}
}

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

func addToInstructionRegisterMSB(origin sumOrigin, setInstructionRegisterMSB bool, setReadBus bool) cyclePostAction {
	return func(cpu *Cpu65C02S) {
		if setInstructionRegisterMSB {
			cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
		}

		if setReadBus {
			cpu.setReadBus(cpu.instructionRegister)
		}

		switch origin {
		case fromXRegister:
			cpu.addToInstructionRegister(uint16(cpu.xRegister))
		case fromYRegister:
			cpu.addToInstructionRegister(uint16(cpu.yRegister))
		}
	}
}

func moveInstructionRegisterToProgramCounter(setInstructionRegisterMSB bool) cyclePostAction {
	return func(cpu *Cpu65C02S) {
		if setInstructionRegisterMSB {
			cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
		}
		cpu.programCounter = cpu.instructionRegister
		cpu.setReadBus(cpu.programCounter)
	}
}

func doNothing() cyclePostAction {
	return func(cpu *Cpu65C02S) {
	}
}

/**********************************************************************************************************
* Address Modes Cycles
***********************************************************************************************************/

var readOpCode cycleActions = cycleActions{
	cycle:     readFromProgramCounter(true),
	postCycle: intoOpCode(),
}

/**********************************
* Implied / Accumulator / Immediate
***********************************/

var addressModeImplicitOrAccumulatorActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(false),
		postCycle: intoDataRegister(true),
	},
}

var addressModeImmediateActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoDataRegister(true),
	},
}

/**********************************
* Absolute
***********************************/

var addressModeAbsoluteJumpActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterMSB(true),
	},
}

var addressModeAbsoluteActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterMSB(false),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
	},
}

var addressModeAbsoluteRMWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterMSB(false),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(false),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(false),
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
	},
}

var addressModeAbsoluteWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterMSB(false),
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
	},
}

/**********************************
* Zero Page
***********************************/

var addressModeZeroPageActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
	},
}

var addressModeZeroPageRMWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(false),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(false),
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
	},
}

var addressModeZeroPageWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
	},
}

/**********************************
* Zero Page Indexed
***********************************/

var addressModeZeroPageXActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: addToInstructionRegisterLSB(fromXRegister),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
	},
}

var addressModeZeroPageXRMWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: addToInstructionRegisterLSB(fromXRegister),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(false),
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
	},
}

var addressModeZeroPageXWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: addToInstructionRegisterLSB(fromXRegister),
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
	},
}

var addressModeZeroPageYActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: addToInstructionRegisterLSB(fromYRegister),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
	},
}

/**********************************
* Absolute Indexed Addressing
***********************************/
var addressModeAbsoluteXActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: addToInstructionRegisterMSB(fromXRegister, true, true),
	},
	{
		cycle:     extraCycleIfCarryInstructionRegister(),
		postCycle: doNothing(),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
	},
}

var addressModeAbsoluteXRMWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: addToInstructionRegisterMSB(fromXRegister, true, true),
	},
	{
		cycle:     extraCycleIfCarryInstructionRegister(),
		postCycle: doNothing(),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(false),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(false),
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
	},
}

var addressModeAbsoluteXWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: addToInstructionRegisterMSB(fromXRegister, true, true),
	},
	{
		cycle:     extraCycleIfCarryInstructionRegister(),
		postCycle: doNothing(),
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
	},
}

var addressModeAbsoluteYActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: addToInstructionRegisterMSB(fromXRegister, true, true),
	},
	{
		cycle:     extraCycleIfCarryInstructionRegister(),
		postCycle: doNothing(),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
	},
}

var addressModeAbsoluteYWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: addToInstructionRegisterMSB(fromYRegister, true, true),
	},
	{
		cycle:     extraCycleIfCarryInstructionRegister(),
		postCycle: doNothing(),
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
	},
}

/**********************************
* Relative
***********************************/
var addressModeRelativeActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoDataRegister(true),
	},
	{
		cycle:     extraCycleIfBranchTaken(),
		postCycle: moveInstructionRegisterToProgramCounter(false),
	},
	{
		cycle:     extraCycleIfCarryInstructionRegister(),
		postCycle: doNothing(),
	},
}

/**********************************
* Indexed Indirect X
***********************************/
var addressModeZeroPageIndexedIndirectXActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: addToInstructionRegisterLSB(fromXRegister),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromNextAddressInBus(),
		postCycle: intoInstructionRegisterMSB(false),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
	},
}

var addressModeZeroPageIndexedIndirectXWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: addToInstructionRegisterLSB(fromXRegister),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromNextAddressInBus(),
		postCycle: intoInstructionRegisterMSB(false),
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
	},
}

/**********************************
* Indirect Indexed
***********************************/

var addressModeZeroPageIndirectIndexedYActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoInstructionRegisterLSB(),
	},
	{ // TODO: If page boundary crossed might require extra cycle (couldn't find documentation might need to check with real hardware)
		cycle:     readFromNextAddressInBus(),
		postCycle: addToInstructionRegisterMSB(fromYRegister, true, true),
	},
	{
		cycle:     extraCycleIfCarryInstructionRegister(),
		postCycle: doNothing(),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
	},
}

var addressModeZeroPageIndirectIndexedYWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromNextAddressInBus(),
		postCycle: addToInstructionRegisterMSB(fromYRegister, true, true),
	},
	{
		cycle:     extraCycleIfCarryInstructionRegister(),
		postCycle: doNothing(),
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
	},
}

/**********************************
* Indirect
***********************************/

var addressModeIndirectActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterMSB(false),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromNextAddressInBus(),
		postCycle: intoInstructionRegisterMSB(false),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
	},
}

/**********************************
* Zero Page Indirect
***********************************/

var addressModeIndirectZeroPageActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromNextAddressInBus(),
		postCycle: intoInstructionRegisterMSB(false),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
	},
}

var addressModeIndirectZeroPageWActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromNextAddressInBus(),
		postCycle: intoInstructionRegisterMSB(false),
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
	},
}

/**********************************
* Absolute Indexed Indirect
***********************************/

var addressModeAbsoluteIndexedIndirectActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: addToInstructionRegisterMSB(fromXRegister, true, false),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromNextAddressInBus(),
		postCycle: intoInstructionRegisterMSB(false),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
	},
}

/**********************************
* Stack pointer instructions
***********************************/
var addressModePushStackActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(false),
		postCycle: intoDataRegister(false),
	},
	{
		cycle:     readFromAddressInBus(true),
		postCycle: doNothing(),
	},
}

var addressModePullStackActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(false),
		postCycle: intoDataRegister(false),
	},
	{
		cycle:     readFromStackPointer(true),
		postCycle: doNothing(),
	},
	{
		cycle:     readFromStackPointer(false),
		postCycle: intoDataRegister(true),
	},
}

var addressModeBreakActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoDataRegister(false),
	},
	{
		cycle:     writeProgramCounterMSBToStack(),
		postCycle: doNothing(),
	},
	{
		cycle:     writeProgramCounterLSBToStack(),
		postCycle: doNothing(),
	},
	{
		cycle:     writeProcessorStatusRegisterToStack(),
		postCycle: doNothing(),
	},
	{
		cycle:     readFromAddress(IRQ_VECTOR_LSB),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromAddress(IRQ_VECTOR_MSB),
		postCycle: moveInstructionRegisterToProgramCounter(true),
	},
}

var addressModeReturnFromInterruptActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(false),
		postCycle: intoDataRegister(false),
	},
	{
		cycle:     readFromStackPointer(true),
		postCycle: doNothing(),
	},
	{
		cycle:     readFromStackPointer(true),
		postCycle: intoStatusRegister(),
	},
	{
		cycle:     readFromStackPointer(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromStackPointer(true),
		postCycle: moveInstructionRegisterToProgramCounter(true),
	},
}

var addressModeJumpToSubroutineActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromStackPointer(false),
		postCycle: doNothing(),
	},
	{
		cycle:     writeProgramCounterMSBToStack(),
		postCycle: doNothing(),
	},
	{
		cycle:     writeProgramCounterLSBToStack(),
		postCycle: doNothing(),
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: moveInstructionRegisterToProgramCounter(true),
	},
}

var addressModeReturnFromSubroutineActions []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(false),
		postCycle: intoDataRegister(false),
	},
	{
		cycle:     readFromStackPointer(true),
		postCycle: doNothing(),
	},
	{
		cycle:     readFromStackPointer(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromStackPointer(true),
		postCycle: moveInstructionRegisterToProgramCounter(true),
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: doNothing(),
	},
}
