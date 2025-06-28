package clementina

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/fran150/clementina-6502/pkg/components/cpu"
	"github.com/fran150/clementina-6502/pkg/components/memory"
	"github.com/fran150/clementina-6502/pkg/components/via"
	"github.com/fran150/clementina-6502/pkg/computers/clementina/modules"
	"github.com/fran150/clementina-6502/pkg/terminal"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

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
	chips       *chips
	circuit     *circuit
	console     *console
	mustReset   bool
	resetCycles uint8
	appConfig   *terminal.ApplicationConfig

	pause bool
	step  bool
}

// NewClementinaComputer creates and initializes a new instance of the Clementina 6502 computer emulation.
// It sets up all hardware components and connects them according to the design
//
// Returns:
//   - A pointer to the initialized ClementinaComputer
//   - An error if initialization fails
func NewClementinaComputer() (*ClementinaComputer, error) {
	chips := &chips{
		cpu:      cpu.NewCpu65C02S(),
		baseram:  memory.NewRam(memory.RAM_SIZE_32K),
		exram:    memory.NewRam(memory.RAM_SIZE_512K),
		hiram:    memory.NewRam(memory.RAM_SIZE_32K),
		via:      via.NewVia65C22(),
		csLogic:  modules.NewClementinaCSLogic(),
		oeRWSync: modules.NewClementinaOERWPHISync(),
	}

	addressBus := buses.New16BitStandaloneBus()
	portABus := buses.New8BitStandaloneBus()
	portBBus := buses.New8BitStandaloneBus()

	// Transform port A in a 16-bit bus
	portAToBigPortA := func(value []uint8) uint16 {
		return uint16(value[0])
	}

	bigPortAtoPortA := func(value uint16, current []uint8) []uint8 {
		return []uint8{uint8(value)}
	}

	// HiRAM mapped bus uses A0 - A11 from the address bus
	// and A12 - A14 is mapped to PORTA 5 - 7
	sourcesToHiRamBus := func(value []uint16) uint16 {
		address := value[0]
		portA := value[1]

		address &= 0x0FFF // Remove A12 - A15
		portA &= 0xE0     // Keep only PA5 - PA7

		return (portA << 8) | address // PA5 - PA7 | A11 - A0
	}

	hiRamToSourceBuses := func(value uint16, current []uint16) []uint16 {
		address := (current[0] & 0xF000) | (value & 0x0FFF)  // Replace A0 - A11
		portA := (current[1] & 0x1F) | ((value & 0xE0) >> 8) // Replace A5 - A7

		return []uint16{address, portA}
	}

	// ExRAM mapped bus uses A0 - A13 from the address bus
	// and A14 - A15 is mapped to PORTA 0 - 2
	sourcesToExRamBus := func(value []uint16) uint16 {
		address := value[0]
		portA := value[1]

		address &= 0x3FFF // Remove A14 - A15
		portA &= 0x03     // Keep only PA0 - PA1

		return (portA << 14) | address // PA0 - PA1 | A13 - A0
	}

	exRamToSourceBuses := func(value uint16, current []uint16) []uint16 {
		address := (current[0] & 0xC000) | (value & 0x3FFF)   // Replace A0 - A13
		portA := (current[1] & 0xFC) | ((value & 0x03) >> 14) // Replace A0 - A1

		return []uint16{address, portA}
	}

	// ExRAMHi uses PORTA 2 - 4 on the pins A0 - A2
	sourcesToExRamBusHigh := func(value []uint16) uint16 {
		portA := value[0] & 0x1C // Keep only PA2 - PA4
		return portA >> 2        // Shift PA2 - PA4 to A0 - A2
	}

	exRamHiToSourceBuses := func(value uint16, current []uint16) []uint16 {
		portA := (current[0] & 0xE3) | (value << 2) // Replace A0 - A2 with PA2 - PA4
		return []uint16{portA}
	}

	bigPortABus := buses.New16BitMappedBus(
		[]buses.Bus[uint8]{portABus},
		bigPortAtoPortA,
		portAToBigPortA,
	)

	hiRamBus := buses.New16BitMappedBus(
		[]buses.Bus[uint16]{addressBus, bigPortABus},
		hiRamToSourceBuses,
		sourcesToHiRamBus,
	)

	exRamBus := buses.New16BitMappedBus(
		[]buses.Bus[uint16]{addressBus, bigPortABus},
		exRamToSourceBuses,
		sourcesToExRamBus,
	)

	exRamBusHigh := buses.New16BitMappedBus(
		[]buses.Bus[uint16]{bigPortABus},
		exRamHiToSourceBuses,
		sourcesToExRamBusHigh,
	)

	circuit := &circuit{
		addressBus:   addressBus,
		dataBus:      buses.New8BitStandaloneBus(),
		cpuIRQ:       buses.NewStandaloneLine(true),
		cpuReset:     buses.NewStandaloneLine(true),
		hiramBus:     hiRamBus,
		exramBus:     exRamBus,
		exramBusHigh: exRamBusHigh,
		portABus:     portABus,
		bigPortA:     bigPortABus,
		portBBus:     portBBus,
		picoHiRAME:   buses.NewStandaloneLine(true),
		vcc:          buses.NewStandaloneLine(true),
		ground:       buses.NewStandaloneLine(false),
	}

	addressBus15 := circuit.addressBus.GetBusLine(15)
	addressBus14 := circuit.addressBus.GetBusLine(14)
	addressBus13 := circuit.addressBus.GetBusLine(13)
	addressBus12 := circuit.addressBus.GetBusLine(12)
	addressBus11 := circuit.addressBus.GetBusLine(11)
	addressBus10 := circuit.addressBus.GetBusLine(10)

	addressBus3 := circuit.addressBus.GetBusLine(3)
	addressBus2 := circuit.addressBus.GetBusLine(2)
	addressBus1 := circuit.addressBus.GetBusLine(1)
	addressBus0 := circuit.addressBus.GetBusLine(0)

	// 6502 CPU connections
	chips.cpu.AddressBus().Connect(circuit.addressBus)
	chips.cpu.DataBus().Connect(circuit.dataBus)
	chips.cpu.Ready().Connect(circuit.vcc)
	chips.cpu.InterruptRequest().Connect(circuit.cpuIRQ)
	chips.cpu.NonMaskableInterrupt().Connect(circuit.vcc)
	chips.cpu.Reset().Connect(circuit.cpuReset)
	chips.cpu.BusEnable().Connect(circuit.vcc)
	chips.cpu.ReadWrite().Connect(circuit.cpuRW)

	// Connect the CPU to the CS Logic
	chips.csLogic.A1(0).Connect(addressBus10)
	chips.csLogic.A1(1).Connect(addressBus11)
	chips.csLogic.A1(2).Connect(addressBus12)
	chips.csLogic.A1(3).Connect(addressBus13)
	chips.csLogic.A1(4).Connect(addressBus14)
	chips.csLogic.A1(5).Connect(addressBus15)
	chips.csLogic.PicoHiRAME().Connect(circuit.picoHiRAME)

	// Connect the CPU to the OE/RW sync module
	chips.oeRWSync.CpuRW().Connect(circuit.cpuRW)

	// Base RAM connections
	chips.baseram.AddressBus().Connect(circuit.addressBus)
	chips.baseram.DataBus().Connect(circuit.dataBus)
	chips.baseram.WriteEnable().Connect(chips.oeRWSync.RW())
	chips.baseram.OutputEnable().Connect(chips.oeRWSync.OE())
	chips.baseram.ChipSelect().Connect(addressBus15)

	// HiRAM connections
	chips.hiram.AddressBus().Connect(circuit.hiramBus)
	chips.hiram.DataBus().Connect(circuit.dataBus)
	chips.hiram.WriteEnable().Connect(chips.oeRWSync.RW())
	chips.hiram.OutputEnable().Connect(chips.oeRWSync.OE())
	chips.hiram.ChipSelect().Connect(chips.csLogic.HiRAME())

	// VIA connections
	chips.via.DataBus().Connect(circuit.dataBus)
	chips.via.IrqRequest().Connect(circuit.cpuIRQ)
	chips.via.ReadWrite().Connect(circuit.cpuRW)
	chips.via.ChipSelect2().Connect(chips.csLogic.IOOE().GetBusLine(0))
	chips.via.ChipSelect1().Connect(circuit.vcc)
	chips.via.Reset().Connect(circuit.cpuReset)
	chips.via.RegisterSelect(3).Connect(addressBus3)
	chips.via.RegisterSelect(2).Connect(addressBus2)
	chips.via.RegisterSelect(1).Connect(addressBus1)
	chips.via.RegisterSelect(0).Connect(addressBus0)
	chips.via.PeripheralPortB().Connect(circuit.portBBus)

	// EXRam connections
	chips.exram.AddressBus().Connect(circuit.exramBus)
	chips.exram.HiAddressBus().Connect(circuit.exramBusHigh)
	chips.exram.DataBus().Connect(circuit.dataBus)
	chips.exram.WriteEnable().Connect(chips.oeRWSync.RW())
	chips.exram.OutputEnable().Connect(chips.oeRWSync.OE())
	chips.exram.ChipSelect().Connect(chips.csLogic.ExRAME())

	return &ClementinaComputer{
		chips:       chips,
		circuit:     circuit,
		mustReset:   false,
		resetCycles: 0,
	}, nil
}

