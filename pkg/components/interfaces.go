package components

import (
	"github.com/fran150/clementina6502/pkg/components/buses"
	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/fran150/clementina6502/pkg/components/cpu"
	"go.bug.st/serial"
)

type Acia6522Chip interface {
	// Pin connections
	DataBus() *buses.BusConnector[uint8]
	IrqRequest() *buses.ConnectorEnabledLow
	ReadWrite() *buses.ConnectorEnabledLow
	ChipSelect0() *buses.ConnectorEnabledHigh
	ChipSelect1() *buses.ConnectorEnabledLow
	RegisterSelect(num uint8) *buses.ConnectorEnabledHigh
	Reset() *buses.ConnectorEnabledLow

	// Configuration methods
	ConnectToPort(port serial.Port)
	ConnectRegisterSelectLines(lines [2]buses.Line)
	Close()

	// Emulation methods
	Tick(context *common.StepContext)

	// Register getters
	GetStatusRegister() uint8
	GetControlRegister() uint8
	GetCommandRegister() uint8
	GetTXRegister() uint8
	GetRXRegister() uint8
	GetTXRegisterEmpty() bool
	GetRXRegisterEmpty() bool
}

type Cpu6502Chip interface {
	// Control Lines
	AddressBus() *buses.BusConnector[uint16]
	BusEnable() *buses.ConnectorEnabledHigh
	DataBus() *buses.BusConnector[uint8]
	InterruptRequest() *buses.ConnectorEnabledLow
	MemoryLock() *buses.ConnectorEnabledLow
	NonMaskableInterrupt() *buses.ConnectorEnabledLow
	Reset() *buses.ConnectorEnabledLow
	SetOverflow() *buses.ConnectorEnabledLow
	ReadWrite() *buses.ConnectorEnabledLow
	Ready() *buses.ConnectorEnabledHigh
	Sync() *buses.ConnectorEnabledHigh
	VectorPull() *buses.ConnectorEnabledLow

	// Timer methods
	Tick(context *common.StepContext)
	PostTick(context *common.StepContext)

	// State getters
	GetAccumulatorRegister() uint8
	GetXRegister() uint8
	GetYRegister() uint8
	GetStackPointer() uint8
	GetProcessorStatusRegister() cpu.StatusRegister
	IsReadingOpcode() bool
	GetCurrentInstruction() *cpu.CpuInstructionData
	GetCurrentAddressMode() *cpu.AddressModeData
	GetProgramCounter() uint16

	// Program counter manipulation
	ForceProgramCounter(value uint16)
}
