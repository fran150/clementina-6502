package cpu

/**************************************************************************************************
* Evaluate values and set processor status flags
**************************************************************************************************/

func setZeroFlag(cpu *Cpu65C02S, value uint8) {
	cpu.processorStatusRegister.SetFlag(ZeroFlagBit, value == 0)
}

func setZeroFlag16(cpu *Cpu65C02S, value uint16) {
	cpu.processorStatusRegister.SetFlag(ZeroFlagBit, value&0x00FF == 0)
}

func setNegativeFlag(cpu *Cpu65C02S, value uint8) {
	cpu.processorStatusRegister.SetFlag(NegativeFlagBit, value&0x80 > 0)
}

func setNegativeFlag16(cpu *Cpu65C02S, value uint16) {
	cpu.processorStatusRegister.SetFlag(NegativeFlagBit, value&0x0080 > 0)
}

func setCarryFlag(cpu *Cpu65C02S, value uint16) {
	cpu.processorStatusRegister.SetFlag(CarryFlagBit, value&0xFF00 > 0)
}

func setCarryFlagDecimal(cpu *Cpu65C02S, value uint16) {
	cpu.processorStatusRegister.SetFlag(CarryFlagBit, value > 0x99)
}

func setCarryFlagSubstraction(cpu *Cpu65C02S, value uint16) {
	cpu.processorStatusRegister.SetFlag(CarryFlagBit, value < 0x100)
}

func setOverflowFlagAddition(cpu *Cpu65C02S, original uint8, register uint8, addedValue uint16) {
	termA := ^(uint16(register) ^ uint16(original))
	termB := (uint16(register) ^ addedValue)
	value := (termA&termB)&0x0080 != 0x0000

	cpu.processorStatusRegister.SetFlag(OverflowFlagBit, value)
}

func setOverflowFlagBit(cpu *Cpu65C02S, value uint8) {
	cpu.processorStatusRegister.SetFlag(OverflowFlagBit, value&(1<<6) != 0)
}

/**************************************************************************************************
* BCD transform functions for decimal operations ADC / SBC
**************************************************************************************************/

func toBCDAddition(accumulator uint8, dataRegister uint8, carry uint8, value uint16) uint16 {
	if ((accumulator & 0xF) + (dataRegister & 0xF) + carry) > 0x09 {
		value += 6
	}

	if value > 0x99 {
		value += 96
	}

	return value
}

func toBCDSubstraction(accumulatorRegister uint8, dataRegister uint8, carry uint8, value uint16) uint16 {
	if ((accumulatorRegister & 0x0F) - (dataRegister & 0x0F) - carry) > 0x09 {
		value -= 6
	}

	if value > 0x99 {
		value -= 0x60
	}

	return value
}

/**************************************************************************************************
* 6502 traditional instruction set
**************************************************************************************************/

// A,Z,C,N = A+M+C
// This instruction adds the contents of a memory location to the accumulator together with the carry bit.
// If overflow occurs the carry bit is set, this enables multiple byte addition to be performed.
// Decimal Mode based on https://github.com/gianlucag/mos6502/blob/master/mos6502.cpp
func actionADC(cpu *Cpu65C02S) {
	var carry uint8 = 0
	var value uint16

	if cpu.processorStatusRegister.Flag(CarryFlagBit) {
		carry = 1
	}

	value = uint16(cpu.dataRegister) + uint16(cpu.accumulatorRegister) + uint16(carry)

	if cpu.processorStatusRegister.Flag(DecimalModeFlagBit) {
		value = toBCDAddition(cpu.accumulatorRegister, cpu.dataRegister, carry, value)

		setCarryFlagDecimal(cpu, value)
	} else {
		setCarryFlag(cpu, value)
	}

	setZeroFlag16(cpu, value)
	setNegativeFlag16(cpu, value)
	setOverflowFlagAddition(cpu, cpu.dataRegister, cpu.accumulatorRegister, value)

	cpu.accumulatorRegister = uint8(value)
}

// A,Z,N = A&M
// A logical AND is performed, bit by bit, on the accumulator contents using the contents of a byte of memory.
func actionAND(cpu *Cpu65C02S) {
	cpu.accumulatorRegister &= cpu.dataRegister

	setZeroFlag(cpu, cpu.accumulatorRegister)
	setNegativeFlag(cpu, cpu.accumulatorRegister)
}

