package cpu

type OpCode uint

type CpuInstructionData struct {
	opcode      OpCode
	mnemonic    Mnemonic
	addressMode AddressMode
}

func (data *CpuInstructionData) OpCode() OpCode {
	return data.opcode
}

func (data *CpuInstructionData) Mnemonic() Mnemonic {
	return data.mnemonic
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
		{0x00, BRK, AddressModeImmediate},
		{0x01, ORA, AddressModeZeroPageIndexedIndirectX},
		{0x04, TSB, AddressModeZeroPage},
		{0x05, ORA, AddressModeZeroPage},
		{0x06, ASL, AddressModeZeroPage},
		{0x07, RMB0, AddressModeZeroPage},
		{0x08, PHP, AddressModeImplicit},
		{0x09, ORA, AddressModeImmediate},
		{0x0A, ASL, AddressModeAccumulator},
		{0x0C, TSB, AddressModeAbsolute},
		{0x0D, ORA, AddressModeAbsolute},
		{0x0E, ASL, AddressModeAbsolute},
		{0x0F, BBR0, AddressModeRelative},

		{0x10, BPL, AddressModeRelative},
		{0x11, ORA, AddressModeZeroPageIndirectIndexedY},
		{0x12, ORA, AddressModeIndirectZeroPage},
		{0x14, TRB, AddressModeZeroPage},
		{0x15, ORA, AddressModeZeroPageX},
		{0x16, ASL, AddressModeZeroPageX},
		{0x17, RMB1, AddressModeZeroPage},
		{0x18, CLC, AddressModeImplicit},
		{0x19, ORA, AddressModeAbsoluteY},
		{0x1A, INC, AddressModeAccumulator},
		{0x1C, TRB, AddressModeAbsolute},
		{0x1D, ORA, AddressModeAbsoluteX},
		{0x1E, ASL, AddressModeAbsoluteX},
		{0x1F, BBR1, AddressModeRelative},

		{0x20, JSR, AddressModeAbsolute},
		{0x21, AND, AddressModeZeroPageIndexedIndirectX},
		{0x24, BIT, AddressModeZeroPage},
		{0x25, AND, AddressModeZeroPage},
		{0x26, ROL, AddressModeZeroPage},
		{0x27, RMB2, AddressModeZeroPage},
		{0x28, PLP, AddressModeImplicit},
		{0x29, AND, AddressModeImmediate},
		{0x2A, ROL, AddressModeAccumulator},
		{0x2C, BIT, AddressModeAbsolute},
		{0x2D, AND, AddressModeAbsolute},
		{0x2E, ROL, AddressModeAbsolute},
		{0x2F, BBR2, AddressModeRelative},

		{0x30, BMI, AddressModeRelative},
		{0x31, AND, AddressModeZeroPageIndirectIndexedY},
		{0x32, AND, AddressModeIndirect},
		{0x34, BIT, AddressModeZeroPageX},
		{0x35, AND, AddressModeZeroPageX},
		{0x36, ROL, AddressModeZeroPageX},
		{0x37, RMB3, AddressModeZeroPage},
		{0x38, SEC, AddressModeImplicit},
		{0x39, AND, AddressModeAbsoluteY},
		{0x3A, DEC, AddressModeAccumulator},
		{0x3C, BIT, AddressModeAbsoluteX},
		{0x3D, AND, AddressModeAbsoluteX},
		{0x3E, ROL, AddressModeAbsoluteX},
		{0x3F, BBR3, AddressModeRelative},

		{0x40, RTI, AddressModeImplicit},
		{0x41, EOR, AddressModeZeroPageIndexedIndirectX},
		{0x45, EOR, AddressModeZeroPage},
		{0x46, LSR, AddressModeZeroPage},
		{0x47, RMB4, AddressModeZeroPage},
		{0x48, PHA, AddressModeImplicit},
		{0x49, EOR, AddressModeImmediate},
		{0x4A, LSR, AddressModeAccumulator},
		{0x4C, JMP, AddressModeAbsolute},
		{0x4D, EOR, AddressModeAbsolute},
		{0x4E, LSR, AddressModeAbsolute},
		{0x4F, BBR4, AddressModeRelative},

		{0x50, BVC, AddressModeRelative},
		{0x51, EOR, AddressModeZeroPageIndirectIndexedY},
		{0x52, EOR, AddressModeIndirect},
		{0x55, EOR, AddressModeZeroPageX},
		{0x56, LSR, AddressModeZeroPageX},
		{0x57, RMB5, AddressModeZeroPage},
		{0x58, CLI, AddressModeImplicit},
		{0x59, EOR, AddressModeAbsoluteY},
		{0x5A, PHY, AddressModeImplicit},
		{0x5D, EOR, AddressModeAbsoluteX},
		{0x5E, LSR, AddressModeAbsoluteX},
		{0x5F, BBR5, AddressModeRelative},

		{0x60, RTS, AddressModeImplicit},
		{0x61, ADC, AddressModeZeroPageIndexedIndirectX},
		{0x64, STZ, AddressModeZeroPage},
		{0x65, ADC, AddressModeZeroPage},
		{0x66, ROR, AddressModeZeroPage},
		{0x67, RMB6, AddressModeZeroPage},
		{0x68, PLA, AddressModeImplicit},
		{0x69, ADC, AddressModeImmediate},
		{0x6A, ROR, AddressModeAccumulator},
		{0x6C, JMP, AddressModeIndirect},
		{0x6D, ADC, AddressModeAbsolute},
		{0x6E, ROR, AddressModeAbsolute},
		{0x6F, BBR6, AddressModeRelative},

		{0x70, BVS, AddressModeRelative},
		{0x71, ADC, AddressModeZeroPageIndirectIndexedY},
		{0x72, ADC, AddressModeIndirect},
		{0x74, STZ, AddressModeZeroPageX},
		{0x75, ADC, AddressModeZeroPageX},
		{0x76, ROR, AddressModeZeroPageX},
		{0x77, RMB7, AddressModeZeroPage},
		{0x78, SEI, AddressModeImplicit},
		{0x79, ADC, AddressModeAbsoluteY},
		{0x7A, PLY, AddressModeImplicit},
		{0x7C, JMP, AddressModeAbsoluteIndexedIndirect},
		{0x7D, ADC, AddressModeAbsoluteX},
		{0x7E, ROR, AddressModeAbsoluteX},
		{0x7F, BBR7, AddressModeRelative},

		{0x80, BRA, AddressModeRelative},
		{0x81, STA, AddressModeZeroPageIndexedIndirectX},
		{0x84, STY, AddressModeZeroPage},
		{0x85, STA, AddressModeZeroPage},
		{0x86, STX, AddressModeZeroPage},
		{0x87, SMB0, AddressModeZeroPage},
		{0x88, DEY, AddressModeImplicit},
		{0x89, BIT, AddressModeImmediate},
		{0x8A, TXA, AddressModeImplicit},
		{0x8C, STY, AddressModeAbsolute},
		{0x8D, STA, AddressModeAbsolute},
		{0x8E, STX, AddressModeAbsolute},
		{0x8F, BBS0, AddressModeRelative},

		{0x90, BCC, AddressModeRelative},
		{0x91, STA, AddressModeZeroPageIndirectIndexedY},
		{0x92, STA, AddressModeIndirect},
		{0x94, STY, AddressModeZeroPageX},
		{0x95, STA, AddressModeZeroPageX},
		{0x96, STX, AddressModeZeroPageX},
		{0x97, SMB1, AddressModeZeroPage},
		{0x98, TYA, AddressModeImplicit},
		{0x99, STA, AddressModeAbsoluteY},
		{0x9A, TXS, AddressModeImplicit},
		{0x9C, STZ, AddressModeAbsolute},
		{0x9D, STA, AddressModeAbsoluteX},
		{0x9E, STZ, AddressModeAbsoluteX},
		{0x9F, BBS1, AddressModeRelative},

		{0xA0, LDY, AddressModeImmediate},
		{0xA1, LDA, AddressModeZeroPageIndexedIndirectX},
		{0xA2, LDX, AddressModeImmediate},
		{0xA4, LDY, AddressModeZeroPage},
		{0xA5, LDA, AddressModeZeroPage},
		{0xA6, LDX, AddressModeZeroPage},
		{0xA7, SMB2, AddressModeZeroPage},
		{0xA8, TAY, AddressModeImplicit},
		{0xA9, LDA, AddressModeImmediate},
		{0xAA, TAX, AddressModeImplicit},
		{0xAC, LDY, AddressModeAbsolute},
		{0xAD, LDA, AddressModeAbsolute},
		{0xAE, LDX, AddressModeAbsolute},
		{0xAF, BBS2, AddressModeRelative},

		{0xB0, BCS, AddressModeRelative},
		{0xB1, LDA, AddressModeZeroPageIndirectIndexedY},
		{0xB2, LDA, AddressModeIndirect},
		{0xB4, LDY, AddressModeZeroPageX},
		{0xB5, LDA, AddressModeZeroPageX},
		{0xB6, LDX, AddressModeZeroPageX},
		{0xB7, SMB3, AddressModeZeroPage},
		{0xB8, CLV, AddressModeImplicit},
		{0xB9, LDA, AddressModeAbsoluteY},
		{0xBA, TSX, AddressModeImplicit},
		{0xBC, LDY, AddressModeAbsoluteX},
		{0xBD, LDA, AddressModeAbsoluteX},
		{0xBE, LDX, AddressModeAbsoluteY},
		{0xBF, BBS3, AddressModeRelative},

		{0xC0, CPY, AddressModeImmediate},
		{0xC1, CMP, AddressModeZeroPageIndexedIndirectX},
		{0xC4, CPY, AddressModeZeroPage},
		{0xC5, CMP, AddressModeZeroPage},
		{0xC6, DEC, AddressModeZeroPage},
		{0xC7, SMB4, AddressModeZeroPage},
		{0xC8, INY, AddressModeImplicit},
		{0xC9, CMP, AddressModeImmediate},
		{0xCA, DEX, AddressModeImplicit},
		{0xCB, WAI, AddressModeImplicit},
		{0xCC, CPY, AddressModeAbsolute},
		{0xCD, CMP, AddressModeAbsolute},
		{0xCE, DEC, AddressModeAbsolute},
		{0xCF, BBS4, AddressModeRelative},

		{0xD0, BNE, AddressModeRelative},
		{0xD1, CMP, AddressModeZeroPageIndirectIndexedY},
		{0xD2, CMP, AddressModeIndirect},
		{0xD5, CMP, AddressModeZeroPageX},
		{0xD6, DEC, AddressModeZeroPageX},
		{0xD7, SMB5, AddressModeZeroPage},
		{0xD8, CLD, AddressModeImplicit},
		{0xD9, CMP, AddressModeAbsoluteY},
		{0xDA, PHX, AddressModeImplicit},
		{0xDB, STP, AddressModeImplicit},
		{0xDD, CMP, AddressModeAbsoluteX},
		{0xDE, DEC, AddressModeAbsoluteX},
		{0xDF, BBS5, AddressModeRelative},

		{0xE0, CPX, AddressModeImmediate},
		{0xE1, SBC, AddressModeZeroPageIndexedIndirectX},
		{0xE4, CPX, AddressModeZeroPage},
		{0xE5, SBC, AddressModeZeroPage},
		{0xE6, INC, AddressModeZeroPage},
		{0xE7, SMB6, AddressModeZeroPage},
		{0xE8, INX, AddressModeImplicit},
		{0xE9, SBC, AddressModeImmediate},
		{0xEA, NOP, AddressModeImplicit},
		{0xEC, CPX, AddressModeAbsolute},
		{0xED, SBC, AddressModeAbsolute},
		{0xEE, INC, AddressModeAbsolute},
		{0xEF, BBS6, AddressModeRelative},

		{0xF0, BEQ, AddressModeRelative},
		{0xF1, SBC, AddressModeZeroPageIndirectIndexedY},
		{0xF2, SBC, AddressModeIndirect},
		{0xF5, SBC, AddressModeZeroPageX},
		{0xF6, INC, AddressModeZeroPageX},
		{0xF7, SMB7, AddressModeZeroPage},
		{0xF8, SED, AddressModeImplicit},
		{0xF9, SBC, AddressModeAbsoluteY},
		{0xFA, PLX, AddressModeImplicit},
		{0xFD, SBC, AddressModeAbsoluteX},
		{0xFE, INC, AddressModeAbsoluteX},
		{0xFF, BBS7, AddressModeRelative},
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
