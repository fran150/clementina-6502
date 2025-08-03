package beneater

import (
	"fmt"
	"os"

	"github.com/fran150/clementina-6502/pkg/components/acia"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/fran150/clementina-6502/pkg/components/cpu"
	"github.com/fran150/clementina-6502/pkg/components/lcd"
	"github.com/fran150/clementina-6502/pkg/components/memory"
	"github.com/fran150/clementina-6502/pkg/components/other/gates"
	"github.com/fran150/clementina-6502/pkg/components/via"
	"github.com/fran150/clementina-6502/pkg/computers"
	"github.com/rivo/tview"
)

// NewBenEaterComputer creates and initializes a new instance of the Ben Eater 6502 computer emulation.
// It sets up all hardware components, connects them according to the original design, and configures
// the serial port for communication.
//
// Parameters:
//   - config: Configuration containing emulation settings, serial port, and modem line options
//
// Returns:
//   - A pointer to the initialized BenEaterComputer
//   - An error if initialization fails
func NewBenEaterComputer(config *BenEaterComputerConfig) (*BenEaterComputer, error) {
	chips := &chips{
		cpu:  cpu.NewCpu65C02S(),
		ram:  memory.NewRam(memory.RAM_SIZE_32K),
		rom:  memory.NewRam(memory.RAM_SIZE_32K),
		via:  via.NewVia65C22(),
		lcd:  lcd.NewLCDController(),
		acia: acia.NewAcia65C51N(config.EmulateModemLines),
		nand: gates.New74HC00(),
	}

	portABus := buses.New8BitStandaloneBus()
	portBBus := buses.New8BitStandaloneBus()
	mapPortBBusToLcdBus := func(value []uint8) uint8 {
		value[0] &= 0x0F
		return value[0] << 4
	}
	mapLcdBusToPortBBus := func(value uint8, current []uint8) []uint8 {
		value &= 0xF0
		return []uint8{value >> 4}
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
		lcdBus:     buses.New8BitMappedBus([]buses.Bus[uint8]{portBBus}, mapLcdBusToPortBBus, mapPortBBusToLcdBus),
		serial:     config.Port,
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

	computer := &BenEaterComputer{
		chips:       chips,
		circuit:     circuit,
		resetCycles: 0,
	}

	computer.BaseComputer = *computers.NewBaseComputer(
		computers.NewEmulationLoopFor(computer, &config.EmulationLoopConfig),
		tview.NewApplication().
			EnableMouse(true).
			EnablePaste(true).
			SetInputCapture(computer.KeyPressed),
	)

	computer.console = newMainConsole(computer)

	computer.Loop().SetPanicHandler(func(loopType string, panicData any) bool {
		fmt.Fprintf(os.Stderr, "%s panic: %v\n", loopType, panicData)
		computer.Stop()
		return false
	})

	return computer, nil
}