// A,Z,C,N = M*2 or M,Z,C,N = M*2
// This operation shifts all the bits of the accumulator or memory contents one bit left.
// Bit 0 is set to 0 and bit 7 is placed in the carry flag. The effect of this operation is to multiply the memory
// contents by 2 (ignoring 2's complement considerations), setting the carry if the result will not fit in 8 bits.
func actionASL(cpu *Cpu65C02S) {
	var value uint16

	if cpu.currentAddressMode.name == AddressModeAccumulator {
		value = uint16(cpu.accumulatorRegister) << 1
		cpu.accumulatorRegister = uint8(value)
	} else {
		value = uint16(cpu.dataRegister) << 1
		cpu.dataRegister = uint8(value)
		cpu.setWriteBus(cpu.instructionRegister, cpu.dataRegister)
	}

	setCarryFlag(cpu, value)
	setZeroFlag16(cpu, value)
	setNegativeFlag16(cpu, value)
}

// If the carry flag is clear then add the relative displacement to the program counter to cause a branch to a new location.
func actionBCC(cpu *Cpu65C02S) {
	if !cpu.processorStatusRegister.Flag(CarryFlagBit) {
		cpu.branchTaken = true
	}
}

// If the carry flag is set then add the relative displacement to the program counter to cause a branch to a new location.
func actionBCS(cpu *Cpu65C02S) {
	if cpu.processorStatusRegister.Flag(CarryFlagBit) {
		cpu.branchTaken = true
	}
}

// If the zero flag is set then add the relative displacement to the program counter to cause a branch to a new location.

func actionBEQ(cpu *Cpu65C02S) {
	if cpu.processorStatusRegister.Flag(ZeroFlagBit) {
		cpu.branchTaken = true
	}
}

// A & M, N = M7, V = M6
// This instructions is used to test if one or more bits are set in a target memory location. The mask pattern in A is ANDed
// with the value in memory to set or clear the zero flag, but the result is not kept. Bits 7 and 6 of the value from memory
// are copied into the N and V flags.
func actionBIT(cpu *Cpu65C02S) {
	value := cpu.dataRegister & cpu.accumulatorRegister
	setZeroFlag(cpu, value)
	setNegativeFlag(cpu, cpu.dataRegister)
	setOverflowFlagBit(cpu, cpu.dataRegister)
}

// If the negative flag is set then add the relative displacement to the program counter to cause a branch to a new location.
func actionBMI(cpu *Cpu65C02S) {
	if cpu.processorStatusRegister.Flag(NegativeFlagBit) {
		cpu.branchTaken = true
	}
}

// If the zero flag is clear then add the relative displacement to the program counter to cause a branch to a new location.
func actionBNE(cpu *Cpu65C02S) {
	if !cpu.processorStatusRegister.Flag(ZeroFlagBit) {
		cpu.branchTaken = true
	}
}

// If the negative flag is clear then add the relative displacement to the program counter to cause a branch to a new location.
func actionBPL(cpu *Cpu65C02S) {
	if !cpu.processorStatusRegister.Flag(NegativeFlagBit) {
		cpu.branchTaken = true
	}
}

// The BRK instruction forces the generation of an interrupt request. The program counter and processor status are pushed on
// the stack then the IRQ interrupt vector at $FFFE/F is loaded into the PC and the break flag in the status set to one.
//func actionBRK(cpu *Cpu65C02S) {
// Handled by the address mode
//}

// If the overflow flag is clear then add the relative displacement to the program counter to cause a branch to a new location.
func actionBVC(cpu *Cpu65C02S) {
	if !cpu.processorStatusRegister.Flag(OverflowFlagBit) {
		cpu.branchTaken = true
	}
}

// If the overflow flag is set then add the relative displacement to the program counter to cause a branch to a new location.
func actionBVS(cpu *Cpu65C02S) {
	if cpu.processorStatusRegister.Flag(OverflowFlagBit) {
		cpu.branchTaken = true
	}
}

// C = 0
// Set the carry flag to zero.
func actionCLC(cpu *Cpu65C02S) {
	cpu.processorStatusRegister.SetFlag(CarryFlagBit, false)
}

