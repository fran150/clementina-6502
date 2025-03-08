package cpu

import (
	"testing"
)

func TestNewInstructionSet(t *testing.T) {
	// Test creation of instruction set
	instructionSet := NewInstructionSet()

	// Test that instruction set is not nil
	if instructionSet == nil {
		t.Error("Expected non-nil instruction set")
	}

	// Test some known opcodes
	testCases := []struct {
		opcode   OpCode
		mnemonic Mnemonic
	}{
		{0xEA, NOP}, // NOP instruction
		{0xA9, LDA}, // LDA immediate
		{0x4C, JMP}, // JMP absolute
	}

	for _, tc := range testCases {
		instruction := instructionSet.GetByOpCode(tc.opcode)
		if instruction == nil {
			t.Errorf("Expected instruction for opcode %02X", tc.opcode)
			continue
		}
		if instruction.Mnemonic() != tc.mnemonic {
			t.Errorf("Opcode %02X: expected mnemonic %v, got %v",
				tc.opcode, tc.mnemonic, instruction.Mnemonic())
		}
	}
}

func TestInvalidOpCode(t *testing.T) {
	instructionSet := NewInstructionSet()

	// Test an invalid opcode (0x02 is not used in the 65C02)
	invalidOpCode := OpCode(0x02)
	instruction := instructionSet.GetByOpCode(invalidOpCode)

	// Should return NOP (0xEA) for invalid opcodes
	if instruction.OpCode() != OpCode(0xEA) {
		t.Errorf("Expected invalid opcode to return NOP (0xEA), got %02X",
			instruction.OpCode())
	}

	if instruction.Mnemonic() != NOP {
		t.Errorf("Expected invalid opcode to return NOP mnemonic, got %v",
			instruction.Mnemonic())
	}
}

func TestInstructionData(t *testing.T) {
	// Create a test instruction
	testInstruction := CpuInstructionData{
		opcode:      0xEA,
		mnemonic:    NOP,
		action:      actionNOP,
		addressMode: AddressModeImplicit,
	}

	// Test getter methods
	if testInstruction.OpCode() != OpCode(0xEA) {
		t.Errorf("Expected opcode 0xEA, got %02X", testInstruction.OpCode())
	}

	if testInstruction.Mnemonic() != NOP {
		t.Errorf("Expected mnemonic NOP, got %v", testInstruction.Mnemonic())
	}

	if testInstruction.AddressMode() != AddressModeImplicit {
		t.Errorf("Expected AddressModeImplicit, got %v",
			testInstruction.AddressMode())
	}
}

func TestInstructionSetCompleteness(t *testing.T) {
	instructionSet := NewInstructionSet()

	// Test that we have all valid opcodes implemented
	expectedInstructions := map[OpCode]bool{
		0xEA: true, // NOP
		0xA9: true, // LDA immediate
		0x4C: true, // JMP absolute
		// Add more known valid opcodes as needed
	}

	for opcode := range expectedInstructions {
		instruction := instructionSet.GetByOpCode(opcode)
		if instruction == nil {
			t.Errorf("Missing implementation for opcode %02X", opcode)
		}
	}
}
