package beneater

import (
	"fmt"
	"strings"
	"time"

	"github.com/fran150/clementina6502/pkg/components/acia"
	"github.com/fran150/clementina6502/pkg/components/buses"
	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/fran150/clementina6502/pkg/components/cpu"
	"github.com/fran150/clementina6502/pkg/components/lcd"
	"github.com/fran150/clementina6502/pkg/components/memory"
	"github.com/fran150/clementina6502/pkg/components/other/gates"
	"github.com/fran150/clementina6502/pkg/components/via"
	"github.com/rivo/tview"
	"go.bug.st/serial"
)

var (
	app   *tview.Application
	grid  *tview.Grid
	code  *tview.TextView
	other *tview.TextView
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

type Computer struct {
	chips   *chips
	circuit *circuit
}

func CreateBenEaterComputer(portName string) *Computer {
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

	return &Computer{
		chips:   chips,
		circuit: circuit,
	}
}

func (c *Computer) Load(romImagePath string) {
	err := c.chips.rom.Load(romImagePath)
	if err != nil {
		panic(err)
	}
}

func (c *Computer) UnReset() {
	c.circuit.cpuReset.Set(true)
}

func (c *Computer) Reset() {
	c.circuit.cpuReset.Set(false)
}

func (c *Computer) Run(context *common.StepContext) {
	//var pc uint16

	var nano int64

	for {
		if nano > 900 {
			context.Next()

			//pc = c.chips.cpu.GetProgramCounter()

			c.chips.cpu.Tick(*context)
			c.chips.nand.Tick(*context)
			c.chips.ram.Tick(*context)
			c.chips.rom.Tick(*context)
			c.chips.via.Tick(*context)
			c.chips.lcd.Tick(*context)
			c.chips.acia.Tick(*context)

			c.chips.cpu.PostTick(*context)

			if c.chips.cpu.IsReadingOpcode() && c.chips.cpu.GetCurrentInstruction() != nil {

				//line := DrawInstruction(c.chips.cpu, c.chips.rom, pc, c.chips.cpu.GetCurrentInstruction())
				//fmt.Fprint(code, line)

				other.Clear()

				//c.DrawCPUState()
				//c.DrawViaRegisters()
				c.DrawLcd()

				app.Draw()
			}

			// Todo this is probably not needed
			if context.Cycle > 7 && context.Cycle < 10 {
				c.circuit.cpuReset.Set(false)
			} else {
				c.circuit.cpuReset.Set(true)
			}
		}

		nano = time.Since(context.T).Nanoseconds()
	}
}

func (c *Computer) DrawCPUState() {
	fmt.Fprintf(other, "[yellow] A: [white]%5d [grey]($%02X)\n", c.chips.cpu.GetAccumulatorRegister(), c.chips.cpu.GetAccumulatorRegister())
	fmt.Fprintf(other, "[yellow] X: [white]%5d [grey]($%02X)\n", c.chips.cpu.GetXRegister(), c.chips.cpu.GetXRegister())
	fmt.Fprintf(other, "[yellow] Y: [white]%5d [grey]($%02X)\n", c.chips.cpu.GetYRegister(), c.chips.cpu.GetYRegister())
	fmt.Fprintf(other, "[yellow]SP: [white]%5d [grey]($%02X)\n", c.chips.cpu.GetStackPointer(), c.chips.cpu.GetStackPointer())
	fmt.Fprintf(other, "[yellow]PC: [white]$%04X [grey](%v)\n", c.chips.cpu.GetProgramCounter(), c.chips.cpu.GetProgramCounter())

	fmt.Fprint(other, "[yellow]Flags: ")
	fmt.Fprintf(other, "%sN", c.getFlagStatusColor(cpu.NegativeFlagBit))
	fmt.Fprintf(other, "%sV", c.getFlagStatusColor(cpu.OverflowFlagBit))
	fmt.Fprintf(other, "%s-", c.getFlagStatusColor(cpu.UnusedFlagBit))
	fmt.Fprintf(other, "%sB", c.getFlagStatusColor(cpu.BreakCommandFlagBit))
	fmt.Fprintf(other, "%sD", c.getFlagStatusColor(cpu.DecimalModeFlagBit))
	fmt.Fprintf(other, "%sI", c.getFlagStatusColor(cpu.IrqDisableFlagBit))
	fmt.Fprintf(other, "%sZ", c.getFlagStatusColor(cpu.ZeroFlagBit))
	fmt.Fprintf(other, "%sC", c.getFlagStatusColor(cpu.CarryFlagBit))
	fmt.Fprint(other, "\n")
}

func (c *Computer) getFlagStatusColor(bit cpu.StatusBit) string {
	status := c.chips.cpu.GetProcessorStatusRegister()

	if status.Flag(bit) {
		return "[green]"
	}

	return "[red]"
}

func (c *Computer) DrawChipSelected() {
	value := "[grey]None"

	if c.chips.ram.ChipSelect().Enabled() {
		value = "[cyan]RAM"
	}

	if c.chips.rom.ChipSelect().Enabled() {
		value = "[blue]ROM"
	}

	if c.chips.via.ChipSelect1().Enabled() && c.chips.via.ChipSelect2().Enabled() {
		value = "[purple]VIA"
	}

	if c.chips.acia.ChipSelect1().Enabled() && c.chips.acia.ChipSelect0().Enabled() {
		value = "[orange]ACIA"
	}

	if c.chips.lcd.Enable().Enabled() {
		value = "[green]LCD"
	}

	fmt.Fprintf(other, "[white]Chip Selected: %s\n", value)
}

func (c *Computer) Close() {
	c.chips.acia.Close()
}

func (c *Computer) DrawViaRegisters() {
	fmt.Fprintf(other, "[yellow]VIA Registers:\n")
	fmt.Fprintf(other, "[yellow] ORA:  [white]$%02X\n", c.chips.via.GetOutputRegisterA())
	fmt.Fprintf(other, "[yellow] ORB:  [white]$%02X\n", c.chips.via.GetOutputRegisterB())
	fmt.Fprintf(other, "[yellow] IRA:  [white]$%02X\n", c.chips.via.GetInputRegisterA())
	fmt.Fprintf(other, "[yellow] IRB:  [white]$%02X\n", c.chips.via.GetInputRegisterB())
	fmt.Fprintf(other, "[yellow] DDRA: [white]$%02X\n", c.chips.via.GetDataDirectionRegisterA())
	fmt.Fprintf(other, "[yellow] DDRB: [white]$%02X\n", c.chips.via.GetDataDirectionRegisterB())
	fmt.Fprintf(other, "[yellow] LL1:  [white]$%02X\n", c.chips.via.GetLowLatches1())
	fmt.Fprintf(other, "[yellow] HL1:  [white]$%02X\n", c.chips.via.GetHighLatches1())
	fmt.Fprintf(other, "[yellow] CTR1: [white]$%04X\n", c.chips.via.GetCounter1())
	fmt.Fprintf(other, "[yellow] LL2:  [white]$%02X\n", c.chips.via.GetLowLatches2())
	fmt.Fprintf(other, "[yellow] HL2:  [white]$%02X\n", c.chips.via.GetHighLatches2())
	fmt.Fprintf(other, "[yellow] CTR2: [white]$%04X\n", c.chips.via.GetCounter2())
	fmt.Fprintf(other, "[yellow] SR:   [white]$%02X\n", c.chips.via.GetShiftRegister())
	fmt.Fprintf(other, "[yellow] ACR:  [white]$%02X\n", c.chips.via.GetAuxiliaryControl())
	fmt.Fprintf(other, "[yellow] PCR:  [white]$%02X\n", c.chips.via.GetPeripheralControl())
	fmt.Fprintf(other, "[yellow] IFR:  [white]$%02X\n", c.chips.via.GetInterruptFlagValue())
	fmt.Fprintf(other, "[yellow] IER:  [white]$%02X\n", c.chips.via.GetInterruptEnabledFlag())

	fmt.Fprintf(other, "[yellow] PortA: [white]$%04X\n", c.circuit.portABus.Read())
	fmt.Fprintf(other, "[yellow] PortB: [white]$%04X\n", c.circuit.portBBus.Read())
}

func (c *Computer) DrawLcd() {
	cursorStatus := c.chips.lcd.GetCursorStatus()
	displayStatus := c.chips.lcd.GetDisplayStatus()

	fmt.Fprintf(other, "LCD Memory:\n")

	for i, data := range displayStatus.DDRAM {
		fmt.Fprintf(other, "%02v: %s ", i, string(data))

		if i%10 == 9 {
			fmt.Fprintf(other, "\n")
		}
	}

	fmt.Fprintf(other, "LCD Screen: \n")

	count := 0
	index := displayStatus.Line1Start

	for count < 16 {
		index %= 40
		fmt.Fprint(other, string(displayStatus.DDRAM[index]))
		index++
		count++
	}

	fmt.Fprint(other, "\n")

	count = 0
	index = displayStatus.Line2Start
	for count < 16 {
		index %= 40
		fmt.Fprint(other, string(displayStatus.DDRAM[index+40]))
		index++
		count++
	}

	fmt.Fprint(other, "\n\n")

	fmt.Fprintf(other, "Display ON: %v\n", displayStatus.DisplayOn)
	fmt.Fprintf(other, "8 Bit Mode: %v\n", displayStatus.Is8BitMode)
	fmt.Fprintf(other, "Line 2 display: %v\n", displayStatus.Is2LineDisplay)
	fmt.Fprintf(other, "Cursor Position: %v\n", cursorStatus.CursorPosition)
	fmt.Fprintf(other, "Bus: %v\n", c.circuit.lcdBus.Read())
	fmt.Fprintf(other, "E: %v\n", c.chips.lcd.Enable().Enabled())
	fmt.Fprintf(other, "RW: %v\n", c.chips.lcd.ReadWrite().Enabled())
	fmt.Fprintf(other, "RS: %v\n", c.chips.lcd.RegisterSelect().Enabled())
	fmt.Fprintf(other, "PB: %08b\n", c.circuit.portBBus.Read())

}

func DrawInstruction(processor *cpu.Cpu65C02S, ram *memory.Ram, address uint16, instruction *cpu.CpuInstructionData) string {
	addressModeDetails := cpu.GetAddressMode(instruction.AddressMode())

	size := addressModeDetails.MemSize() - 1

	sb := strings.Builder{}

	// Write current address
	sb.WriteString(fmt.Sprintf("[blue]$%04X: [red]%s [white]", address, instruction.Mnemonic()))

	address = address & 0x7FFF

	// Write operands
	switch size {
	case 0:
	case 1:
		sb.WriteString(fmt.Sprintf(addressModeDetails.Format(), ram.Peek(address+1)))
	case 2:
		msb := uint16(ram.Peek(address+2)) << 8
		lsb := uint16(ram.Peek(address + 1))
		sb.WriteString(fmt.Sprintf(addressModeDetails.Format(), msb|lsb))
	default:
		sb.WriteString("Unrecognized Instruction or Address Mode")
	}

	sb.WriteString("\r\n")

	return sb.String()
}

func (c *Computer) RunUI() {
	app = tview.NewApplication()

	newPrimitive := func(text string) *tview.TextView {
		return tview.NewTextView().
			SetTextAlign(tview.AlignLeft).
			SetScrollable(false).
			SetDynamicColors(true).
			SetText(text)
	}

	code = newPrimitive("Main content")
	other = newPrimitive("Side Bar")

	grid = tview.NewGrid().
		SetRows(0).
		SetColumns(25, 0).
		SetBorders(true)

	// Layout for screens narrower than 100 cells (menu and side bar are hidden).
	grid.AddItem(code, 0, 0, 1, 1, 0, 0, false).
		AddItem(other, 0, 1, 1, 1, 0, 0, false)

	context := common.CreateStepContext()
	t := time.Now()

	defer func() {
		elapsed := time.Since(t)
		total := (float64(context.Cycle) / elapsed.Seconds()) / 1_000_000

		fmt.Printf("Executed %v cycles in %v seconds\n", context.Cycle, elapsed)
		fmt.Printf("Computer ran at %v mhz\n", total)
	}()

	go c.Run(&context)

	if err := app.SetRoot(grid, true).Run(); err != nil {
		panic(err)
	}
}
