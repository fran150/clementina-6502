package clementina

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/fran150/clementina-6502/pkg/computers/clementina/modules"
)

/*******************************************************************************************
* Structs definition
********************************************************************************************/

type mapperFunctions[T uint8 | uint16, S uint8 | uint16] struct {
	MapToSource   func(value T, current []S) []S
	MapFromSource func(value []S) T
}

type mappers struct {
	portA   mapperFunctions[uint16, uint8]
	mia     mapperFunctions[uint8, uint16]
	exRam   mapperFunctions[uint16, uint16]
	exRamHi mapperFunctions[uint16, uint16]
}

type chips struct {
	cpu      components.Cpu65C02
	baseram  components.Memory
	exram    components.Memory
	via      components.Via65C22
	mia      components.MiaChip
	csLogic  *modules.ClementinaCSLogic
	oeRWSync *modules.ClementinaOERWPHISync
}

type circuit struct {
	addressBus buses.Bus[uint16]
	dataBus    buses.Bus[uint8]
	cpuIRQ     *buses.StandaloneLine
	cpuReset   *buses.StandaloneLine
	cpuRW      *buses.StandaloneLine

	miaBus       buses.Bus[uint8]
	exramBus     buses.Bus[uint16]
	exramBusHigh buses.Bus[uint16]
	portABus     buses.Bus[uint8]
	bigPortA     buses.Bus[uint16]
	portBBus     buses.Bus[uint8]

	miaCS *buses.StandaloneLine

	vcc    *buses.StandaloneLine
	ground *buses.StandaloneLine
}

// Clementina represents a complete emulation of Clementina 6502 computer.
// It contains all the necessary components and connections to simulate the hardware.
type ClementinaComputer struct {
	chips   *chips
	circuit *circuit

	mappers mappers
}

/*******************************************************************************************
* ComputerCore Interface methods
********************************************************************************************/

// Tick advances the computer's state by one cycle.
//
// Parameters:
//   - context: The current step context
func (c *ClementinaComputer) Tick(context *common.StepContext) {
	// Core emulation - keep this tight for performance
	c.chips.cpu.Tick(context)

	c.chips.csLogic.Tick(context)
	c.chips.oeRWSync.Tick(context)

	c.chips.via.Tick(context)

	c.chips.baseram.Tick(context)
	c.chips.mia.Tick(context)
	c.chips.exram.Tick(context)

	c.chips.cpu.PostTick(context)
}

// GetProgramCounter returns the current program counter value from the CPU.
//
// Returns:
//   - The current program counter as a uint16 value
func (c *ClementinaComputer) GetProgramCounter() uint16 {
	return c.chips.cpu.GetProgramCounter()
}

// Reset sets the reset state of the computer.
//
// Parameters:
//   - status: true to reset the computer, false to release from reset
func (c *ClementinaComputer) Reset(status bool) {
	c.circuit.cpuReset.Set(!status)
}

/*******************************************************************************************
* Miscellaneous functions
********************************************************************************************/

// getPotentialOperators retrieves the next two bytes from memory at the given program counter.
// It handles mapped Clementina memory regions that support side-effect-free peeking.
//
// Parameters:
//   - programCounter: The 16-bit program counter address
//
// Returns:
//   - An array of two bytes representing the potential operands
func (c *ClementinaComputer) getPotentialOperators(programCounter uint16) [2]uint8 {
	op1, _ := c.peekMappedMemory(programCounter)
	op2, _ := c.peekMappedMemory(programCounter + 1)

	return [2]uint8{op1, op2}
}

// peekMappedMemory returns a byte from the current Clementina memory map without bus side effects.
func (c *ClementinaComputer) peekMappedMemory(address uint16) (uint8, bool) {
	switch {
	case address < 0x8000:
		return c.chips.baseram.Peek(uint32(address & 0x7FFF)), true

	case address >= 0x8000 && address < 0xC000:
		return c.chips.exram.Peek(c.mapExRAMAddress(address)), true

	case address >= 0xE000:
		if mia, ok := c.chips.mia.(interface{ Peek(uint16) uint8 }); ok {
			return mia.Peek(address), true
		}
	}

	return 0, false
}

// mapExRAMAddress maps a CPU address in the extended RAM window to the physical RAM offset.
func (c *ClementinaComputer) mapExRAMAddress(address uint16) uint32 {
	portA := c.circuit.portABus.Read()
	portA16Bits := c.mappers.portA.MapFromSource([]uint8{portA})
	addressLow := c.mappers.exRam.MapFromSource([]uint16{address, portA16Bits})
	addressHi := c.mappers.exRamHi.MapFromSource([]uint16{portA16Bits})

	return uint32(addressLow) | (uint32(addressHi) << 16)
}

// BaseRamPoke writes a value directly to the base RAM at the specified address.
// This bypasses normal CPU memory access and is used for debugging or initialization.
//
// Parameters:
//   - address: The 16-bit address in base RAM to write to
//   - value: The 8-bit value to write
func (c *ClementinaComputer) BaseRamPoke(address uint16, value uint8) {
	c.chips.baseram.Poke(address, value)
}

// ExRamPoke writes a value directly to the extended RAM at the specified address and bank.
// This bypasses normal CPU memory access and is used for debugging or initialization.
//
// Parameters:
//   - address: The 16-bit address in extended RAM to write to
//   - bank: The bank number (32 banks of 16K)
//   - value: The 8-bit value to write
func (c *ClementinaComputer) ExRamPoke(address uint16, bank uint8, value uint8) {
	bank = bank & 0x1F
	mapped := c.mappers.exRam.MapFromSource([]uint16{address, uint16(bank)})
	c.chips.exram.Poke(mapped, value)
}