// D = 0
// Sets the decimal mode flag to zero.
func actionCLD(cpu *Cpu65C02S) {
	cpu.processorStatusRegister.SetFlag(DecimalModeFlagBit, false)
}

// I = 0
// Clears the interrupt disable flag allowing normal interrupt requests to be serviced.
func actionCLI(cpu *Cpu65C02S) {
	cpu.processorStatusRegister.SetFlag(IrqDisableFlagBit, false)
}

// V = 0
// Clears the overflow flag.
func actionCLV(cpu *Cpu65C02S) {
	cpu.processorStatusRegister.SetFlag(OverflowFlagBit, false)
}

// Z,C,N = A-M
// This instruction compares the contents of the accumulator with another memory held value and sets the
// zero and carry flags as appropriate.
func actionCMP(cpu *Cpu65C02S) {
	// We will use the substract instruction to make the comparison and set the appropriate flags
	// before that we store the current status of affected registers and flags.
	tempA := cpu.accumulatorRegister
	tempV := cpu.processorStatusRegister.Flag(OverflowFlagBit)
	tempD := cpu.processorStatusRegister.Flag(DecimalModeFlagBit)

	// CMP will have carry set to start comparison and will not be decimal
	cpu.processorStatusRegister.SetFlag(CarryFlagBit, true)
	cpu.processorStatusRegister.SetFlag(DecimalModeFlagBit, false)
	actionSBC(cpu)

	// Set the values back to the flags and the accumulator
	cpu.processorStatusRegister.SetFlag(OverflowFlagBit, tempV)
	cpu.processorStatusRegister.SetFlag(DecimalModeFlagBit, tempD)
	cpu.accumulatorRegister = tempA
}

// Z,C,N = X-M
// This instruction compares the contents of the X register with another memory held value and sets the
// zero and carry flags as appropriate.
func actionCPX(cpu *Cpu65C02S) {
	temp := cpu.accumulatorRegister
	cpu.accumulatorRegister = cpu.xRegister
	actionCMP(cpu)
	cpu.accumulatorRegister = temp
}

// Z,C,N = Y-M
// This instruction compares the contents of the Y register with another memory held value and sets the
// zero and carry flags as appropriate.
func actionCPY(cpu *Cpu65C02S) {
	temp := cpu.accumulatorRegister
	cpu.accumulatorRegister = cpu.yRegister
	actionCMP(cpu)
	cpu.accumulatorRegister = temp
}

// Subtracts one from the value held at a specified memory location setting the zero and negative
// flags as appropriate.
func actionDEC(cpu *Cpu65C02S) {
	var value uint8

	if cpu.currentAddressMode.Name() != AddressModeAccumulator {
		cpu.dataRegister--
		value = cpu.dataRegister
		cpu.setWriteBus(cpu.instructionRegister, cpu.dataRegister)
	} else {
		cpu.accumulatorRegister--
		value = cpu.accumulatorRegister
	}

	setZeroFlag(cpu, value)
	setNegativeFlag(cpu, value)
}

// X,Z,N = X-1
// Subtracts one from the X register setting the zero and negative flags as appropriate.
func actionDEX(cpu *Cpu65C02S) {
	cpu.xRegister--

	setZeroFlag(cpu, cpu.xRegister)
	setNegativeFlag(cpu, cpu.xRegister)

}

// Y,Z,N = Y-1
// Subtracts one from the Y register setting the zero and negative flags as appropriate.
func actionDEY(cpu *Cpu65C02S) {
	cpu.yRegister--

	setZeroFlag(cpu, cpu.yRegister)
	setNegativeFlag(cpu, cpu.yRegister)
}

// A,Z,N = A^M
// An exclusive OR is performed, bit by bit, on the accumulator contents using the contents of a byte of memory.
func actionEOR(cpu *Cpu65C02S) {
	cpu.accumulatorRegister = cpu.accumulatorRegister ^ cpu.dataRegister

	setZeroFlag(cpu, cpu.accumulatorRegister)
	setNegativeFlag(cpu, cpu.accumulatorRegister)
}

