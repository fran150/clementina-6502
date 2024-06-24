package cpu

import (
	"fmt"
	"strings"
	"testing"

	"github.com/fran150/clementina6502/buses"
	"github.com/fran150/clementina6502/memory"
)

// These are the values to validate after each cycle. The tests will check if
// after each cycle the values on the CPU matches the ones in this struct
// Uppercase flag records means that the flag needs to be checked for set, lower case
// will check if the flag is unset.
// Typical characters for the flags are used NV-BDIZC.
// For example "nvZ" will check that flag negative and overflow are NOT set and the
// zero flag is set
type testData struct {
	addressBus          uint16
	dataBus             uint8
	writeEnable         bool
	accumulatorRegister uint8
	xRegister           uint8
	yRegister           uint8
	flags               string
	programCounter      uint16
}

// Creates a computer for testing the CPU emulation.
// It is only 64K of RAM memory connected to the bus, processor lines are wired
// to always high or low lines.
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

// Evaluates the current status of the CPU on a given cycle and
// compares it with the expceted value from the test data.
func evaluateCycle(cycle int, cpu *Cpu65C02S, step *testData, t *testing.T) {
	evaluateRegister(cycle, cpu.addressBus.Read(), step.addressBus, t, "address bus")
	evaluateRegister(cycle, cpu.dataBus.Read(), step.dataBus, t, "data bus")

	evaluateLine(cycle, cpu.readWrite.GetLine().Status(), step.writeEnable, t, "R/W")

	evaluateRegister(cycle, cpu.accumulatorRegister, step.accumulatorRegister, t, "A")
	evaluateRegister(cycle, cpu.xRegister, step.xRegister, t, "X")
	evaluateRegister(cycle, cpu.xRegister, step.xRegister, t, "Y")

	evaluateFlag(cycle, cpu, step, t, "N", NegativeFlagBit)
	evaluateFlag(cycle, cpu, step, t, "Z", ZeroFlagBit)

	evaluateRegister(cycle, cpu.programCounter, step.programCounter, t, "program counter")
}

// Evaluates the value of the specified CPU line
func evaluateLine(cycle int, status bool, stepStatus bool, t *testing.T, lineName string) {
	if status != stepStatus {
		t.Errorf("Cycle %v - Current %v and expected %v status %s line don´t match", cycle, status, stepStatus, lineName)
	}
}

// Evaluates the value of the sepcified register values
func evaluateRegister[U uint8 | uint16](cycle int, registerValue U, stepValue U, t *testing.T, registerName string) {
	var text string

	if registerValue != stepValue {
		switch any(registerValue).(type) {
		case uint8:
			text = "Cycle %v - Current %02X and expected %02X values for %s don't match"
		case uint16:
			text = "Cycle %v - Current %04X and expected %04X values for %s don't match"
		}

		t.Errorf(text, cycle, registerValue, stepValue, registerName)
	}
}

// Evaluates if the specified flag is set or not.
// The flag string specifies the value that it will be searched in the testData.flags
// field. If this letter is found then if it's uppercase it will validate if the flag is set
// if it's lowercase it will validate if it's unset.
func evaluateFlag(cycle int, cpu *Cpu65C02S, step *testData, t *testing.T, flagString string, flag StatusBit) {
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

func runTest(cpu *Cpu65C02S, ram *memory.Ram, steps []testData, t *testing.T) {
	fmt.Printf("Cycle \t Addr \t Data \t R/W \t PC \t A \t X \t Y \t SP \t Flags \n")
	fmt.Printf("---- \t ---- \t ---- \t ---- \t ---- \t -- \t -- \t -- \t -- \t ----- \n")

	for cycle, step := range steps {
		cpu.Tick(100)
		ram.Tick(100)
		cpu.PostTick(100)

		fmt.Printf("%v \t %04X \t %02X \t %v \t %04X \t %02X \t %02X \t %02X \t %02X \t %08b \n", cycle, cpu.addressBus.Read(), cpu.dataBus.Read(), cpu.readWrite.GetLine().Status(), cpu.programCounter, cpu.accumulatorRegister, cpu.xRegister, cpu.yRegister, cpu.stackPointer, uint8(cpu.processorStatusRegister))

		evaluateCycle(cycle, cpu, &step, t)
	}

}

/**************************************************************************
* Address modes tests
* -------------------
* This section will tests if the address modes have the right output on
* each cycle. For reference on the behaviour of each cycle see:
* https://www.atarihq.com/danb/files/64doc.txt
**************************************************************************/

/*****************************************
* Implicit / Accumulator modes / Immediate
******************************************/

func TestImplicitAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0x18) // CLC
	ram.Poke(0xC001, 0xea) // NOP

	steps := []testData{
		{0xC000, 0x18, true, 0x00, 0x00, 0x00, "", 0xC001},
		{0xC001, 0xEA, true, 0x00, 0x00, 0x00, "", 0xC001},
	}

	runTest(cpu, ram, steps, t)
}

func TestAccumulatorAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0x2A) // ROL a
	ram.Poke(0xC001, 0xEA) // NOP

	steps := []testData{
		{0xC000, 0x2A, true, 0x00, 0x00, 0x00, "", 0xC001},
		{0xC001, 0xEA, true, 0x00, 0x00, 0x00, "", 0xC001},
	}

	runTest(cpu, ram, steps, t)
}

func TestImmediateAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0xA9) // LDA #FF
	ram.Poke(0xC001, 0xFF)
	ram.Poke(0xC002, 0xEA) // NOP

	steps := []testData{
		{0xC000, 0xA9, true, 0x00, 0x00, 0x00, "", 0xC001},
		{0xC001, 0xFF, true, 0xFF, 0x00, 0x00, "", 0xC002},
		{0xC002, 0xEA, true, 0xFF, 0x00, 0x00, "", 0xC003},
	}

	runTest(cpu, ram, steps, t)
}

/***************
* Absolute modes
****************/

func TestAbsoluteJumpAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0x4C)
	ram.Poke(0xC001, 0x00)
	ram.Poke(0xC002, 0xD0)
	ram.Poke(0xD000, 0xEA)

	steps := []testData{
		{0xC000, 0x4C, true, 0x00, 0x00, 0x00, "", 0xC001},
		{0xC001, 0x00, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0xC002, 0xD0, true, 0x00, 0x00, 0x00, "", 0xD000},
		{0xD000, 0xEA, true, 0x00, 0x00, 0x00, "", 0xD001},
	}

	runTest(cpu, ram, steps, t)
}

func TestAbsoluteAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0xAD) // LDA $D000
	ram.Poke(0xC001, 0x00)
	ram.Poke(0xC002, 0xD0)
	ram.Poke(0xC003, 0xEA) // NOP
	ram.Poke(0xD000, 0xFA) // This value should to to A

	steps := []testData{
		{0xC000, 0xAD, true, 0x00, 0x00, 0x00, "", 0xC001},
		{0xC001, 0x00, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0xC002, 0xD0, true, 0x00, 0x00, 0x00, "", 0xC003},
		{0xD000, 0xFA, true, 0xFA, 0x00, 0x00, "", 0xC003},
		{0xC003, 0xEA, true, 0xFA, 0x00, 0x00, "", 0xC004},
	}

	runTest(cpu, ram, steps, t)
}

func TestAbsoluteAddressRMWMode(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0xEE) // INC $D000
	ram.Poke(0xC001, 0x00)
	ram.Poke(0xC002, 0xD0)
	ram.Poke(0xC003, 0xEA) // NOP
	ram.Poke(0xD000, 0xFA) // This value should be added 1

	steps := []testData{
		{0xC000, 0xEE, true, 0x00, 0x00, 0x00, "", 0xC001},
		{0xC001, 0x00, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0xC002, 0xD0, true, 0x00, 0x00, 0x00, "", 0xC003},
		{0xD000, 0xFA, true, 0x00, 0x00, 0x00, "", 0xC003},
		{0xD000, 0xFA, true, 0x00, 0x00, 0x00, "", 0xC003},
		{0xD000, 0xFB, false, 0x00, 0x00, 0x00, "", 0xC003},
		{0xC003, 0xEA, true, 0x00, 0x00, 0x00, "", 0xC004},
	}

	runTest(cpu, ram, steps, t)
}

func TestWriteAbsoluteAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xFF

	ram.Poke(0xC000, 0x8D) // STA $D000
	ram.Poke(0xC001, 0x00)
	ram.Poke(0xC002, 0xD0)
	ram.Poke(0xC003, 0xEA) // NOP

	steps := []testData{
		{0xC000, 0x8D, true, 0xFF, 0x00, 0x00, "", 0xC001},
		{0xC001, 0x00, true, 0xFF, 0x00, 0x00, "", 0xC002},
		{0xC002, 0xD0, true, 0xFF, 0x00, 0x00, "", 0xC003},
		{0xD000, 0xFF, false, 0xFF, 0x00, 0x00, "", 0xC003},
		{0xC003, 0xEA, true, 0xFF, 0x00, 0x00, "", 0xC004},
	}

	runTest(cpu, ram, steps, t)
}

