package beneater

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/fran150/clementina-6502/pkg/components/acia"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/fran150/clementina-6502/pkg/components/cpu"
	"github.com/fran150/clementina-6502/pkg/components/lcd"
	"github.com/fran150/clementina-6502/pkg/components/memory"
	"github.com/fran150/clementina-6502/pkg/components/other/gates"
	"github.com/fran150/clementina-6502/pkg/components/via"
	"github.com/fran150/clementina-6502/pkg/terminal"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"go.bug.st/serial"
)

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

type BenEaterComputer struct {
	chips       *chips
	circuit     *circuit
	console     *console
	mustReset   bool
	resetCycles uint8
	appConfig   *terminal.ApplicationConfig

	pause bool
	step  bool
}

func NewBenEaterComputer(port serial.Port, emulateModemLines bool) (*BenEaterComputer, error) {
	chips := &chips{
		cpu:  cpu.NewCpu65C02S(),
		ram:  memory.NewRamWithLessPins(memory.RAM_SIZE_32K, 0x7FFF),
		rom:  memory.NewRamWithLessPins(memory.RAM_SIZE_32K, 0x7FFF),
		via:  via.NewVia65C22(),
		lcd:  lcd.NewLCDController(),
		acia: acia.NewAcia65C51N(emulateModemLines),
		nand: gates.New74HC00(),
	}

	portABus := buses.New8BitStandaloneBus()
	portBBus := buses.New8BitStandaloneBus()
	mapPortBBusToLcdBus := func(value uint8) uint8 {
		value &= 0x0F
		return value << 4
	}
	mapLcdBusToPortBBus := func(value uint8) uint8 {
		value &= 0xF0
		return value >> 4
	}

	circuit := &circuit{
		addressBus: buses.New16BitStandaloneBus(),
		dataBus:    buses.New8BitStandaloneBus(),
		cpuIRQ:     buses.NewStandaloneLine(true),
		cpuReset:   buses.NewStandaloneLine(true),
		cpuRW:      buses.NewStandaloneLine(false),
		u4dOut:     buses.NewStandaloneLine(false),
		u4cOut:     buses.NewStandaloneLine(false),
		u4bOut:     buses.NewStandaloneLine(false),
		fiveVolts:  buses.NewStandaloneLine(true),
		ground:     buses.NewStandaloneLine(false),
		portABus:   portABus,
		portBBus:   portBBus,
		lcdBus:     buses.New8BitMappedBus(portBBus, mapLcdBusToPortBBus, mapPortBBusToLcdBus),
		serial:     port,
	}

	addressBus15 := circuit.addressBus.GetBusLine(15)
	addressBus14 := circuit.addressBus.GetBusLine(14)
	addressBus13 := circuit.addressBus.GetBusLine(13)
	addressBus12 := circuit.addressBus.GetBusLine(12)
	addressBus3 := circuit.addressBus.GetBusLine(3)
	addressBus2 := circuit.addressBus.GetBusLine(2)
	addressBus1 := circuit.addressBus.GetBusLine(1)
	addressBus0 := circuit.addressBus.GetBusLine(0)

	chips.cpu.AddressBus().Connect(circuit.addressBus)
	chips.cpu.DataBus().Connect(circuit.dataBus)
	chips.cpu.Ready().Connect(circuit.fiveVolts)
	chips.cpu.InterruptRequest().Connect(circuit.cpuIRQ)
	chips.cpu.NonMaskableInterrupt().Connect(circuit.fiveVolts)
	chips.cpu.Reset().Connect(circuit.cpuReset)
	chips.cpu.BusEnable().Connect(circuit.fiveVolts)
	chips.cpu.ReadWrite().Connect(circuit.cpuRW)

	chips.rom.AddressBus().Connect(circuit.addressBus)
	chips.rom.DataBus().Connect(circuit.dataBus)
	chips.rom.WriteEnable().Connect(circuit.fiveVolts)
	chips.rom.OutputEnable().Connect(circuit.ground)
	chips.rom.ChipSelect().Connect(circuit.u4dOut)

	chips.ram.AddressBus().Connect(circuit.addressBus)
	chips.ram.DataBus().Connect(circuit.dataBus)
	chips.ram.WriteEnable().Connect(circuit.cpuRW)
	chips.ram.OutputEnable().Connect(addressBus14)
	chips.ram.ChipSelect().Connect(circuit.u4cOut)

	chips.via.DataBus().Connect(circuit.dataBus)
	chips.via.IrqRequest().Connect(circuit.cpuIRQ)
	chips.via.ReadWrite().Connect(circuit.cpuRW)
	chips.via.ChipSelect2().Connect(circuit.u4bOut)
	chips.via.ChipSelect1().Connect(addressBus13)
	chips.via.Reset().Connect(circuit.cpuReset)
	chips.via.RegisterSelect(3).Connect(addressBus3)
	chips.via.RegisterSelect(2).Connect(addressBus2)
	chips.via.RegisterSelect(1).Connect(addressBus1)
	chips.via.RegisterSelect(0).Connect(addressBus0)
	chips.via.PeripheralPortB().Connect(circuit.portBBus)

	viaPBAddress6 := circuit.portBBus.GetBusLine(6)
	viaPBAddress5 := circuit.portBBus.GetBusLine(5)
	viaPBAddress4 := circuit.portBBus.GetBusLine(4)

	chips.lcd.Enable().Connect(viaPBAddress6)
	chips.lcd.ReadWrite().Connect(viaPBAddress5)
	chips.lcd.RegisterSelect().Connect(viaPBAddress4)
	chips.lcd.DataBus().Connect(circuit.lcdBus)

	chips.acia.DataBus().Connect(circuit.dataBus)
	chips.acia.IrqRequest().Connect(circuit.cpuIRQ)
	chips.acia.ReadWrite().Connect(circuit.cpuRW)
	chips.acia.RegisterSelect(0).Connect(addressBus0)
	chips.acia.RegisterSelect(1).Connect(addressBus1)
	chips.acia.ChipSelect1().Connect(circuit.u4bOut)
	chips.acia.ChipSelect0().Connect(addressBus12)

	chips.nand.APin(0).Connect(addressBus15)
	chips.nand.BPin(0).Connect(addressBus15)
	chips.nand.YPin(0).Connect(circuit.u4dOut)

	// In reality goes to PHI2
	chips.nand.APin(1).Connect(circuit.fiveVolts)
	chips.nand.BPin(1).Connect(circuit.u4dOut)
	chips.nand.YPin(1).Connect(circuit.u4cOut)

	chips.nand.APin(2).Connect(addressBus14)
	chips.nand.BPin(2).Connect(circuit.u4dOut)
	chips.nand.YPin(2).Connect(circuit.u4bOut)

	if circuit.serial != nil {
		if err := chips.acia.ConnectToPort(circuit.serial); err != nil {
			return nil, err
		}
	}

	return &BenEaterComputer{
		chips:       chips,
		circuit:     circuit,
		mustReset:   false,
		resetCycles: 0,
	}, nil
}

