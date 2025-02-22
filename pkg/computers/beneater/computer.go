package beneater

import (
	"github.com/fran150/clementina6502/pkg/components/acia"
	"github.com/fran150/clementina6502/pkg/components/buses"
	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/fran150/clementina6502/pkg/components/cpu"
	"github.com/fran150/clementina6502/pkg/components/lcd"
	"github.com/fran150/clementina6502/pkg/components/memory"
	"github.com/fran150/clementina6502/pkg/components/other/gates"
	"github.com/fran150/clementina6502/pkg/components/via"
	"github.com/fran150/clementina6502/pkg/terminal"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"go.bug.st/serial"
)

type chips struct {
	cpu  *cpu.Cpu65C02S
	ram  *memory.Ram
	rom  *memory.Ram
	via  *via.Via65C22S
	lcd  *lcd.LcdHD44780U
	acia *acia.Acia65C51N
	nand *gates.Nand74HC00
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
}

func NewBenEaterComputer(portName string) *BenEaterComputer {
	port, err := serial.Open(portName, &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	})
	if err != nil {
		panic(err)
	}

	chips := &chips{
		cpu:  cpu.NewCPU(),
		ram:  memory.NewRamWithLessPins(memory.RAM_SIZE_32K, 0x7FFF),
		rom:  memory.NewRamWithLessPins(memory.RAM_SIZE_32K, 0x7FFF),
		via:  via.NewVia65C22(),
		lcd:  lcd.NewLCDController(),
		acia: acia.NewAcia65C51N(false),
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

	chips.acia.ConnectToPort(circuit.serial)

	return &BenEaterComputer{
		chips:       chips,
		circuit:     circuit,
		mustReset:   false,
		resetCycles: 0,
	}
}

func (c *BenEaterComputer) LoadRom(romImagePath string) {
	err := c.chips.rom.Load(romImagePath)
	if err != nil {
		panic(err)
	}
}

func (c *BenEaterComputer) Init(tvApp *tview.Application, appConfig *terminal.ApplicationConfig) {
	c.appConfig = appConfig
	c.console = newMainConsole(c, tvApp)
}

func (c *BenEaterComputer) Tick(context *common.StepContext) {
	c.chips.cpu.Tick(context)
	c.chips.nand.Tick(context)
	c.chips.ram.Tick(context)
	c.chips.rom.Tick(context)
	c.chips.via.Tick(context)
	c.chips.lcd.Tick(context)
	c.chips.acia.Tick(context)

	if c.console != nil {
		c.console.Tick(context)
	}

	c.chips.cpu.PostTick(context)

	c.checkReset()
}

func (c *BenEaterComputer) Draw(context *common.StepContext) {
	c.console.Draw(context)
}

func (c *BenEaterComputer) KeyPressed(event *tcell.EventKey, context *common.StepContext) *tcell.EventKey {
	if event.Key() == tcell.KeyEsc {
		context.Stop = true
	}

	if event.Rune() == 'r' {
		c.mustReset = true
	}

	if event.Rune() == '=' {
		c.appConfig.TargetSpeedMhz += 0.2
	}

	if event.Rune() == '-' {
		c.appConfig.TargetSpeedMhz -= 0.2
	}

	return event
}

func (c *BenEaterComputer) Close() {
	c.chips.acia.Close()
	c.circuit.serial.Close()
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
