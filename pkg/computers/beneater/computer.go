package beneater

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"go.bug.st/serial"
)

/*******************************************************************************************
* Structs definition
********************************************************************************************/

type chips struct {
	cpu  components.Cpu65C02
	ram  components.Memory
	rom  components.Memory
	via  components.Via65C22
	lcd  components.LCDController
	acia components.Acia65C51
	nand components.LogicGateArray
}

type circuit struct {
	addressBus buses.Bus[uint16]
	dataBus    buses.Bus[uint8]
	cpuIRQ     *buses.StandaloneLine
	cpuReset   *buses.StandaloneLine
	cpuRW      *buses.StandaloneLine
	u4dOut     *buses.StandaloneLine
	u4cOut     *buses.StandaloneLine
	u4bOut     *buses.StandaloneLine
	fiveVolts  *buses.StandaloneLine
	ground     *buses.StandaloneLine
	portABus   buses.Bus[uint8]
	portBBus   buses.Bus[uint8]
	lcdBus     buses.Bus[uint8]
	serial     serial.Port
}

// BenEaterComputerConfig holds configuration options for creating a new BenEaterComputer.
// It specifies the serial port and modem line emulation settings.
type BenEaterComputerConfig struct {
	Port              serial.Port
	EmulateModemLines bool
}

// BenEaterComputer represents a complete emulation of Ben Eater's 6502 computer.
// It contains all the necessary components and connections to simulate the hardware.
type BenEaterComputer struct {
	chips   *chips
	circuit *circuit
}

/*******************************************************************************************
* ComputerCore Interface methods
********************************************************************************************/

// Tick advances the computer's state by one cycle.
//
// Parameters:
//   - context: The current step context
func (c *BenEaterComputer) Tick(context *common.StepContext) {
	// Core emulation - keep this tight for performance
	c.chips.cpu.Tick(context)
	c.chips.nand.Tick(context)
	c.chips.ram.Tick(context)
	c.chips.rom.Tick(context)
	c.chips.via.Tick(context)
	c.chips.lcd.Tick(context)
	c.chips.acia.Tick(context)

	c.chips.cpu.PostTick(context)
}

// GetProgramCounter returns the current program counter value from the CPU.
//
// Returns:
//   - The current program counter as a uint16 value
func (c *BenEaterComputer) GetProgramCounter() uint16 {
	return c.chips.cpu.GetProgramCounter()
}

// Reset sets the reset state of the computer.
//
// Parameters:
//   - status: true to reset the computer, false to release from reset
func (c *BenEaterComputer) Reset(status bool) {
	c.circuit.cpuReset.Set(!status)
}

/*******************************************************************************************
* Miscellaneous functions
********************************************************************************************/

// Close performs cleanup operations when shutting down the computer.
// It ensures that the ACIA component is properly closed to release resources.
func (c *BenEaterComputer) Close() {
	c.chips.acia.Close()
}

// getPotentialOperators retrieves the next two bytes from ROM at the given program counter.
func (c *BenEaterComputer) getPotentialOperators(programCounter uint16) [2]uint8 {
	rom := c.chips.rom
	programCounter &= 0x7FFF
	operand1Address := programCounter & 0x7FFF
	operand2Address := (programCounter + 1) & 0x7FFF
	return [2]uint8{rom.Peek(uint32(operand1Address)), rom.Peek(uint32(operand2Address))}
}

// LoadRom loads a ROM image from the specified file path into the computer's ROM.
//
// Parameters:
//   - romImagePath: The path to the ROM image file
//
// Returns:
//   - An error if the ROM image could not be loaded, nil otherwise
func (c *BenEaterComputer) LoadRom(romImagePath string) error {
	err := c.chips.rom.Load(romImagePath)
	if err != nil {
		return err
	}

	return nil
}
