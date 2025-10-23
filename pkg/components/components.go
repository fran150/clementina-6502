package components

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/fran150/clementina-6502/pkg/core"
	"go.bug.st/serial"
)

// Core interfaces for composition

// PostTicker defines components that need post-tick processing
type PostTicker interface {
	PostTick(context *common.StepContext)
}

// Resettable defines components that can be reset
type Resettable interface {
	Reset() *buses.ConnectorEnabledLow
}

// DataBusConnected defines components connected to the data bus
type DataBusConnected interface {
	DataBus() *buses.BusConnector[uint8]
}

// AddressBusConnected defines components connected to the address bus
type AddressBusConnected interface {
	AddressBus() *buses.BusConnector[uint16]
}

// ReadWriteControlled defines components with read/write control
type ReadWriteControlled interface {
	ReadWrite() *buses.ConnectorEnabledLow
}

// ChipSelectable defines components with chip select functionality in where the
// lines are numbered 0 and 1
type ChipSelectable01 interface {
	ChipSelect0() *buses.ConnectorEnabledHigh
	ChipSelect1() *buses.ConnectorEnabledLow
}

// ChipSelectable defines components with chip select functionality in where the
// lines are numbered 1 and 2
type ChipSelectable12 interface {
	ChipSelect1() *buses.ConnectorEnabledHigh
	ChipSelect2() *buses.ConnectorEnabledLow
}

// InterruptCapable defines components that can generate interrupts
type InterruptCapable interface {
	IrqRequest() *buses.ConnectorEnabledLow
}

// RegisterSelectable defines components with register selection
type RegisterSelectable interface {
	RegisterSelect(num uint8) *buses.ConnectorEnabledHigh
}

// BusComponent combines common bus-related interfaces
type BusComponent interface {
	DataBusConnected
	ReadWriteControlled
	core.Ticker
}

// SerialPortConnectable defines components that can connect to serial ports
type SerialPortConnectable interface {
	ConnectToPort(port serial.Port) error
	Close()
}

// AciaRegisters defines ACIA register access methods
type AciaRegisters interface {
	GetStatusRegister() uint8
	GetControlRegister() uint8
	GetCommandRegister() uint8
	GetTXRegister() uint8
	GetRXRegister() uint8
	IsTXRegisterEmpty() bool
	IsRXRegisterEmpty() bool
}

// Acia65C51 defines the interface for the 65C51N Asynchronous Communications Interface Adapter.
// This chip provides serial communication capabilities to the 6502 computer system.
type Acia65C51 interface {
	BusComponent
	ChipSelectable01
	Resettable
	InterruptCapable
	RegisterSelectable
	SerialPortConnectable
	AciaRegisters

	// ACIA-specific methods
	ConnectRegisterSelectLines(lines [2]buses.Line)
}

// AddressMode represents the different addressing modes available in the 6502 processor.
// Each mode determines how the CPU accesses memory for an instruction.
type AddressMode int

// AddressModeData defines the interface for CPU addressing mode information.
// It provides details about how the CPU accesses memory for different instruction types.
type AddressModeData interface {
	Name() AddressMode
	Text() string
	Format() string
	Cycles() int
	MemSize() uint8
}

// OpCode represents a unique identifier for each instruction and addressing mode combination.
// Values range from 0x00 to 0xFF (0-255), covering all possible 6502 instructions.
// See http://www.6502.org/users/obelisk/65C02/reference.html
type OpCode uint

// Mnemonic represents the human-readable assembly language representation of 6502 CPU instructions.
// Each mnemonic can be used with different addressing modes to form complete instructions.
// For example, LDA $C010 uses absolute addressing to load a value from memory address $C010,
// while LDA #$FF uses immediate addressing to load the literal value $FF.
type Mnemonic string

type CpuInstructionData interface {
	OpCode() OpCode
	Mnemonic() Mnemonic
	AddressMode() AddressMode
}

// StatusBit defines the bit positions for each flag in the status register.
// These constants are used to access and modify individual flags.
type StatusBit uint8

type StatusRegister interface {
	Flag(bit StatusBit) bool
}

// CpuControlLines defines CPU control signal interfaces
type CpuControlLines interface {
	BusEnable() *buses.ConnectorEnabledHigh
	InterruptRequest() *buses.ConnectorEnabledLow
	MemoryLock() *buses.ConnectorEnabledLow
	NonMaskableInterrupt() *buses.ConnectorEnabledLow
	SetOverflow() *buses.ConnectorEnabledLow
	Ready() *buses.ConnectorEnabledHigh
	Sync() *buses.ConnectorEnabledHigh
	VectorPull() *buses.ConnectorEnabledLow
}

