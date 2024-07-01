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

func readFromProgramCounter(incrementProgramCounter bool) cycleAction {
	return func(cpu *Cpu65C02S) bool {
		cpu.setReadBus(cpu.programCounter)
		if incrementProgramCounter {
			cpu.programCounter++
		}

		return true
	}
}

func readFromInstructionRegister() cycleAction {
	return func(cpu *Cpu65C02S) bool {
		cpu.setReadBus(cpu.instructionRegister)

		return true
	}
}

func readFromBus(performAction bool) cycleAction {
	return func(cpu *Cpu65C02S) bool {
		if performAction {
			cpu.performAction()
		}

		return true
	}
}

func readFromAddress(address uint16) cycleAction {
	return func(cpu *Cpu65C02S) bool {
		cpu.setReadBus(address)

		return true
	}
}

func readFromNextAddressInBus() cycleAction {
	return func(cpu *Cpu65C02S) bool {
		cpu.setReadBus(uint16(cpu.addressBus.Read() + 1))

		return true
	}
}

func readFromStackPointer(increaseStackPointer bool) cycleAction {
	return func(cpu *Cpu65C02S) bool {
		cpu.readFromStack()

		if increaseStackPointer {
			cpu.stackPointer++
		}

		return true
	}
}

func extraCycleIfCarryInstructionRegister() cycleAction {
	return func(cpu *Cpu65C02S) bool {
		if cpu.instructionRegisterCarry {
			cpu.instructionRegisterCarry = false
			return true
		} else {
			return false
		}
	}
}

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

func writeProgramCounterMSBToStack() cycleAction {
	return func(cpu *Cpu65C02S) bool {
		counterMSB := cpu.programCounter & 0xFF00
		counterMSB = counterMSB >> 8
		cpu.writeToStack(uint8(counterMSB))
		cpu.stackPointer--
		return true
	}
}

func writeProgramCounterLSBToStack() cycleAction {
	return func(cpu *Cpu65C02S) bool {
		counterLSB := cpu.programCounter & 0x00FF
		cpu.writeToStack(uint8(counterLSB))
		cpu.stackPointer--
		return true
	}
}

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

var actionImplicitOrAccumulator []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(false),
		postCycle: intoDataRegister(true),
	},
}

var actionImmediate []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoDataRegister(true),
	},
}

/**********************************
* Absolute
***********************************/

var actionAbsoluteJump []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterMSB(true),
	},
}

var actionAbsolute []cycleActions = []cycleActions{
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

var actionAbsoluteRMW []cycleActions = []cycleActions{
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
		cycle:     readFromBus(true),
		postCycle: doNothing(),
	},
}

var actionAbsoluteWrite []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterMSB(false),
	},
	{
		cycle:     readFromBus(true),
		postCycle: doNothing(),
	},
}

/**********************************
* Zero Page
***********************************/

var actionZeroPage []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: intoDataRegister(true),
	},
}

var actionZeroPageRMW []cycleActions = []cycleActions{
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
		cycle:     readFromBus(true),
		postCycle: doNothing(),
	},
}

var actionZeroPageWrite []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromBus(true),
		postCycle: doNothing(),
	},
}

/**********************************
* Zero Page Indexed
***********************************/

var actionZeroPageX []cycleActions = []cycleActions{
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

var actionZeroPageXRMW []cycleActions = []cycleActions{
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
		cycle:     readFromBus(true),
		postCycle: doNothing(),
	},
}

var actionZeroPageXWrite []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(true),
		postCycle: intoInstructionRegisterLSB(),
	},
	{
		cycle:     readFromInstructionRegister(),
		postCycle: addToInstructionRegisterLSB(fromXRegister),
	},
	{
		cycle:     readFromBus(true),
		postCycle: doNothing(),
	},
}

var actionZeroPageY []cycleActions = []cycleActions{
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
var actionAbsoluteX []cycleActions = []cycleActions{
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

var actionAbsoluteXRMW []cycleActions = []cycleActions{
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
		cycle:     readFromBus(true),
		postCycle: doNothing(),
	},
}

var actionAbsoluteXWrite []cycleActions = []cycleActions{
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
		cycle:     readFromBus(true),
		postCycle: doNothing(),
	},
}

var actionAbsoluteY []cycleActions = []cycleActions{
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

var actionAbsoluteYWrite []cycleActions = []cycleActions{
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
		cycle:     readFromBus(true),
		postCycle: doNothing(),
	},
}

/**********************************
* Relative
***********************************/
var actionRelative []cycleActions = []cycleActions{
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
var actionIndexedIndirectX []cycleActions = []cycleActions{
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

var actionIndexedIndirectXW []cycleActions = []cycleActions{
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
		cycle:     readFromBus(true),
		postCycle: doNothing(),
	},
}

/**********************************
* Indirect Indexed
***********************************/

var actionIndirectIndexedY []cycleActions = []cycleActions{
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

var actionIndirectIndexedYW []cycleActions = []cycleActions{
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
		cycle:     readFromBus(true),
		postCycle: doNothing(),
	},
}

/**********************************
* Indirect
***********************************/

var actionIndirect []cycleActions = []cycleActions{
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

var actionZeroPageIndirect []cycleActions = []cycleActions{
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

var actionZeroPageIndirectWrite []cycleActions = []cycleActions{
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
		cycle:     readFromBus(true),
		postCycle: doNothing(),
	},
}

/**********************************
* Absolute Indexed Indirect
***********************************/

var actionAbsoluteIndexedIndirectX []cycleActions = []cycleActions{
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
var actionPushStack []cycleActions = []cycleActions{
	{
		cycle:     readFromProgramCounter(false),
		postCycle: intoDataRegister(false),
	},
	{
		cycle:     readFromBus(true),
		postCycle: doNothing(),
	},
}

var actionPullStack []cycleActions = []cycleActions{
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

var actionBreak []cycleActions = []cycleActions{
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

var actionReturnFromInterrupt []cycleActions = []cycleActions{
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

var actionJumpToSubroutine []cycleActions = []cycleActions{
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

var actionReturnFromSubroutine []cycleActions = []cycleActions{
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
