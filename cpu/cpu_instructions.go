package cpu

type CpuInstructionData struct {
	opcode      uint8
	Mnemonic    string
	AddressMode AddressMode
}

type CpuInstructionSet struct {
	opCodeIndex    map[uint8]CpuInstructionData
	mnemonicsIndex map[string]CpuInstructionData
}

func (instruction *CpuInstructionSet) GetInstructionByMnemonic(mnemonic string) CpuInstructionData {
	return instruction.mnemonicsIndex[mnemonic]
}

func (instruction *CpuInstructionSet) GetInstructionByOpCode(opCode uint8) CpuInstructionData {
	return instruction.opCodeIndex[opCode]
}

func CreateInstructionSet() *CpuInstructionSet {
	var data = []CpuInstructionData{
		{0x00, "BRK", Stack},
		{0x01, "ORA", ZeroPageIndexedIndirect},
		{0x04, "TSB", ZeroPage},
		{0x05, "ORA", ZeroPage},
		{0x06, "ASL", ZeroPage},
		{0x07, "RMB0", ZeroPage},
		{0x08, "PHP", Stack},
		{0x09, "ORA", Immediate},
		{0x0A, "ASL", Accumulator},
		{0x0C, "TSB", Absolute},
		{0x0D, "ORA", Absolute},
		{0x0E, "ASL", Absolute},
		{0x0F, "BBR0", ProgramCounterRelative},

		{0x10, "BPL", ProgramCounterRelative},
		{0x11, "ORA", ZeroPageIndirectIndexedY},
		{0x12, "ORA", ZeroPageIndirect},
		{0x14, "TRB", ZeroPage},
		{0x15, "ORA", ZeroPageIndexedX},
		{0x16, "ASL", ZeroPageIndexedX},
		{0x17, "RMB1", ZeroPage},
		{0x18, "CLC", Implied},
		{0x19, "ORA", AbsoluteIndexedY},
		{0x1A, "INC", Accumulator},
		{0x1C, "TRB", Absolute},
		{0x1D, "ORA", AbsoluteIndexedX},
		{0x1E, "ASL", AbsoluteIndexedX},
		{0x1F, "BBR1", ProgramCounterRelative},

		{0x20, "JSR", Absolute},
		{0x21, "AND", ZeroPageIndexedIndirect},
		{0x24, "BIT", ZeroPage},
		{0x25, "AND", ZeroPage},
		{0x26, "ROL", ZeroPage},
		{0x27, "RMB2", ZeroPage},
		{0x28, "PLP", Stack},
		{0x29, "AND", Immediate},
		{0x2A, "ROL", Accumulator},
		{0x2C, "BIT", Absolute},
		{0x2D, "AND", Absolute},
		{0x2E, "ROL", Absolute},
		{0x2F, "BBR2", ProgramCounterRelative},

		{0x30, "BMI", ProgramCounterRelative},
		{0x31, "AND", ZeroPageIndirectIndexedY},
		{0x32, "AND", ZeroPageIndirect},
		{0x34, "BIT", ZeroPageIndexedX},
		{0x35, "AND", ZeroPageIndexedX},
		{0x36, "ROL", ZeroPageIndexedX},
		{0x37, "RMB3", ZeroPage},
		{0x38, "SEC", Implied},
		{0x39, "AND", AbsoluteIndexedY},
		{0x3A, "DEC", Accumulator},
		{0x3C, "BIT", AbsoluteIndexedX},
		{0x3D, "AND", AbsoluteIndexedX},
		{0x3E, "ROL", AbsoluteIndexedX},
		{0x3F, "BBR3", ProgramCounterRelative},

		{0x40, "RTI", Stack},
		{0x41, "EOR", ZeroPageIndexedIndirect},
		{0x45, "EOR", ZeroPage},
		{0x46, "LSR", ZeroPage},
		{0x47, "RMB4", ZeroPage},
		{0x48, "PHA", Stack},
		{0x49, "EOR", Immediate},
		{0x4A, "LSR", Accumulator},
		{0x4C, "JMP", Absolute},
		{0x4D, "EOR", Absolute},
		{0x4E, "LSR", Absolute},
		{0x4F, "BBR4", ProgramCounterRelative},

		{0x50, "BVC", ProgramCounterRelative},
		{0x51, "EOR", ZeroPageIndirectIndexedY},
		{0x52, "EOR", ZeroPageIndirect},
		{0x55, "EOR", ZeroPageIndexedX},
		{0x56, "LSR", ZeroPageIndexedX},
		{0x57, "RMB5", ZeroPage},
		{0x58, "CLI", Implied},
		{0x59, "EOR", AbsoluteIndexedY},
		{0x5A, "PHY", Stack},
		{0x5D, "EOR", AbsoluteIndexedX},
		{0x5E, "LSR", AbsoluteIndexedX},
		{0x5F, "BBR5", ProgramCounterRelative},

		{0x60, "RTS", Stack},
		{0x61, "ADC", ZeroPageIndexedIndirect},
		{0x64, "STZ", ZeroPage},
		{0x65, "ADC", ZeroPage},
		{0x66, "ROR", ZeroPage},
		{0x67, "RMB6", ZeroPage},
		{0x68, "PLA", Stack},
		{0x69, "ADC", Immediate},
		{0x6A, "ROR", Accumulator},
		{0x6C, "JMP", AbsoluteIndirect},
		{0x6D, "ADC", Absolute},
		{0x6E, "ROR", Absolute},
		{0x6F, "BBR6", ProgramCounterRelative},

		{0x70, "BVS", ProgramCounterRelative},
		{0x71, "ADC", ZeroPageIndirectIndexedY},
		{0x72, "ADC", ZeroPageIndirect},
		{0x74, "STZ", ZeroPageIndexedX},
		{0x75, "ADC", ZeroPageIndexedX},
		{0x76, "ROR", ZeroPageIndexedX},
		{0x77, "RMB7", ZeroPage},
		{0x78, "SEI", Implied},
		{0x79, "ADC", AbsoluteIndexedY},
		{0x7A, "PLY", Stack},
		{0x7C, "JMP", AbsoluteIndexedIndirect},
		{0x7D, "ADC", AbsoluteIndexedX},
		{0x7E, "ROR", AbsoluteIndexedX},
		{0x7F, "BBR7", ProgramCounterRelative},

		{0x80, "BRA", ProgramCounterRelative},
		{0x81, "STA", ZeroPageIndexedIndirect},
		{0x84, "STY", ZeroPage},
		{0x85, "STA", ZeroPage},
		{0x86, "STX", ZeroPage},
		{0x87, "SMB0", ZeroPage},
		{0x88, "DEY", Implied},
		{0x89, "BIT", Immediate},
		{0x8A, "TXA", Implied},
		{0x8C, "STY", Absolute},
		{0x8D, "STA", Absolute},
		{0x8E, "STX", Absolute},
		{0x8F, "BBS0", ProgramCounterRelative},

		{0x90, "BCC", ProgramCounterRelative},
		{0x91, "STA", ZeroPageIndirectIndexedY},
		{0x92, "STA", ZeroPageIndirect},
		{0x94, "STY", ZeroPageIndexedX},
		{0x95, "STA", ZeroPageIndexedX},
		{0x96, "STX", ZeroPageIndexedX},
		{0x97, "SMB1", ZeroPage},
		{0x98, "TYA", Implied},
		{0x99, "STA", AbsoluteIndexedY},
		{0x9A, "TXS", Implied},
		{0x9C, "STZ", Absolute},
		{0x9D, "STA", AbsoluteIndexedX},
		{0x9E, "STZ", AbsoluteIndexedX},
		{0x9F, "BBS1", ProgramCounterRelative},

		{0xA0, "LDY", Immediate},
		{0xA1, "LDA", ZeroPageIndexedIndirect},
		{0xA2, "LDX", Immediate},
		{0xA4, "LDY", ZeroPage},
		{0xA5, "LDA", ZeroPage},
		{0xA6, "LDX", ZeroPage},
		{0xA7, "SMB2", ZeroPage},
		{0xA8, "TAY", Implied},
		{0xA9, "LDA", Immediate},
		{0xAA, "TAX", Implied},
		{0xAC, "LDY", Absolute},
		{0xAD, "LDA", Absolute},
		{0xAE, "LDX", Absolute},
		{0xAF, "BBS2", ProgramCounterRelative},

		{0xB0, "BCS", ProgramCounterRelative},
		{0xB1, "LDA", ZeroPageIndirectIndexedY},
		{0xB2, "LDA", ZeroPageIndirect},
		{0xB4, "LDY", ZeroPageIndexedX},
		{0xB5, "LDA", ZeroPageIndexedX},
		{0xB6, "LDX", ZeroPageIndexedX},
		{0xB7, "SMB3", ZeroPage},
		{0xB8, "CLV", Implied},
		{0xB9, "LDA", AbsoluteIndexedY},
		{0xBA, "TSX", Implied},
		{0xBC, "LDY", AbsoluteIndexedX},
		{0xBD, "LDA", AbsoluteIndexedX},
		{0xBE, "LDX", AbsoluteIndexedY},
		{0xBF, "BBS3", ProgramCounterRelative},

		{0xC0, "CPY", Immediate},
		{0xC1, "CMP", ZeroPageIndexedIndirect},
		{0xC4, "CPY", ZeroPage},
		{0xC5, "CMP", ZeroPage},
		{0xC6, "DEC", ZeroPage},
		{0xC7, "SMB4", ZeroPage},
		{0xC8, "INY", Implied},
		{0xC9, "CMP", Immediate},
		{0xCA, "DEX", Implied},
		{0xCB, "WAI", Implied},
		{0xCC, "CPY", Absolute},
		{0xCD, "CMP", Absolute},
		{0xCE, "DEC", Absolute},
		{0xCF, "BBS4", ProgramCounterRelative},

		{0xD0, "BNE", ProgramCounterRelative},
		{0xD1, "CMP", ZeroPageIndirectIndexedY},
		{0xD2, "CMP", ZeroPageIndirect},
		{0xD5, "CMP", ZeroPageIndexedX},
		{0xD6, "DEC", ZeroPageIndexedX},
		{0xD7, "SMB5", ZeroPage},
		{0xD8, "CLD", Implied},
		{0xD9, "CMP", AbsoluteIndexedY},
		{0xDA, "PHX", Stack},
		{0xDB, "STP", Implied},
		{0xDD, "CMP", AbsoluteIndexedX},
		{0xDE, "DEC", AbsoluteIndexedX},
		{0xDF, "BBS5", ProgramCounterRelative},

		{0xE0, "CPX", Immediate},
		{0xE1, "SBC", ZeroPageIndexedIndirect},
		{0xE4, "CPX", ZeroPage},
		{0xE5, "SBC", ZeroPage},
		{0xE6, "INC", ZeroPage},
		{0xE7, "SMB6", ZeroPage},
		{0xE8, "INX", Implied},
		{0xE9, "SBC", Immediate},
		{0xEA, "NOP", Implied},
		{0xEC, "CPX", Absolute},
		{0xED, "SBC", Absolute},
		{0xEE, "INC", Absolute},
		{0xEF, "BBS6", ProgramCounterRelative},

		{0xF0, "BEQ", ProgramCounterRelative},
		{0xF1, "SBC", ZeroPageIndirectIndexedY},
		{0xF2, "SBC", ZeroPageIndirect},
		{0xF5, "SBC", ZeroPageIndexedX},
		{0xF6, "INC", ZeroPageIndexedX},
		{0xF7, "SMB7", ZeroPage},
		{0xF8, "SED", Implied},
		{0xF9, "SBC", AbsoluteIndexedY},
		{0xFA, "PLX", Stack},
		{0xFD, "SBC", AbsoluteIndexedX},
		{0xFE, "INC", AbsoluteIndexedX},
		{0xFF, "BBS7", ProgramCounterRelative},
	}

	instructionSet := CpuInstructionSet{
		opCodeIndex:    map[uint8]CpuInstructionData{},
		mnemonicsIndex: map[string]CpuInstructionData{},
	}

	for _, data := range data {
		instructionSet.mnemonicsIndex[data.Mnemonic] = data
		instructionSet.opCodeIndex[data.opcode] = data
	}

	return &instructionSet
}
