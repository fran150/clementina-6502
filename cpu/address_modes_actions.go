package cpu

type cycleAction func(cpu *Cpu65C02S) func()

type sumOrigin uint8

const (
	doNotSum         sumOrigin = 0
	fromXRegister    sumOrigin = 1
	fromYRegister    sumOrigin = 2
	fromDataRegister sumOrigin = 3
)

var readOpCode cycleAction = func(cpu *Cpu65C02S) func() {
	// read next instruction byte (and throw it away)
	cpu.setReadBus(cpu.programCounter)
	cpu.programCounter++

	return func() {
		cpu.currentOpCode = OpCode(cpu.dataBus.Read())
	}
}

func readNextInstructionAndThrowAway() cycleAction {
	return func(cpu *Cpu65C02S) func() {
		// read next instruction byte (and throw it away)
		cpu.setReadBus(cpu.programCounter)

		return func() {
			cpu.performAction()
		}
	}
}

func readFromProgramCounter(performAction bool) cycleAction {
	return func(cpu *Cpu65C02S) func() {
		// fetch value, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.dataRegister = cpu.dataBus.Read()
			if performAction {
				cpu.performAction()
			}
		}
	}
}

func readInstructionRegisterLSB() cycleAction {
	return func(cpu *Cpu65C02S) func() {
		// fetch low address byte, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	}
}

func readInstructionRegisterMSB() cycleAction {
	return func(cpu *Cpu65C02S) func() {
		// fetch high address byte to PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
		}
	}
}

func readFromInstructionRegister(performAction bool) cycleAction {
	return func(cpu *Cpu65C02S) func() {
		// read from effective address
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.dataRegister = cpu.dataBus.Read()

			if performAction {
				cpu.performAction()
			}
		}
	}
}

func addToInstructionRegisterLSB(origin sumOrigin) cycleAction {
	return func(cpu *Cpu65C02S) func() {
		// read from address, add index register to it
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			switch origin {
			case fromXRegister:
				cpu.addToInstructionRegisterLSB(cpu.xRegister)
			case fromYRegister:
				cpu.addToInstructionRegisterLSB(cpu.yRegister)
			case fromDataRegister:
				cpu.addToInstructionRegisterLSB(cpu.dataRegister)
			}

		}
	}
}

func addToInstructionRegister(origin sumOrigin, setInstructionRegisterMSB bool, setReadBus bool) cycleAction {
	return func(cpu *Cpu65C02S) func() {
		// fetch high byte of address, add index register to low address byte, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
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
			case fromDataRegister:
				cpu.addToInstructionRegister(uint16(cpu.dataRegister))
			}
		}
	}
}

func performActionOnTick() cycleAction {
	return func(cpu *Cpu65C02S) func() {
		// Write the new value to effective address
		cpu.performAction()

		return func() {
		}
	}
}

func performInstructionRegisterCarryCycle() cycleAction {
	return func(cpu *Cpu65C02S) func() {
		if cpu.instructionRegisterCarry {
			cpu.instructionRegisterCarry = false
			// Previous cycle already set the address in the bus
			return func() {
			}
		} else {
			return nil
		}
	}
}

/**********************************
* Implied / Accumulator / Immediate
***********************************/

var actionImplicitOrAccumulator []cycleAction = []cycleAction{
	readNextInstructionAndThrowAway(),
}

var actionImmediate []cycleAction = []cycleAction{
	readFromProgramCounter(true),
}

/**********************************
* Absolute
***********************************/

var actionAbsoluteJump []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	func(cpu *Cpu65C02S) func() {
		// fetch high address byte to PC
		cpu.setReadBus(cpu.programCounter)

		return func() {
			cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
			cpu.performAction()
		}
	},
}

var actionAbsolute []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	readInstructionRegisterMSB(),
	readFromInstructionRegister(true),
}

var actionAbsoluteRMW []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	readInstructionRegisterMSB(),
	readFromInstructionRegister(false),
	readFromInstructionRegister(false),
	performActionOnTick(),
}

var actionAbsoluteWrite []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	readInstructionRegisterMSB(),
	performActionOnTick(),
}

/**********************************
* Zero Page
***********************************/

var actionZeroPage []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	readFromInstructionRegister(true),
}

var actionZeroPageRMW []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	readFromInstructionRegister(false),
	readFromInstructionRegister(false),
	performActionOnTick(),
}

var actionZeroPageWrite []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	performActionOnTick(),
}

/**********************************
* Zero Page Indexed
***********************************/

var actionZeroPageX []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	addToInstructionRegisterLSB(fromXRegister),
	readFromInstructionRegister(true),
}

var actionZeroPageXRMW []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	addToInstructionRegisterLSB(fromXRegister),
	readFromInstructionRegister(false),
	performActionOnTick(),
}

var actionZeroPageXWrite []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	addToInstructionRegisterLSB(fromXRegister),
	performActionOnTick(),
}

var actionZeroPageY []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	addToInstructionRegisterLSB(fromYRegister),
	readFromInstructionRegister(true),
}

/**********************************
* Absolute Indexed Addressing
***********************************/
var actionAbsoluteX []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	addToInstructionRegister(fromXRegister, true, true),
	performInstructionRegisterCarryCycle(),
	readFromInstructionRegister(true),
}

