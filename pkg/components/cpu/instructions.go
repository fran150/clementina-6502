package cpu

// Each instruction and it's different address mode is represented by a number from 0 to 0xFF called OpCode
// See http://www.6502.org/users/obelisk/65C02/reference.html
type OpCode uint

// Contains information about each instruction.
type CpuInstructionData struct {
	opcode      OpCode
	mnemonic    Mnemonic
	action      func(cpu *Cpu65C02S)
	addressMode AddressMode
}

// Returns the OpCode value of this instruction and address mode
func (data *CpuInstructionData) OpCode() OpCode {
	return data.opcode
}

// Returns the Mnemonic for the instruction.
func (data *CpuInstructionData) Mnemonic() Mnemonic {
	return data.mnemonic
}

// Executes the instruction actions modifying the CPU state. For example, INX will increment
// the value in the X register.
func (data *CpuInstructionData) execute(cpu *Cpu65C02S) {
	data.action(cpu)
}

// Returns the address mode corresponding to this instruction's OpCode
func (data *CpuInstructionData) AddressMode() AddressMode {
	return data.addressMode
}

// ----------------------------------------------------------------------

// Returns all the instructions available for the CPU
type CpuInstructionSet struct {
	opCodeIndex [0x100]*CpuInstructionData
}

// Gets data about the instruction and address mode represented by the specified Opcode.
func (instruction *CpuInstructionSet) GetByOpCode(opCode OpCode) *CpuInstructionData {
	return instruction.opCodeIndex[opCode]
}

