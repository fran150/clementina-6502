package cpu

func setZeroFlag(cpu *Cpu65C02S, value uint8) {
	cpu.processorStatusRegister.SetFlag(ZeroFlagBit, value == 0)
}

func setNegativeFlag(cpu *Cpu65C02S, value uint8) {
	cpu.processorStatusRegister.SetFlag(NegativeFlagBit, value&0x80 > 0)
}

func ActionADC(cpu *Cpu65C02S) {

}

func ActionAND(cpu *Cpu65C02S) {
	cpu.accumulatorRegister &= cpu.dataRegister

	setZeroFlag(cpu, cpu.accumulatorRegister)
	setNegativeFlag(cpu, cpu.accumulatorRegister)
}

func ActionASL(cpu *Cpu65C02S) {
	temp := uint16(cpu.dataRegister) << 1

	if temp > 0xFF {
		cpu.processorStatusRegister.SetFlag(CarryFlagBit, true)
	}

	cpu.dataRegister = uint8(temp)
	cpu.setWriteBus(cpu.instructionRegister, cpu.dataRegister)
}

func ActionBCC(cpu *Cpu65C02S) {
	if !cpu.processorStatusRegister.Flag(CarryFlagBit) {
		cpu.branchTaken = true
	}
}

func ActionBCS(cpu *Cpu65C02S) {

}

func ActionBEQ(cpu *Cpu65C02S) {

}

func ActionBIT(cpu *Cpu65C02S) {

}

func ActionBMI(cpu *Cpu65C02S) {

}

func ActionBNE(cpu *Cpu65C02S) {

}

func ActionBPL(cpu *Cpu65C02S) {

}

func ActionBRA(cpu *Cpu65C02S) {

}

func ActionBRK(cpu *Cpu65C02S) {

}

func ActionBVC(cpu *Cpu65C02S) {

}

func ActionBVS(cpu *Cpu65C02S) {

}

func ActionCLC(cpu *Cpu65C02S) {

}

func ActionCLD(cpu *Cpu65C02S) {

}

func ActionCLI(cpu *Cpu65C02S) {

}

func ActionCLV(cpu *Cpu65C02S) {

}

func ActionCMP(cpu *Cpu65C02S) {

}

func ActionCPX(cpu *Cpu65C02S) {

}

func ActionCPY(cpu *Cpu65C02S) {

}

func ActionDEC(cpu *Cpu65C02S) {

}

func ActionDEX(cpu *Cpu65C02S) {

}

func ActionDEY(cpu *Cpu65C02S) {

}

func ActionEOR(cpu *Cpu65C02S) {

}

func ActionINC(cpu *Cpu65C02S) {
	cpu.dataRegister++

	if cpu.getCurrentAddressMode().Name() != AddressModeAccumulator {
		cpu.setWriteBus(cpu.instructionRegister, cpu.dataRegister)
	} else {
		cpu.accumulatorRegister = cpu.dataRegister
	}

}

func ActionINX(cpu *Cpu65C02S) {

}

func ActionINY(cpu *Cpu65C02S) {

}

func ActionJMP(cpu *Cpu65C02S) {
	cpu.programCounter = cpu.instructionRegister
}

func ActionJSR(cpu *Cpu65C02S) {

}

func ActionLDA(cpu *Cpu65C02S) {
	cpu.accumulatorRegister = cpu.dataRegister
}

func ActionLDX(cpu *Cpu65C02S) {
	cpu.xRegister = cpu.dataRegister
}

func ActionLDY(cpu *Cpu65C02S) {
	cpu.yRegister = cpu.dataRegister
}

func ActionLSR(cpu *Cpu65C02S) {

}

func ActionNOP(cpu *Cpu65C02S) {

}

func ActionORA(cpu *Cpu65C02S) {

}

func ActionPHA(cpu *Cpu65C02S) {

}

func ActionPHP(cpu *Cpu65C02S) {

}

func ActionPHX(cpu *Cpu65C02S) {

}

func ActionPHY(cpu *Cpu65C02S) {

}

func ActionPLA(cpu *Cpu65C02S) {

}

func ActionPLP(cpu *Cpu65C02S) {

}

func ActionPLX(cpu *Cpu65C02S) {

}

func ActionPLY(cpu *Cpu65C02S) {

}

func ActionROL(cpu *Cpu65C02S) {

}

func ActionROR(cpu *Cpu65C02S) {

}

func ActionRTI(cpu *Cpu65C02S) {

}

func ActionRTS(cpu *Cpu65C02S) {

}

func ActionSBC(cpu *Cpu65C02S) {

}

func ActionSEC(cpu *Cpu65C02S) {

}

func ActionSED(cpu *Cpu65C02S) {

}

func ActionSEI(cpu *Cpu65C02S) {

}

func ActionSTA(cpu *Cpu65C02S) {
	cpu.setWriteBus(cpu.instructionRegister, cpu.accumulatorRegister)
}

func ActionSTP(cpu *Cpu65C02S) {

}

func ActionSTX(cpu *Cpu65C02S) {
	cpu.setWriteBus(cpu.instructionRegister, cpu.xRegister)
}

func ActionSTY(cpu *Cpu65C02S) {
	cpu.setWriteBus(cpu.instructionRegister, cpu.yRegister)
}

func ActionSTZ(cpu *Cpu65C02S) {

}

func ActionTAX(cpu *Cpu65C02S) {

}

func ActionTAY(cpu *Cpu65C02S) {

}

func ActionTRB(cpu *Cpu65C02S) {

}

func ActionTSB(cpu *Cpu65C02S) {

}

func ActionTSX(cpu *Cpu65C02S) {

}

func ActionTXA(cpu *Cpu65C02S) {

}

func ActionTXS(cpu *Cpu65C02S) {

}

func ActionTYA(cpu *Cpu65C02S) {

}

func ActionWAI(cpu *Cpu65C02S) {

}

func ActionRMB(cpu *Cpu65C02S) {

}

func ActionSMB(cpu *Cpu65C02S) {

}

func ActionBBS(cpu *Cpu65C02S) {

}

func ActionBBR(cpu *Cpu65C02S) {

}