// LoadRom loads a ROM image from the specified file path into the computer's base RAM.
//
// Parameters:
//   - romImagePath: The path to the ROM image file
//
// Returns:
//   - An error if the ROM image could not be loaded, nil otherwise
func (c *ClementinaComputer) LoadRom(romImagePath string) error {
	err := c.chips.baseram.Load(romImagePath)
	if err != nil {
		return err
	}

	return nil
}

// Init initializes the computer with the provided application and configuration.
// It sets up the console interface and enables mouse and paste functionality.
//
// Parameters:
//   - tvApp: The tview application instance
//   - appConfig: The application configuration
func (c *ClementinaComputer) Init(tvApp *tview.Application, appConfig *terminal.ApplicationConfig) {
	tvApp.EnableMouse(true).EnablePaste(true)

	c.appConfig = appConfig
	c.console = newMainConsole(c, tvApp)
}

// Tick advances the computer's state by one cycle.
// It updates all components if the computer is not paused or if a single step is requested.
// It also handles breakpoints and resets.
//
// Parameters:
//   - context: The current step context
func (c *ClementinaComputer) Tick(context *common.StepContext) {
	if !c.pause || c.step {
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

		c.step = false

		if c.chips.cpu.IsReadingOpcode() {
			if breakpointForm := GetWindow[ui.BreakPointForm](c.console, "breakpoint"); breakpointForm != nil {
				if breakpointForm.CheckBreakpoint(c.chips.cpu.GetProgramCounter() - 1) {
					c.pause = true
				}
			}
		}
	}
}

