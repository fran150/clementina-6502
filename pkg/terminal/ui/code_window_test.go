package ui

import (
	"testing"

	"github.com/fran150/clementina6502/pkg/common"
	"github.com/fran150/clementina6502/pkg/components/buses"
	"github.com/fran150/clementina6502/pkg/components/cpu"
	"github.com/stretchr/testify/assert"
)

// Mock implementation of ICpu65C02S
type mockCpu struct {
	programCounter     uint16
	currentInstruction *cpu.CpuInstructionData
	isReadingOpcode    bool
}

// Implement all interface methods
func (m *mockCpu) AddressBus() *buses.BusConnector[uint16]          { return nil }
func (m *mockCpu) BusEnable() *buses.ConnectorEnabledHigh           { return nil }
func (m *mockCpu) DataBus() *buses.BusConnector[uint8]              { return nil }
func (m *mockCpu) InterruptRequest() *buses.ConnectorEnabledLow     { return nil }
func (m *mockCpu) MemoryLock() *buses.ConnectorEnabledLow           { return nil }
func (m *mockCpu) NonMaskableInterrupt() *buses.ConnectorEnabledLow { return nil }
func (m *mockCpu) Reset() *buses.ConnectorEnabledLow                { return nil }
func (m *mockCpu) SetOverflow() *buses.ConnectorEnabledLow          { return nil }
func (m *mockCpu) ReadWrite() *buses.ConnectorEnabledLow            { return nil }
func (m *mockCpu) Ready() *buses.ConnectorEnabledHigh               { return nil }
func (m *mockCpu) Sync() *buses.ConnectorEnabledHigh                { return nil }
func (m *mockCpu) VectorPull() *buses.ConnectorEnabledLow           { return nil }
func (m *mockCpu) Tick(*common.StepContext)                         {}
func (m *mockCpu) PostTick(*common.StepContext)                     {}
func (m *mockCpu) GetAccumulatorRegister() uint8                    { return 0 }
func (m *mockCpu) GetXRegister() uint8                              { return 0 }
func (m *mockCpu) GetYRegister() uint8                              { return 0 }
func (m *mockCpu) GetStackPointer() uint8                           { return 0 }
func (m *mockCpu) GetProcessorStatusRegister() cpu.StatusRegister   { return 0 }
func (m *mockCpu) GetCurrentAddressMode() *cpu.AddressModeData      { return nil }

// Implement the methods that we'll actually use in tests
func (m *mockCpu) IsReadingOpcode() bool                          { return m.isReadingOpcode }
func (m *mockCpu) GetCurrentInstruction() *cpu.CpuInstructionData { return m.currentInstruction }
func (m *mockCpu) GetProgramCounter() uint16                      { return m.programCounter }
func (m *mockCpu) ForceProgramCounter(value uint16)               { m.programCounter = value }

func TestCodeWindow_NewCodeWindow(t *testing.T) {
	mockCpu := &mockCpu{}
	operandsGetter := func(pc uint16) [2]uint8 { return [2]uint8{0, 0} }

	codeWindow := NewCodeWindow(mockCpu, operandsGetter)

	assert.NotNil(t, codeWindow)
	assert.NotNil(t, codeWindow.text)
	assert.NotNil(t, codeWindow.lines)
	assert.NotNil(t, codeWindow.processor)
	assert.NotNil(t, codeWindow.operandsGetter)
}

func TestCodeWindow_Tick_NoInstruction(t *testing.T) {
	mockCpu := &mockCpu{
		programCounter:     0x1000,
		currentInstruction: nil,
		isReadingOpcode:    true,
	}

	codeWindow := NewCodeWindow(mockCpu, func(pc uint16) [2]uint8 { return [2]uint8{0, 0} })
	context := &common.StepContext{}

	codeWindow.Tick(context)

	assert.Equal(t, 0, codeWindow.lines.Size())
}

func TestCodeWindow_Tick_WithInstruction(t *testing.T) {
	instructions := cpu.NewInstructionSet()
	instruction := instructions.GetByOpCode(cpu.OpCode(0xA9))

	mockCpu := &mockCpu{
		programCounter:     0x1000,
		currentInstruction: instruction,
		isReadingOpcode:    true,
	}

	codeWindow := NewCodeWindow(mockCpu, func(pc uint16) [2]uint8 { return [2]uint8{0x42, 0} })
	context := &common.StepContext{}

	codeWindow.Tick(context)

	assert.Equal(t, 1, codeWindow.lines.Size())
}

func TestCodeWindow_MaxLines(t *testing.T) {
	instructions := cpu.NewInstructionSet()
	instruction := instructions.GetByOpCode(cpu.OpCode(0xEA))

	mockCpu := &mockCpu{
		programCounter:     0x1000,
		currentInstruction: instruction,
		isReadingOpcode:    true,
	}

	codeWindow := NewCodeWindow(mockCpu, func(pc uint16) [2]uint8 { return [2]uint8{0, 0} })
	context := &common.StepContext{}

	// Add more lines than maxLinesOfCode
	for i := 0; i < maxLinesOfCode+10; i++ {
		mockCpu.programCounter = uint16(0x1000 + i)
		codeWindow.Tick(context)
	}

	assert.Equal(t, maxLinesOfCode, codeWindow.lines.Size())
}

