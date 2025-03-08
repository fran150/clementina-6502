package cpu

import (
	"testing"
)

func TestCpu65C02S_GetterMethods(t *testing.T) {
	// Setup
	cpu := NewCpu65C02S()

	// Test cases structure for register values
	tests := []struct {
		name     string
		setup    func(*Cpu65C02S)
		getValue func(*Cpu65C02S) uint8
		want     uint8
	}{
		{
			name: "GetAccumulatorRegister returns correct value",
			setup: func(c *Cpu65C02S) {
				c.accumulatorRegister = 0x42
			},
			getValue: (*Cpu65C02S).GetAccumulatorRegister,
			want:     0x42,
		},
		{
			name: "GetXRegister returns correct value",
			setup: func(c *Cpu65C02S) {
				c.xRegister = 0x55
			},
			getValue: (*Cpu65C02S).GetXRegister,
			want:     0x55,
		},
		{
			name: "GetYRegister returns correct value",
			setup: func(c *Cpu65C02S) {
				c.yRegister = 0xAA
			},
			getValue: (*Cpu65C02S).GetYRegister,
			want:     0xAA,
		},
		{
			name: "GetStackPointer returns correct value",
			setup: func(c *Cpu65C02S) {
				c.stackPointer = 0xFF
			},
			getValue: (*Cpu65C02S).GetStackPointer,
			want:     0xFF,
		},
	}

	// Run register tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(cpu)
			if got := tt.getValue(cpu); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}

	// Test program counter separately since it's uint16
	t.Run("GetProgramCounter returns correct value", func(t *testing.T) {
		wantPC := uint16(0x1234)
		cpu.ForceProgramCounter(wantPC)
		if got := cpu.GetProgramCounter(); got != wantPC {
			t.Errorf("GetProgramCounter() = %v, want %v", got, wantPC)
		}
	})

	// Test IsReadingOpcode
	t.Run("IsReadingOpcode returns correct state", func(t *testing.T) {
		cpu.currentCycle.signaling.sync = true
		if !cpu.IsReadingOpcode() {
			t.Error("IsReadingOpcode() = false, want true")
		}

		cpu.currentCycle.signaling.sync = false
		if cpu.IsReadingOpcode() {
			t.Error("IsReadingOpcode() = true, want false")
		}
	})

	// Test GetProcessorStatusRegister
	t.Run("GetProcessorStatusRegister returns correct value", func(t *testing.T) {
		expectedStatus := StatusRegister(0b10110001)
		cpu.processorStatusRegister = expectedStatus
		if got := cpu.GetProcessorStatusRegister(); got != expectedStatus {
			t.Errorf("GetProcessorStatusRegister() = %v, want %v", got, expectedStatus)
		}
	})

	// Test GetCurrentInstruction and GetCurrentAddressMode
	t.Run("GetCurrentInstruction returns correct instruction data", func(t *testing.T) {
		instructionData := &CpuInstructionData{} // Fill with appropriate test data
		cpu.currentInstruction = instructionData
		if got := cpu.GetCurrentInstruction(); got != instructionData {
			t.Errorf("GetCurrentInstruction() returned unexpected value")
		}
	})

	t.Run("GetCurrentAddressMode returns correct address mode data", func(t *testing.T) {
		addressModeData := &AddressModeData{} // Fill with appropriate test data
		cpu.currentAddressMode = addressModeData
		if got := cpu.GetCurrentAddressMode(); got != addressModeData {
			t.Errorf("GetCurrentAddressMode() returned unexpected value")
		}
	})
}
