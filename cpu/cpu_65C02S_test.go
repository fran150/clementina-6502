package cpu

import (
	"fmt"
	"strings"
	"testing"

	"github.com/fran150/clementina6502/buses"
	"github.com/fran150/clementina6502/memory"
)

func createComputer() (*Cpu65C02S, *memory.Ram) {
	addressBus := buses.CreateBus[uint16]()
	dataBus := buses.CreateBus[uint8]()

	alwaysHighLine := buses.CreateStandaloneLine(true)
	alwaysLowLine := buses.CreateStandaloneLine(false)

	writeEnableLine := buses.CreateStandaloneLine(true)

	ram := memory.CreateRam()
	ram.Connect(addressBus, dataBus, writeEnableLine, alwaysLowLine, alwaysLowLine)

	cpu := CreateCPU()
	cpu.ConnectAddressBus(addressBus)
	cpu.ConnectDataBus(dataBus)

	cpu.BusEnable().Connect(alwaysHighLine)
	cpu.ReadWrite().Connect(writeEnableLine)

	cpu.programCounter = 0xC000

	return cpu, ram
}

type TestData struct {
	addressBus          uint16
	dataBus             uint8
	writeEnable         bool
	accumulatorRegister uint8
	flags               string
}

func TestCpuReadOpCodeCycle(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xAA

	ram.Poke(0xC000, 0x29)
	ram.Poke(0xC001, 0x0F)
	ram.Poke(0xC002, 0xEA)

	steps := []TestData{
		{0xC000, 0x29, true, 0xAA, ""},
		{0xC001, 0x0F, true, 0x0A, "zn"},
		{0xC002, 0xEA, true, 0x0A, ""},
	}

	for _, step := range steps {
		cpu.Tick(100)
		ram.Tick(100)
		cpu.PostTick(100)

		fmt.Printf("%X \t %X \t %v \t %X \n", cpu.addressBus.Read(), cpu.dataBus.Read(), cpu.readWrite.GetLine().Status(), cpu.accumulatorRegister)

		if cpu.addressBus.Read() != step.addressBus {
			t.Errorf("Current %x and expected %x addresses don't match", cpu.addressBus.Read(), step.addressBus)
		}

		if cpu.dataBus.Read() != step.dataBus {
			t.Errorf("Current %x and expected %x values on data bus don't match", cpu.dataBus.Read(), step.dataBus)
		}

		if cpu.readWrite.GetLine().Status() != step.writeEnable {
			t.Errorf("Current %v and expected %v values on data bus don't match", cpu.readWrite.GetLine().Status(), step.writeEnable)
		}

		if cpu.accumulatorRegister != step.accumulatorRegister {
			t.Errorf("Current %x and expected %x values on accumulator don't match", cpu.addressBus.Read(), step.addressBus)
		}

		if strings.Contains(step.flags, "Z") {
			if !cpu.processorStatusRegister.Flag(ZeroFlagBit) {
				t.Error("Expected 0 flag to be set")
			}
		}

		if strings.Contains(step.flags, "z") {
			if cpu.processorStatusRegister.Flag(ZeroFlagBit) {
				t.Error("Expected Zero flag to be NOT set")
			}
		}

		if strings.Contains(step.flags, "N") {
			if !cpu.processorStatusRegister.Flag(NegativeFlagBit) {
				t.Error("Expected 0 flag to be set")
			}
		}

		if strings.Contains(step.flags, "n") {
			if cpu.processorStatusRegister.Flag(NegativeFlagBit) {
				t.Error("Expected Zero flag to be NOT set")
			}
		}
	}
}
