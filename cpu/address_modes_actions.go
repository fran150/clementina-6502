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

/**********************************
* Implied / Accumulator / Immediate
***********************************/

var actionImplicitOrAccumulator []cycleAction = []cycleAction{
	func(cpu *Cpu65C02S) func() {
		// read next instruction byte (and throw it away)
		cpu.setReadBus(cpu.programCounter)

		return func() {
			cpu.performAction()
		}
	},
}

var actionImmediate []cycleAction = []cycleAction{
	func(cpu *Cpu65C02S) func() {
		// fetch value, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.dataRegister = cpu.dataBus.Read()
			cpu.performAction()
		}
	},
}

/**********************************
* Absolute
***********************************/

var actionAbsoluteJump []cycleAction = []cycleAction{
	func(cpu *Cpu65C02S) func() {
		// fetch low address byte, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	},
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
	func(cpu *Cpu65C02S) func() {
		// fetch low byte of address, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	},
	func(cpu *Cpu65C02S) func() {
		// fetch high byte of address, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
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

var actionAbsoluteRMW []cycleAction = []cycleAction{
	func(cpu *Cpu65C02S) func() {
		// fetch low byte of address, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	},
	func(cpu *Cpu65C02S) func() {
		// fetch high byte of address, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
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
		// Write the new value to effective address
		cpu.performAction()

		return func() {
		}
	},
}

var actionAbsoluteWrite []cycleAction = []cycleAction{
	func(cpu *Cpu65C02S) func() {
		// fetch low byte of address, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	},
	func(cpu *Cpu65C02S) func() {
		// fetch high byte of address, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterMSB(cpu.dataBus.Read())
		}
	},
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
	func(cpu *Cpu65C02S) func() {
		// fetch address, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
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

var actionZeroPageRMW []cycleAction = []cycleAction{
	func(cpu *Cpu65C02S) func() {
		// fetch address, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
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
	func(cpu *Cpu65C02S) func() {
		// fetch address, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	},
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
	func(cpu *Cpu65C02S) func() {
		// fetch address, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	},
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
	func(cpu *Cpu65C02S) func() {
		// fetch address, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	},
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
	func(cpu *Cpu65C02S) func() {
		// fetch address, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	},
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
	func(cpu *Cpu65C02S) func() {
		// fetch address, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	},
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
	func(cpu *Cpu65C02S) func() {
		// fetch low byte of address, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	},
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
	func(cpu *Cpu65C02S) func() {
		// fetch low byte of address, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	},
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
	func(cpu *Cpu65C02S) func() {
		// fetch low byte of address, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	},
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
	func(cpu *Cpu65C02S) func() {
		// fetch low byte of address, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	},
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
	func(cpu *Cpu65C02S) func() {
		// fetch low byte of address, increment PC
		cpu.setReadBus(cpu.programCounter)
		cpu.programCounter++

		return func() {
			cpu.setInstructionRegisterLSB(cpu.dataBus.Read())
		}
	},
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
