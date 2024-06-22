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

func evaluateResult(cycle int, cpu *Cpu65C02S, step *TestData, t *testing.T) {
	if cpu.addressBus.Read() != step.addressBus {
		t.Errorf("Cycle %v - Current %04X and expected %04X addresses don't match", cycle, cpu.addressBus.Read(), step.addressBus)
	}

	if cpu.dataBus.Read() != step.dataBus {
		t.Errorf("Cycle %v - Current %02X and expected %02X values on data bus don't match", cycle, cpu.dataBus.Read(), step.dataBus)
	}

	if cpu.readWrite.GetLine().Status() != step.writeEnable {
		t.Errorf("Cycle %v - Current %v and expected %v status R/W line don´t match", cycle, cpu.readWrite.GetLine().Status(), step.writeEnable)
	}

	if cpu.accumulatorRegister != step.accumulatorRegister {
		t.Errorf("Cycle %v - Current %02X and expected %02X values on accumulator don't match", cycle, cpu.accumulatorRegister, step.accumulatorRegister)
	}

	evaluateFlag(cycle, cpu, step, t, "N", NegativeFlagBit)
	evaluateFlag(cycle, cpu, step, t, "Z", ZeroFlagBit)
}

func evaluateFlag(cycle int, cpu *Cpu65C02S, step *TestData, t *testing.T, flagString string, flag StatusBit) {
	ucFlag := strings.ToUpper(flagString)
	lcFlag := strings.ToLower(flagString)

	if strings.Contains(step.flags, ucFlag) {
		if !cpu.processorStatusRegister.Flag(flag) {
			t.Errorf("Cycle %v - Expected %s flag to be set", cycle, ucFlag)
		}
	}

	if strings.Contains(step.flags, lcFlag) {
		if cpu.processorStatusRegister.Flag(flag) {
			t.Errorf("Cycle %v - Expected %s flag to be NOT set", cycle, ucFlag)
		}
	}
}

func TestCpuReadOpCodeCycle(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xAA

	ram.Poke(0xC000, 0x29)
	ram.Poke(0xC001, 0xF0)
	ram.Poke(0xC002, 0x29)
	ram.Poke(0xC003, 0x0F)

	steps := []TestData{
		{0xC000, 0x29, true, 0xAA, ""},
		{0xC001, 0xF0, true, 0xA0, "zN"},
		{0xC002, 0x29, true, 0xA0, ""},
		{0xC003, 0x0F, true, 0x00, "Zn"},
	}

	fmt.Printf("Cycle \t Addr \t Data \t R/W \t PC \t A \t X \t Y \t SP \t Flags \n")
	fmt.Printf("---- \t ---- \t ---- \t ---- \t ---- \t -- \t -- \t -- \t -- \t ----- \n")

	for cycle, step := range steps {
		cpu.Tick(100)
		ram.Tick(100)
		cpu.PostTick(100)

		fmt.Printf("%v \t %04X \t %02X \t %v \t %04X \t %02X \t %02X \t %02X \t %02X \t %08b \n", cycle, cpu.addressBus.Read(), cpu.dataBus.Read(), cpu.readWrite.GetLine().Status(), cpu.programCounter, cpu.accumulatorRegister, cpu.xRegister, cpu.yRegister, cpu.stackPointer, uint8(cpu.processorStatusRegister))

		evaluateResult(cycle, cpu, &step, t)
	}
}