// M,Z,N = M+1
// Adds one to the value held at a specified memory location setting the zero and negative flags as appropriate.
func actionINC(cpu *Cpu65C02S) {
	var value uint8

	if cpu.currentAddressMode.Name() != AddressModeAccumulator {
		cpu.dataRegister++
		value = cpu.dataRegister
		cpu.setWriteBus(cpu.instructionRegister, cpu.dataRegister)
	} else {
		cpu.accumulatorRegister++
		value = cpu.accumulatorRegister
	}

	setZeroFlag(cpu, value)
	setNegativeFlag(cpu, value)
}

// X,Z,N = X+1
// Adds one to the X register setting the zero and negative flags as appropriate.
func actionINX(cpu *Cpu65C02S) {
	cpu.xRegister++

	setZeroFlag(cpu, cpu.xRegister)
	setNegativeFlag(cpu, cpu.xRegister)

}

// Y,Z,N = Y+1
// Adds one to the Y register setting the zero and negative flags as appropriate.
func actionINY(cpu *Cpu65C02S) {
	cpu.yRegister++

	setZeroFlag(cpu, cpu.yRegister)
	setNegativeFlag(cpu, cpu.yRegister)
}

// Sets the program counter to the address specified by the operand.
func actionJMP(cpu *Cpu65C02S) {
	cpu.programCounter = cpu.instructionRegister
}

// The JSR instruction pushes the address (minus one) of the return point on to the stack and then sets
// the program counter to the target memory address.
//func actionJSR(cpu *Cpu65C02S) {
// Handled by the address mode
//}

// A,Z,N = M
// Loads a byte of memory into the accumulator setting the zero and negative flags as appropriate.
func actionLDA(cpu *Cpu65C02S) {
	cpu.accumulatorRegister = cpu.dataRegister

	setZeroFlag(cpu, cpu.accumulatorRegister)
	setNegativeFlag(cpu, cpu.accumulatorRegister)
}

// X,Z,N = M
// Loads a byte of memory into the X register setting the zero and negative flags as appropriate.
func actionLDX(cpu *Cpu65C02S) {
	cpu.xRegister = cpu.dataRegister

	setZeroFlag(cpu, cpu.xRegister)
	setNegativeFlag(cpu, cpu.xRegister)
}

// Y,Z,N = M
// Loads a byte of memory into the Y register setting the zero and negative flags as appropriate.
func actionLDY(cpu *Cpu65C02S) {
	cpu.yRegister = cpu.dataRegister

	setZeroFlag(cpu, cpu.yRegister)
	setNegativeFlag(cpu, cpu.yRegister)
}

// A,C,Z,N = A/2 or M,C,Z,N = M/2
// Each of the bits in A or M is shift one place to the right. The bit that was in bit 0 is shifted
// into the carry flag. Bit 7 is set to zero.
func actionLSR(cpu *Cpu65C02S) {
	var value uint8
	if cpu.currentAddressMode.Name() != AddressModeAccumulator {
		cpu.processorStatusRegister.SetFlag(CarryFlagBit, cpu.dataRegister&0x01 > 0)
		cpu.dataRegister = cpu.dataRegister >> 1
		value = cpu.dataRegister
		cpu.setWriteBus(cpu.instructionRegister, cpu.dataRegister)
	} else {
		cpu.processorStatusRegister.SetFlag(CarryFlagBit, cpu.accumulatorRegister&0x01 > 0)
		cpu.accumulatorRegister = cpu.accumulatorRegister >> 1
		value = cpu.accumulatorRegister
	}

	setZeroFlag(cpu, value)
	setNegativeFlag(cpu, value)
}

// The NOP instruction causes no changes to the processor other than the normal incrementing of the program counter to the next instruction.
func actionNOP(cpu *Cpu65C02S) {
	// Do nothing
}

// A,Z,N = A|M
// An inclusive OR is performed, bit by bit, on the accumulator contents using the contents of a byte of memory.
func actionORA(cpu *Cpu65C02S) {
	cpu.accumulatorRegister = cpu.accumulatorRegister | cpu.dataRegister

	setZeroFlag(cpu, cpu.accumulatorRegister)
	setNegativeFlag(cpu, cpu.accumulatorRegister)
}

