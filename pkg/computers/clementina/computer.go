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
	chips       *chips
	circuit     *circuit
	console     *console
	mustReset   bool
	resetCycles uint8
	appConfig   *terminal.ApplicationConfig

	mappers mappers

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

	// Create the address bus and via's port A and B buses
	addressBus := buses.New16BitStandaloneBus()
	portABus := buses.New8BitStandaloneBus()
	portBBus := buses.New8BitStandaloneBus()

	mappers := mappers{
		// Mapped only to convert portA from 8 bit to 16 bit
		portA: mapperFunctions[uint16, uint8]{
			MapToSource: func(value uint16, current []uint8) []uint8 {
				return []uint8{uint8(value)}
			},
			MapFromSource: func(value []uint8) uint16 {
				return uint16(value[0])
			},
		},

		// HiRAM mapped bus uses A0 - A12 from the address bus
		// and A13 - A14 is mapped to PORTA 5 - 6
		hiRam: mapperFunctions[uint16, uint16]{
			MapToSource: func(value uint16, current []uint16) []uint16 {
				address := (current[0] & 0xE000) | (value & 0x1FFF)  // Replace A0 - A12
				portA := (current[1] & 0x9F) | ((value & 0x60) >> 8) // Replace A5 - A6

				return []uint16{address, portA}
			},
			MapFromSource: func(value []uint16) uint16 {
				address := value[0]
				portA := value[1]

				address &= 0x1FFF // Remove A13 - A15
				portA &= 0x60     // Keep only PA5 - PA6

				return (portA << 8) | address // PA5 - PA6 | A12 - A0
			},
		},
		// ExRAM mapped bus uses A0 - A13 from the address bus
		// and A14 - A15 is mapped to PORTA 0 - 2
		exRam: mapperFunctions[uint16, uint16]{
			MapToSource: func(value uint16, current []uint16) []uint16 {
				address := (current[0] & 0xC000) | (value & 0x3FFF)   // Replace A0 - A13
				portA := (current[1] & 0xFC) | ((value & 0x03) >> 14) // Replace A0 - A1

				return []uint16{address, portA}
			},

			MapFromSource: func(value []uint16) uint16 {
				address := value[0]
				portA := value[1]

				address &= 0x3FFF // Remove A14 - A15
				portA &= 0x03     // Keep only PA0 - PA1

				return (portA << 14) | address // PA0 - PA1 | A13 - A0
			},
		},
		// ExRAMHi uses PORTA 2 - 4 on the pins A0 - A2
		exRamHi: mapperFunctions[uint16, uint16]{
			MapToSource: func(value uint16, current []uint16) []uint16 {
				portA := (current[0] & 0xE3) | (value << 2) // Replace A0 - A2 with PA2 - PA4
				return []uint16{portA}
			},
			MapFromSource: func(value []uint16) uint16 {
				portA := value[0] & 0x1C // Keep only PA2 - PA4
				return portA >> 2        // Shift PA2 - PA4 to A0 - A2
			},
		},
	}

	// Create the big port A bus which is a 16-bit bus mapped to port A
	// This is only used to be able to interface with the HiRAM and ExRAM buses
	// As currently we can only connect buses with the same size
	bigPortABus := buses.New16BitMappedBus(
		[]buses.Bus[uint8]{portABus},
		mappers.portA.MapToSource,
		mappers.portA.MapFromSource,
	)

	// Create the bus for the HiRAM
	hiRamBus := buses.New16BitMappedBus(
		[]buses.Bus[uint16]{addressBus, bigPortABus},
		mappers.hiRam.MapToSource,
		mappers.hiRam.MapFromSource,
	)

	// Create the bus for the ExRAM (connects to pins 0 to 15 of the chip)
	exRamBus := buses.New16BitMappedBus(
		[]buses.Bus[uint16]{addressBus, bigPortABus},
		mappers.exRam.MapToSource,
		mappers.exRam.MapFromSource,
	)

	// Create the bus for the ExRAMHigh (connects to pins 16 to 18 of the chip)
	exRamBusHigh := buses.New16BitMappedBus(
		[]buses.Bus[uint16]{bigPortABus},
		mappers.exRamHi.MapToSource,
		mappers.exRamHi.MapFromSource,
	)

	// Create the circuit which contains all the buses and lines
	circuit := &circuit{
		addressBus:   addressBus,
		dataBus:      buses.New8BitStandaloneBus(),
		cpuIRQ:       buses.NewStandaloneLine(true),
		cpuReset:     buses.NewStandaloneLine(true),
		cpuRW:        buses.NewStandaloneLine(true),
		hiramBus:     hiRamBus,
		exramBus:     exRamBus,
		exramBusHigh: exRamBusHigh,
		portABus:     portABus,
		bigPortA:     bigPortABus,
		portBBus:     portBBus,
		picoHiRAME:   buses.NewStandaloneLine(false), // TODO: Must default to true
		vcc:          buses.NewStandaloneLine(true),
		ground:       buses.NewStandaloneLine(false),
	}

	// Get references to the specific address bus lines
	// we will use these to connect the CPU and other components
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
		mappers:     mappers,
	}, nil
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

// SpeedUp increases the emulation speed of the computer.
// It uses a non-linear scale for speeds below 0.5 MHz and a linear scale above.
//
// Parameters:
//   - context: The current step context
func (c *ClementinaComputer) SpeedUp(context *common.StepContext) {
	currentSpeed := c.appConfig.TargetSpeedMhz

	if currentSpeed < 0.5 {
		// Non-linear increase below 0.5 MHz
		// Increase by 20% of current speed
		increase := currentSpeed * 0.2
		if increase < 0.000001 {
			// Ensure minimum increase to avoid tiny increments
			increase = 0.000001
		}
		c.appConfig.TargetSpeedMhz += increase
	} else {
		// Linear increase above 0.5 MHz
		c.appConfig.TargetSpeedMhz += 0.1
	}
}

// SpeedDown decreases the emulation speed of the computer.
// It uses a linear scale for speeds above 0.5 MHz and a non-linear scale below,
// ensuring the speed never goes below a minimum threshold.
//
// Parameters:
//   - context: The current step context
func (c *ClementinaComputer) SpeedDown(context *common.StepContext) {
	currentSpeed := c.appConfig.TargetSpeedMhz

	if currentSpeed > 0.5 {
		// Linear reduction above 0.5 MHz
		c.appConfig.TargetSpeedMhz -= 0.1
	} else {
		// Non-linear reduction below 0.5 MHz to avoid reaching 0
		// This will reduce by a fraction of the current speed
		reduction := currentSpeed * 0.2
		if reduction < 0.000001 {
			// Ensure minimum reduction to avoid tiny decrements
			reduction = 0.000001
		}
		c.appConfig.TargetSpeedMhz -= reduction
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
