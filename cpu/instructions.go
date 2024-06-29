package cpu

type OpCode uint

type CpuInstructionData struct {
	opcode      OpCode
	mnemonic    Mnemonic
	action      func(cpu *Cpu65C02S)
	addressMode AddressMode
}

func (data *CpuInstructionData) OpCode() OpCode {
	return data.opcode
}

func (data *CpuInstructionData) Mnemonic() Mnemonic {
	return data.mnemonic
}

func (data *CpuInstructionData) Execute(cpu *Cpu65C02S) {
	data.action(cpu)
}

func (data *CpuInstructionData) AddressMode() AddressMode {
	return data.addressMode
}

// ----------------------------------------------------------------------

type CpuInstructionSet struct {
	opCodeIndex    map[OpCode]*CpuInstructionData
	mnemonicsIndex map[Mnemonic]*CpuInstructionData
}

func (instruction *CpuInstructionSet) GetByMnemonic(mnemonic Mnemonic) *CpuInstructionData {
	return instruction.mnemonicsIndex[mnemonic]
}

func (instruction *CpuInstructionSet) GetByOpCode(opCode OpCode) *CpuInstructionData {
	return instruction.opCodeIndex[opCode]
}

func CreateInstructionSet() *CpuInstructionSet {
	var data = []CpuInstructionData{
		{0x00, BRK, ActionBRK, AddressModeImplicit},
		{0x01, ORA, ActionORA, AddressModeZeroPageIndexedIndirectX},
		{0x04, TSB, ActionTSB, AddressModeZeroPageRMW},
		{0x05, ORA, ActionORA, AddressModeZeroPage},
		{0x06, ASL, ActionASL, AddressModeZeroPageRMW},
		{0x07, RMB0, ActionRMB, AddressModeZeroPageRMW},
		{0x08, PHP, ActionPHP, AddressModeImplicit},
		{0x09, ORA, ActionORA, AddressModeImmediate},
		{0x0A, ASL, ActionASL, AddressModeAccumulator},
		{0x0C, TSB, ActionTSB, AddressModeAbsoluteRMW},
		{0x0D, ORA, ActionORA, AddressModeAbsolute},
		{0x0E, ASL, ActionASL, AddressModeAbsoluteRMW},
		{0x0F, BBR0, ActionBBR, AddressModeRelative},

		{0x10, BPL, ActionBPL, AddressModeRelative},
		{0x11, ORA, ActionORA, AddressModeZeroPageIndirectIndexedY},
		{0x12, ORA, ActionORA, AddressModeIndirectZeroPage},
		{0x14, TRB, ActionTRB, AddressModeZeroPageRMW},
		{0x15, ORA, ActionORA, AddressModeZeroPageX},
		{0x16, ASL, ActionASL, AddressModeZeroPageXRMW},
		{0x17, RMB1, ActionRMB, AddressModeZeroPageRMW},
		{0x18, CLC, ActionCLC, AddressModeImplicit},
		{0x19, ORA, ActionORA, AddressModeAbsoluteY},
		{0x1A, INC, ActionINC, AddressModeAccumulator},
		{0x1C, TRB, ActionTRB, AddressModeAbsoluteRMW},
		{0x1D, ORA, ActionORA, AddressModeAbsoluteX},
		{0x1E, ASL, ActionASL, AddressModeAbsoluteXRMW},
		{0x1F, BBR1, ActionBBR, AddressModeRelative},

		{0x20, JSR, ActionJSR, AddressModeAbsolute},
		{0x21, AND, ActionAND, AddressModeZeroPageIndexedIndirectX},
		{0x24, BIT, ActionBIT, AddressModeZeroPage},
		{0x25, AND, ActionAND, AddressModeZeroPage},
		{0x26, ROL, ActionROL, AddressModeZeroPageRMW},
		{0x27, RMB2, ActionRMB, AddressModeZeroPageRMW},
		{0x28, PLP, ActionPLP, AddressModeImplicit},
		{0x29, AND, ActionAND, AddressModeImmediate},
		{0x2A, ROL, ActionROL, AddressModeAccumulator},
		{0x2C, BIT, ActionBIT, AddressModeAbsolute},
		{0x2D, AND, ActionAND, AddressModeAbsolute},
		{0x2E, ROL, ActionROL, AddressModeAbsoluteRMW},
		{0x2F, BBR2, ActionBBR, AddressModeRelative},

		{0x30, BMI, ActionBMI, AddressModeRelative},
		{0x31, AND, ActionAND, AddressModeZeroPageIndirectIndexedY},
		{0x32, AND, ActionAND, AddressModeIndirectZeroPage},
		{0x34, BIT, ActionBIT, AddressModeZeroPageX},
		{0x35, AND, ActionAND, AddressModeZeroPageX},
		{0x36, ROL, ActionROL, AddressModeZeroPageXRMW},
		{0x37, RMB3, ActionRMB, AddressModeZeroPageRMW},
		{0x38, SEC, ActionSEC, AddressModeImplicit},
		{0x39, AND, ActionAND, AddressModeAbsoluteY},
		{0x3A, DEC, ActionDEC, AddressModeAccumulator},
		{0x3C, BIT, ActionBIT, AddressModeAbsoluteX},
		{0x3D, AND, ActionAND, AddressModeAbsoluteX},
		{0x3E, ROL, ActionROL, AddressModeAbsoluteXRMW},
		{0x3F, BBR3, ActionBBR, AddressModeRelative},

		{0x40, RTI, ActionRTI, AddressModeImplicit},
		{0x41, EOR, ActionEOR, AddressModeZeroPageIndexedIndirectX},
		{0x45, EOR, ActionEOR, AddressModeZeroPage},
		{0x46, LSR, ActionLSR, AddressModeZeroPageRMW},
		{0x47, RMB4, ActionRMB, AddressModeZeroPageRMW},
		{0x48, PHA, ActionPHA, AddressModeImplicit},
		{0x49, EOR, ActionEOR, AddressModeImmediate},
		{0x4A, LSR, ActionLSR, AddressModeAccumulator},
		{0x4C, JMP, ActionJMP, AddressModeAbsoluteJump},
		{0x4D, EOR, ActionEOR, AddressModeAbsolute},
		{0x4E, LSR, ActionLSR, AddressModeAbsoluteRMW},
		{0x4F, BBR4, ActionBBR, AddressModeRelative},

		{0x50, BVC, ActionBVC, AddressModeRelative},
		{0x51, EOR, ActionEOR, AddressModeZeroPageIndirectIndexedY},
		{0x52, EOR, ActionEOR, AddressModeIndirectZeroPage},
		{0x55, EOR, ActionEOR, AddressModeZeroPageX},
		{0x56, LSR, ActionLSR, AddressModeZeroPageXRMW},
		{0x57, RMB5, ActionRMB, AddressModeZeroPageRMW},
		{0x58, CLI, ActionCLI, AddressModeImplicit},
		{0x59, EOR, ActionEOR, AddressModeAbsoluteY},
		{0x5A, PHY, ActionPHY, AddressModeImplicit},
		{0x5D, EOR, ActionEOR, AddressModeAbsoluteX},
		{0x5E, LSR, ActionLSR, AddressModeAbsoluteXRMW},
		{0x5F, BBR5, ActionBBR, AddressModeRelative},

		{0x60, RTS, ActionRTS, AddressModeImplicit},
		{0x61, ADC, ActionADC, AddressModeZeroPageIndexedIndirectX},
		{0x64, STZ, ActionSTZ, AddressModeZeroPage},
		{0x65, ADC, ActionADC, AddressModeZeroPage},
		{0x66, ROR, ActionROR, AddressModeZeroPageRMW},
		{0x67, RMB6, ActionRMB, AddressModeZeroPageRMW},
		{0x68, PLA, ActionPLA, AddressModeImplicit},
		{0x69, ADC, ActionADC, AddressModeImmediate},
		{0x6A, ROR, ActionROR, AddressModeAccumulator},
		{0x6C, JMP, ActionJMP, AddressModeIndirect},
		{0x6D, ADC, ActionADC, AddressModeAbsolute},
		{0x6E, ROR, ActionROR, AddressModeAbsoluteRMW},
		{0x6F, BBR6, ActionBBR, AddressModeRelative},

		{0x70, BVS, ActionBVS, AddressModeRelative},
		{0x71, ADC, ActionADC, AddressModeZeroPageIndirectIndexedY},
		{0x72, ADC, ActionADC, AddressModeIndirectZeroPage},
		{0x74, STZ, ActionSTZ, AddressModeZeroPageX},
		{0x75, ADC, ActionADC, AddressModeZeroPageX},
		{0x76, ROR, ActionROR, AddressModeZeroPageXRMW},
		{0x77, RMB7, ActionRMB, AddressModeZeroPageRMW},
		{0x78, SEI, ActionSEI, AddressModeImplicit},
		{0x79, ADC, ActionADC, AddressModeAbsoluteY},
		{0x7A, PLY, ActionPLY, AddressModeImplicit},
		{0x7C, JMP, ActionJMP, AddressModeAbsoluteIndexedIndirect},
		{0x7D, ADC, ActionADC, AddressModeAbsoluteX},
		{0x7E, ROR, ActionROR, AddressModeAbsoluteXRMW},
		{0x7F, BBR7, ActionBBR, AddressModeRelative},

		{0x80, BRA, ActionBRA, AddressModeRelative},
		{0x81, STA, ActionSTA, AddressModeZeroPageIndexedIndirectXW},
		{0x84, STY, ActionSTY, AddressModeZeroPageW},
		{0x85, STA, ActionSTA, AddressModeZeroPageW},
		{0x86, STX, ActionSTX, AddressModeZeroPageW},
		{0x87, SMB0, ActionSMB, AddressModeZeroPageRMW},
		{0x88, DEY, ActionDEY, AddressModeImplicit},
		{0x89, BIT, ActionBIT, AddressModeImmediate},
		{0x8A, TXA, ActionTXA, AddressModeImplicit},
		{0x8C, STY, ActionSTY, AddressModeAbsoluteW},
		{0x8D, STA, ActionSTA, AddressModeAbsoluteW},
		{0x8E, STX, ActionSTX, AddressModeAbsoluteW},
		{0x8F, BBS0, ActionBBS, AddressModeRelative},

		{0x90, BCC, ActionBCC, AddressModeRelative},
		{0x91, STA, ActionSTA, AddressModeZeroPageIndirectIndexedYW},
		{0x92, STA, ActionSTA, AddressModeIndirectZeroPageW},
		{0x94, STY, ActionSTY, AddressModeZeroPageXW},
		{0x95, STA, ActionSTA, AddressModeZeroPageXW},
		{0x96, STX, ActionSTX, AddressModeZeroPageY},
		{0x97, SMB1, ActionSMB, AddressModeZeroPageRMW},
		{0x98, TYA, ActionTYA, AddressModeImplicit},
		{0x99, STA, ActionSTA, AddressModeAbsoluteYW},
		{0x9A, TXS, ActionTXS, AddressModeImplicit},
		{0x9C, STZ, ActionSTZ, AddressModeAbsolute},
		{0x9D, STA, ActionSTA, AddressModeAbsoluteXW},
		{0x9E, STZ, ActionSTZ, AddressModeAbsoluteX},
		{0x9F, BBS1, ActionBBS, AddressModeRelative},

		{0xA0, LDY, ActionLDY, AddressModeImmediate},
		{0xA1, LDA, ActionLDA, AddressModeZeroPageIndexedIndirectX},
		{0xA2, LDX, ActionLDX, AddressModeImmediate},
		{0xA4, LDY, ActionLDY, AddressModeZeroPage},
		{0xA5, LDA, ActionLDA, AddressModeZeroPage},
		{0xA6, LDX, ActionLDX, AddressModeZeroPage},
		{0xA7, SMB2, ActionSMB, AddressModeZeroPageRMW},
		{0xA8, TAY, ActionTAY, AddressModeImplicit},
		{0xA9, LDA, ActionLDA, AddressModeImmediate},
		{0xAA, TAX, ActionTAX, AddressModeImplicit},
		{0xAC, LDY, ActionLDY, AddressModeAbsolute},
		{0xAD, LDA, ActionLDA, AddressModeAbsolute},
		{0xAE, LDX, ActionLDX, AddressModeAbsolute},
		{0xAF, BBS2, ActionBBS, AddressModeRelative},

		{0xB0, BCS, ActionBCS, AddressModeRelative},
		{0xB1, LDA, ActionLDA, AddressModeZeroPageIndirectIndexedY},
		{0xB2, LDA, ActionLDA, AddressModeIndirectZeroPage},
		{0xB4, LDY, ActionLDY, AddressModeZeroPageX},
		{0xB5, LDA, ActionLDA, AddressModeZeroPageX},
		{0xB6, LDX, ActionLDX, AddressModeZeroPageY},
		{0xB7, SMB3, ActionSMB, AddressModeZeroPageRMW},
		{0xB8, CLV, ActionCLV, AddressModeImplicit},
		{0xB9, LDA, ActionLDA, AddressModeAbsoluteY},
		{0xBA, TSX, ActionTSX, AddressModeImplicit},
		{0xBC, LDY, ActionLDY, AddressModeAbsoluteX},
		{0xBD, LDA, ActionLDA, AddressModeAbsoluteX},
		{0xBE, LDX, ActionLDX, AddressModeAbsoluteY},
		{0xBF, BBS3, ActionBBS, AddressModeRelative},

		{0xC0, CPY, ActionCPY, AddressModeImmediate},
		{0xC1, CMP, ActionCMP, AddressModeZeroPageIndexedIndirectX},
		{0xC4, CPY, ActionCPY, AddressModeZeroPage},
		{0xC5, CMP, ActionCMP, AddressModeZeroPage},
		{0xC6, DEC, ActionDEC, AddressModeZeroPageRMW},
		{0xC7, SMB4, ActionSMB, AddressModeZeroPageRMW},
		{0xC8, INY, ActionINY, AddressModeImplicit},
		{0xC9, CMP, ActionCMP, AddressModeImmediate},
		{0xCA, DEX, ActionDEX, AddressModeImplicit},
		{0xCB, WAI, ActionWAI, AddressModeImplicit},
		{0xCC, CPY, ActionCPY, AddressModeAbsolute},
		{0xCD, CMP, ActionCMP, AddressModeAbsolute},
		{0xCE, DEC, ActionDEC, AddressModeAbsoluteRMW},
		{0xCF, BBS4, ActionBBS, AddressModeRelative},

		{0xD0, BNE, ActionBNE, AddressModeRelative},
		{0xD1, CMP, ActionCMP, AddressModeZeroPageIndirectIndexedY},
		{0xD2, CMP, ActionCMP, AddressModeIndirectZeroPage},
		{0xD5, CMP, ActionCMP, AddressModeZeroPageX},
		{0xD6, DEC, ActionDEC, AddressModeZeroPageXRMW},
		{0xD7, SMB5, ActionSMB, AddressModeZeroPageRMW},
		{0xD8, CLD, ActionCLD, AddressModeImplicit},
		{0xD9, CMP, ActionCMP, AddressModeAbsoluteY},
		{0xDA, PHX, ActionPHX, AddressModeImplicit},
		{0xDB, STP, ActionSTP, AddressModeImplicit},
		{0xDD, CMP, ActionCMP, AddressModeAbsoluteX},
		{0xDE, DEC, ActionDEC, AddressModeAbsoluteXRMW},
		{0xDF, BBS5, ActionBBS, AddressModeRelative},

		{0xE0, CPX, ActionCPX, AddressModeImmediate},
		{0xE1, SBC, ActionSBC, AddressModeZeroPageIndexedIndirectX},
		{0xE4, CPX, ActionCPX, AddressModeZeroPage},
		{0xE5, SBC, ActionSBC, AddressModeZeroPage},
		{0xE6, INC, ActionINC, AddressModeZeroPageRMW},
		{0xE7, SMB6, ActionSMB, AddressModeZeroPageRMW},
		{0xE8, INX, ActionINX, AddressModeImplicit},
		{0xE9, SBC, ActionSBC, AddressModeImmediate},
		{0xEA, NOP, ActionNOP, AddressModeImplicit},
		{0xEC, CPX, ActionCPX, AddressModeAbsolute},
		{0xED, SBC, ActionSBC, AddressModeAbsolute},
		{0xEE, INC, ActionINC, AddressModeAbsoluteRMW},
		{0xEF, BBS6, ActionBBS, AddressModeRelative},

		{0xF0, BEQ, ActionBEQ, AddressModeRelative},
		{0xF1, SBC, ActionSBC, AddressModeZeroPageIndirectIndexedY},
		{0xF2, SBC, ActionSBC, AddressModeIndirectZeroPage},
		{0xF5, SBC, ActionSBC, AddressModeZeroPageX},
		{0xF6, INC, ActionINC, AddressModeZeroPageXRMW},
		{0xF7, SMB7, ActionSMB, AddressModeZeroPageRMW},
		{0xF8, SED, ActionSED, AddressModeImplicit},
		{0xF9, SBC, ActionSBC, AddressModeAbsoluteY},
		{0xFA, PLX, ActionPLX, AddressModeImplicit},
		{0xFD, SBC, ActionSBC, AddressModeAbsoluteX},
		{0xFE, INC, ActionINC, AddressModeAbsoluteXRMW},
		{0xFF, BBS7, ActionBBS, AddressModeRelative},
	}

	instructionSet := CpuInstructionSet{
		opCodeIndex:    map[OpCode]*CpuInstructionData{},
		mnemonicsIndex: map[Mnemonic]*CpuInstructionData{},
	}

	for _, data := range data {
		instructionSet.mnemonicsIndex[data.mnemonic] = &data
		instructionSet.opCodeIndex[data.opcode] = &data
	}

	return &instructionSet
}
