package cpu

func setZeroFlag(cpu *Cpu65C02S, value uint8) {
	cpu.processorStatusRegister.SetFlag(ZeroFlagBit, value == 0)
}

func setNegativeFlag(cpu *Cpu65C02S, value uint8) {
	cpu.processorStatusRegister.SetFlag(NegativeFlagBit, value&0x80 > 0)
}

func actionADC(cpu *Cpu65C02S) {

}

func actionAND(cpu *Cpu65C02S) {
	cpu.accumulatorRegister &= cpu.dataRegister

	setZeroFlag(cpu, cpu.accumulatorRegister)
	setNegativeFlag(cpu, cpu.accumulatorRegister)
}

func actionASL(cpu *Cpu65C02S) {
	temp := uint16(cpu.dataRegister) << 1

	if temp > 0xFF {
		cpu.processorStatusRegister.SetFlag(CarryFlagBit, true)
	}

	cpu.dataRegister = uint8(temp)
	cpu.setWriteBus(cpu.instructionRegister, cpu.dataRegister)
}

func actionBCC(cpu *Cpu65C02S) {
	if !cpu.processorStatusRegister.Flag(CarryFlagBit) {
		cpu.branchTaken = true
	}
}

func actionBCS(cpu *Cpu65C02S) {

}

func actionBEQ(cpu *Cpu65C02S) {

}

func actionBIT(cpu *Cpu65C02S) {

}

func actionBMI(cpu *Cpu65C02S) {

}

func actionBNE(cpu *Cpu65C02S) {

}

func actionBPL(cpu *Cpu65C02S) {

}

func actionBRA(cpu *Cpu65C02S) {

}

func actionBRK(cpu *Cpu65C02S) {

}

func actionBVC(cpu *Cpu65C02S) {

}

func actionBVS(cpu *Cpu65C02S) {

}

func actionCLC(cpu *Cpu65C02S) {

}

func actionCLD(cpu *Cpu65C02S) {

}

func actionCLI(cpu *Cpu65C02S) {

}

func actionCLV(cpu *Cpu65C02S) {

}

func actionCMP(cpu *Cpu65C02S) {

}

func actionCPX(cpu *Cpu65C02S) {

}

func actionCPY(cpu *Cpu65C02S) {

}

func actionDEC(cpu *Cpu65C02S) {

}

func actionDEX(cpu *Cpu65C02S) {

}

func actionDEY(cpu *Cpu65C02S) {

}

func actionEOR(cpu *Cpu65C02S) {

}

func actionINC(cpu *Cpu65C02S) {
	cpu.dataRegister++

	if cpu.getCurrentAddressMode().Name() != AddressModeAccumulator {
		cpu.setWriteBus(cpu.instructionRegister, cpu.dataRegister)
	} else {
		cpu.accumulatorRegister = cpu.dataRegister
	}

}

func actionINX(cpu *Cpu65C02S) {

}

func actionINY(cpu *Cpu65C02S) {

}

func actionJMP(cpu *Cpu65C02S) {
	cpu.programCounter = cpu.instructionRegister
}

func actionJSR(cpu *Cpu65C02S) {

}

func actionLDA(cpu *Cpu65C02S) {
	cpu.accumulatorRegister = cpu.dataRegister
}

func actionLDX(cpu *Cpu65C02S) {
	cpu.xRegister = cpu.dataRegister
}

func actionLDY(cpu *Cpu65C02S) {
	cpu.yRegister = cpu.dataRegister
}

func actionLSR(cpu *Cpu65C02S) {

}

func actionNOP(cpu *Cpu65C02S) {

}

func actionORA(cpu *Cpu65C02S) {

}

func actionPHA(cpu *Cpu65C02S) {
	cpu.writeToStack(cpu.accumulatorRegister)
	cpu.stackPointer--
}

func actionPHP(cpu *Cpu65C02S) {
	cpu.writeToStack(uint8(cpu.processorStatusRegister))
	cpu.stackPointer--
}

func actionPHX(cpu *Cpu65C02S) {
	cpu.writeToStack(cpu.xRegister)
	cpu.stackPointer--
}

func actionPHY(cpu *Cpu65C02S) {
	cpu.writeToStack(cpu.yRegister)
	cpu.stackPointer--
}

func actionPLA(cpu *Cpu65C02S) {
	cpu.accumulatorRegister = cpu.dataRegister
}

func actionPLP(cpu *Cpu65C02S) {
	cpu.processorStatusRegister = StatusRegister(cpu.dataRegister)
}

func actionPLX(cpu *Cpu65C02S) {
	cpu.xRegister = cpu.dataRegister
}

func actionPLY(cpu *Cpu65C02S) {
	cpu.yRegister = cpu.dataRegister
}

func actionROL(cpu *Cpu65C02S) {

}

func actionROR(cpu *Cpu65C02S) {

}

func actionRTI(cpu *Cpu65C02S) {

}

func actionRTS(cpu *Cpu65C02S) {

}

func actionSBC(cpu *Cpu65C02S) {

}

func actionSEC(cpu *Cpu65C02S) {

}

func actionSED(cpu *Cpu65C02S) {

}

func actionSEI(cpu *Cpu65C02S) {

}

func actionSTA(cpu *Cpu65C02S) {
	cpu.setWriteBus(cpu.instructionRegister, cpu.accumulatorRegister)
}

func actionSTP(cpu *Cpu65C02S) {

}

func actionSTX(cpu *Cpu65C02S) {
	cpu.setWriteBus(cpu.instructionRegister, cpu.xRegister)
}

func actionSTY(cpu *Cpu65C02S) {
	cpu.setWriteBus(cpu.instructionRegister, cpu.yRegister)
}

func actionSTZ(cpu *Cpu65C02S) {

}

func actionTAX(cpu *Cpu65C02S) {

}

func actionTAY(cpu *Cpu65C02S) {

}

func actionTRB(cpu *Cpu65C02S) {

}

func actionTSB(cpu *Cpu65C02S) {

}

func actionTSX(cpu *Cpu65C02S) {

}

func actionTXA(cpu *Cpu65C02S) {

}

func actionTXS(cpu *Cpu65C02S) {

}

func actionTYA(cpu *Cpu65C02S) {

}

func actionWAI(cpu *Cpu65C02S) {

}

func actionRMB(cpu *Cpu65C02S) {

}

func actionSMB(cpu *Cpu65C02S) {

}

func actionBBS(cpu *Cpu65C02S) {

}

func actionBBR(cpu *Cpu65C02S) {

}
