package cpu

// Mnemonic represents the human-readable assembly language representation of 6502 CPU instructions.
// Each mnemonic can be used with different addressing modes to form complete instructions.
// For example, LDA $C010 uses absolute addressing to load a value from memory address $C010,
// while LDA #$FF uses immediate addressing to load the literal value $FF.
type Mnemonic string

const (
	// Standard 6502 instructions
	ADC Mnemonic = "ADC" // Add with Carry
	AND Mnemonic = "AND" // Logical AND
	ASL Mnemonic = "ASL" // Arithmetic Shift Left
	BCC Mnemonic = "BCC" // Branch if Carry Clear
	BCS Mnemonic = "BCS" // Branch if Carry Set
	BEQ Mnemonic = "BEQ" // Branch if Equal (Zero set)
	BIT Mnemonic = "BIT" // Bit Test
	BMI Mnemonic = "BMI" // Branch if Minus (Negative set)
	BNE Mnemonic = "BNE" // Branch if Not Equal (Zero clear)
	BPL Mnemonic = "BPL" // Branch if Plus (Negative clear)
	BRA Mnemonic = "BRA" // Branch Always (65C02 only)
	BRK Mnemonic = "BRK" // Force Break / Interrupt
	BVC Mnemonic = "BVC" // Branch if Overflow Clear
	BVS Mnemonic = "BVS" // Branch if Overflow Set
	CLC Mnemonic = "CLC" // Clear Carry Flag
	CLD Mnemonic = "CLD" // Clear Decimal Mode
	CLI Mnemonic = "CLI" // Clear Interrupt Disable
	CLV Mnemonic = "CLV" // Clear Overflow Flag
	CMP Mnemonic = "CMP" // Compare (with Accumulator)
	CPX Mnemonic = "CPX" // Compare with X Register
	CPY Mnemonic = "CPY" // Compare with Y Register
	DEC Mnemonic = "DEC" // Decrement
	DEX Mnemonic = "DEX" // Decrement X Register
	DEY Mnemonic = "DEY" // Decrement Y Register
	EOR Mnemonic = "EOR" // Exclusive OR
	INC Mnemonic = "INC" // Increment
	INX Mnemonic = "INX" // Increment X Register
	INY Mnemonic = "INY" // Increment Y Register
	JMP Mnemonic = "JMP" // Jump
	JSR Mnemonic = "JSR" // Jump to Subroutine
	LDA Mnemonic = "LDA" // Load Accumulator
	LDX Mnemonic = "LDX" // Load X Register
	LDY Mnemonic = "LDY" // Load Y Register
	LSR Mnemonic = "LSR" // Logical Shift Right
	NOP Mnemonic = "NOP" // No Operation
	ORA Mnemonic = "ORA" // Logical OR
	PHA Mnemonic = "PHA" // Push Accumulator
	PHP Mnemonic = "PHP" // Push Processor Status
	PHX Mnemonic = "PHX" // Push X Register (65C02 only)
	PHY Mnemonic = "PHY" // Push Y Register (65C02 only)
	PLA Mnemonic = "PLA" // Pull Accumulator
	PLP Mnemonic = "PLP" // Pull Processor Status
	PLX Mnemonic = "PLX" // Pull X Register (65C02 only)
	PLY Mnemonic = "PLY" // Pull Y Register (65C02 only)
	ROL Mnemonic = "ROL" // Rotate Left
	ROR Mnemonic = "ROR" // Rotate Right
	RTI Mnemonic = "RTI" // Return from Interrupt
	RTS Mnemonic = "RTS" // Return from Subroutine
	SBC Mnemonic = "SBC" // Subtract with Carry
	SEC Mnemonic = "SEC" // Set Carry Flag
	SED Mnemonic = "SED" // Set Decimal Mode
	SEI Mnemonic = "SEI" // Set Interrupt Disable
	STA Mnemonic = "STA" // Store Accumulator
	STP Mnemonic = "STP" // Stop (65C02 only)
	STX Mnemonic = "STX" // Store X Register
	STY Mnemonic = "STY" // Store Y Register
	STZ Mnemonic = "STZ" // Store Zero (65C02 only)
	TAX Mnemonic = "TAX" // Transfer Accumulator to X
	TAY Mnemonic = "TAY" // Transfer Accumulator to Y
	TRB Mnemonic = "TRB" // Test and Reset Bits (65C02 only)
	TSB Mnemonic = "TSB" // Test and Set Bits (65C02 only)
	TSX Mnemonic = "TSX" // Transfer Stack Pointer to X
	TXA Mnemonic = "TXA" // Transfer X to Accumulator
	TXS Mnemonic = "TXS" // Transfer X to Stack Pointer
	TYA Mnemonic = "TYA" // Transfer Y to Accumulator
	WAI Mnemonic = "WAI" // Wait for Interrupt (65C02 only)

	// Rockwell/WDC 65C02 bit manipulation instructions
	RMB0 Mnemonic = "RMB0" // Reset Memory Bit 0
	RMB1 Mnemonic = "RMB1" // Reset Memory Bit 1
	RMB2 Mnemonic = "RMB2" // Reset Memory Bit 2
	RMB3 Mnemonic = "RMB3" // Reset Memory Bit 3
	RMB4 Mnemonic = "RMB4" // Reset Memory Bit 4
	RMB5 Mnemonic = "RMB5" // Reset Memory Bit 5
	RMB6 Mnemonic = "RMB6" // Reset Memory Bit 6
	RMB7 Mnemonic = "RMB7" // Reset Memory Bit 7

	SMB0 Mnemonic = "SMB0" // Set Memory Bit 0
	SMB1 Mnemonic = "SMB1" // Set Memory Bit 1
	SMB2 Mnemonic = "SMB2" // Set Memory Bit 2
	SMB3 Mnemonic = "SMB3" // Set Memory Bit 3
	SMB4 Mnemonic = "SMB4" // Set Memory Bit 4
	SMB5 Mnemonic = "SMB5" // Set Memory Bit 5
	SMB6 Mnemonic = "SMB6" // Set Memory Bit 6
	SMB7 Mnemonic = "SMB7" // Set Memory Bit 7

	BBR0 Mnemonic = "BBR0" // Branch if Bit 0 Reset
	BBR1 Mnemonic = "BBR1" // Branch if Bit 1 Reset
	BBR2 Mnemonic = "BBR2" // Branch if Bit 2 Reset
	BBR3 Mnemonic = "BBR3" // Branch if Bit 3 Reset
	BBR4 Mnemonic = "BBR4" // Branch if Bit 4 Reset
	BBR5 Mnemonic = "BBR5" // Branch if Bit 5 Reset
	BBR6 Mnemonic = "BBR6" // Branch if Bit 6 Reset
	BBR7 Mnemonic = "BBR7" // Branch if Bit 7 Reset

	BBS0 Mnemonic = "BBS0" // Branch if Bit 0 Set
	BBS1 Mnemonic = "BBS1" // Branch if Bit 1 Set
	BBS2 Mnemonic = "BBS2" // Branch if Bit 2 Set
	BBS3 Mnemonic = "BBS3" // Branch if Bit 3 Set
	BBS4 Mnemonic = "BBS4" // Branch if Bit 4 Set
	BBS5 Mnemonic = "BBS5" // Branch if Bit 5 Set
	BBS6 Mnemonic = "BBS6" // Branch if Bit 6 Set
	BBS7 Mnemonic = "BBS7" // Branch if Bit 7 Set
)