// Creates the instruction set supported by this CPU
func CreateInstructionSet() *CpuInstructionSet {
	var data = []CpuInstructionData{
		{0x00, BRK, nil, AddressModeBreak},
		{0x01, ORA, actionORA, AddressModeZeroPageIndexedIndirectX},
		{0x04, TSB, actionTSB, AddressModeZeroPageRMW},
		{0x05, ORA, actionORA, AddressModeZeroPage},
		{0x06, ASL, actionASL, AddressModeZeroPageRMW},
		{0x07, RMB0, actionRMB, AddressModeZeroPageRMW},
		{0x08, PHP, actionPHP, AddressModePushStack},
		{0x09, ORA, actionORA, AddressModeImmediate},
		{0x0A, ASL, actionASL, AddressModeAccumulator},
		{0x0C, TSB, actionTSB, AddressModeAbsoluteRMW},
		{0x0D, ORA, actionORA, AddressModeAbsolute},
		{0x0E, ASL, actionASL, AddressModeAbsoluteRMW},
		{0x0F, BBR0, actionBBR, AddressModeRelativeExtended},

		{0x10, BPL, actionBPL, AddressModeRelative},
		{0x11, ORA, actionORA, AddressModeZeroPageIndirectIndexedY},
		{0x12, ORA, actionORA, AddressModeIndirectZeroPage},
		{0x14, TRB, actionTRB, AddressModeZeroPageRMW},
		{0x15, ORA, actionORA, AddressModeZeroPageX},
		{0x16, ASL, actionASL, AddressModeZeroPageXRMW},
		{0x17, RMB1, actionRMB, AddressModeZeroPageRMW},
		{0x18, CLC, actionCLC, AddressModeImplicit},
		{0x19, ORA, actionORA, AddressModeAbsoluteY},
		{0x1A, INC, actionINC, AddressModeAccumulator},
		{0x1C, TRB, actionTRB, AddressModeAbsoluteRMW},
		{0x1D, ORA, actionORA, AddressModeAbsoluteX},
		{0x1E, ASL, actionASL, AddressModeAbsoluteXRMW},
		{0x1F, BBR1, actionBBR, AddressModeRelativeExtended},

		{0x20, JSR, nil, AddressModeJumpToSubroutine},
		{0x21, AND, actionAND, AddressModeZeroPageIndexedIndirectX},
		{0x24, BIT, actionBIT, AddressModeZeroPage},
		{0x25, AND, actionAND, AddressModeZeroPage},
		{0x26, ROL, actionROL, AddressModeZeroPageRMW},
		{0x27, RMB2, actionRMB, AddressModeZeroPageRMW},
		{0x28, PLP, actionPLP, AddressModePullStack},
		{0x29, AND, actionAND, AddressModeImmediate},
		{0x2A, ROL, actionROL, AddressModeAccumulator},
		{0x2C, BIT, actionBIT, AddressModeAbsolute},
		{0x2D, AND, actionAND, AddressModeAbsolute},
		{0x2E, ROL, actionROL, AddressModeAbsoluteRMW},
		{0x2F, BBR2, actionBBR, AddressModeRelativeExtended},

		{0x30, BMI, actionBMI, AddressModeRelative},
		{0x31, AND, actionAND, AddressModeZeroPageIndirectIndexedY},
		{0x32, AND, actionAND, AddressModeIndirectZeroPage},
		{0x34, BIT, actionBIT, AddressModeZeroPageX},
		{0x35, AND, actionAND, AddressModeZeroPageX},
		{0x36, ROL, actionROL, AddressModeZeroPageXRMW},
		{0x37, RMB3, actionRMB, AddressModeZeroPageRMW},
		{0x38, SEC, actionSEC, AddressModeImplicit},
		{0x39, AND, actionAND, AddressModeAbsoluteY},
		{0x3A, DEC, actionDEC, AddressModeAccumulator},
		{0x3C, BIT, actionBIT, AddressModeAbsoluteX},
		{0x3D, AND, actionAND, AddressModeAbsoluteX},
		{0x3E, ROL, actionROL, AddressModeAbsoluteXRMW},
		{0x3F, BBR3, actionBBR, AddressModeRelativeExtended},

		{0x40, RTI, nil, AddressModeReturnFromInterrupt},
		{0x41, EOR, actionEOR, AddressModeZeroPageIndexedIndirectX},
		{0x45, EOR, actionEOR, AddressModeZeroPage},
		{0x46, LSR, actionLSR, AddressModeZeroPageRMW},
		{0x47, RMB4, actionRMB, AddressModeZeroPageRMW},
		{0x48, PHA, actionPHA, AddressModePushStack},
		{0x49, EOR, actionEOR, AddressModeImmediate},
		{0x4A, LSR, actionLSR, AddressModeAccumulator},
		{0x4C, JMP, actionJMP, AddressModeAbsoluteJump},
		{0x4D, EOR, actionEOR, AddressModeAbsolute},
		{0x4E, LSR, actionLSR, AddressModeAbsoluteRMW},
		{0x4F, BBR4, actionBBR, AddressModeRelativeExtended},

		{0x50, BVC, actionBVC, AddressModeRelative},
		{0x51, EOR, actionEOR, AddressModeZeroPageIndirectIndexedY},
		{0x52, EOR, actionEOR, AddressModeIndirectZeroPage},
		{0x55, EOR, actionEOR, AddressModeZeroPageX},
		{0x56, LSR, actionLSR, AddressModeZeroPageXRMW},
		{0x57, RMB5, actionRMB, AddressModeZeroPageRMW},
		{0x58, CLI, actionCLI, AddressModeImplicit},
		{0x59, EOR, actionEOR, AddressModeAbsoluteY},
		{0x5A, PHY, actionPHY, AddressModePushStack},
		{0x5D, EOR, actionEOR, AddressModeAbsoluteX},
		{0x5E, LSR, actionLSR, AddressModeAbsoluteXRMW},
		{0x5F, BBR5, actionBBR, AddressModeRelativeExtended},

		{0x60, RTS, nil, AddressModeReturnFromSubroutine},
		{0x61, ADC, actionADC, AddressModeZeroPageIndexedIndirectX},
		{0x64, STZ, actionSTZ, AddressModeZeroPageW},
		{0x65, ADC, actionADC, AddressModeZeroPage},
		{0x66, ROR, actionROR, AddressModeZeroPageRMW},
		{0x67, RMB6, actionRMB, AddressModeZeroPageRMW},
		{0x68, PLA, actionPLA, AddressModePullStack},
		{0x69, ADC, actionADC, AddressModeImmediate},
		{0x6A, ROR, actionROR, AddressModeAccumulator},
		{0x6C, JMP, actionJMP, AddressModeIndirect},
		{0x6D, ADC, actionADC, AddressModeAbsolute},
		{0x6E, ROR, actionROR, AddressModeAbsoluteRMW},
		{0x6F, BBR6, actionBBR, AddressModeRelativeExtended},

		{0x70, BVS, actionBVS, AddressModeRelative},
		{0x71, ADC, actionADC, AddressModeZeroPageIndirectIndexedY},
		{0x72, ADC, actionADC, AddressModeIndirectZeroPage},
		{0x74, STZ, actionSTZ, AddressModeZeroPageXW},
		{0x75, ADC, actionADC, AddressModeZeroPageX},
		{0x76, ROR, actionROR, AddressModeZeroPageXRMW},
		{0x77, RMB7, actionRMB, AddressModeZeroPageRMW},
		{0x78, SEI, actionSEI, AddressModeImplicit},
		{0x79, ADC, actionADC, AddressModeAbsoluteY},
		{0x7A, PLY, actionPLY, AddressModePullStack},
		{0x7C, JMP, actionJMP, AddressModeAbsoluteIndexedIndirect},
		{0x7D, ADC, actionADC, AddressModeAbsoluteX},
		{0x7E, ROR, actionROR, AddressModeAbsoluteXRMW},
		{0x7F, BBR7, actionBBR, AddressModeRelativeExtended},

		{0x80, BRA, actionBRA, AddressModeRelative},
		{0x81, STA, actionSTA, AddressModeZeroPageIndexedIndirectXW},
		{0x84, STY, actionSTY, AddressModeZeroPageW},
		{0x85, STA, actionSTA, AddressModeZeroPageW},
		{0x86, STX, actionSTX, AddressModeZeroPageW},
		{0x87, SMB0, actionSMB, AddressModeZeroPageRMW},
		{0x88, DEY, actionDEY, AddressModeImplicit},
		{0x89, BIT, actionBIT, AddressModeImmediate},
		{0x8A, TXA, actionTXA, AddressModeImplicit},
		{0x8C, STY, actionSTY, AddressModeAbsoluteW},
		{0x8D, STA, actionSTA, AddressModeAbsoluteW},
		{0x8E, STX, actionSTX, AddressModeAbsoluteW},
		{0x8F, BBS0, actionBBS, AddressModeRelativeExtended},

		{0x90, BCC, actionBCC, AddressModeRelative},
		{0x91, STA, actionSTA, AddressModeZeroPageIndirectIndexedYW},
		{0x92, STA, actionSTA, AddressModeIndirectZeroPageW},
		{0x94, STY, actionSTY, AddressModeZeroPageXW},
		{0x95, STA, actionSTA, AddressModeZeroPageXW},
		{0x96, STX, actionSTX, AddressModeZeroPageYW},
		{0x97, SMB1, actionSMB, AddressModeZeroPageRMW},
		{0x98, TYA, actionTYA, AddressModeImplicit},
		{0x99, STA, actionSTA, AddressModeAbsoluteYW},
		{0x9A, TXS, actionTXS, AddressModeImplicit},
		{0x9C, STZ, actionSTZ, AddressModeAbsoluteW},
		{0x9D, STA, actionSTA, AddressModeAbsoluteXW},
		{0x9E, STZ, actionSTZ, AddressModeAbsoluteXW},
		{0x9F, BBS1, actionBBS, AddressModeRelativeExtended},

		{0xA0, LDY, actionLDY, AddressModeImmediate},
		{0xA1, LDA, actionLDA, AddressModeZeroPageIndexedIndirectX},
		{0xA2, LDX, actionLDX, AddressModeImmediate},
		{0xA4, LDY, actionLDY, AddressModeZeroPage},
		{0xA5, LDA, actionLDA, AddressModeZeroPage},
		{0xA6, LDX, actionLDX, AddressModeZeroPage},
		{0xA7, SMB2, actionSMB, AddressModeZeroPageRMW},
		{0xA8, TAY, actionTAY, AddressModeImplicit},
		{0xA9, LDA, actionLDA, AddressModeImmediate},
		{0xAA, TAX, actionTAX, AddressModeImplicit},
		{0xAC, LDY, actionLDY, AddressModeAbsolute},
		{0xAD, LDA, actionLDA, AddressModeAbsolute},
		{0xAE, LDX, actionLDX, AddressModeAbsolute},
		{0xAF, BBS2, actionBBS, AddressModeRelativeExtended},

		{0xB0, BCS, actionBCS, AddressModeRelative},
		{0xB1, LDA, actionLDA, AddressModeZeroPageIndirectIndexedY},
		{0xB2, LDA, actionLDA, AddressModeIndirectZeroPage},
		{0xB4, LDY, actionLDY, AddressModeZeroPageX},
		{0xB5, LDA, actionLDA, AddressModeZeroPageX},
		{0xB6, LDX, actionLDX, AddressModeZeroPageY},
		{0xB7, SMB3, actionSMB, AddressModeZeroPageRMW},
		{0xB8, CLV, actionCLV, AddressModeImplicit},
		{0xB9, LDA, actionLDA, AddressModeAbsoluteY},
		{0xBA, TSX, actionTSX, AddressModeImplicit},
		{0xBC, LDY, actionLDY, AddressModeAbsoluteX},
		{0xBD, LDA, actionLDA, AddressModeAbsoluteX},
		{0xBE, LDX, actionLDX, AddressModeAbsoluteY},
		{0xBF, BBS3, actionBBS, AddressModeRelativeExtended},

		{0xC0, CPY, actionCPY, AddressModeImmediate},
		{0xC1, CMP, actionCMP, AddressModeZeroPageIndexedIndirectX},
		{0xC4, CPY, actionCPY, AddressModeZeroPage},
		{0xC5, CMP, actionCMP, AddressModeZeroPage},
		{0xC6, DEC, actionDEC, AddressModeZeroPageRMW},
		{0xC7, SMB4, actionSMB, AddressModeZeroPageRMW},
		{0xC8, INY, actionINY, AddressModeImplicit},
		{0xC9, CMP, actionCMP, AddressModeImmediate},
		{0xCA, DEX, actionDEX, AddressModeImplicit},
		{0xCB, WAI, actionWAI, AddressModeImplicit},
		{0xCC, CPY, actionCPY, AddressModeAbsolute},
		{0xCD, CMP, actionCMP, AddressModeAbsolute},
		{0xCE, DEC, actionDEC, AddressModeAbsoluteRMW},
		{0xCF, BBS4, actionBBS, AddressModeRelativeExtended},

		{0xD0, BNE, actionBNE, AddressModeRelative},
		{0xD1, CMP, actionCMP, AddressModeZeroPageIndirectIndexedY},
		{0xD2, CMP, actionCMP, AddressModeIndirectZeroPage},
		{0xD5, CMP, actionCMP, AddressModeZeroPageX},
		{0xD6, DEC, actionDEC, AddressModeZeroPageXRMW},
		{0xD7, SMB5, actionSMB, AddressModeZeroPageRMW},
		{0xD8, CLD, actionCLD, AddressModeImplicit},
		{0xD9, CMP, actionCMP, AddressModeAbsoluteY},
		{0xDA, PHX, actionPHX, AddressModePushStack},
		{0xDB, STP, actionSTP, AddressModeImplicit},
		{0xDD, CMP, actionCMP, AddressModeAbsoluteX},
		{0xDE, DEC, actionDEC, AddressModeAbsoluteXRMW},
		{0xDF, BBS5, actionBBS, AddressModeRelativeExtended},

		{0xE0, CPX, actionCPX, AddressModeImmediate},
		{0xE1, SBC, actionSBC, AddressModeZeroPageIndexedIndirectX},
		{0xE4, CPX, actionCPX, AddressModeZeroPage},
		{0xE5, SBC, actionSBC, AddressModeZeroPage},
		{0xE6, INC, actionINC, AddressModeZeroPageRMW},
		{0xE7, SMB6, actionSMB, AddressModeZeroPageRMW},
		{0xE8, INX, actionINX, AddressModeImplicit},
		{0xE9, SBC, actionSBC, AddressModeImmediate},
		{0xEA, NOP, actionNOP, AddressModeImplicit},
		{0xEC, CPX, actionCPX, AddressModeAbsolute},
		{0xED, SBC, actionSBC, AddressModeAbsolute},
		{0xEE, INC, actionINC, AddressModeAbsoluteRMW},
		{0xEF, BBS6, actionBBS, AddressModeRelativeExtended},

		{0xF0, BEQ, actionBEQ, AddressModeRelative},
		{0xF1, SBC, actionSBC, AddressModeZeroPageIndirectIndexedY},
		{0xF2, SBC, actionSBC, AddressModeIndirectZeroPage},
		{0xF5, SBC, actionSBC, AddressModeZeroPageX},
		{0xF6, INC, actionINC, AddressModeZeroPageXRMW},
		{0xF7, SMB7, actionSMB, AddressModeZeroPageRMW},
		{0xF8, SED, actionSED, AddressModeImplicit},
		{0xF9, SBC, actionSBC, AddressModeAbsoluteY},
		{0xFA, PLX, actionPLX, AddressModePullStack},
		{0xFD, SBC, actionSBC, AddressModeAbsoluteX},
		{0xFE, INC, actionINC, AddressModeAbsoluteXRMW},
		{0xFF, BBS7, actionBBS, AddressModeRelativeExtended},
	}

	instructionSet := CpuInstructionSet{
		opCodeIndex: [0x100]*CpuInstructionData{},
	}

	for _, data := range data {
		instructionSet.opCodeIndex[data.opcode] = &data
	}

	return &instructionSet
}
