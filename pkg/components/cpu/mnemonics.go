package cpu

import "github.com/fran150/clementina-6502/pkg/components"

const (
	// Standard 6502 instructions
	ADC components.Mnemonic = "ADC" // Add with Carry
	AND components.Mnemonic = "AND" // Logical AND
	ASL components.Mnemonic = "ASL" // Arithmetic Shift Left
	BCC components.Mnemonic = "BCC" // Branch if Carry Clear
	BCS components.Mnemonic = "BCS" // Branch if Carry Set
	BEQ components.Mnemonic = "BEQ" // Branch if Equal (Zero set)
	BIT components.Mnemonic = "BIT" // Bit Test
	BMI components.Mnemonic = "BMI" // Branch if Minus (Negative set)
	BNE components.Mnemonic = "BNE" // Branch if Not Equal (Zero clear)
	BPL components.Mnemonic = "BPL" // Branch if Plus (Negative clear)
	BRA components.Mnemonic = "BRA" // Branch Always (65C02 only)
	BRK components.Mnemonic = "BRK" // Force Break / Interrupt
	BVC components.Mnemonic = "BVC" // Branch if Overflow Clear
	BVS components.Mnemonic = "BVS" // Branch if Overflow Set
	CLC components.Mnemonic = "CLC" // Clear Carry Flag
	CLD components.Mnemonic = "CLD" // Clear Decimal Mode
	CLI components.Mnemonic = "CLI" // Clear Interrupt Disable
	CLV components.Mnemonic = "CLV" // Clear Overflow Flag
	CMP components.Mnemonic = "CMP" // Compare (with Accumulator)
	CPX components.Mnemonic = "CPX" // Compare with X Register
	CPY components.Mnemonic = "CPY" // Compare with Y Register
	DEC components.Mnemonic = "DEC" // Decrement
	DEX components.Mnemonic = "DEX" // Decrement X Register
	DEY components.Mnemonic = "DEY" // Decrement Y Register
	EOR components.Mnemonic = "EOR" // Exclusive OR
	INC components.Mnemonic = "INC" // Increment
	INX components.Mnemonic = "INX" // Increment X Register
	INY components.Mnemonic = "INY" // Increment Y Register
	JMP components.Mnemonic = "JMP" // Jump
	JSR components.Mnemonic = "JSR" // Jump to Subroutine
	LDA components.Mnemonic = "LDA" // Load Accumulator
	LDX components.Mnemonic = "LDX" // Load X Register
	LDY components.Mnemonic = "LDY" // Load Y Register
	LSR components.Mnemonic = "LSR" // Logical Shift Right
	NOP components.Mnemonic = "NOP" // No Operation
	ORA components.Mnemonic = "ORA" // Logical OR
	PHA components.Mnemonic = "PHA" // Push Accumulator
	PHP components.Mnemonic = "PHP" // Push Processor Status
	PHX components.Mnemonic = "PHX" // Push X Register (65C02 only)
	PHY components.Mnemonic = "PHY" // Push Y Register (65C02 only)
	PLA components.Mnemonic = "PLA" // Pull Accumulator
	PLP components.Mnemonic = "PLP" // Pull Processor Status
	PLX components.Mnemonic = "PLX" // Pull X Register (65C02 only)
	PLY components.Mnemonic = "PLY" // Pull Y Register (65C02 only)
	ROL components.Mnemonic = "ROL" // Rotate Left
	ROR components.Mnemonic = "ROR" // Rotate Right
	RTI components.Mnemonic = "RTI" // Return from Interrupt
	RTS components.Mnemonic = "RTS" // Return from Subroutine
	SBC components.Mnemonic = "SBC" // Subtract with Carry
	SEC components.Mnemonic = "SEC" // Set Carry Flag
	SED components.Mnemonic = "SED" // Set Decimal Mode
	SEI components.Mnemonic = "SEI" // Set Interrupt Disable
	STA components.Mnemonic = "STA" // Store Accumulator
	STP components.Mnemonic = "STP" // Stop (65C02 only)
	STX components.Mnemonic = "STX" // Store X Register
	STY components.Mnemonic = "STY" // Store Y Register
	STZ components.Mnemonic = "STZ" // Store Zero (65C02 only)
	TAX components.Mnemonic = "TAX" // Transfer Accumulator to X
	TAY components.Mnemonic = "TAY" // Transfer Accumulator to Y
	TRB components.Mnemonic = "TRB" // Test and Reset Bits (65C02 only)
	TSB components.Mnemonic = "TSB" // Test and Set Bits (65C02 only)
	TSX components.Mnemonic = "TSX" // Transfer Stack Pointer to X
	TXA components.Mnemonic = "TXA" // Transfer X to Accumulator
	TXS components.Mnemonic = "TXS" // Transfer X to Stack Pointer
	TYA components.Mnemonic = "TYA" // Transfer Y to Accumulator
	WAI components.Mnemonic = "WAI" // Wait for Interrupt (65C02 only)

	// Rockwell/WDC 65C02 bit manipulation instructions
	RMB0 components.Mnemonic = "RMB0" // Reset Memory Bit 0
	RMB1 components.Mnemonic = "RMB1" // Reset Memory Bit 1
	RMB2 components.Mnemonic = "RMB2" // Reset Memory Bit 2
	RMB3 components.Mnemonic = "RMB3" // Reset Memory Bit 3
	RMB4 components.Mnemonic = "RMB4" // Reset Memory Bit 4
	RMB5 components.Mnemonic = "RMB5" // Reset Memory Bit 5
	RMB6 components.Mnemonic = "RMB6" // Reset Memory Bit 6
	RMB7 components.Mnemonic = "RMB7" // Reset Memory Bit 7

	SMB0 components.Mnemonic = "SMB0" // Set Memory Bit 0
	SMB1 components.Mnemonic = "SMB1" // Set Memory Bit 1
	SMB2 components.Mnemonic = "SMB2" // Set Memory Bit 2
	SMB3 components.Mnemonic = "SMB3" // Set Memory Bit 3
	SMB4 components.Mnemonic = "SMB4" // Set Memory Bit 4
	SMB5 components.Mnemonic = "SMB5" // Set Memory Bit 5
	SMB6 components.Mnemonic = "SMB6" // Set Memory Bit 6
	SMB7 components.Mnemonic = "SMB7" // Set Memory Bit 7

	BBR0 components.Mnemonic = "BBR0" // Branch if Bit 0 Reset
	BBR1 components.Mnemonic = "BBR1" // Branch if Bit 1 Reset
	BBR2 components.Mnemonic = "BBR2" // Branch if Bit 2 Reset
	BBR3 components.Mnemonic = "BBR3" // Branch if Bit 3 Reset
	BBR4 components.Mnemonic = "BBR4" // Branch if Bit 4 Reset
	BBR5 components.Mnemonic = "BBR5" // Branch if Bit 5 Reset
	BBR6 components.Mnemonic = "BBR6" // Branch if Bit 6 Reset
	BBR7 components.Mnemonic = "BBR7" // Branch if Bit 7 Reset

	BBS0 components.Mnemonic = "BBS0" // Branch if Bit 0 Set
	BBS1 components.Mnemonic = "BBS1" // Branch if Bit 1 Set
	BBS2 components.Mnemonic = "BBS2" // Branch if Bit 2 Set
	BBS3 components.Mnemonic = "BBS3" // Branch if Bit 3 Set
	BBS4 components.Mnemonic = "BBS4" // Branch if Bit 4 Set
	BBS5 components.Mnemonic = "BBS5" // Branch if Bit 5 Set
	BBS6 components.Mnemonic = "BBS6" // Branch if Bit 6 Set
	BBS7 components.Mnemonic = "BBS7" // Branch if Bit 7 Set
)