// Pause stops the execution of the computer.
// The computer will remain paused until Resume or Step is called.
//
// Parameters:
//   - context: The current step context
func (c *ClementinaComputer) Pause(context *common.StepContext) {
	c.pause = true
}

// Resume continues the execution of the computer after being paused.
// It sets the step flag to true to ensure at least one cycle executes.
//
// Parameters:
//   - context: The current step context
func (c *ClementinaComputer) Resume(context *common.StepContext) {
	c.pause = false
	c.step = true
}

// Step executes a single cycle of the computer while in pause mode.
// This allows for step-by-step debugging of the computer's operation.
//
// Parameters:
//   - context: The current step context
func (c *ClementinaComputer) Step(context *common.StepContext) {
	c.step = true
}

// Draw renders the computer's UI to the terminal.
// It delegates the drawing to the console component.
//
// Parameters:
//   - context: The current step context
func (c *ClementinaComputer) Draw(context *common.StepContext) {
	c.console.Draw(context)
}

// Stop signals that the computer should stop execution.
// This sets the Stop flag in the context to true.
//
// Parameters:
//   - context: The current step context
func (c *ClementinaComputer) Stop(context *common.StepContext) {
	context.Stop = true
}

// Reset signals that the computer should be reset.
// This sets the mustReset flag which will be processed during the next tick.
//
// Parameters:
//   - context: The current step context
func (c *ClementinaComputer) Reset(context *common.StepContext) {
	c.mustReset = true
}

// SkipUp increases the number of cycles to skip during execution
func (c *ClementinaComputer) SkipUp(context *common.StepContext, size int64) {
	c.appConfig.SkipCycles += size
}

// SkipDown decreases the number of cycles to skip during execution.
func (c *ClementinaComputer) SkipDown(context *common.StepContext, size int64) {
	c.appConfig.SkipCycles -= size

	if c.appConfig.SkipCycles < 0 {
		c.appConfig.SkipCycles = 0
	}
}

// KeyPressed handles keyboard events for the computer.
// It processes key presses through the options window if available,
// or returns the event unchanged.
//
// Parameters:
//   - event: The keyboard event to process
//   - context: The current step context
//
// Returns:
//   - The processed event or the original event if not handled
func (c *ClementinaComputer) KeyPressed(event *tcell.EventKey, context *common.StepContext) *tcell.EventKey {
	if options := GetWindow[ui.OptionsWindow](c.console, "options"); options != nil {
		return options.ProcessKey(event, context)
	}

	return event
}

// Close performs cleanup operations when shutting down the computer.
// It ensures that the ACIA component is properly closed to release resources.
func (c *ClementinaComputer) Close() {
}

func (c *ClementinaComputer) getPotentialOperators(programCounter uint16) [2]uint8 {
	// TODO: Fix to use correct mapping
	rom := c.chips.baseram
	programCounter &= 0x7FFF
	operand1Address := programCounter & 0x7FFF
	operand2Address := (programCounter + 1) & 0x7FFF
	return [2]uint8{rom.Peek(uint32(operand1Address)), rom.Peek(uint32(operand2Address))}
}

func (c *ClementinaComputer) checkReset() {
	if c.mustReset {
		c.circuit.cpuReset.Set(false)
		c.resetCycles++
		if c.resetCycles > 5 {
			c.mustReset = false
			c.resetCycles = 0
		}
	} else {
		c.circuit.cpuReset.Set(true)
	}
}
