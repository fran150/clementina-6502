package cpu

// Mnemonics are used in assembly langugate to represent the different OpCodes of all
// available instructions in a more human readable way. Each mnemonic can be written differently
// depending on the different address modes. For example LDA $C010 uses absolute address mode to
// fetch the byte from $C010 address and store it in the accumulator, it is the OpCode $AD.
// While LDA #$FF is the immediate version of LDA in where it directly stores $FF in the accumulator
// and the OpCode is $A9.
type Mnemonic string

const (
	ADC Mnemonic = "ADC"
	AND Mnemonic = "AND"
	ASL Mnemonic = "ASL"
	BCC Mnemonic = "BCC"
	BCS Mnemonic = "BCS"
	BEQ Mnemonic = "BEQ"
	BIT Mnemonic = "BIT"
	BMI Mnemonic = "BMI"
	BNE Mnemonic = "BNE"
	BPL Mnemonic = "BPL"
	BRA Mnemonic = "BRA"
	BRK Mnemonic = "BRK"
	BVC Mnemonic = "BVC"
	BVS Mnemonic = "BVS"
	CLC Mnemonic = "CLC"
	CLD Mnemonic = "CLD"
	CLI Mnemonic = "CLI"
	CLV Mnemonic = "CLV"
	CMP Mnemonic = "CMP"
	CPX Mnemonic = "CPX"
	CPY Mnemonic = "CPY"
	DEC Mnemonic = "DEC"
	DEX Mnemonic = "DEX"
	DEY Mnemonic = "DEY"
	EOR Mnemonic = "EOR"
	INC Mnemonic = "INC"
	INX Mnemonic = "INX"
	INY Mnemonic = "INY"
	JMP Mnemonic = "JMP"
	JSR Mnemonic = "JSR"
	LDA Mnemonic = "LDA"
	LDX Mnemonic = "LDX"
	LDY Mnemonic = "LDY"
	LSR Mnemonic = "LSR"
	NOP Mnemonic = "NOP"
	ORA Mnemonic = "ORA"
	PHA Mnemonic = "PHA"
	PHP Mnemonic = "PHP"
	PHX Mnemonic = "PHX"
	PHY Mnemonic = "PHY"
	PLA Mnemonic = "PLA"
	PLP Mnemonic = "PLP"
	PLX Mnemonic = "PLX"
	PLY Mnemonic = "PLY"
	ROL Mnemonic = "ROL"
	ROR Mnemonic = "ROR"
	RTI Mnemonic = "RTI"
	RTS Mnemonic = "RTS"
	SBC Mnemonic = "SBC"
	SEC Mnemonic = "SEC"
	SED Mnemonic = "SED"
	SEI Mnemonic = "SEI"
	STA Mnemonic = "STA"
	STP Mnemonic = "STP"
	STX Mnemonic = "STX"
	STY Mnemonic = "STY"
	STZ Mnemonic = "STZ"
	TAX Mnemonic = "TAX"
	TAY Mnemonic = "TAY"
	TRB Mnemonic = "TRB"
	TSB Mnemonic = "TSB"
	TSX Mnemonic = "TSX"
	TXA Mnemonic = "TXA"
	TXS Mnemonic = "TXS"
	TYA Mnemonic = "TYA"
	WAI Mnemonic = "WAI"

	RMB0 Mnemonic = "RMB0"
	RMB1 Mnemonic = "RMB1"
	RMB2 Mnemonic = "RMB2"
	RMB3 Mnemonic = "RMB3"
	RMB4 Mnemonic = "RMB4"
	RMB5 Mnemonic = "RMB5"
	RMB6 Mnemonic = "RMB6"
	RMB7 Mnemonic = "RMB7"

	SMB0 Mnemonic = "SMB0"
	SMB1 Mnemonic = "SMB1"
	SMB2 Mnemonic = "SMB2"
	SMB3 Mnemonic = "SMB3"
	SMB4 Mnemonic = "SMB4"
	SMB5 Mnemonic = "SMB5"
	SMB6 Mnemonic = "SMB6"
	SMB7 Mnemonic = "SMB7"

	BBR0 Mnemonic = "BBR0"
	BBR1 Mnemonic = "BBR1"
	BBR2 Mnemonic = "BBR2"
	BBR3 Mnemonic = "BBR3"
	BBR4 Mnemonic = "BBR4"
	BBR5 Mnemonic = "BBR5"
	BBR6 Mnemonic = "BBR6"
	BBR7 Mnemonic = "BBR7"

	BBS0 Mnemonic = "BBS0"
	BBS1 Mnemonic = "BBS1"
	BBS2 Mnemonic = "BBS2"
	BBS3 Mnemonic = "BBS3"
	BBS4 Mnemonic = "BBS4"
	BBS5 Mnemonic = "BBS5"
	BBS6 Mnemonic = "BBS6"
	BBS7 Mnemonic = "BBS7"
)