func TestCodeWindow_Clear(t *testing.T) {
	mockCpu := &mockCpu{}
	codeWindow := NewCodeWindow(mockCpu, func(pc uint16) [2]uint8 { return [2]uint8{0, 0} })

	codeWindow.Clear()

	// Get the text content after clearing
	text := codeWindow.text.GetText(true)
	assert.Empty(t, text)
}

func TestCodeWindow_Draw(t *testing.T) {
	instructions := cpu.NewInstructionSet()
	instruction := instructions.GetByOpCode(cpu.OpCode(0xA9))

	mockCpu := &mockCpu{
		programCounter:     0x1000,
		currentInstruction: instruction,
		isReadingOpcode:    true,
	}

	codeWindow := NewCodeWindow(mockCpu, func(pc uint16) [2]uint8 { return [2]uint8{0x42, 0} })
	context := &common.StepContext{}

	codeWindow.Tick(context)
	codeWindow.Draw(context)

	// Verify that the text view contains content
	text := codeWindow.text.GetText(true)
	assert.NotEmpty(t, text)
}

func TestCodeWindow_GetDrawArea(t *testing.T) {
	mockCpu := &mockCpu{}
	codeWindow := NewCodeWindow(mockCpu, func(pc uint16) [2]uint8 { return [2]uint8{0, 0} })

	primitive := codeWindow.GetDrawArea()

	assert.NotNil(t, primitive)
	assert.Equal(t, codeWindow.text, primitive)
}

func TestCodeWindow_InstructionDecoding(t *testing.T) {
	mockCpu := &mockCpu{
		programCounter:  0x1000,
		isReadingOpcode: true,
	}

	instructions := cpu.NewInstructionSet()

	tests := []struct {
		name          string
		opcode        uint8
		operands      [2]uint8
		expectedValue string
	}{
		{
			name:          "LDA Immediate",
			opcode:        0xA9, // LDA #$nn
			operands:      [2]uint8{0x42, 0x00},
			expectedValue: "$0FFF: LDA #$42",
		},
		{
			name:          "JMP Absolute",
			opcode:        0x4C, // JMP $nnnn
			operands:      [2]uint8{0x34, 0x12},
			expectedValue: "$0FFF: JMP $1234",
		},
		{
			name:          "BEQ Relative",
			opcode:        0xF0, // BEQ $nn
			operands:      [2]uint8{0x10, 0x00},
			expectedValue: "$0FFF: BEQ $10 ($1011)", // PC + 2 + offset
		},
		{
			name:          "BEQ Relative Backwards",
			opcode:        0xF0, // BEQ $nn
			operands:      [2]uint8{0xFA, 0x00},
			expectedValue: "$0FFF: BEQ $FA ($0FFB)", // PC + 2 + offset
		},
		{
			name:          "STA Zero Page",
			opcode:        0x85, // STA $nn
			operands:      [2]uint8{0x42, 0x00},
			expectedValue: "$0FFF: STA $42",
		},
		{
			name:          "Non Existing to NOP",
			opcode:        0xF3, // Unexisting code
			operands:      [2]uint8{0x42, 0x00},
			expectedValue: "$0FFF: NOP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCpu.currentInstruction = instructions.GetByOpCode(cpu.OpCode(tt.opcode))

			operandsGetter := func(pc uint16) [2]uint8 {
				return tt.operands
			}

			codeWindow := NewCodeWindow(mockCpu, operandsGetter)
			context := &common.StepContext{}

			codeWindow.Tick(context)
			codeWindow.Draw(context)

			// Verify that the text view contains content
			text := codeWindow.text.GetText(true)

			assert.Contains(t, text, tt.expectedValue)
		})
	}
}

func TestCodeWindow_Tick(t *testing.T) {
	mockCpu := &mockCpu{
		programCounter:  0x1000,
		isReadingOpcode: true,
	}

	instructions := cpu.NewInstructionSet()
	mockCpu.currentInstruction = instructions.GetByOpCode(cpu.OpCode(0xA9)) // LDA Immediate

	operandsGetter := func(pc uint16) [2]uint8 {
		return [2]uint8{0x42, 0x00}
	}

	codeWindow := NewCodeWindow(mockCpu, operandsGetter)
	codeWindow.Tick(&common.StepContext{})

	// Verify that a line was added to the queue
	assert.Equal(t, 1, codeWindow.lines.Size())

	// Verify queue management when exceeding maxLinesOfCode
	for i := 0; i < maxLinesOfCode; i++ {
		codeWindow.Tick(&common.StepContext{})
	}

	// Queue should not exceed maxLinesOfCode
	assert.LessOrEqual(t, codeWindow.lines.Size(), maxLinesOfCode)
}

func TestCodeWindow_BRKInstruction(t *testing.T) {
	mockCpu := &mockCpu{
		programCounter:  0x1000,
		isReadingOpcode: true,
	}

	instructions := cpu.NewInstructionSet()
	mockCpu.currentInstruction = instructions.GetByOpCode(cpu.OpCode(0x00)) // BRK

	operandsGetter := func(pc uint16) [2]uint8 {
		return [2]uint8{0x00, 0x00}
	}

	codeWindow := NewCodeWindow(mockCpu, operandsGetter)
	context := &common.StepContext{}

	codeWindow.Tick(context)
	codeWindow.Draw(context)

	// Verify that the text view contains content
	text := codeWindow.text.GetText(true)

	// BRK should be shown as a single byte instruction
	assert.Contains(t, text, "BRK")
}
