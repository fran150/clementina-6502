package components

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/fran150/clementina-6502/pkg/components/cpu"
	"github.com/fran150/clementina-6502/pkg/components/lcd"
	"go.bug.st/serial"
)

// Acia6522Chip defines the interface for the 65C51N Asynchronous Communications Interface Adapter.
// This chip provides serial communication capabilities to the 6502 computer system.
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
	ConnectToPort(port serial.Port) error
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

// Cpu6502Chip defines the interface for the 65C02S CPU emulation.
// It provides access to all CPU pins, internal registers, and execution control.
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

// LCDControllerChip defines the interface for the HD44780U LCD controller.
// This chip manages a character LCD display with cursor control and display settings.
type LCDControllerChip interface {
	// Bus connection methods
	Enable() *buses.ConnectorEnabledHigh
	ReadWrite() *buses.ConnectorEnabledLow
	RegisterSelect() *buses.ConnectorEnabledHigh
	DataBus() *buses.BusConnector[uint8]

	// Emulation method
	Tick(context *common.StepContext)

	// Status methods (based on usage in lcd_controller_window.go)
	GetCursorStatus() lcd.CursorStatus
	GetDisplayStatus() lcd.DisplayStatus
}

// MemoryChip defines the interface for RAM and ROM memory components.
// It provides access to memory contents and control signals for memory operations.
type MemoryChip interface {
	// Bus and control signal connections
	HiAddressBus() *buses.BusConnector[uint16]
	AddressBus() *buses.BusConnector[uint16]
	DataBus() *buses.BusConnector[uint8]
	WriteEnable() *buses.ConnectorEnabledLow
	ChipSelect() *buses.ConnectorEnabledLow
	OutputEnable() *buses.ConnectorEnabledLow

	// Utility methods
	Peek(address uint32) uint8
	PeekRange(startAddress uint16, endAddress uint16) []uint8
	Poke(address uint16, value uint8)
	Load(binFilePath string) error
	Size() int

	// Emulation method
	Tick(context *common.StepContext)
}

// ViaChip defines the interface for the 65C22S Versatile Interface Adapter.
// This chip provides parallel I/O ports, timers, and shift register functionality.
type ViaChip interface {
	// Pin Getters / Setters
	PeripheralAControlLines(num int) *buses.ConnectorEnabledHigh
	PeripheralBControlLines(num int) *buses.ConnectorEnabledHigh
	ChipSelect1() *buses.ConnectorEnabledHigh
	ChipSelect2() *buses.ConnectorEnabledLow
	DataBus() *buses.BusConnector[uint8]
	IrqRequest() *buses.ConnectorEnabledLow
	PeripheralPortA() *buses.BusConnector[uint8]
	PeripheralPortB() *buses.BusConnector[uint8]
	Reset() *buses.ConnectorEnabledLow
	RegisterSelect(num uint8) *buses.ConnectorEnabledHigh
	ReadWrite() *buses.ConnectorEnabledLow
	ConnectRegisterSelectLines(lines [4]buses.Line)

	// Tick method
	Tick(context *common.StepContext)

	// Internal Registers Getters
	GetOutputRegisterA() uint8
	GetOutputRegisterB() uint8
	GetInputRegisterA() uint8
	GetInputRegisterB() uint8
	GetDataDirectionRegisterA() uint8
	GetDataDirectionRegisterB() uint8
	GetLowLatches2() uint8
	GetLowLatches1() uint8
	GetHighLatches2() uint8
	GetHighLatches1() uint8
	GetCounter2() uint16
	GetCounter1() uint16
	GetShiftRegister() uint8
	GetAuxiliaryControl() uint8
	GetPeripheralControl() uint8
	GetInterruptFlagValue() uint8
	GetInterruptEnabledFlag() uint8
}

// NANDGatesChip defines the interface for the 74HC00 quad NAND gate chip.
// This chip contains four independent two-input NAND gates.
type NANDGatesChip interface {
	// Returns the connector for pin A at the specified index (0-3)
	APin(index int) buses.LineConnector

	// Returns the connector for pin B at the specified index (0-3)
	BPin(index int) buses.LineConnector

	// Returns the connector for pin Y at the specified index (0-3)
	YPin(index int) buses.LineConnector

	// Processes one clock tick
	Tick(context *common.StepContext)
}