// CpuRegisters defines CPU register access methods
type CpuRegisters interface {
	GetAccumulatorRegister() uint8
	GetXRegister() uint8
	GetYRegister() uint8
	GetStackPointer() uint8
	GetProcessorStatusRegister() StatusRegister
	GetProgramCounter() uint16
	ForceProgramCounter(value uint16)
}

// CpuState defines CPU execution state methods
type CpuState interface {
	IsReadingOpcode() bool
	GetCurrentInstruction() CpuInstructionData
	GetCurrentAddressMode() AddressModeData
}

// Cpu65C02 defines the interface for the 65C02S CPU emulation.
// It provides access to all CPU pins, internal registers, and execution control.
type Cpu65C02 interface {
	AddressBusConnected
	DataBusConnected
	ReadWriteControlled
	Resettable
	core.Ticker
	PostTicker
	CpuControlLines
	CpuRegisters
	CpuState
}

// Returns the data needed to display the cursor on the LCD
type CursorStatus struct {
	CursorVisible      bool
	CursorPosition     uint8 // Position of the cursor in the DDRAM
	BlinkStatusShowing bool
}

// DisplayStatus contains information about the current state of the LCD display.
// It includes configuration settings and display parameters.
type DisplayStatus struct {
	DisplayOn      bool
	Is2LineDisplay bool
	Is5x10Font     bool
	Line1Start     uint8
	Line2Start     uint8
	Is8BitMode     bool
	CGRAM          []uint8
	DDRAM          []uint8
}

// LCDStatus defines LCD status access methods
type LCDStatus interface {
	GetCursorStatus() CursorStatus
	GetDisplayStatus() DisplayStatus
}

// LCDController defines the interface for the HD44780U LCD controller.
// This chip manages a character LCD display with cursor control and display settings.
type LCDController interface {
	DataBusConnected
	ReadWriteControlled
	core.Ticker
	LCDStatus

	// LCD-specific control lines
	Enable() *buses.ConnectorEnabledHigh
	RegisterSelect() *buses.ConnectorEnabledHigh
}

// MemoryAccess defines memory access methods
type MemoryAccess interface {
	Peek(address uint32) uint8
	PeekRange(startAddress uint16, endAddress uint16) []uint8
	Poke(address uint16, value uint8)
	Load(binFilePath string) error
	Size() int
}

// MemoryControlLines defines memory control signal interfaces
type MemoryControlLines interface {
	WriteEnable() *buses.ConnectorEnabledLow
	ChipSelect() *buses.ConnectorEnabledLow
	OutputEnable() *buses.ConnectorEnabledLow
}

// Memory defines the interface for RAM and ROM memory components.
// It provides access to memory contents and control signals for memory operations.
type Memory interface {
	AddressBusConnected
	DataBusConnected
	core.Ticker
	MemoryAccess
	MemoryControlLines

	// Memory-specific address bus
	HiAddressBus() *buses.BusConnector[uint16]
}

// ViaPeripheralPorts defines VIA peripheral port access
type ViaPeripheralPorts interface {
	PeripheralPortA() *buses.BusConnector[uint8]
	PeripheralPortB() *buses.BusConnector[uint8]
	PeripheralAControlLines(num int) *buses.ConnectorEnabledHigh
	PeripheralBControlLines(num int) *buses.ConnectorEnabledHigh
}

// ViaRegisters defines VIA register access methods
type ViaRegisters interface {
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

// Via65C22 defines the interface for the 65C22S Versatile Interface Adapter.
// This chip provides parallel I/O ports, timers, and shift register functionality.
type Via65C22 interface {
	BusComponent
	ChipSelectable12
	Resettable
	InterruptCapable
	RegisterSelectable
	ViaPeripheralPorts
	ViaRegisters

	// VIA-specific methods
	ConnectRegisterSelectLines(lines [4]buses.Line)
}

// Decoder74HC138 defines the interface for a 3-to-8 line decoder
type Decoder74HC138 interface {
	core.Ticker

	APin(index int) buses.LineConnector
	YPin() *buses.BusConnector[uint8]
	EPin(index int) buses.LineConnector
}

// QuadLogicGate defines the interface for a quad logic gate component.
// It provides methods to access the A, B, and Y pins and to execute a tick.
// It is used by various logic gate implementations like NAND, AND, etc.
type QuadLogicGate interface {
	core.Ticker

	APin(index int) buses.LineConnector
	BPin(index int) buses.LineConnector
	YPin(index int) buses.LineConnector
}