func (c *BenEaterComputer) LoadRom(romImagePath string) error {
	err := c.chips.rom.Load(romImagePath)
	if err != nil {
		return err
	}

	return nil
}

func (c *BenEaterComputer) Init(tvApp *tview.Application, appConfig *terminal.ApplicationConfig) {
	tvApp.EnableMouse(true).EnablePaste(true)

	c.appConfig = appConfig
	c.console = newMainConsole(c, tvApp)
}

func (c *BenEaterComputer) Tick(context *common.StepContext) {
	if !c.pause || c.step {
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

func (c *BenEaterComputer) Pause(context *common.StepContext) {
	c.pause = true
}

func (c *BenEaterComputer) Resume(context *common.StepContext) {
	c.pause = false
	c.step = true
}

func (c *BenEaterComputer) Step(context *common.StepContext) {
	c.step = true
}

func (c *BenEaterComputer) Draw(context *common.StepContext) {
	c.console.Draw(context)
}

func (c *BenEaterComputer) Stop(context *common.StepContext) {
	context.Stop = true
}

func (c *BenEaterComputer) Reset(context *common.StepContext) {
	c.mustReset = true
}

func (c *BenEaterComputer) SpeedUp(context *common.StepContext) {
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

func (c *BenEaterComputer) SpeedDown(context *common.StepContext) {
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

	// Ensure we never go below a minimum threshold (e.g., 0.000001 MHz)
	if c.appConfig.TargetSpeedMhz < 0.000001 {
		c.appConfig.TargetSpeedMhz = 0.000001
	}
}

func (c *BenEaterComputer) KeyPressed(event *tcell.EventKey, context *common.StepContext) *tcell.EventKey {
	if options := GetWindow[ui.OptionsWindow](c.console, "options"); options != nil {
		return options.ProcessKey(event, context)
	}

	return event
}

func (c *BenEaterComputer) Close() {
	c.chips.acia.Close()
}

func (c *BenEaterComputer) getPotentialOperators(programCounter uint16) [2]uint8 {
	rom := c.chips.rom
	programCounter &= 0x7FFF
	operand1Address := programCounter & 0x7FFF
	operand2Address := (programCounter + 1) & 0x7FFF
	return [2]uint8{rom.Peek(operand1Address), rom.Peek(operand2Address)}
}

func (c *BenEaterComputer) checkReset() {
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
