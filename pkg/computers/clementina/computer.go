package clementina

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/fran150/clementina-6502/pkg/computers"
	"github.com/fran150/clementina-6502/pkg/computers/clementina/modules"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
)

type mapperFunctions[T uint8 | uint16, S uint8 | uint16] struct {
	MapToSource   func(value T, current []S) []S
	MapFromSource func(value []S) T
}

type mappers struct {
	portA   mapperFunctions[uint16, uint8]
	hiRam   mapperFunctions[uint16, uint16]
	exRam   mapperFunctions[uint16, uint16]
	exRamHi mapperFunctions[uint16, uint16]
}

type chips struct {
	cpu      components.Cpu6502Chip
	baseram  components.MemoryChip
	hiram    components.MemoryChip
	exram    components.MemoryChip
	via      components.ViaChip
	csLogic  *modules.ClementinaCSLogic
	oeRWSync *modules.ClementinaOERWPHISync
}

type circuit struct {
	addressBus buses.Bus[uint16]
	dataBus    buses.Bus[uint8]
	cpuIRQ     *buses.StandaloneLine
	cpuReset   *buses.StandaloneLine
	cpuRW      *buses.StandaloneLine

	hiramBus     buses.Bus[uint16]
	exramBus     buses.Bus[uint16]
	exramBusHigh buses.Bus[uint16]
	portABus     buses.Bus[uint8]
	bigPortA     buses.Bus[uint16]
	portBBus     buses.Bus[uint8]

	picoHiRAME *buses.StandaloneLine

	vcc    *buses.StandaloneLine
	ground *buses.StandaloneLine
}

// Clementina represents a complete emulation of Clementina 6502 computer.
// It contains all the necessary components and connections to simulate the hardware.
type ClementinaComputer struct {
	computers.BaseComputer

	chips       *chips
	circuit     *circuit
	console     *console
	resetCycles uint8

	mappers mappers
}

/*******************************************************************************************
* Computer Interface methods
********************************************************************************************/

// Run starts the emulation loop and runs the console application.
func (c *ClementinaComputer) Run() (*common.StepContext, error) {
	context := c.BaseComputer.Run()

	if err := c.console.Run(); err != nil {
		c.BaseComputer.Stop()
		return nil, err
	}

	return context, nil
}

// Stop stops computer execution and finishes the console application.
func (c *ClementinaComputer) Stop() {
	c.BaseComputer.Stop()
	c.console.Stop()
}

// Tick advances the computer's state by one cycle.
// It updates all components if the computer is not paused or if a single step is requested.
// It also handles breakpoints and resets.
//
// Parameters:
//   - context: The current step context
func (c *ClementinaComputer) Tick(context *common.StepContext) {
	if !c.IsPaused() || c.IsStepping() {
		c.chips.cpu.Tick(context)

		c.chips.csLogic.Tick(context)
		c.chips.oeRWSync.Tick(context)

		c.chips.via.Tick(context)

		c.chips.baseram.Tick(context)
		c.chips.hiram.Tick(context)
		c.chips.exram.Tick(context)

		c.chips.cpu.PostTick(context)

		c.checkReset()

		if c.console != nil {
			c.console.Tick(context)
		}

		c.ClearStepping()

		if c.chips.cpu.IsReadingOpcode() {
			if breakpointForm := computers.GetWindow[ui.BreakPointForm](&c.console.BaseConsole, "breakpoint"); breakpointForm != nil {
				if breakpointForm.CheckBreakpoint(c.chips.cpu.GetProgramCounter() - 1) {
					c.Pause()
				}
			}
		}
	}
}

// Draw renders the computer's UI to the terminal.
// It delegates the drawing to the console component.
//
// Parameters:
//   - context: The current step context
func (c *ClementinaComputer) Draw(context *common.StepContext) {
	c.console.Draw(context)
}

// Close performs cleanup operations when shutting down the computer.
func (c *ClementinaComputer) Close() {
}

// getPotentialOperators retrieves the next two bytes from memory at the given program counter.
// It handles different memory regions (base RAM, extended RAM, high RAM) based on the address.
//
// Parameters:
//   - programCounter: The 16-bit program counter address
//
// Returns:
//   - An array of two bytes representing the potential operands
func (c *ClementinaComputer) getPotentialOperators(programCounter uint16) [2]uint8 {
	var chip components.MemoryChip
	var address uint32

	switch {
	case programCounter < 0x8000:
		address = uint32(programCounter & 0x7FFF)
		chip = c.chips.baseram

	case programCounter >= 0x8000 && programCounter < 0xC000:
		portA := c.circuit.portABus.Read()
		portA16Bits := c.mappers.portA.MapFromSource([]uint8{portA})
		addressLow := c.mappers.exRam.MapFromSource([]uint16{programCounter, portA16Bits})
		addressHi := c.mappers.exRamHi.MapFromSource([]uint16{portA16Bits})

		address = uint32(addressLow) | (uint32(addressHi) << 16)

		chip = c.chips.exram
	case programCounter >= 0xE000:
		portA := c.circuit.portABus.Read()
		portA16Bits := c.mappers.portA.MapFromSource([]uint8{portA})

		address = uint32(c.mappers.hiRam.MapFromSource([]uint16{programCounter, portA16Bits}))

		chip = c.chips.hiram

		// TODO: Add logic for reading when PICO is enabled
	}

	op1 := chip.Peek(address)
	op2 := chip.Peek(address + 1)

	return [2]uint8{op1, op2}
}

// checkReset handles the reset signal timing for the CPU.
// On 6502 systems the reset signal must be held low for a certain number of cycles
func (c *ClementinaComputer) checkReset() {
	if c.IsResetting() {
		c.circuit.cpuReset.Set(false)
		c.resetCycles++
		if c.resetCycles > 5 {
			c.Unreset()
			c.resetCycles = 0
		}
	} else {
		c.circuit.cpuReset.Set(true)
	}
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

// HiRamPoke writes a value directly to the high RAM at the specified address and bank.
// This bypasses normal CPU memory access and is used for debugging or initialization.
//
// Parameters:
//   - address: The 16-bit address in high RAM to write to
//   - bank: The bank number (4 banks of 8K)
//   - value: The 8-bit value to write
func (c *ClementinaComputer) HiRamPoke(address uint16, bank uint8, value uint8) {
	bank = (bank & 0x03) << 5
	mapped := c.mappers.hiRam.MapFromSource([]uint16{address, uint16(bank)})
	c.chips.hiram.Poke(mapped, value)
}