var actionAbsoluteXRMW []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	addToInstructionRegister(fromXRegister, true, true),
	performInstructionRegisterCarryCycle(),
	readFromInstructionRegister(false),
	readFromInstructionRegister(false),
	performActionOnTick(),
}

var actionAbsoluteXWrite []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	addToInstructionRegister(fromXRegister, true, true),
	performInstructionRegisterCarryCycle(),
	performActionOnTick(),
}

var actionAbsoluteY []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	addToInstructionRegister(fromXRegister, true, true),
	performInstructionRegisterCarryCycle(),
	readFromInstructionRegister(true),
}

var actionAbsoluteYWrite []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	addToInstructionRegister(fromYRegister, true, true),
	performInstructionRegisterCarryCycle(),
	performActionOnTick(),
}

/**********************************
* Relative
***********************************/
var actionRelative []cycleAction = []cycleAction{
	func(cpu *Cpu65C02S) func() {
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.dataRegister = cpu.dataBus.Read()
			cpu.performAction()
		}
	},
	func(cpu *Cpu65C02S) func() {
		if cpu.branchTaken {
			cpu.branchTaken = false
			cpu.setReadBus(cpu.programCounter)
			cpu.instructionRegister = cpu.programCounter
			cpu.addToInstructionRegister(uint16(cpu.dataRegister))
			return func() {
				cpu.programCounter = cpu.instructionRegister
				cpu.setReadBus(cpu.programCounter)
			}
		} else {
			return nil
		}
	},
	performInstructionRegisterCarryCycle(),
}

/**********************************
* Indexed Indirect X
***********************************/
var actionIndexedIndirectX []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	func(cpu *Cpu65C02S) func() {
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.addToInstructionRegisterLSB(cpu.xRegister)
		}
	},
	func(cpu *Cpu65C02S) func() {
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	},
	func(cpu *Cpu65C02S) func() {
		cpu.setReadBus(uint16(cpu.addressBus.Read() + 1))

		return func() {
			cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
		}
	},
	readFromInstructionRegister(true),
}

var actionIndexedIndirectXW []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	func(cpu *Cpu65C02S) func() {
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.addToInstructionRegisterLSB(cpu.xRegister)
		}
	},
	func(cpu *Cpu65C02S) func() {
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	},
	func(cpu *Cpu65C02S) func() {
		cpu.setReadBus(uint16(cpu.addressBus.Read() + 1))

		return func() {
			cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
		}
	},
	performActionOnTick(),
}

/**********************************
* Indirect Indexed
***********************************/

var actionIndirectIndexedY []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	func(cpu *Cpu65C02S) func() {
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	},
	func(cpu *Cpu65C02S) func() {
		// TODO: If page boundary crossed might require extra cycle (couldn't find documentation might need to check with real hardware)
		cpu.setReadBus(cpu.addressBus.Read() + 1)

		return func() {
			cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
			cpu.setReadBus(cpu.instructionRegister)
			cpu.addToInstructionRegister(uint16(cpu.yRegister))
		}
	},
	performInstructionRegisterCarryCycle(),
	readFromInstructionRegister(true),
}

var actionIndirectIndexedYW []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	func(cpu *Cpu65C02S) func() {
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	},
	func(cpu *Cpu65C02S) func() {
		cpu.setReadBus(cpu.addressBus.Read() + 1)

		return func() {
			cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
			cpu.setReadBus(cpu.instructionRegister)
			cpu.addToInstructionRegister(uint16(cpu.yRegister))
		}
	},
	performInstructionRegisterCarryCycle(),
	performActionOnTick(),
}

/**********************************
* Indirect
***********************************/

var actionIndirect []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	readInstructionRegisterMSB(),
	func(cpu *Cpu65C02S) func() {
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	},
	func(cpu *Cpu65C02S) func() {
		cpu.setReadBus(cpu.addressBus.Read() + 1)

		return func() {
			cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
		}
	},
	readFromInstructionRegister(true),
}

/**********************************
* Zero Page Indirect
***********************************/

var actionZeroPageIndirect []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	func(cpu *Cpu65C02S) func() {
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	},
	func(cpu *Cpu65C02S) func() {
		cpu.setReadBus(cpu.addressBus.Read() + 1)

		return func() {
			cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
		}
	},
	readFromInstructionRegister(true),
}

var actionZeroPageIndirectWrite []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	func(cpu *Cpu65C02S) func() {
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	},
	func(cpu *Cpu65C02S) func() {
		cpu.setReadBus(cpu.addressBus.Read() + 1)

		return func() {
			cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
		}
	},
	performActionOnTick(),
}

/**********************************
* Absolute Indexed Indirect
***********************************/

var actionAbsoluteIndexedIndirectX []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	func(cpu *Cpu65C02S) func() {
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
			cpu.addToInstructionRegister(uint16(cpu.xRegister))
		}
	},
	func(cpu *Cpu65C02S) func() {
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	},
	func(cpu *Cpu65C02S) func() {
		cpu.setReadBus(uint16(cpu.addressBus.Read() + 1))

		return func() {
			cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
		}
	},
	readFromInstructionRegister(true),
}
