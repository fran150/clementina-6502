package clementina

import (
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/fran150/clementina-6502/pkg/components/cpu"
	"github.com/fran150/clementina-6502/pkg/components/memory"
	"github.com/fran150/clementina-6502/pkg/components/mia"
	"github.com/fran150/clementina-6502/pkg/components/via"
	"github.com/fran150/clementina-6502/pkg/computers/clementina/modules"
)

func NewClementinaComputer() (*ClementinaComputer, error) {
	return newClementinaComputer(mia.NewEmulatedMia())
}

func NewClementinaGPIOComputer(chipName string) (*ClementinaComputer, error) {
	chip, err := mia.NewPicoMia(chipName)
	if err != nil {
		return nil, err
	}

	return newClementinaComputer(chip)
}

func newClementinaComputer(mia components.MiaChip) (*ClementinaComputer, error) {
	chips := &chips{
		cpu:      cpu.NewCpu65C02S(),
		baseram:  memory.NewRam(memory.RAM_SIZE_32K),
		exram:    memory.NewRam(memory.RAM_SIZE_512K),
		via:      via.NewVia65C22S(),
		mia:      mia,
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

		// Bus for MIA uses only 5 pins from address bus
		mia: mapperFunctions[uint8, uint16]{
			MapToSource: func(value uint8, current []uint16) []uint16 {
				return []uint16{uint16(value) & 0x1F}
			},
			MapFromSource: func(value []uint16) uint8 {
				return uint8(value[0])
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
	miaBus := buses.New8BitMappedBus(
		[]buses.Bus[uint16]{addressBus, bigPortABus},
		mappers.mia.MapToSource,
		mappers.mia.MapFromSource,
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
		miaBus:       miaBus,
		exramBus:     exRamBus,
		exramBusHigh: exRamBusHigh,
		portABus:     portABus,
		bigPortA:     bigPortABus,
		portBBus:     portBBus,
		miaCS:        buses.NewStandaloneLine(true),
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

	// Connect the CPU to the OE/RW sync module
	chips.oeRWSync.CpuRW().Connect(circuit.cpuRW)

	// Base RAM connections
	chips.baseram.AddressBus().Connect(circuit.addressBus)
	chips.baseram.DataBus().Connect(circuit.dataBus)
	chips.baseram.WriteEnable().Connect(chips.oeRWSync.RW())
	chips.baseram.OutputEnable().Connect(chips.oeRWSync.OE())
	chips.baseram.ChipSelect().Connect(addressBus15)

	// MIA connections
	chips.mia.AddressBus().Connect(circuit.miaBus)
	chips.mia.DataBus().Connect(circuit.dataBus)
	chips.mia.MiaCS().Connect(chips.csLogic.MiaCS())
	chips.mia.Reset().Connect(circuit.cpuReset)
	chips.mia.WriteEnable().Connect(chips.oeRWSync.RW())
	chips.mia.Irq().Connect(circuit.cpuIRQ)

	// VIA connections
	chips.via.DataBus().Connect(circuit.dataBus)
	chips.via.IrqRequest().Connect(circuit.cpuIRQ)
	chips.via.ReadWrite().Connect(circuit.cpuRW)
	chips.via.ChipSelect2().Connect(chips.csLogic.IOCS().GetBusLine(0))
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
	chips.exram.ChipSelect().Connect(chips.csLogic.ExRAMCS())

	computer := &ClementinaComputer{
		chips:   chips,
		circuit: circuit,
		mappers: mappers,
	}

	return computer, nil
}