/*************************
* Zero page absolute modes
**************************/

func TestZeroPageAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0x00C0, 0xFF)
	ram.Poke(0xC000, 0xA5) // LDA $C0
	ram.Poke(0xC001, 0xC0)
	ram.Poke(0xC002, 0xEA) // NOP

	steps := []testData{
		{0xC000, 0xA5, true, 0x00, 0x00, 0x00, "", 0xC001},
		{0xC001, 0xC0, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0x00C0, 0xFF, true, 0xFF, 0x00, 0x00, "", 0xC002},
		{0xC002, 0xEA, true, 0xFF, 0x00, 0x00, "", 0xC003},
	}

	runTest(cpu, ram, steps, t)
}

func TestZeroPageRMWAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0x00C0, 0xFA)
	ram.Poke(0xC000, 0xE6) // INC $C0
	ram.Poke(0xC001, 0xC0)
	ram.Poke(0xC002, 0xEA) // NOP

	steps := []testData{
		{0xC000, 0xE6, true, 0x00, 0x00, 0x00, "", 0xC001},
		{0xC001, 0xC0, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0x00C0, 0xFA, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0x00C0, 0xFA, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0x00C0, 0xFB, false, 0x00, 0x00, 0x00, "", 0xC002},
		{0xC002, 0xEA, true, 0x00, 0x00, 0x00, "", 0xC003},
	}

	runTest(cpu, ram, steps, t)
}

func TestZeroPageAddressWriteMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xDD

	ram.Poke(0xC000, 0x85) // STA $C0
	ram.Poke(0xC001, 0xC0)
	ram.Poke(0xC002, 0xEA) // NOP

	steps := []testData{
		{0xC000, 0x85, true, 0xDD, 0x00, 0x00, "", 0xC001},
		{0xC001, 0xC0, true, 0xDD, 0x00, 0x00, "", 0xC002},
		{0x00C0, 0xDD, false, 0xDD, 0x00, 0x00, "", 0xC002},
		{0xC002, 0xEA, true, 0xDD, 0x00, 0x00, "", 0xC003},
	}

	runTest(cpu, ram, steps, t)
}

/************************
* Zero page indexed modes
*************************/

func TestZeroPageIndexedAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.xRegister = 5

	ram.Poke(0x00C0, 0xFF)
	ram.Poke(0x00C5, 0xFA)

	ram.Poke(0xC000, 0xB5) // LDA $C0,X
	ram.Poke(0xC001, 0xC0)
	ram.Poke(0xC002, 0xEA) // NOP

	steps := []testData{
		{0xC000, 0xB5, true, 0x00, 0x05, 0x00, "", 0xC001},
		{0xC001, 0xC0, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0x00C0, 0xFF, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0x00C5, 0xFA, true, 0xFA, 0x05, 0x00, "", 0xC002},
		{0xC002, 0xEA, true, 0xFA, 0x05, 0x00, "", 0xC003},
	}

	runTest(cpu, ram, steps, t)
}

func TestZeroPageIndexedYAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.yRegister = 5

	ram.Poke(0x00C0, 0xFF)
	ram.Poke(0x00C5, 0xFA)

	ram.Poke(0xC000, 0xB6) // LDX $C0,Y
	ram.Poke(0xC001, 0xC0)
	ram.Poke(0xC002, 0xEA) // NOP

	steps := []testData{
		{0xC000, 0xB6, true, 0x00, 0x00, 0x05, "", 0xC001},
		{0xC001, 0xC0, true, 0x00, 0x00, 0x05, "", 0xC002},
		{0x00C0, 0xFF, true, 0x00, 0x00, 0x05, "", 0xC002},
		{0x00C5, 0xFA, true, 0x00, 0xFA, 0x05, "", 0xC002},
		{0xC002, 0xEA, true, 0x00, 0xFA, 0x05, "", 0xC003},
	}

	runTest(cpu, ram, steps, t)
}

func TestZeroPageIndexedRMWAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.xRegister = 0x05

	ram.Poke(0x00C0, 0xFF)
	ram.Poke(0x00C5, 0xFA)

	ram.Poke(0xC000, 0xF6) // INC $C0,X
	ram.Poke(0xC001, 0xC0)
	ram.Poke(0xC002, 0xEA) // NOP

	steps := []testData{
		{0xC000, 0xF6, true, 0x00, 0x05, 0x00, "", 0xC001},
		{0xC001, 0xC0, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0x00C0, 0xFF, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0x00C5, 0xFA, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0x00C5, 0xFB, false, 0x00, 0x05, 0x00, "", 0xC002},
		{0xC002, 0xEA, true, 0x00, 0x05, 0x00, "", 0xC003},
	}

	runTest(cpu, ram, steps, t)
}

func TestZeroPageIndexedWriteAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xBB
	cpu.xRegister = 5

	ram.Poke(0x00C0, 0xFF)
	ram.Poke(0x00C5, 0xFA)

	ram.Poke(0xC000, 0x95) // STA $C0,X
	ram.Poke(0xC001, 0xC0)
	ram.Poke(0xC002, 0xEA) // NOP

	steps := []testData{
		{0xC000, 0x95, true, 0xBB, 0x05, 0x00, "", 0xC001},
		{0xC001, 0xC0, true, 0xBB, 0x05, 0x00, "", 0xC002},
		{0x00C0, 0xFF, true, 0xBB, 0x05, 0x00, "", 0xC002},
		{0x00C5, 0xBB, false, 0xBB, 0x05, 0x00, "", 0xC002},
		{0xC002, 0xEA, true, 0xBB, 0x05, 0x00, "", 0xC003},
	}

	runTest(cpu, ram, steps, t)
}

/***********************
* Absolute indexed modes
************************/

func TestAbsoluteIndexedWithExtraCycleAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.xRegister = 5

	ram.Poke(0xCFFF, 0xFF)
	ram.Poke(0xD004, 0xFA)

	ram.Poke(0xC000, 0xBD) // LDA $CFFF,X
	ram.Poke(0xC001, 0xFF)
	ram.Poke(0xC002, 0xCF)
	ram.Poke(0xC003, 0xEA) // NOP

	steps := []testData{
		{0xC000, 0xBD, true, 0x00, 0x05, 0x00, "", 0xC001},
		{0xC001, 0xFF, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0xC002, 0xCF, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xCFFF, 0xFF, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xD004, 0xFA, true, 0xFA, 0x05, 0x00, "", 0xC003},
		{0xC003, 0xEA, true, 0xFA, 0x05, 0x00, "", 0xC004},
	}

	runTest(cpu, ram, steps, t)
}

func TestAbsoluteIndexedAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.xRegister = 5

	ram.Poke(0xD000, 0xFF)
	ram.Poke(0xD005, 0xFA)

	ram.Poke(0xC000, 0xBD) // LDA $D000,X
	ram.Poke(0xC001, 0x00)
	ram.Poke(0xC002, 0xD0)
	ram.Poke(0xC003, 0xEA) // NOP

	steps := []testData{
		{0xC000, 0xBD, true, 0x00, 0x05, 0x00, "", 0xC001},
		{0xC001, 0x00, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0xC002, 0xD0, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xD005, 0xFA, true, 0xFA, 0x05, 0x00, "", 0xC003},
		{0xC003, 0xEA, true, 0xFA, 0x05, 0x00, "", 0xC004},
	}

	runTest(cpu, ram, steps, t)
}

func TestAbsoluteIndexedRMWWithExtraCycleAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.xRegister = 5

	ram.Poke(0xCFFF, 0xFF)
	ram.Poke(0xD004, 0xFA)

	ram.Poke(0xC000, 0x1E) // ASL $CFFF,X
	ram.Poke(0xC001, 0xFF)
	ram.Poke(0xC002, 0xCF)
	ram.Poke(0xC003, 0xEA) // NOP

	steps := []testData{
		{0xC000, 0x1E, true, 0x00, 0x05, 0x00, "", 0xC001},
		{0xC001, 0xFF, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0xC002, 0xCF, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xCFFF, 0xFF, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xD004, 0xFA, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xD004, 0xFA, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xD004, 0xF4, false, 0x00, 0x05, 0x00, "C", 0xC003},
		{0xC003, 0xEA, true, 0x00, 0x05, 0x00, "C", 0xC004},
	}

	runTest(cpu, ram, steps, t)
}

func TestAbsoluteIndexedRMWAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.xRegister = 5

	ram.Poke(0xD000, 0xFF)
	ram.Poke(0xD005, 0xFA)

	ram.Poke(0xC000, 0x1E) // ASL $D000,X
	ram.Poke(0xC001, 0x00)
	ram.Poke(0xC002, 0xD0)
	ram.Poke(0xC003, 0xEA) // NOP

	steps := []testData{
		{0xC000, 0x1E, true, 0x00, 0x05, 0x00, "", 0xC001},
		{0xC001, 0x00, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0xC002, 0xD0, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xD005, 0xFA, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xD005, 0xFA, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xD005, 0xF4, false, 0x00, 0x05, 0x00, "C", 0xC003},
		{0xC003, 0xEA, true, 0x00, 0x05, 0x00, "C", 0xC004},
	}

	runTest(cpu, ram, steps, t)
}

func TestAbsoluteIndexedRMWIncAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.xRegister = 5

	ram.Poke(0xD000, 0xFF)
	ram.Poke(0xD005, 0xFA)

	ram.Poke(0xC000, 0xFE) // INC $D000,X
	ram.Poke(0xC001, 0x00)
	ram.Poke(0xC002, 0xD0)
	ram.Poke(0xC003, 0xEA) // NOP

	steps := []testData{
		{0xC000, 0xFE, true, 0x00, 0x05, 0x00, "", 0xC001},
		{0xC001, 0x00, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0xC002, 0xD0, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xD000, 0xFF, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xD005, 0xFA, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xD005, 0xFA, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xD005, 0xFB, false, 0x00, 0x05, 0x00, "", 0xC003},
		{0xC003, 0xEA, true, 0x00, 0x05, 0x00, "", 0xC004},
	}

	runTest(cpu, ram, steps, t)
}

func TestAbsoluteIndexedWithExtraCycleWriteAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xCC
	cpu.yRegister = 5

	ram.Poke(0xCFFF, 0xFF)
	ram.Poke(0xD004, 0xFA)

	ram.Poke(0xC000, 0x99) // STA $CFFF,Y
	ram.Poke(0xC001, 0xFF)
	ram.Poke(0xC002, 0xCF)
	ram.Poke(0xC003, 0xEA) // NOP

	steps := []testData{
		{0xC000, 0x99, true, 0xCC, 0x00, 0x05, "", 0xC001},
		{0xC001, 0xFF, true, 0xCC, 0x00, 0x05, "", 0xC002},
		{0xC002, 0xCF, true, 0xCC, 0x00, 0x05, "", 0xC003},
		{0xCFFF, 0xFF, true, 0xCC, 0x00, 0x05, "", 0xC003},
		{0xD004, 0xCC, false, 0xCC, 0x00, 0x05, "", 0xC003},
		{0xC003, 0xEA, true, 0xCC, 0x00, 0x05, "", 0xC004},
	}

	runTest(cpu, ram, steps, t)
}

func TestAbsoluteIndexedWriteAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xCC
	cpu.yRegister = 5

	ram.Poke(0xD000, 0xFF)
	ram.Poke(0xD005, 0xFA)

	ram.Poke(0xC000, 0x99) // STA $D000,Y
	ram.Poke(0xC001, 0x00)
	ram.Poke(0xC002, 0xD0)
	ram.Poke(0xC003, 0xEA) // NOP

	steps := []testData{
		{0xC000, 0x99, true, 0xCC, 0x00, 0x05, "", 0xC001},
		{0xC001, 0x00, true, 0xCC, 0x00, 0x05, "", 0xC002},
		{0xC002, 0xD0, true, 0xCC, 0x00, 0x05, "", 0xC003},
		{0xD005, 0xCC, false, 0xCC, 0x00, 0x05, "", 0xC003},
		{0xC003, 0xEA, true, 0xCC, 0x00, 0x05, "", 0xC004},
	}

	runTest(cpu, ram, steps, t)
}

func TestAbsoluteIndexedSTAAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xCC
	cpu.xRegister = 5

	ram.Poke(0xD000, 0xFF)
	ram.Poke(0xD005, 0xFA)

	ram.Poke(0xC000, 0x9D) // STA $D000,X
	ram.Poke(0xC001, 0x00)
	ram.Poke(0xC002, 0xD0)
	ram.Poke(0xC003, 0xEA) // NOP

	steps := []testData{
		{0xC000, 0x9D, true, 0xCC, 0x05, 0x00, "", 0xC001},
		{0xC001, 0x00, true, 0xCC, 0x05, 0x00, "", 0xC002},
		{0xC002, 0xD0, true, 0xCC, 0x05, 0x00, "", 0xC003},
		{0xD000, 0xFF, true, 0xCC, 0x05, 0x00, "", 0xC003},
		{0xD005, 0xCC, false, 0xCC, 0x05, 0x00, "", 0xC003},
		{0xC003, 0xEA, true, 0xCC, 0x05, 0x00, "", 0xC004},
	}

	runTest(cpu, ram, steps, t)
}

/**************************************************************************
* Other tests
* -----------
* This section will tests if the address modes have the right output on
* each cycle. For reference on the behaviour of each cycle see:
* https://www.atarihq.com/danb/files/64doc.txt
**************************************************************************/
func TestCpuReadOpCodeCycle(t *testing.T) {
}