// Pushes a copy of the accumulator on to the stack.
func actionPHA(cpu *Cpu65C02S) {
	cpu.writeToStack(cpu.accumulatorRegister)
	cpu.stackPointer--
}

// Pushes a copy of the status flags on to the stack.
func actionPHP(cpu *Cpu65C02S) {
	cpu.writeToStack(cpu.processorStatusRegister.ReadValue())
	cpu.stackPointer--
}

// Pulls an 8 bit value from the stack and into the accumulator. The zero and negative flags are set as appropriate.
func actionPLA(cpu *Cpu65C02S) {
	cpu.accumulatorRegister = cpu.dataRegister

	setZeroFlag(cpu, cpu.accumulatorRegister)
	setNegativeFlag(cpu, cpu.accumulatorRegister)
}

// Pulls an 8 bit value from the stack and into the processor flags. The flags will take on new states
// as determined by the value pulled.
func actionPLP(cpu *Cpu65C02S) {
	cpu.processorStatusRegister.SetValue(cpu.dataRegister)
}

// Move each of the bits in either A or M one place to the left. Bit 0 is filled with the current value
// of the carry flag whilst the old bit 7 becomes the new carry flag value.
func actionROL(cpu *Cpu65C02S) {
	var carry uint16 = 0
	var value uint16

	if cpu.processorStatusRegister.Flag(CarryFlagBit) {
		carry = 1
	}

	if cpu.currentAddressMode.Name() != AddressModeAccumulator {
		value = uint16(cpu.dataRegister)<<1 | carry
		cpu.dataRegister = uint8(value)
		cpu.setWriteBus(cpu.instructionRegister, cpu.dataRegister)

	} else {
		value = uint16(cpu.accumulatorRegister)<<1 | carry
		cpu.accumulatorRegister = uint8(value)
	}

	setCarryFlag(cpu, value)
	setZeroFlag16(cpu, value)
	setNegativeFlag16(cpu, value)
}

// Move each of the bits in either A or M one place to the right. Bit 7 is filled with the current value of the carry flag
// whilst the old bit 0 becomes the new carry flag value.
func actionROR(cpu *Cpu65C02S) {
	var carry uint16 = 0
	var value uint16

	if cpu.processorStatusRegister.Flag(CarryFlagBit) {
		carry = 1 << 7
	}

	if cpu.currentAddressMode.Name() != AddressModeAccumulator {
		cpu.processorStatusRegister.SetFlag(CarryFlagBit, cpu.dataRegister&0x01 > 0)
		value = carry | (uint16(cpu.dataRegister) >> 1)
		cpu.dataRegister = uint8(value)
		cpu.setWriteBus(cpu.instructionRegister, cpu.dataRegister)
	} else {
		cpu.processorStatusRegister.SetFlag(CarryFlagBit, cpu.accumulatorRegister&0x01 > 0)
		value = carry | (uint16(cpu.accumulatorRegister) >> 1)
		cpu.accumulatorRegister = uint8(value)
	}

	setZeroFlag16(cpu, value)
	setNegativeFlag16(cpu, value)
}

// The RTI instruction is used at the end of an interrupt processing routine. It pulls the processor flags from the stack
// followed by the program counter.
// func actionRTI(cpu *Cpu65C02S) {
// Handled by the address mode
// }

// The RTS instruction is used at the end of a subroutine to return to the calling routine.
// It pulls the program counter (minus one) from the stack.
// func actionRTS(cpu *Cpu65C02S) {
// Handled by the address mode
// }

// A,Z,C,N = A-M-(1-C)
// This instruction subtracts the contents of a memory location to the accumulator together with the not of the carry bit.
// If overflow occurs the carry bit is clear, this enables multiple byte subtraction to be performed.
// Decimal Mode based on https://github.com/gianlucag/mos6502/blob/master/mos6502.cpp
func actionSBC(cpu *Cpu65C02S) {
	var carry uint8 = 0
	var value uint16

	if !cpu.processorStatusRegister.Flag(CarryFlagBit) {
		carry = 1
	}

	value = uint16(cpu.accumulatorRegister) - uint16(cpu.dataRegister) - uint16(carry)

	if cpu.processorStatusRegister.Flag(DecimalModeFlagBit) {
		value = toBCDSubstraction(cpu.accumulatorRegister, cpu.dataRegister, carry, value)
	}

	setCarryFlagSubstraction(cpu, value)

	setOverflowFlagAddition(cpu, ^cpu.dataRegister, cpu.accumulatorRegister, value)
	setZeroFlag16(cpu, value)
	setNegativeFlag16(cpu, value)

	cpu.accumulatorRegister = uint8(value)
}

