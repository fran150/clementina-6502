package cpu

type cycleAction func(cpu *Cpu65C02S) func()

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

func readFromProgramCounterAndPerformAction() cycleAction {
	return func(cpu *Cpu65C02S) func() {
		// fetch value, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.dataRegister = cpu.dataBus.Read()
			cpu.performAction()
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

/**********************************
* Implied / Accumulator / Immediate
***********************************/

var actionImplicitOrAccumulator []cycleAction = []cycleAction{
	readNextInstructionAndThrowAway(),
}

var actionImmediate []cycleAction = []cycleAction{
	readFromProgramCounterAndPerformAction(),
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
	func(cpu *Cpu65C02S) func() {
		// read from effective address
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.dataRegister = cpu.dataBus.Read()
			cpu.performAction()
		}
	},
}

var actionAbsoluteRMW []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	readInstructionRegisterMSB(),
	func(cpu *Cpu65C02S) func() {
		// read from effective address
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
		}
	},
	func(cpu *Cpu65C02S) func() {
		// read from effective address
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.dataRegister = cpu.dataBus.Read()
		}
	},
	func(cpu *Cpu65C02S) func() {
		// Write the new value to effective address
		cpu.performAction()

		return func() {
		}
	},
}

var actionAbsoluteWrite []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	readInstructionRegisterMSB(),
	func(cpu *Cpu65C02S) func() {
		// write register to effective address
		cpu.performAction()

		return func() {
		}
	},
}

/**********************************
* Zero Page
***********************************/

var actionZeroPage []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	func(cpu *Cpu65C02S) func() {
		// read from effective address
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.dataRegister = cpu.dataBus.Read()
			cpu.performAction()
		}
	},
}

var actionZeroPageRMW []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	func(cpu *Cpu65C02S) func() {
		// read from effective address
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.dataRegister = cpu.dataBus.Read()
		}
	},
	func(cpu *Cpu65C02S) func() {
		// read from effective address
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.dataRegister = cpu.dataBus.Read()
		}
	},
	func(cpu *Cpu65C02S) func() {
		// write the new value to effective address
		cpu.performAction()

		return func() {
		}
	},
}

var actionZeroPageWrite []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	func(cpu *Cpu65C02S) func() {
		// write register to effective address
		cpu.performAction()

		return func() {
		}
	},
}

/**********************************
* Zero Page Indexed
***********************************/

var actionZeroPageX []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	func(cpu *Cpu65C02S) func() {
		// read from address, add index register to it
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.addToInstructionRegisterLSB(cpu.xRegister)
		}
	},
	func(cpu *Cpu65C02S) func() {
		// read from effective address
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.dataRegister = cpu.dataBus.Read()
			cpu.performAction()
		}
	},
}

var actionZeroPageXRMW []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	func(cpu *Cpu65C02S) func() {
		// read from address, add index register to it
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.addToInstructionRegisterLSB(cpu.xRegister)
		}
	},
	func(cpu *Cpu65C02S) func() {
		// read from effective address
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.dataRegister = cpu.dataBus.Read()
		}
	},
	func(cpu *Cpu65C02S) func() {
		cpu.performAction()

		return func() {
		}
	},
}

var actionZeroPageXWrite []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	func(cpu *Cpu65C02S) func() {
		// read from address, add index register to it
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.addToInstructionRegisterLSB(cpu.xRegister)
		}
	},
	func(cpu *Cpu65C02S) func() {
		// write to effective address
		cpu.performAction()

		return func() {
		}
	},
}

var actionZeroPageY []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	func(cpu *Cpu65C02S) func() {
		// read from address, add index register to it
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.addToInstructionRegisterLSB(cpu.yRegister)
		}
	},
	func(cpu *Cpu65C02S) func() {
		// read from effective address
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.dataRegister = cpu.dataBus.Read()
			cpu.performAction()
		}
	},
}

/**********************************
* Absolute Indexed Addressing
***********************************/
var actionAbsoluteX []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	func(cpu *Cpu65C02S) func() {
		// fetch high byte of address, add index register to low address byte, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
			cpu.setReadBus(cpu.instructionRegister)
			cpu.addToInstructionRegister(uint16(cpu.xRegister))
		}
	},
	func(cpu *Cpu65C02S) func() {
		if cpu.extraCycleEnabled {
			cpu.extraCycleEnabled = false
			// Previous cycle already set the address in the bus
			return func() {
			}
		} else {
			return nil
		}
	},
	func(cpu *Cpu65C02S) func() {
		// read from effective address
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.dataRegister = cpu.dataBus.Read()
			cpu.performAction()
		}
	},
}

var actionAbsoluteXRMW []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	func(cpu *Cpu65C02S) func() {
		// fetch high byte of address, add index register to low address byte, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
			cpu.setReadBus(cpu.instructionRegister)
			cpu.addToInstructionRegister(uint16(cpu.xRegister))
		}
	},
	func(cpu *Cpu65C02S) func() {
		if cpu.extraCycleEnabled {
			cpu.extraCycleEnabled = false
			// Previous cycle already set the address in the bus
			return func() {
			}
		} else {
			return nil
		}
	},
	func(cpu *Cpu65C02S) func() {
		// read from effective address
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
		}
	},
	func(cpu *Cpu65C02S) func() {
		// read from effective address
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.dataRegister = cpu.dataBus.Read()
		}
	},
	func(cpu *Cpu65C02S) func() {
		cpu.performAction()

		return func() {
		}
	},
}

var actionAbsoluteXWrite []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	func(cpu *Cpu65C02S) func() {
		// fetch high byte of address, add index register to low address byte, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
			cpu.setReadBus(cpu.instructionRegister)
			cpu.addToInstructionRegister(uint16(cpu.xRegister))
		}
	},
	func(cpu *Cpu65C02S) func() {
		if cpu.extraCycleEnabled {
			cpu.extraCycleEnabled = false
			// Previous cycle already set the address in the bus
			return func() {
			}
		} else {
			return nil
		}
	},
	func(cpu *Cpu65C02S) func() {
		// write to effective address
		cpu.performAction()

		return func() {
		}
	},
}

var actionAbsoluteY []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	func(cpu *Cpu65C02S) func() {
		// fetch high byte of address, add index register to low address byte, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
			cpu.setReadBus(cpu.instructionRegister)
			cpu.addToInstructionRegister(uint16(cpu.xRegister))
		}
	},
	func(cpu *Cpu65C02S) func() {
		if cpu.extraCycleEnabled {
			cpu.extraCycleEnabled = false
			// Previous cycle already set the address in the bus
			return func() {
			}
		} else {
			return nil
		}
	},
	func(cpu *Cpu65C02S) func() {
		// read from effective address
		cpu.setReadBus(cpu.instructionRegister)

		return func() {
			cpu.dataRegister = cpu.dataBus.Read()
			cpu.performAction()
		}
	},
}

var actionAbsoluteYWrite []cycleAction = []cycleAction{
	readInstructionRegisterLSB(),
	func(cpu *Cpu65C02S) func() {
		// fetch high byte of address, add index register to low address byte, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
			cpu.setReadBus(cpu.instructionRegister)
			cpu.addToInstructionRegister(uint16(cpu.yRegister))
		}
	},
	func(cpu *Cpu65C02S) func() {
		if cpu.extraCycleEnabled {
			cpu.extraCycleEnabled = false
			// Previous cycle already set the address in the bus
			return func() {
			}
		} else {
			return nil
		}
	},
	func(cpu *Cpu65C02S) func() {
		// write to effective address
		cpu.performAction()

		return func() {
		}
	},
}
