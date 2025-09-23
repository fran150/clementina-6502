package beneater

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/fran150/clementina-6502/pkg/computers"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"go.bug.st/serial"
)

/*******************************************************************************************
* Structs definition
********************************************************************************************/

type chips struct {
	cpu  components.Cpu6502Chip
	ram  components.MemoryChip
	rom  components.MemoryChip
	via  components.ViaChip
	lcd  components.LCDControllerChip
	acia components.Acia6522Chip
	nand components.NANDGatesChip
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

// BenEaterComputer represents a complete emulation of Ben Eater's 6502 computer.
// It contains all the necessary components and connections to simulate the hardware.
type BenEaterComputer struct {
	computers.BaseComputer

	chips       *chips
	circuit     *circuit
	console     *console
	resetCycles uint8
}

type BenEaterComputerConfig struct {
	computers.EmulationLoopConfig
	Port              serial.Port
	EmulateModemLines bool
}

/*******************************************************************************************
* Computer Interface methods
********************************************************************************************/

// Run starts the emulation loop and runs the console application.
func (c *BenEaterComputer) Run() (*common.StepContext, error) {
	context := c.BaseComputer.Run()

	if err := c.console.Run(); err != nil {
		c.BaseComputer.Stop()
		return nil, err
	}

	return context, nil
}

// Stop stops computer execution and finishes the console application.
func (c *BenEaterComputer) Stop() {
	c.BaseComputer.Stop()
	c.console.Stop()
}

// Tick advances the computer's state by one cycle.
// It updates all components if the computer is not paused or if a single step is requested.
// It also handles breakpoints and resets.
//
// Parameters:
//   - context: The current step context
func (c *BenEaterComputer) Tick(context *common.StepContext) {
	if !c.IsPaused() || c.IsStepping() {
		c.chips.cpu.Tick(context)
		c.chips.nand.Tick(context)
		c.chips.ram.Tick(context)
		c.chips.rom.Tick(context)
		c.chips.via.Tick(context)
		c.chips.lcd.Tick(context)
		c.chips.acia.Tick(context)

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
func (c *BenEaterComputer) Draw(context *common.StepContext) {
	c.console.Draw(context)
}

/*******************************************************************************************
* Miscellaneous functions
********************************************************************************************/

// Close performs cleanup operations when shutting down the computer.
// It ensures that the ACIA component is properly closed to release resources.
func (c *BenEaterComputer) Close() {
	c.chips.acia.Close()
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

// getPotentialOperators retrieves the next two bytes from ROM at the given program counter.
func (c *BenEaterComputer) getPotentialOperators(programCounter uint16) [2]uint8 {
	rom := c.chips.rom
	programCounter &= 0x7FFF
	operand1Address := programCounter & 0x7FFF
	operand2Address := (programCounter + 1) & 0x7FFF
	return [2]uint8{rom.Peek(uint32(operand1Address)), rom.Peek(uint32(operand2Address))}
}

// checkReset handles the reset signal timing for the CPU.
// In order to reset the CPU, it must be held low for a certain number of cycles.
func (c *BenEaterComputer) checkReset() {
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