// C = 1
// Set the carry flag to one.
func actionSEC(cpu *Cpu65C02S) {
	cpu.processorStatusRegister.SetFlag(CarryFlagBit, true)
}

// D = 1
// Set the decimal mode flag to one.
func actionSED(cpu *Cpu65C02S) {
	cpu.processorStatusRegister.SetFlag(DecimalModeFlagBit, true)
}

// I = 1
// Set the interrupt disable flag to one.
func actionSEI(cpu *Cpu65C02S) {
	cpu.processorStatusRegister.SetFlag(IrqDisableFlagBit, true)
}

// M = A
// Stores the contents of the accumulator into memory.
func actionSTA(cpu *Cpu65C02S) {
	cpu.setWriteBus(cpu.instructionRegister, cpu.accumulatorRegister)
}

// M = X
// Stores the contents of the X register into memory.
func actionSTX(cpu *Cpu65C02S) {
	cpu.setWriteBus(cpu.instructionRegister, cpu.xRegister)
}

// M = Y
// Stores the contents of the Y register into memory.
func actionSTY(cpu *Cpu65C02S) {
	cpu.setWriteBus(cpu.instructionRegister, cpu.yRegister)
}

// X = A
// Copies the current contents of the accumulator into the X register and sets the zero and negative flags as appropriate.
func actionTAX(cpu *Cpu65C02S) {
	cpu.xRegister = cpu.accumulatorRegister

	setZeroFlag(cpu, cpu.xRegister)
	setNegativeFlag(cpu, cpu.xRegister)
}

// Y = A
// Copies the current contents of the accumulator into the Y register and sets the zero and negative flags as appropriate.
func actionTAY(cpu *Cpu65C02S) {
	cpu.yRegister = cpu.accumulatorRegister

	setZeroFlag(cpu, cpu.yRegister)
	setNegativeFlag(cpu, cpu.yRegister)

}

// X = S
// Copies the current contents of the stack register into the X register and sets the zero and negative flags as appropriate.
func actionTSX(cpu *Cpu65C02S) {
	cpu.xRegister = cpu.stackPointer

	setZeroFlag(cpu, cpu.xRegister)
	setNegativeFlag(cpu, cpu.xRegister)
}

// A = X
// Copies the current contents of the X register into the accumulator and sets the zero and negative flags as appropriate.
func actionTXA(cpu *Cpu65C02S) {
	cpu.accumulatorRegister = cpu.xRegister

	setZeroFlag(cpu, cpu.accumulatorRegister)
	setNegativeFlag(cpu, cpu.accumulatorRegister)
}

// S = X
// Copies the current contents of the X register into the stack register.
func actionTXS(cpu *Cpu65C02S) {
	cpu.stackPointer = cpu.xRegister
}

// A = Y
// Copies the current contents of the Y register into the accumulator and sets the zero and negative flags as appropriate.
func actionTYA(cpu *Cpu65C02S) {
	cpu.accumulatorRegister = cpu.yRegister

	setZeroFlag(cpu, cpu.accumulatorRegister)
	setNegativeFlag(cpu, cpu.accumulatorRegister)
}

/**************************************************************************************************
* 65C02S new instruction set
**************************************************************************************************/
// This a actually a set of 8 instructions. Each tests a specific bit of a byte held on zero page and causes a branch of the bit is resest (0).
// They are 3 bytes long: OPCODE ZEROPAGE, BRANCH, for example BBR7 $10, $1A (Check if byte 7 of $0010 is reset and add $1A to PC)
func actionBBR(cpu *Cpu65C02S) {
	var bit uint8 = (uint8(cpu.currentOpCode) / 0x10)
	var mask uint8 = 0x01 << bit

	if (cpu.dataRegister & mask) == 0 {
		cpu.branchTaken = true
	}
}

