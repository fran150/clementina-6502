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
	"github.com/fran150/clementina6502/pkg/computers"
	"github.com/gdamore/tcell/v2"
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
	executor    *computers.RateExecutor
}

func CreateBenEaterComputer(portName string) *BenEaterComputer {
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
		cpu:  cpu.CreateCPU(),
		ram:  memory.CreateRamWithLessPins(memory.RAM_SIZE_32K, 0x7FFF),
		rom:  memory.CreateRamWithLessPins(memory.RAM_SIZE_32K, 0x7FFF),
		via:  via.CreateVia65C22(),
		lcd:  lcd.CreateLCD(),
		acia: acia.CreateAcia65C51N(false),
		nand: gates.Create74HC00(),
	}

	portABus := buses.Create8BitStandaloneBus()
	portBBus := buses.Create8BitStandaloneBus()
	mapPortBBusToLcdBus := func(value uint8) uint8 {
		value &= 0x0F
		return value << 4
	}
	mapLcdBusToPortBBus := func(value uint8) uint8 {
		value &= 0xF0
		return value >> 4
	}

	circuit := &circuit{
		addressBus: buses.Create16BitStandaloneBus(),
		dataBus:    buses.Create8BitStandaloneBus(),
		cpuIRQ:     buses.CreateStandaloneLine(true),
		cpuReset:   buses.CreateStandaloneLine(true),
		cpuRW:      buses.CreateStandaloneLine(false),
		u4dOut:     buses.CreateStandaloneLine(false),
		u4cOut:     buses.CreateStandaloneLine(false),
		u4bOut:     buses.CreateStandaloneLine(false),
		fiveVolts:  buses.CreateStandaloneLine(true),
		ground:     buses.CreateStandaloneLine(false),
		portABus:   portABus,
		portBBus:   portBBus,
		lcdBus:     buses.Create8BitMappedBus(portBBus, mapLcdBusToPortBBus, mapPortBBusToLcdBus),
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

	computer := &BenEaterComputer{
		chips:       chips,
		circuit:     circuit,
		mustReset:   false,
		resetCycles: 0,
	}

	computer.console = createMainConsole(computer)

	executorHandlers := computers.RateExecutorHandlers{
		Tick: func(context *common.StepContext) {
			chips.cpu.Tick(context)
			chips.nand.Tick(context)
			chips.ram.Tick(context)
			chips.rom.Tick(context)
			chips.via.Tick(context)
			chips.lcd.Tick(context)
			chips.acia.Tick(context)

			computer.console.Tick(context)

			chips.cpu.PostTick(context)

			computer.checkReset()
		},
		UpdateDisplay: func(context *common.StepContext) {
			computer.console.Draw(context)

		},
	}

	executorConfig := computers.RateExecutorConfig{
		TargetSpeedMhz: 1.2,
		DisplayFps:     10,
	}

	computer.executor = computers.CreateExecutor(&executorHandlers, &executorConfig)

	return computer
}

func (c *BenEaterComputer) Load(romImagePath string) {
	err := c.chips.rom.Load(romImagePath)
	if err != nil {
		panic(err)
	}
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

func (c *BenEaterComputer) RunEventLoop() {
	c.console.Run(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			c.console.app.Stop()
		}

		if event.Rune() == 'r' {
			c.mustReset = true
		}

		return event
	})
}

func (e *BenEaterComputer) Run() *common.StepContext {
	context := e.executor.Start()

	e.RunEventLoop()

	e.executor.Stop(context)

	return context
}
