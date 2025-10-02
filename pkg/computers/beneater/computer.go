package beneater

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/fran150/clementina-6502/pkg/computers"
	"github.com/fran150/clementina-6502/pkg/core/interfaces"
	"github.com/fran150/clementina-6502/pkg/core/managers"
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
	system      *computers.ComputerSystem
	chips       *chips
	circuit     *circuit
	console     *console
	resetCycles uint8

	// Performance optimization: cache frequently accessed state
	stateManager        *managers.StateManager
	lastBreakpointCheck uint64
}

type BenEaterComputerConfig struct {
	DisplayFps        int
	Port              serial.Port
	EmulateModemLines bool
}

/*******************************************************************************************
* ComputerCore Interface methods (Emulator + Renderer)
********************************************************************************************/

// Run starts the emulation loop and runs the console application.
func (c *BenEaterComputer) Run() (*common.StepContext, error) {
	context, err := c.system.Start()
	if err != nil {
		return nil, err
	}

	if err := c.console.Run(); err != nil {
		c.system.Stop()
		return nil, err
	}

	return context, nil
}

// Stop stops computer execution and finishes the console application.
func (c *BenEaterComputer) Stop() {
	c.system.Stop()
	c.console.Stop()
}

// Tick advances the computer's state by one cycle.
// It updates all components if the computer is not paused or if a single step is requested.
// It also handles breakpoints and resets.
//
// Parameters:
//   - context: The current step context
func (c *BenEaterComputer) Tick(context *common.StepContext) {
	// Cache state manager for performance
	if c.stateManager == nil {
		c.stateManager = c.system.GetStateManager()
	}

	if !c.stateManager.IsPaused() || c.stateManager.IsStepping() {
		// Core emulation - keep this tight for performance
		c.chips.cpu.Tick(context)
		c.chips.nand.Tick(context)
		c.chips.ram.Tick(context)
		c.chips.rom.Tick(context)
		c.chips.via.Tick(context)
		c.chips.lcd.Tick(context)
		c.chips.acia.Tick(context)

		c.chips.cpu.PostTick(context)
		c.checkReset()

		// Clear stepping state
		if c.stateManager.IsStepping() {
			c.stateManager.ClearStepping()
		}

		// Only check breakpoints every 100 cycles for performance
		if c.chips.cpu.IsReadingOpcode() && (context.Cycle-c.lastBreakpointCheck) > 100 {
			c.lastBreakpointCheck = context.Cycle
			if breakpointWindow := managers.GetWindow[ui.BreakPointForm](c.console.GetWindowManager(), "breakpoint"); breakpointWindow != nil {
				if breakpointWindow.CheckBreakpoint(c.chips.cpu.GetProgramCounter() - 1) {
					c.stateManager.Pause()
				}
			}
		}

		// Console tick is expensive, do it less frequently
		if (context.Cycle%10) == 0 && c.console != nil {
			c.console.Tick(context)
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
	if c.stateManager.IsResetting() {
		c.circuit.cpuReset.Set(false)
		c.resetCycles++
		if c.resetCycles > 5 {
			c.stateManager.Unreset()
			c.resetCycles = 0
		}
	} else {
		c.circuit.cpuReset.Set(true)
	}
}

/*******************************************************************************************
* Controller Interface methods (delegated to system)
********************************************************************************************/

// Pause stops the execution of the computer.
func (c *BenEaterComputer) Pause() {
	c.system.Pause()
}

// Resume continues the execution of the computer after being paused.
func (c *BenEaterComputer) Resume() {
	c.system.Resume()
}

// Reset triggers a reset of the computer.
func (c *BenEaterComputer) Reset() {
	c.system.Reset()
}

// Step signals that the computer should step through one cycle.
func (c *BenEaterComputer) Step() {
	c.system.Step()
}

// SpeedUp increases the emulation speed.
func (c *BenEaterComputer) SpeedUp() {
	c.system.SpeedUp()
}

// SpeedDown decreases the emulation speed.
func (c *BenEaterComputer) SpeedDown() {
	c.system.SpeedDown()
}

// IsRunning checks if the computer is currently running.
func (c *BenEaterComputer) IsRunning() bool {
	return c.system.IsRunning()
}

// IsPaused checks if the computer is currently paused.
func (c *BenEaterComputer) IsPaused() bool {
	return c.system.IsPaused()
}

// GetTargetSpeed returns the current target speed in MHz.
func (c *BenEaterComputer) GetTargetSpeed() float64 {
	return c.system.GetTargetSpeed()
}

// GetSpeedController returns the speed controller for direct access.
func (c *BenEaterComputer) GetSpeedController() interfaces.SpeedController {
	return c.system.GetSpeedController()
}