// This a actually a set of 8 instructions. Each tests a specific bit of a byte held on zero page and causes a branch of the bit is set (1).
// They are 3 bytes long: OPCODE ZEROPAGE, BRANCH, for example BBS2 $70, $10 (Check if byte 2 of $0070 is set and add $10 to PC)
func actionBBS(cpu *Cpu65C02S) {
	var bit uint8 = (uint8(cpu.currentOpCode) / 0x10) - 8
	var mask uint8 = 0x01 << bit

	if (cpu.dataRegister & mask) > 0 {
		cpu.branchTaken = true
	}
}

// Adds the relative displacement to the program counter to cause a branch to a new location.
func actionBRA(cpu *Cpu65C02S) {
	cpu.branchTaken = true
}

// Pushes a copy of the X register  on to the stack.
func actionPHX(cpu *Cpu65C02S) {
	cpu.writeToStack(cpu.xRegister)
	cpu.stackPointer--
}

// Pushes a copy of the Y register on to the stack.
func actionPHY(cpu *Cpu65C02S) {
	cpu.writeToStack(cpu.yRegister)
	cpu.stackPointer--
}

// Pulls an 8 bit value from the stack and into the X register. The zero and negative flags are set as appropriate.
func actionPLX(cpu *Cpu65C02S) {
	cpu.xRegister = cpu.dataRegister

	setZeroFlag(cpu, cpu.xRegister)
	setNegativeFlag(cpu, cpu.xRegister)
}

// Pulls an 8 bit value from the stack and into the Y register. The zero and negative flags are set as appropriate.
func actionPLY(cpu *Cpu65C02S) {
	cpu.yRegister = cpu.dataRegister

	setZeroFlag(cpu, cpu.yRegister)
	setNegativeFlag(cpu, cpu.yRegister)
}

// M = 0
// Stores a zero byte value into memory.
func actionSTZ(cpu *Cpu65C02S) {
	cpu.setWriteBus(cpu.instructionRegister, 0x00)
}

// This a actually a set of 8 instructions. Each resets a specific bit of a byte held on zero page.
// For example RMB0 $10 (resets bit 0 of value in $0010)
func actionRMB(cpu *Cpu65C02S) {
	var bit uint8 = (uint8(cpu.currentOpCode) / 0x10)
	var mask uint8 = 0x01 << bit
	mask = ^mask

	cpu.dataRegister = cpu.dataRegister & mask
	cpu.setWriteBus(cpu.instructionRegister, cpu.dataRegister)
}

// This a actually a set of 8 instructions. Each sets a specific bit of a byte held on zero page.
// For example SMB2 $20 (sets bit 2 of value in $0020)
func actionSMB(cpu *Cpu65C02S) {
	var bit uint8 = (uint8(cpu.currentOpCode) / 0x10) - 8
	var mask uint8 = 0x01 << bit

	cpu.dataRegister = cpu.dataRegister | mask
	cpu.setWriteBus(cpu.instructionRegister, cpu.dataRegister)
}

// Z = M & A
// M = M & ~A
// The memory byte is tested to see if it contains any of the bits indicated by the value in the accumulator
// then the bits are reset in the memory byte.
func actionTRB(cpu *Cpu65C02S) {
	cpu.processorStatusRegister.SetFlag(ZeroFlagBit, cpu.dataRegister&cpu.accumulatorRegister > 0)
	cpu.dataRegister = cpu.dataRegister & (^cpu.accumulatorRegister)
	cpu.setWriteBus(cpu.instructionRegister, cpu.dataRegister)
}

// Z = M & A
// M = M | A
// The memory byte is tested to see if it contains any of the bits indicated by the value in the accumul
func actionTSB(cpu *Cpu65C02S) {
	cpu.processorStatusRegister.SetFlag(ZeroFlagBit, cpu.dataRegister&cpu.accumulatorRegister > 0)
	cpu.dataRegister = cpu.dataRegister | cpu.accumulatorRegister
	cpu.setWriteBus(cpu.instructionRegister, cpu.dataRegister)
}

func actionWAI(cpu *Cpu65C02S) {
	cpu.processorPaused = true
}

func actionSTP(cpu *Cpu65C02S) {
	cpu.processorStopped = true
}
