package cpu

import (
	"strings"
	"testing"
	"unicode"

	"github.com/fran150/clementina6502/pkg/components/buses"
	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/fran150/clementina6502/pkg/components/memory"
)

// These are the values to validate after each cycle. The tests will check if
// after each cycle the values on the CPU matches the ones in this struct
// Uppercase flag records means that the flag needs to be checked for set, lower case
// will check if the flag is unset.
// Typical characters for the flags are used NV-BDIZC.
// For example "nvZ" will check that flag negative and overflow are NOT set and the
// zero flag is set
type addressModeTestData struct {
	addressBus          uint16
	dataBus             uint8
	writeEnable         bool
	accumulatorRegister uint8
	xRegister           uint8
	yRegister           uint8
	signalLines         string
	programCounter      uint16
}

// See addressModeTestData. This data is what it will be validated after each cycle.
// This structs includes also values to trigger NMI, regular interrupts or reset.
type addressModeTestDataWithControlLines struct {
	addressModeTestData

	triggerIRQ   bool
	triggerNMI   bool
	triggerReset bool
	setNotReady  bool
}

// Creates a computer for testing the CPU emulation.
// It is only 64K of RAM memory connected to the bus, processor lines are wired
// to always high or low lines.
func createComputer() (*Cpu65C02S, *memory.Ram) {
	addressBus := buses.Create16BitBus()
	dataBus := buses.Create8BitBus()

	alwaysHighLine := buses.CreateStandaloneLine(true)
	alwaysLowLine := buses.CreateStandaloneLine(false)

	writeEnableLine := buses.CreateStandaloneLine(true)

	memoryLockLine := buses.CreateStandaloneLine(false)
	syncLine := buses.CreateStandaloneLine(false)
	vectorPullLine := buses.CreateStandaloneLine(false)

	ram := memory.CreateRam(memory.RAM_SIZE_64K)
	ram.AddressBus().Connect(addressBus)
	ram.DataBus().Connect(dataBus)
	ram.WriteEnable().Connect(writeEnableLine)
	ram.ChipSelect().Connect(alwaysLowLine)
	ram.OutputEnable().Connect(alwaysLowLine)

	cpu := CreateCPU()
	cpu.AddressBus().Connect(addressBus)
	cpu.DataBus().Connect(dataBus)

	cpu.BusEnable().Connect(alwaysHighLine)
	cpu.ReadWrite().Connect(writeEnableLine)
	cpu.MemoryLock().Connect(memoryLockLine)
	cpu.Sync().Connect(syncLine)
	cpu.Ready().Connect(alwaysHighLine)
	cpu.VectorPull().Connect(vectorPullLine)
	cpu.SetOverflow().Connect(alwaysHighLine)
	cpu.Reset().Connect(alwaysHighLine)

	cpu.InterruptRequest().Connect(alwaysHighLine)
	cpu.NonMaskableInterrupt().Connect(alwaysHighLine)

	cpu.programCounter = 0xC000

	return cpu, ram
}

// Similar to the createComputer() function but in this case it returns the NMI, IRQ, RESET and RDY lines
// to allow testing
func createComputerWithControlLines() (*Cpu65C02S, *memory.Ram, *buses.StandaloneLine, *buses.StandaloneLine, *buses.StandaloneLine, *buses.StandaloneLine) {
	cpu, ram := createComputer()

	nmiLine := buses.CreateStandaloneLine(true)
	irqLine := buses.CreateStandaloneLine(true)
	resetLine := buses.CreateStandaloneLine(true)
	readyLine := buses.CreateStandaloneLine(true)

	cpu.NonMaskableInterrupt().Connect(nmiLine)
	cpu.InterruptRequest().Connect(irqLine)
	cpu.Reset().Connect(resetLine)
	cpu.Ready().Connect(readyLine)

	return cpu, ram, nmiLine, irqLine, resetLine, readyLine
}

// Evaluates the current status of the CPU on a given cycle and
// compares it with the expceted value from the test data.
func evaluateCycle(cycle int, cpu *Cpu65C02S, step *addressModeTestData, t *testing.T) {
	evaluateLine(cycle, cpu.readWrite.GetLine().Status(), step.writeEnable, t, "R/W")

	evaluateRegister(cycle, cpu.accumulatorRegister, step.accumulatorRegister, t, "A")
	evaluateRegister(cycle, cpu.xRegister, step.xRegister, t, "X")
	evaluateRegister(cycle, cpu.xRegister, step.xRegister, t, "Y")

	evaluateRegister(cycle, cpu.programCounter, step.programCounter, t, "program counter")

	evaluateSignalLines(t, cpu, step.signalLines)
}

// Evaluates the value of the specified CPU line
func evaluateLine(cycle int, status bool, stepStatus bool, t *testing.T, lineName string) {
	if status != stepStatus {
		t.Errorf("Cycle %v - Current %v and expected %v status %s line don't match", cycle, status, stepStatus, lineName)
	}
}

// Returns the appropriate signal line based on the letter specified for the test
func getSignalLine(cpu *Cpu65C02S, signalCode rune) buses.LineConnector {
	switch string(unicode.ToLower(signalCode)) {
	case "m":
		return cpu.memoryLock
	case "s":
		return cpu.sync
	case "v":
		return cpu.vectorPull
	case "r":
		return cpu.ready
	}

	return nil
}

// Evaluates the status of the signal lines according to the letter specified in the signalString parameter.
// If the letter is specified upper case, then the line is expected to be enabled, if the letter is specified lower case the line
// is expected to be disabled.
// If not specified M (Memory Lock), S (Sync) and V (lines) are expected to be disabled. If not specified, R (Ready) line is
// expected to be enabled. For example, specifying "" will expect M,S and V disabled and R enabled.
func evaluateSignalLines(t *testing.T, cpu *Cpu65C02S, signalString string) {
	// Lines to be evaluated and their default expected status (upper case -> enabled, lower case -> disabled)
	const signals string = "msvR"
	// Line names to show when reporting error
	lineNames := []string{"Memory Lock", "Sync", "Vector Pull", "Ready"}

	// Gets the current instruction and address mode
	instruction := cpu.instructionSet.GetByOpCode(cpu.currentOpCode)
	addressMode := cpu.addressModeSet.GetByName(instruction.addressMode)

	// For each line to be evaluated
	for i, signal := range signals {
		// Get the signal line.
		line := getSignalLine(cpu, signal)

		// Get the upper and lower case values of the signal.
		uc := unicode.ToUpper(signal)
		lc := unicode.ToLower(signal)

		// If it's specified, it Forces the signal expected status to the expected value
		if strings.Contains(signalString, string(uc)) {
			signal = uc
		}

		if strings.Contains(signalString, string(lc)) {
			signal = lc
		}

		// Throw error if the signal is expected enabled but is disabled
		if signal == uc && !line.Enabled() {
			t.Errorf("%s - %s - Expected %s line to be enabled", instruction.Mnemonic(), addressMode.Text(), lineNames[i])
		}

		// Throw error if the signal is expected disabled but is enabled
		if signal == lc && line.Enabled() {
			t.Errorf("%s - %s - Expected %s line NOT to be enabled", instruction.Mnemonic(), addressMode.Text(), lineNames[i])
		}
	}
}

// Evaluates the value of the specified register values.
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

// Iterates over the specified steps comparting the status of the CPU with the expected values.
func runTest(cpu *Cpu65C02S, ram *memory.Ram, steps []addressModeTestData, t *testing.T) {
	t.Logf("Cycle \t Addr \t Data \t R/W \t PC \t A \t X \t Y \t SP \t Flags \n")
	t.Logf("---- \t ---- \t ---- \t ---- \t ---- \t -- \t -- \t -- \t -- \t ----- \n")

	context := common.CreateStepContext()

	for cycle, step := range steps {
		cpu.Tick(context)
		ram.Tick(context)

		evaluateRegister(cycle, cpu.addressBus.Read(), step.addressBus, t, "address bus")
		evaluateRegister(cycle, cpu.dataBus.Read(), step.dataBus, t, "data bus")

		cpu.PostTick(context)

		t.Logf("%v \t %04X \t %02X \t %v \t %04X \t %02X \t %02X \t %02X \t %02X \t %08b \n", cycle, cpu.addressBus.Read(), cpu.dataBus.Read(), cpu.readWrite.GetLine().Status(), cpu.programCounter, cpu.accumulatorRegister, cpu.xRegister, cpu.yRegister, cpu.stackPointer, cpu.processorStatusRegister.ReadValue())

		evaluateCycle(cycle, cpu, &step, t)

		context.Next()
	}
}

// Same as ruuTest function but allows to evaluate and test the status of the CPU with respect
// to IRQ, NMI, Reset and RDY control lines. IRQ, NMI and NMI triggers interruptions. Reset is used
// to send the CPU to an initial known state, and RDY is and both input output, if it's pulled to disabled
// halts the CPU leaving the address bus in the last status. When the processor halts during WAI and STP
// instructions, it pulls this line to disable
func runTestWithInterrupts(cpu *Cpu65C02S, ram *memory.Ram, irqLine *buses.StandaloneLine, nmiLine *buses.StandaloneLine, resetLine *buses.StandaloneLine, readyLine *buses.StandaloneLine, steps []addressModeTestDataWithControlLines, t *testing.T) {
	t.Logf("Cycle \t Addr \t Data \t R/W \t PC \t A \t X \t Y \t SP \t Flags \n")
	t.Logf("---- \t ---- \t ---- \t ---- \t ---- \t -- \t -- \t -- \t -- \t ----- \n")

	context := common.CreateStepContext()

	lines := []*buses.StandaloneLine{irqLine, nmiLine, resetLine, readyLine}

	for cycle, step := range steps {
		stepLinesActions := []bool{step.triggerIRQ, step.triggerNMI, step.triggerReset, step.setNotReady}

		for i := 0; i < len(lines); i++ {
			if stepLinesActions[i] {
				lines[i].Set(false)
			}
		}

		cpu.Tick(context)
		ram.Tick(context)

		evaluateRegister(cycle, cpu.addressBus.Read(), step.addressBus, t, "address bus")
		evaluateRegister(cycle, cpu.dataBus.Read(), step.dataBus, t, "data bus")

		cpu.PostTick(context)

		t.Logf("%v \t %04X \t %02X \t %v \t %04X \t %02X \t %02X \t %02X \t %02X \t %08b \n", cycle, cpu.addressBus.Read(), cpu.dataBus.Read(), cpu.readWrite.GetLine().Status(), cpu.programCounter, cpu.accumulatorRegister, cpu.xRegister, cpu.yRegister, cpu.stackPointer, cpu.processorStatusRegister.ReadValue())

		evaluateCycle(cycle, cpu, &step.addressModeTestData, t)

		for i := 0; i < len(lines); i++ {
			lines[i].Set(true)
		}

		context.Next()
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

	steps := []addressModeTestData{
		{0xC000, 0x18, true, 0x00, 0x00, 0x00, "S", 0xC001},
		{0xC001, 0xEA, true, 0x00, 0x00, 0x00, "", 0xC001},
	}

	runTest(cpu, ram, steps, t)
}

func TestAccumulatorAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0x2A) // ROL a
	ram.Poke(0xC001, 0xEA) // NOP

	steps := []addressModeTestData{
		{0xC000, 0x2A, true, 0x00, 0x00, 0x00, "S", 0xC001},
		{0xC001, 0xEA, true, 0x00, 0x00, 0x00, "", 0xC001},
	}

	runTest(cpu, ram, steps, t)
}

func TestImmediateAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0xA9) // LDA #FF
	ram.Poke(0xC001, 0xFF)
	ram.Poke(0xC002, 0xEA) // NOP

	steps := []addressModeTestData{
		{0xC000, 0xA9, true, 0x00, 0x00, 0x00, "S", 0xC001},
		{0xC001, 0xFF, true, 0xFF, 0x00, 0x00, "", 0xC002},
		{0xC002, 0xEA, true, 0xFF, 0x00, 0x00, "S", 0xC003},
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

	steps := []addressModeTestData{
		{0xC000, 0x4C, true, 0x00, 0x00, 0x00, "S", 0xC001},
		{0xC001, 0x00, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0xC002, 0xD0, true, 0x00, 0x00, 0x00, "", 0xD000},
		{0xD000, 0xEA, true, 0x00, 0x00, 0x00, "S", 0xD001},
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

	steps := []addressModeTestData{
		{0xC000, 0xAD, true, 0x00, 0x00, 0x00, "S", 0xC001},
		{0xC001, 0x00, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0xC002, 0xD0, true, 0x00, 0x00, 0x00, "", 0xC003},
		{0xD000, 0xFA, true, 0xFA, 0x00, 0x00, "", 0xC003},
		{0xC003, 0xEA, true, 0xFA, 0x00, 0x00, "S", 0xC004},
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

	steps := []addressModeTestData{
		{0xC000, 0xEE, true, 0x00, 0x00, 0x00, "S", 0xC001},
		{0xC001, 0x00, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0xC002, 0xD0, true, 0x00, 0x00, 0x00, "", 0xC003},
		{0xD000, 0xFA, true, 0x00, 0x00, 0x00, "M", 0xC003},
		{0xD000, 0xFA, true, 0x00, 0x00, 0x00, "M", 0xC003},
		{0xD000, 0xFB, false, 0x00, 0x00, 0x00, "M", 0xC003},
		{0xC003, 0xEA, true, 0x00, 0x00, 0x00, "S", 0xC004},
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

	steps := []addressModeTestData{
		{0xC000, 0x8D, true, 0xFF, 0x00, 0x00, "S", 0xC001},
		{0xC001, 0x00, true, 0xFF, 0x00, 0x00, "", 0xC002},
		{0xC002, 0xD0, true, 0xFF, 0x00, 0x00, "", 0xC003},
		{0xD000, 0xFF, false, 0xFF, 0x00, 0x00, "", 0xC003},
		{0xC003, 0xEA, true, 0xFF, 0x00, 0x00, "S", 0xC004},
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

	steps := []addressModeTestData{
		{0xC000, 0xA5, true, 0x00, 0x00, 0x00, "S", 0xC001},
		{0xC001, 0xC0, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0x00C0, 0xFF, true, 0xFF, 0x00, 0x00, "", 0xC002},
		{0xC002, 0xEA, true, 0xFF, 0x00, 0x00, "S", 0xC003},
	}

	runTest(cpu, ram, steps, t)
}

func TestZeroPageRMWAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0x00C0, 0xFA)
	ram.Poke(0xC000, 0xE6) // INC $C0
	ram.Poke(0xC001, 0xC0)
	ram.Poke(0xC002, 0xEA) // NOP

	steps := []addressModeTestData{
		{0xC000, 0xE6, true, 0x00, 0x00, 0x00, "S", 0xC001},
		{0xC001, 0xC0, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0x00C0, 0xFA, true, 0x00, 0x00, 0x00, "M", 0xC002},
		{0x00C0, 0xFA, true, 0x00, 0x00, 0x00, "M", 0xC002},
		{0x00C0, 0xFB, false, 0x00, 0x00, 0x00, "M", 0xC002},
		{0xC002, 0xEA, true, 0x00, 0x00, 0x00, "S", 0xC003},
	}

	runTest(cpu, ram, steps, t)
}

func TestZeroPageAddressWriteMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xDD

	ram.Poke(0xC000, 0x85) // STA $C0
	ram.Poke(0xC001, 0xC0)
	ram.Poke(0xC002, 0xEA) // NOP

	steps := []addressModeTestData{
		{0xC000, 0x85, true, 0xDD, 0x00, 0x00, "S", 0xC001},
		{0xC001, 0xC0, true, 0xDD, 0x00, 0x00, "", 0xC002},
		{0x00C0, 0xDD, false, 0xDD, 0x00, 0x00, "", 0xC002},
		{0xC002, 0xEA, true, 0xDD, 0x00, 0x00, "S", 0xC003},
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

	steps := []addressModeTestData{
		{0xC000, 0xB5, true, 0x00, 0x05, 0x00, "S", 0xC001},
		{0xC001, 0xC0, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0x00C0, 0xFF, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0x00C5, 0xFA, true, 0xFA, 0x05, 0x00, "", 0xC002},
		{0xC002, 0xEA, true, 0xFA, 0x05, 0x00, "S", 0xC003},
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

	steps := []addressModeTestData{
		{0xC000, 0xB6, true, 0x00, 0x00, 0x05, "S", 0xC001},
		{0xC001, 0xC0, true, 0x00, 0x00, 0x05, "", 0xC002},
		{0x00C0, 0xFF, true, 0x00, 0x00, 0x05, "", 0xC002},
		{0x00C5, 0xFA, true, 0x00, 0xFA, 0x05, "", 0xC002},
		{0xC002, 0xEA, true, 0x00, 0xFA, 0x05, "S", 0xC003},
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

	steps := []addressModeTestData{
		{0xC000, 0xF6, true, 0x00, 0x05, 0x00, "S", 0xC001},
		{0xC001, 0xC0, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0x00C0, 0xFF, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0x00C5, 0xFA, true, 0x00, 0x05, 0x00, "M", 0xC002},
		{0x00C5, 0xFA, true, 0x00, 0x05, 0x00, "M", 0xC002},
		{0x00C5, 0xFB, false, 0x00, 0x05, 0x00, "M", 0xC002},
		{0xC002, 0xEA, true, 0x00, 0x05, 0x00, "S", 0xC003},
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

	steps := []addressModeTestData{
		{0xC000, 0x95, true, 0xBB, 0x05, 0x00, "S", 0xC001},
		{0xC001, 0xC0, true, 0xBB, 0x05, 0x00, "", 0xC002},
		{0x00C0, 0xFF, true, 0xBB, 0x05, 0x00, "", 0xC002},
		{0x00C5, 0xBB, false, 0xBB, 0x05, 0x00, "", 0xC002},
		{0xC002, 0xEA, true, 0xBB, 0x05, 0x00, "S", 0xC003},
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

	steps := []addressModeTestData{
		{0xC000, 0xBD, true, 0x00, 0x05, 0x00, "S", 0xC001},
		{0xC001, 0xFF, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0xC002, 0xCF, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xCFFF, 0xFF, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xD004, 0xFA, true, 0xFA, 0x05, 0x00, "", 0xC003},
		{0xC003, 0xEA, true, 0xFA, 0x05, 0x00, "S", 0xC004},
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

	steps := []addressModeTestData{
		{0xC000, 0xBD, true, 0x00, 0x05, 0x00, "S", 0xC001},
		{0xC001, 0x00, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0xC002, 0xD0, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xD005, 0xFA, true, 0xFA, 0x05, 0x00, "", 0xC003},
		{0xC003, 0xEA, true, 0xFA, 0x05, 0x00, "S", 0xC004},
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

	steps := []addressModeTestData{
		{0xC000, 0x1E, true, 0x00, 0x05, 0x00, "S", 0xC001},
		{0xC001, 0xFF, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0xC002, 0xCF, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xCFFF, 0xFF, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xD004, 0xFA, true, 0x00, 0x05, 0x00, "M", 0xC003},
		{0xD004, 0xFA, true, 0x00, 0x05, 0x00, "M", 0xC003},
		{0xD004, 0xF4, false, 0x00, 0x05, 0x00, "M", 0xC003},
		{0xC003, 0xEA, true, 0x00, 0x05, 0x00, "S", 0xC004},
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

	steps := []addressModeTestData{
		{0xC000, 0x1E, true, 0x00, 0x05, 0x00, "S", 0xC001},
		{0xC001, 0x00, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0xC002, 0xD0, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xD005, 0xFA, true, 0x00, 0x05, 0x00, "M", 0xC003},
		{0xD005, 0xFA, true, 0x00, 0x05, 0x00, "M", 0xC003},
		{0xD005, 0xF4, false, 0x00, 0x05, 0x00, "M", 0xC003},
		{0xC003, 0xEA, true, 0x00, 0x05, 0x00, "S", 0xC004},
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

	steps := []addressModeTestData{
		{0xC000, 0xFE, true, 0x00, 0x05, 0x00, "S", 0xC001},
		{0xC001, 0x00, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0xC002, 0xD0, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xD000, 0xFF, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xD005, 0xFA, true, 0x00, 0x05, 0x00, "M", 0xC003},
		{0xD005, 0xFA, true, 0x00, 0x05, 0x00, "M", 0xC003},
		{0xD005, 0xFB, false, 0x00, 0x05, 0x00, "M", 0xC003},
		{0xC003, 0xEA, true, 0x00, 0x05, 0x00, "S", 0xC004},
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

	steps := []addressModeTestData{
		{0xC000, 0x99, true, 0xCC, 0x00, 0x05, "S", 0xC001},
		{0xC001, 0xFF, true, 0xCC, 0x00, 0x05, "", 0xC002},
		{0xC002, 0xCF, true, 0xCC, 0x00, 0x05, "", 0xC003},
		{0xCFFF, 0xFF, true, 0xCC, 0x00, 0x05, "", 0xC003},
		{0xD004, 0xCC, false, 0xCC, 0x00, 0x05, "", 0xC003},
		{0xC003, 0xEA, true, 0xCC, 0x00, 0x05, "S", 0xC004},
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

	steps := []addressModeTestData{
		{0xC000, 0x99, true, 0xCC, 0x00, 0x05, "S", 0xC001},
		{0xC001, 0x00, true, 0xCC, 0x00, 0x05, "", 0xC002},
		{0xC002, 0xD0, true, 0xCC, 0x00, 0x05, "", 0xC003},
		{0xD005, 0xCC, false, 0xCC, 0x00, 0x05, "", 0xC003},
		{0xC003, 0xEA, true, 0xCC, 0x00, 0x05, "S", 0xC004},
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

	steps := []addressModeTestData{
		{0xC000, 0x9D, true, 0xCC, 0x05, 0x00, "S", 0xC001},
		{0xC001, 0x00, true, 0xCC, 0x05, 0x00, "", 0xC002},
		{0xC002, 0xD0, true, 0xCC, 0x05, 0x00, "", 0xC003},
		{0xD000, 0xFF, true, 0xCC, 0x05, 0x00, "", 0xC003},
		{0xD005, 0xCC, false, 0xCC, 0x05, 0x00, "", 0xC003},
		{0xC003, 0xEA, true, 0xCC, 0x05, 0x00, "S", 0xC004},
	}

	runTest(cpu, ram, steps, t)
}

/***********************
* Relative
************************/

func TestRelativeBranchTakenAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.processorStatusRegister.SetFlag(CarryFlagBit, false)

	ram.Poke(0xC000, 0x90) // BCC $03
	ram.Poke(0xC001, 0x03)
	ram.Poke(0xC002, 0x9D) // STA $0000,x
	ram.Poke(0xC003, 0x00) //
	ram.Poke(0xC004, 0x00) //
	ram.Poke(0xC005, 0x18) // CLC

	steps := []addressModeTestData{
		{0xC000, 0x90, true, 0x00, 0x00, 0x00, "S", 0xC001},
		{0xC001, 0x03, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0xC002, 0x9D, true, 0x00, 0x00, 0x00, "", 0xC005},
		{0xC005, 0x18, true, 0x00, 0x00, 0x00, "S", 0xC006},
	}

	runTest(cpu, ram, steps, t)
}

func TestRelativeBranchTakenPageBoundaryAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.programCounter = 0xC0FD
	cpu.processorStatusRegister.SetFlag(CarryFlagBit, false)

	ram.Poke(0xC0FD, 0x90) // BCC $03
	ram.Poke(0xC0FE, 0x03)
	ram.Poke(0xC0FF, 0x9D) // STA $0000,x
	ram.Poke(0xC100, 0x00) //
	ram.Poke(0xC101, 0x00) //
	ram.Poke(0xC102, 0x18) // CLC

	steps := []addressModeTestData{
		{0xC0FD, 0x90, true, 0x00, 0x00, 0x00, "S", 0xC0FE},
		{0xC0FE, 0x03, true, 0x00, 0x00, 0x00, "", 0xC0FF},
		{0xC0FF, 0x9D, true, 0x00, 0x00, 0x00, "", 0xC102},
		{0xC102, 0x18, true, 0x00, 0x00, 0x00, "", 0xC102},
		{0xC102, 0x18, true, 0x00, 0x00, 0x00, "S", 0xC103},
	}

	runTest(cpu, ram, steps, t)
}

func TestRelativeBranchNotTakenAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.processorStatusRegister.SetFlag(CarryFlagBit, true)

	ram.Poke(0xC000, 0x90) // BCC $03
	ram.Poke(0xC001, 0x03)
	ram.Poke(0xC002, 0x9D) // STA $0000,x
	ram.Poke(0xC003, 0x00) //
	ram.Poke(0xC004, 0x00) //
	ram.Poke(0xC005, 0x18) // CLC

	steps := []addressModeTestData{
		{0xC000, 0x90, true, 0x00, 0x00, 0x00, "S", 0xC001},
		{0xC001, 0x03, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0xC002, 0x9D, true, 0x00, 0x00, 0x00, "S", 0xC003},
		{0xC003, 0x00, true, 0x00, 0x00, 0x00, "", 0xC004},
		{0xC004, 0x00, true, 0x00, 0x00, 0x00, "", 0xC005},
	}

	runTest(cpu, ram, steps, t)
}

/*******************************************************
* Relative Extended
* See https://github.com/SingleStepTests/65x02/pull/3
********************************************************/

func TestRelativeExtendedBranchTakenAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0x0010, 0xF0)

	ram.Poke(0xC000, 0x0F) // BBR0 $10,$05
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC002, 0x05)
	ram.Poke(0xC003, 0xEA) // NOP

	ram.Poke(0xC008, 0x18) // CLC

	steps := []addressModeTestData{
		{0xC000, 0x0F, true, 0x00, 0x00, 0x00, "S", 0xC001},
		{0xC001, 0x10, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0x0010, 0xF0, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0x0010, 0xF0, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0xC002, 0x05, true, 0x00, 0x00, 0x00, "", 0xC003},
		{0xC003, 0xEA, true, 0x00, 0x00, 0x00, "", 0xC008},
		{0xC008, 0x18, true, 0x00, 0x00, 0x00, "S", 0xC009},
	}

	runTest(cpu, ram, steps, t)
}

func TestRelativeExtendedBranchTakenPageBoundaryAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.programCounter = 0xC0FC

	ram.Poke(0x0010, 0xFD)

	ram.Poke(0xC0FC, 0x1F) // BBR1 $03
	ram.Poke(0xC0FD, 0x10)
	ram.Poke(0xC0FE, 0x05)
	ram.Poke(0xC0FF, 0xEA) // NOP

	ram.Poke(0xC104, 0x18) // CLC

	steps := []addressModeTestData{
		{0xC0FC, 0x1F, true, 0x00, 0x00, 0x00, "S", 0xC0FD},
		{0xC0FD, 0x10, true, 0x00, 0x00, 0x00, "", 0xC0FE},
		{0x0010, 0xFD, true, 0x00, 0x00, 0x00, "", 0xC0FE},
		{0x0010, 0xFD, true, 0x00, 0x00, 0x00, "", 0xC0FE},
		{0xC0FE, 0x05, true, 0x00, 0x00, 0x00, "", 0xC0FF},
		{0xC0FF, 0xEA, true, 0x00, 0x00, 0x00, "", 0xC0FF},
		{0xC0FF, 0xEA, true, 0x00, 0x00, 0x00, "", 0xC104},
		{0xC104, 0x18, true, 0x00, 0x00, 0x00, "S", 0xC105},
	}

	runTest(cpu, ram, steps, t)
}

func TestRelativeExtendedBranchNotTakenAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0x0010, 0xFF)

	ram.Poke(0xC000, 0x1F) // BBR1 $10,$05
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC002, 0x05)
	ram.Poke(0xC003, 0xEA) // NOP

	steps := []addressModeTestData{
		{0xC000, 0x1F, true, 0x00, 0x00, 0x00, "S", 0xC001},
		{0xC001, 0x10, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0x0010, 0xFF, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0x0010, 0xFF, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0xC002, 0x05, true, 0x00, 0x00, 0x00, "", 0xC003},
		{0xC003, 0xEA, true, 0x00, 0x00, 0x00, "S", 0xC004},
	}

	runTest(cpu, ram, steps, t)
}

/***********************
* Indexed Indirect X
************************/

func TestZeroPageIndexedIndirectXAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.xRegister = 5

	ram.Poke(0x0005, 0xFF)
	ram.Poke(0x000A, 0x00)
	ram.Poke(0x000B, 0xD0)
	ram.Poke(0xD000, 0xFA)

	ram.Poke(0xC000, 0xA1) // LDA ($05,X)
	ram.Poke(0xC001, 0x05)
	ram.Poke(0xC002, 0xEA) // NOP

	steps := []addressModeTestData{
		{0xC000, 0xA1, true, 0x00, 0x05, 0x00, "S", 0xC001},
		{0xC001, 0x05, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0x0005, 0xFF, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0x000A, 0x00, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0x000B, 0xD0, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0xD000, 0xFA, true, 0xFA, 0x05, 0x00, "", 0xC002},
		{0xC002, 0xEA, true, 0xFA, 0x05, 0x00, "S", 0xC003},
	}

	runTest(cpu, ram, steps, t)
}

func TestZeroPageIndexedIndirectXWriteAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xDD
	cpu.xRegister = 5

	ram.Poke(0x0005, 0xFF)
	ram.Poke(0x000A, 0x00)
	ram.Poke(0x000B, 0xD0)

	ram.Poke(0xC000, 0x81) // STA ($05,X)
	ram.Poke(0xC001, 0x05)
	ram.Poke(0xC002, 0xEA) // NOP

	steps := []addressModeTestData{
		{0xC000, 0x81, true, 0xDD, 0x05, 0x00, "S", 0xC001},
		{0xC001, 0x05, true, 0xDD, 0x05, 0x00, "", 0xC002},
		{0x0005, 0xFF, true, 0xDD, 0x05, 0x00, "", 0xC002},
		{0x000A, 0x00, true, 0xDD, 0x05, 0x00, "", 0xC002},
		{0x000B, 0xD0, true, 0xDD, 0x05, 0x00, "", 0xC002},
		{0xD000, 0xDD, false, 0xDD, 0x05, 0x00, "", 0xC002},
		{0xC002, 0xEA, true, 0xDD, 0x05, 0x00, "S", 0xC003},
	}

	runTest(cpu, ram, steps, t)
}

/***********************
* Indirect Indexed Y
************************/

func TestIndirectIndexedYAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.yRegister = 5

	ram.Poke(0x0005, 0x00)
	ram.Poke(0x0006, 0xD0)
	ram.Poke(0xD005, 0xFA)

	ram.Poke(0xC000, 0xB1) // LDA ($05),Y
	ram.Poke(0xC001, 0x05)
	ram.Poke(0xC002, 0xEA) // NOP

	steps := []addressModeTestData{
		{0xC000, 0xB1, true, 0x00, 0x00, 0x05, "S", 0xC001},
		{0xC001, 0x05, true, 0x00, 0x00, 0x05, "", 0xC002},
		{0x0005, 0x00, true, 0x00, 0x00, 0x05, "", 0xC002},
		{0x0006, 0xD0, true, 0x00, 0x00, 0x05, "", 0xC002},
		{0xD005, 0xFA, true, 0xFA, 0x00, 0x05, "", 0xC002},
		{0xC002, 0xEA, true, 0xFA, 0x00, 0x05, "S", 0xC003},
	}

	runTest(cpu, ram, steps, t)
}

func TestIndirectIndexedYPageBoundaryAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.yRegister = 5

	ram.Poke(0x0005, 0xFF)
	ram.Poke(0x0006, 0xD0)
	ram.Poke(0xD0FF, 0xAA)
	ram.Poke(0xD104, 0xFA)

	ram.Poke(0xC000, 0xB1) // LDA ($05),Y
	ram.Poke(0xC001, 0x05)
	ram.Poke(0xC002, 0xEA) // NOP

	steps := []addressModeTestData{
		{0xC000, 0xB1, true, 0x00, 0x00, 0x05, "S", 0xC001},
		{0xC001, 0x05, true, 0x00, 0x00, 0x05, "", 0xC002},
		{0x0005, 0xFF, true, 0x00, 0x00, 0x05, "", 0xC002},
		{0x0006, 0xD0, true, 0x00, 0x00, 0x05, "", 0xC002},
		{0xD0FF, 0xAA, true, 0x00, 0x00, 0x05, "", 0xC002},
		{0xD104, 0xFA, true, 0xFA, 0x00, 0x05, "", 0xC002},
		{0xC002, 0xEA, true, 0xFA, 0x00, 0x05, "S", 0xC003},
	}

	runTest(cpu, ram, steps, t)
}

func TestIndirectIndexedYWriteAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xFA
	cpu.yRegister = 5

	ram.Poke(0x0005, 0x00)
	ram.Poke(0x0006, 0xD0)

	ram.Poke(0xC000, 0x91) // STA ($05),Y
	ram.Poke(0xC001, 0x05)
	ram.Poke(0xC002, 0xEA) // NOP

	steps := []addressModeTestData{
		{0xC000, 0x91, true, 0xFA, 0x00, 0x05, "S", 0xC001},
		{0xC001, 0x05, true, 0xFA, 0x00, 0x05, "", 0xC002},
		{0x0005, 0x00, true, 0xFA, 0x00, 0x05, "", 0xC002},
		{0x0006, 0xD0, true, 0xFA, 0x00, 0x05, "", 0xC002},
		{0xD005, 0xFA, false, 0xFA, 0x00, 0x05, "", 0xC002},
		{0xC002, 0xEA, true, 0xFA, 0x00, 0x05, "S", 0xC003},
	}

	runTest(cpu, ram, steps, t)
}

func TestIndirectIndexedYWritePageBoundaryAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xFA
	cpu.yRegister = 5

	ram.Poke(0x0005, 0xFF)
	ram.Poke(0x0006, 0xD0)
	ram.Poke(0xD0FF, 0xAA)

	ram.Poke(0xC000, 0x91) // LDA ($05),Y
	ram.Poke(0xC001, 0x05)
	ram.Poke(0xC002, 0xEA) // NOP

	steps := []addressModeTestData{
		{0xC000, 0x91, true, 0xFA, 0x00, 0x05, "S", 0xC001},
		{0xC001, 0x05, true, 0xFA, 0x00, 0x05, "", 0xC002},
		{0x0005, 0xFF, true, 0xFA, 0x00, 0x05, "", 0xC002},
		{0x0006, 0xD0, true, 0xFA, 0x00, 0x05, "", 0xC002},
		{0xD0FF, 0xAA, true, 0xFA, 0x00, 0x05, "", 0xC002},
		{0xD104, 0xFA, false, 0xFA, 0x00, 0x05, "", 0xC002},
		{0xC002, 0xEA, true, 0xFA, 0x00, 0x05, "S", 0xC003},
	}

	runTest(cpu, ram, steps, t)
}

/***********************
* Indirect
************************/

func TestIndirectAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0x6C) // JMP ($C100)
	ram.Poke(0xC001, 0x00)
	ram.Poke(0xC002, 0xC1)
	ram.Poke(0xC003, 0xEA) // NOP

	ram.Poke(0xC100, 0x00)
	ram.Poke(0xC101, 0xD0)

	ram.Poke(0xD000, 0xEA)

	steps := []addressModeTestData{
		{0xC000, 0x6C, true, 0x00, 0x00, 0x00, "S", 0xC001},
		{0xC001, 0x00, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0xC002, 0xC1, true, 0x00, 0x00, 0x00, "", 0xC003},
		{0xC100, 0x00, true, 0x00, 0x00, 0x00, "", 0xC003},
		{0xC101, 0xD0, true, 0x00, 0x00, 0x00, "", 0xC003},
		{0xD000, 0xEA, true, 0x00, 0x00, 0x00, "", 0xD000},
		{0xD000, 0xEA, true, 0x00, 0x00, 0x00, "S", 0xD001},
	}

	runTest(cpu, ram, steps, t)
}

// A bug that is present in all NMOS variants of the 6502 involves the jump instruction when using indirect addressing.
// In this addressing mode, the target address of the JMP instruction is fetched from memory, the jump vector, rather than
// being an operand to the JMP instruction. For example, JMP ($1234) would fetch the value in memory locations $1234
// (least significant byte) and $1235 (most significant byte) and load those values into the program counter, which would then
// cause the processor to continue execution at the address stored in the vector.

// The bug appears when the vector address ends in $FF, which is the boundary of a memory page. In this case, JMP will fetch the
// most significant byte of the target address from $00 of the original page rather than $00 of the new page. Hence JMP ($12FF)
// would get the least significant byte of the target address at $12FF and the most significant byte of the target address from $1200
// rather than $1300. The 65C02 corrected this issue
func TestIndirectAddressMode65C02BugFix(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0x6C) // JMP ($12FF)
	ram.Poke(0xC001, 0xFF)
	ram.Poke(0xC002, 0x12)
	ram.Poke(0xC003, 0xEA) // NOP

	ram.Poke(0x12FF, 0x00)
	ram.Poke(0x1300, 0xD0)

	ram.Poke(0xD000, 0xEA)

	steps := []addressModeTestData{
		{0xC000, 0x6C, true, 0x00, 0x00, 0x00, "S", 0xC001},
		{0xC001, 0xFF, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0xC002, 0x12, true, 0x00, 0x00, 0x00, "", 0xC003},
		{0x12FF, 0x00, true, 0x00, 0x00, 0x00, "", 0xC003},
		{0x1300, 0xD0, true, 0x00, 0x00, 0x00, "", 0xC003},
		{0xD000, 0xEA, true, 0x00, 0x00, 0x00, "", 0xD000},
		{0xD000, 0xEA, true, 0x00, 0x00, 0x00, "S", 0xD001},
	}

	runTest(cpu, ram, steps, t)
}

/***********************
* Zero Page Indirect
************************/

func TestZeroPageIndirectAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0x00C0, 0x00)
	ram.Poke(0x00C1, 0xD0)

	ram.Poke(0xC000, 0xB2) // LDA ($C0)
	ram.Poke(0xC001, 0xC0)
	ram.Poke(0xC002, 0xEA) // NOP

	ram.Poke(0xD000, 0xFA)

	steps := []addressModeTestData{
		{0xC000, 0xB2, true, 0x00, 0x00, 0x00, "S", 0xC001},
		{0xC001, 0xC0, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0x00C0, 0x00, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0x00C1, 0xD0, true, 0x00, 0x00, 0x00, "", 0xC002},
		{0xD000, 0xFA, true, 0xFA, 0x00, 0x00, "", 0xC002},
		{0xC002, 0xEA, true, 0xFA, 0x00, 0x00, "S", 0xC003},
	}

	runTest(cpu, ram, steps, t)
}

func TestZeroPageIndirectWriteAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xAA

	ram.Poke(0x00C0, 0x00)
	ram.Poke(0x00C1, 0xD0)

	ram.Poke(0xC000, 0x92) // STA ($C0)
	ram.Poke(0xC001, 0xC0)
	ram.Poke(0xC002, 0xEA) // NOP

	steps := []addressModeTestData{
		{0xC000, 0x92, true, 0xAA, 0x00, 0x00, "S", 0xC001},
		{0xC001, 0xC0, true, 0xAA, 0x00, 0x00, "", 0xC002},
		{0x00C0, 0x00, true, 0xAA, 0x00, 0x00, "", 0xC002},
		{0x00C1, 0xD0, true, 0xAA, 0x00, 0x00, "", 0xC002},
		{0xD000, 0xAA, false, 0xAA, 0x00, 0x00, "", 0xC002},
		{0xC002, 0xEA, true, 0xAA, 0x00, 0x00, "S", 0xC003},
	}

	runTest(cpu, ram, steps, t)
}

/****************************
* Absolute Indexed Indirect X
*****************************/

func TestAbsoluteIndexedIndirectXAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.xRegister = 5

	ram.Poke(0xC105, 0x00)
	ram.Poke(0xC106, 0xD0)
	ram.Poke(0xD000, 0xEA)

	ram.Poke(0xC000, 0x7C) // LDA ($C100,X)
	ram.Poke(0xC001, 0x00)
	ram.Poke(0xC002, 0xC1)
	ram.Poke(0xC003, 0xEA) // NOP

	steps := []addressModeTestData{
		{0xC000, 0x7C, true, 0x00, 0x05, 0x00, "S", 0xC001},
		{0xC001, 0x00, true, 0x00, 0x05, 0x00, "", 0xC002},
		{0xC002, 0xC1, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xC105, 0x00, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xC106, 0xD0, true, 0x00, 0x05, 0x00, "", 0xC003},
		{0xD000, 0xEA, true, 0x00, 0x05, 0x00, "", 0xD000},
		{0xD000, 0xEA, true, 0x00, 0x05, 0x00, "S", 0xD001},
	}

	runTest(cpu, ram, steps, t)
}

/****************************
* Stack
*****************************/

func TestPushStackAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xFF

	ram.Poke(0xC000, 0x48) // PHA
	ram.Poke(0xC001, 0xEA) // NOP

	steps := []addressModeTestData{
		{0xC000, 0x48, true, 0xFF, 0x00, 0x00, "S", 0xC001},
		{0xC001, 0xEA, true, 0xFF, 0x00, 0x00, "", 0xC001},
		{0x01FD, 0xFF, false, 0xFF, 0x00, 0x00, "", 0xC001},
		{0xC001, 0xEA, true, 0xFF, 0x00, 0x00, "S", 0xC002},
	}

	runTest(cpu, ram, steps, t)
}

func TestPullStackAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	cpu.stackPointer = 0xFC

	ram.Poke(0x01FC, 0xFF)
	ram.Poke(0x01FD, 0xFA)

	ram.Poke(0xC000, 0x68) // PLA
	ram.Poke(0xC001, 0xEA) // NOP

	steps := []addressModeTestData{
		{0xC000, 0x68, true, 0x00, 0x00, 0x00, "S", 0xC001},
		{0xC001, 0xEA, true, 0x00, 0x00, 0x00, "", 0xC001},
		{0x01FC, 0xFF, true, 0x00, 0x00, 0x00, "", 0xC001},
		{0x01FD, 0xFA, true, 0xFA, 0x00, 0x00, "", 0xC001},
		{0xC001, 0xEA, true, 0xFA, 0x00, 0x00, "S", 0xC002},
	}

	runTest(cpu, ram, steps, t)
}

func TestBreakAndReturnFromInterruptAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0x00) // BRK
	ram.Poke(0xC001, 0xEA) // NOP
	ram.Poke(0xC002, 0xEA) // NOP

	ram.Poke(0xD000, 0xEA) // NOP
	ram.Poke(0xD001, 0x40) // RTI
	ram.Poke(0xD002, 0xAA)

	ram.Poke(0xFFFE, 0x00)
	ram.Poke(0xFFFF, 0xD0)

	steps := []addressModeTestData{
		{0xC000, 0x00, true, 0x00, 0x00, 0x00, "S", 0xC001}, // 0 - BRK (7 cycles)
		{0xC001, 0xEA, true, 0x00, 0x00, 0x00, "", 0xC002},  // 1
		{0x01FD, 0xC0, false, 0x00, 0x00, 0x00, "", 0xC002}, // 2
		{0x01FC, 0x02, false, 0x00, 0x00, 0x00, "", 0xC002}, // 3
		{0x01FB, 0x34, false, 0x00, 0x00, 0x00, "", 0xC002}, // 4
		{0xFFFE, 0x00, true, 0x00, 0x00, 0x00, "V", 0xC002}, // 5
		{0xFFFF, 0xD0, true, 0x00, 0x00, 0x00, "V", 0xD000}, // 6
		{0xD000, 0xEA, true, 0x00, 0x00, 0x00, "S", 0xD001}, // 7 -- NOP (2 cycles)
		{0xD001, 0x40, true, 0x00, 0x00, 0x00, "", 0xD001},  // 8
		{0xD001, 0x40, true, 0x00, 0x00, 0x00, "S", 0xD002}, // 9 -- RTI (6 cycles)
		{0xD002, 0xAA, true, 0x00, 0x00, 0x00, "", 0xD002},  // 10
		{0x01FA, 0x00, true, 0x00, 0x00, 0x00, "", 0xD002},  // 11
		{0x01FB, 0x34, true, 0x00, 0x00, 0x00, "", 0xD002},  // 12
		{0x01FC, 0x02, true, 0x00, 0x00, 0x00, "", 0xD002},  // 13
		{0x01FD, 0xC0, true, 0x00, 0x00, 0x00, "", 0xC002},  // 14
		{0xC002, 0xEA, true, 0x00, 0x00, 0x00, "S", 0xC003}, // 15 -- NOP
	}

	runTest(cpu, ram, steps, t)
}

func TestJumpAndReturnFromSubroutineAddressMode(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0x20) // JSR $D000
	ram.Poke(0xC001, 0x00)
	ram.Poke(0xC002, 0xD0)
	ram.Poke(0xC003, 0xEA) // NOP

	ram.Poke(0xD000, 0xEA) // NOP
	ram.Poke(0xD001, 0x60) // RTS
	ram.Poke(0xD002, 0xAA)

	steps := []addressModeTestData{
		{0xC000, 0x20, true, 0x00, 0x00, 0x00, "S", 0xC001}, // 0 - JSR (6 cycles)
		{0xC001, 0x00, true, 0x00, 0x00, 0x00, "", 0xC002},  // 1
		{0x01FD, 0x00, true, 0x00, 0x00, 0x00, "", 0xC002},  // 2
		{0x01FD, 0xC0, false, 0x00, 0x00, 0x00, "", 0xC002}, // 3
		{0x01FC, 0x02, false, 0x00, 0x00, 0x00, "", 0xC002}, // 4
		{0xC002, 0xD0, true, 0x00, 0x00, 0x00, "", 0xD000},  // 5
		{0xD000, 0xEA, true, 0x00, 0x00, 0x00, "S", 0xD001}, // 6 - NOP (2 cycles)
		{0xD001, 0x60, true, 0x00, 0x00, 0x00, "", 0xD001},  // 7
		{0xD001, 0x60, true, 0x00, 0x00, 0x00, "S", 0xD002}, // 8 - RTS (6 cycles)
		{0xD002, 0xAA, true, 0x00, 0x00, 0x00, "", 0xD002},  // 9
		{0x01FB, 0x00, true, 0x00, 0x00, 0x00, "", 0xD002},  // 11
		{0x01FC, 0x02, true, 0x00, 0x00, 0x00, "", 0xD002},  // 12
		{0x01FD, 0xC0, true, 0x00, 0x00, 0x00, "", 0xC002},  // 13
		{0xC002, 0xD0, true, 0x00, 0x00, 0x00, "", 0xC003},  // 13
		{0xC003, 0xEA, true, 0x00, 0x00, 0x00, "S", 0xC004}, // 14 -- NOP*/
	}

	runTest(cpu, ram, steps, t)
}

func TestIRQAndReturnFromInterruptAddressMode(t *testing.T) {
	cpu, ram, nmiLine, irqLine, resetLine, readyLine := createComputerWithControlLines()

	cpu.processorStatusRegister.SetFlag(IrqDisableFlagBit, false)

	ram.Poke(0xC000, 0xA9) // LDA #FF
	ram.Poke(0xC001, 0xFF)
	ram.Poke(0xC002, 0xA9) // LDA #AA
	ram.Poke(0xC003, 0xAA)
	ram.Poke(0xC004, 0xEA) // NOP
	ram.Poke(0xC005, 0xEA) // NOP

	ram.Poke(0xD000, 0xEA) // NOP
	ram.Poke(0xD001, 0x40) // RTI
	ram.Poke(0xD002, 0xAA)

	ram.Poke(0xFFFE, 0x00)
	ram.Poke(0xFFFF, 0xD0)

	steps := []addressModeTestDataWithControlLines{
		{addressModeTestData{0xC000, 0xA9, true, 0x00, 0x00, 0x00, "S", 0xC001}, true, false, false, false},  // 0 LDA #FF IRQ Trigger
		{addressModeTestData{0xC001, 0xFF, true, 0xFF, 0x00, 0x00, "", 0xC002}, true, false, false, false},   // 1
		{addressModeTestData{0xC002, 0xA9, true, 0xFF, 0x00, 0x00, "", 0xC002}, true, false, false, false},   // 2 - BRK (7 cycles)
		{addressModeTestData{0xC003, 0xAA, true, 0xFF, 0x00, 0x00, "", 0xC002}, true, false, false, false},   // 3
		{addressModeTestData{0x01FD, 0xC0, false, 0xFF, 0x00, 0x00, "", 0xC002}, true, false, false, false},  // 4
		{addressModeTestData{0x01FC, 0x02, false, 0xFF, 0x00, 0x00, "", 0xC002}, true, false, false, false},  // 5
		{addressModeTestData{0x01FB, 0xA0, false, 0xFF, 0x00, 0x00, "", 0xC002}, true, false, false, false},  // 6
		{addressModeTestData{0xFFFE, 0x00, true, 0xFF, 0x00, 0x00, "V", 0xC002}, true, false, false, false},  // 7
		{addressModeTestData{0xFFFF, 0xD0, true, 0xFF, 0x00, 0x00, "V", 0xD000}, true, false, false, false},  // 8
		{addressModeTestData{0xD000, 0xEA, true, 0xFF, 0x00, 0x00, "S", 0xD001}, true, false, false, false},  // 9 -- NOP (2 cycles) IRQ is trying to trigger but flag is disabled
		{addressModeTestData{0xD001, 0x40, true, 0xFF, 0x00, 0x00, "", 0xD001}, false, false, false, false},  // 10
		{addressModeTestData{0xD001, 0x40, true, 0xFF, 0x00, 0x00, "S", 0xD002}, false, false, false, false}, // 11 -- RTI (6 cycles)
		{addressModeTestData{0xD002, 0xAA, true, 0xFF, 0x00, 0x00, "", 0xD002}, false, false, false, false},  // 12
		{addressModeTestData{0x01FA, 0x00, true, 0xFF, 0x00, 0x00, "", 0xD002}, false, false, false, false},  // 13
		{addressModeTestData{0x01FB, 0xA0, true, 0xFF, 0x00, 0x00, "", 0xD002}, false, false, false, false},  // 14
		{addressModeTestData{0x01FC, 0x02, true, 0xFF, 0x00, 0x00, "", 0xD002}, false, false, false, false},  // 15
		{addressModeTestData{0x01FD, 0xC0, true, 0xFF, 0x00, 0x00, "", 0xC002}, false, false, false, false},  // 16

		{addressModeTestData{0xC002, 0xA9, true, 0xFF, 0x00, 0x00, "S", 0xC003}, true, false, false, false},  // 17 -- LDA #AA // IRQ triggered again as I flag was cleared on status restore
		{addressModeTestData{0xC003, 0xAA, true, 0xAA, 0x00, 0x00, "", 0xC004}, true, false, false, false},   // 18
		{addressModeTestData{0xC004, 0xEA, true, 0xAA, 0x00, 0x00, "", 0xC004}, true, false, false, false},   // 19 - BRK (7 cycles)
		{addressModeTestData{0xC005, 0xEA, true, 0xAA, 0x00, 0x00, "", 0xC004}, true, false, false, false},   // 20
		{addressModeTestData{0x01FD, 0xC0, false, 0xAA, 0x00, 0x00, "", 0xC004}, true, false, false, false},  // 21
		{addressModeTestData{0x01FC, 0x04, false, 0xAA, 0x00, 0x00, "", 0xC004}, true, false, false, false},  // 22
		{addressModeTestData{0x01FB, 0xA0, false, 0xAA, 0x00, 0x00, "", 0xC004}, false, false, false, false}, // 23
		{addressModeTestData{0xFFFE, 0x00, true, 0xAA, 0x00, 0x00, "V", 0xC004}, false, false, false, false}, // 24
		{addressModeTestData{0xFFFF, 0xD0, true, 0xAA, 0x00, 0x00, "V", 0xD000}, false, false, false, false}, // 25
		{addressModeTestData{0xD000, 0xEA, true, 0xAA, 0x00, 0x00, "S", 0xD001}, false, false, false, false}, // 26 -- NOP (2 cycles) IRQ is trying to trigger but flag is disabled
		{addressModeTestData{0xD001, 0x40, true, 0xAA, 0x00, 0x00, "", 0xD001}, false, false, false, false},  // 27

	}

	runTestWithInterrupts(cpu, ram, irqLine, nmiLine, resetLine, readyLine, steps, t)
}

func TestNMIAndReturnFromInterruptAddressMode(t *testing.T) {
	cpu, ram, nmiLine, irqLine, resetLine, readyLine := createComputerWithControlLines()

	// Interrupt disable is on by default

	ram.Poke(0xC000, 0xA9) // LDA #FF
	ram.Poke(0xC001, 0xFF)
	ram.Poke(0xC002, 0xA9) // LDA #AA
	ram.Poke(0xC003, 0xAA)
	ram.Poke(0xC004, 0xEA) // NOP
	ram.Poke(0xC005, 0xEA) // NOP

	ram.Poke(0xD000, 0xEA) // NOP
	ram.Poke(0xD001, 0x40) // RTI
	ram.Poke(0xD002, 0xAA)

	ram.Poke(0xFFFA, 0x00)
	ram.Poke(0xFFFB, 0xD0)

	steps := []addressModeTestDataWithControlLines{
		{addressModeTestData{0xC000, 0xA9, true, 0x00, 0x00, 0x00, "S", 0xC001}, false, false, false, false}, // 0 LDA #FF
		{addressModeTestData{0xC001, 0xFF, true, 0xFF, 0x00, 0x00, "", 0xC002}, false, true, false, false},   // 1 // NMI Trigger even with I flag set
		{addressModeTestData{0xC002, 0xA9, true, 0xFF, 0x00, 0x00, "", 0xC002}, false, true, false, false},   // 2 - BRK (7 cycles)
		{addressModeTestData{0xC003, 0xAA, true, 0xFF, 0x00, 0x00, "", 0xC002}, false, true, false, false},   // 3
		{addressModeTestData{0x01FD, 0xC0, false, 0xFF, 0x00, 0x00, "", 0xC002}, false, true, false, false},  // 4
		{addressModeTestData{0x01FC, 0x02, false, 0xFF, 0x00, 0x00, "", 0xC002}, false, true, false, false},  // 5
		{addressModeTestData{0x01FB, 0xA4, false, 0xFF, 0x00, 0x00, "", 0xC002}, false, true, false, false},  // 6
		{addressModeTestData{0xFFFA, 0x00, true, 0xFF, 0x00, 0x00, "V", 0xC002}, false, true, false, false},  // 7
		{addressModeTestData{0xFFFB, 0xD0, true, 0xFF, 0x00, 0x00, "V", 0xD000}, false, true, false, false},  // 8
		{addressModeTestData{0xD000, 0xEA, true, 0xFF, 0x00, 0x00, "S", 0xD001}, false, true, false, false},  // 9 -- NOP (2 cycles) NMI is edge triggered, if signal stays low should not re-trigger
		{addressModeTestData{0xD001, 0x40, true, 0xFF, 0x00, 0x00, "", 0xD001}, false, true, false, false},   // 10
		{addressModeTestData{0xD001, 0x40, true, 0xFF, 0x00, 0x00, "S", 0xD002}, false, true, false, false},  // 11 -- RTI (6 cycles)
		{addressModeTestData{0xD002, 0xAA, true, 0xFF, 0x00, 0x00, "", 0xD002}, false, true, false, false},   // 12
		{addressModeTestData{0x01FA, 0x00, true, 0xFF, 0x00, 0x00, "", 0xD002}, false, true, false, false},   // 13
		{addressModeTestData{0x01FB, 0xA4, true, 0xFF, 0x00, 0x00, "", 0xD002}, false, true, false, false},   // 14
		{addressModeTestData{0x01FC, 0x02, true, 0xFF, 0x00, 0x00, "", 0xD002}, false, true, false, false},   // 15
		{addressModeTestData{0x01FD, 0xC0, true, 0xFF, 0x00, 0x00, "", 0xC002}, false, true, false, false},   // 16

		{addressModeTestData{0xC002, 0xA9, true, 0xFF, 0x00, 0x00, "S", 0xC003}, false, false, false, false}, // 17 -- LDA #AA
		{addressModeTestData{0xC003, 0xAA, true, 0xAA, 0x00, 0x00, "", 0xC004}, false, true, false, false},   // 18 -- NMI triggered again as signal transitioned to high and low again
		{addressModeTestData{0xC004, 0xEA, true, 0xAA, 0x00, 0x00, "", 0xC004}, false, false, false, false},  // 19 - BRK (7 cycles)
		{addressModeTestData{0xC005, 0xEA, true, 0xAA, 0x00, 0x00, "", 0xC004}, false, false, false, false},  // 20
		{addressModeTestData{0x01FD, 0xC0, false, 0xAA, 0x00, 0x00, "", 0xC004}, false, false, false, false}, // 21
		{addressModeTestData{0x01FC, 0x04, false, 0xAA, 0x00, 0x00, "", 0xC004}, false, false, false, false}, // 22
		{addressModeTestData{0x01FB, 0xA4, false, 0xAA, 0x00, 0x00, "", 0xC004}, false, false, false, false}, // 23
		{addressModeTestData{0xFFFA, 0x00, true, 0xAA, 0x00, 0x00, "V", 0xC004}, false, false, false, false}, // 24
		{addressModeTestData{0xFFFB, 0xD0, true, 0xAA, 0x00, 0x00, "V", 0xD000}, false, false, false, false}, // 25
		{addressModeTestData{0xD000, 0xEA, true, 0xAA, 0x00, 0x00, "S", 0xD001}, false, false, false, false}, // 26 -- NOP (2 cycles) IRQ is trying to trigger but flag is disabled
		{addressModeTestData{0xD001, 0x40, true, 0xAA, 0x00, 0x00, "", 0xD001}, false, false, false, false},  // 27

	}

	runTestWithInterrupts(cpu, ram, irqLine, nmiLine, resetLine, readyLine, steps, t)
}

func TestResetAddressMode(t *testing.T) {
	cpu, ram, nmiLine, irqLine, resetLine, readyLine := createComputerWithControlLines()

	ram.Poke(0xC000, 0xA9) // LDA #FF
	ram.Poke(0xC001, 0xFF)
	ram.Poke(0xC002, 0xA9) // LDA #AA
	ram.Poke(0xC003, 0xAA)
	ram.Poke(0xC004, 0xEA) // NOP
	ram.Poke(0xC005, 0xEA) // NOP

	ram.Poke(0xFFFC, 0x04)
	ram.Poke(0xFFFD, 0xC0)

	steps := []addressModeTestDataWithControlLines{
		{addressModeTestData{0xC000, 0xA9, true, 0x00, 0x00, 0x00, "S", 0xC001}, false, false, true, false},  // 0 LDA #FF -- No interrupt as it must be held at least 2 cycles
		{addressModeTestData{0xC001, 0xFF, true, 0xFF, 0x00, 0x00, "", 0xC002}, false, false, false, false},  // 1
		{addressModeTestData{0xC002, 0xA9, true, 0xFF, 0x00, 0x00, "S", 0xC003}, false, false, true, false},  // 2 LDA #AA
		{addressModeTestData{0xC003, 0xAA, true, 0xAA, 0x00, 0x00, "", 0xC004}, false, false, true, false},   // 3 Reset triggered
		{addressModeTestData{0xC004, 0xEA, true, 0xAA, 0x00, 0x00, "", 0xC004}, false, false, false, false},  // 4
		{addressModeTestData{0xC004, 0xEA, true, 0xAA, 0x00, 0x00, "", 0xC005}, false, false, false, false},  // 5
		{addressModeTestData{0xC005, 0xEA, true, 0xAA, 0x00, 0x00, "", 0xC006}, false, false, false, false},  // 6
		{addressModeTestData{0xC006, 0x00, true, 0xAA, 0x00, 0x00, "", 0xC007}, false, false, false, false},  // 7
		{addressModeTestData{0xC007, 0x00, true, 0xAA, 0x00, 0x00, "", 0xC008}, false, false, false, false},  // 8
		{addressModeTestData{0xC008, 0x00, true, 0xAA, 0x00, 0x00, "", 0xC009}, false, false, false, false},  // 9
		{addressModeTestData{0xC009, 0x00, true, 0xAA, 0x00, 0x00, "", 0xC00A}, false, false, false, false},  // 10
		{addressModeTestData{0xFFFC, 0x04, true, 0xAA, 0x00, 0x00, "V", 0xC00A}, false, false, false, false}, // 11 -- Read Vector
		{addressModeTestData{0xFFFD, 0xC0, true, 0xAA, 0x00, 0x00, "V", 0xC004}, false, false, false, false}, // 12
		{addressModeTestData{0xC004, 0xEA, true, 0xAA, 0x00, 0x00, "S", 0xC005}, false, false, false, false}, // 13
		{addressModeTestData{0xC005, 0xEA, true, 0xAA, 0x00, 0x00, "", 0xC005}, false, false, false, false},  // 14
	}

	runTestWithInterrupts(cpu, ram, nmiLine, irqLine, resetLine, readyLine, steps, t)
}

func TestPausingProcessorByDroppingReadyLine(t *testing.T) {
	cpu, ram, nmiLine, irqLine, resetLine, readyLine := createComputerWithControlLines()

	cpu.stackPointer = 0xFC

	ram.Poke(0xC000, 0xA9) // LDA #FF
	ram.Poke(0xC001, 0xFF)
	ram.Poke(0xC002, 0xA9) // LDA #AA
	ram.Poke(0xC003, 0xAA)
	ram.Poke(0xC004, 0xEA) // NOP
	ram.Poke(0xC005, 0xEA) // NOP

	steps := []addressModeTestDataWithControlLines{
		{addressModeTestData{0xC000, 0xA9, true, 0x00, 0x00, 0x00, "S", 0xC001}, false, false, false, false},
		{addressModeTestData{0xC001, 0xFF, true, 0xFF, 0x00, 0x00, "", 0xC002}, false, false, false, false},
		{addressModeTestData{0xC002, 0xA9, true, 0xFF, 0x00, 0x00, "S", 0xC003}, false, false, false, false},
		{addressModeTestData{0xC002, 0xA9, true, 0xFF, 0x00, 0x00, "Sr", 0xC003}, false, false, false, true},
		{addressModeTestData{0xC002, 0xA9, true, 0xFF, 0x00, 0x00, "Sr", 0xC003}, false, false, false, true},
		{addressModeTestData{0xC002, 0xA9, true, 0xFF, 0x00, 0x00, "Sr", 0xC003}, false, false, false, true},
		{addressModeTestData{0xC002, 0xA9, true, 0xFF, 0x00, 0x00, "Sr", 0xC003}, false, false, false, true},
		{addressModeTestData{0xC003, 0xAA, true, 0xAA, 0x00, 0x00, "", 0xC004}, false, false, false, false},
		{addressModeTestData{0xC004, 0xEA, true, 0xAA, 0x00, 0x00, "S", 0xC005}, false, false, false, false},
		{addressModeTestData{0xC005, 0xEA, true, 0xAA, 0x00, 0x00, "", 0xC005}, false, false, false, false},
	}

	runTestWithInterrupts(cpu, ram, nmiLine, irqLine, resetLine, readyLine, steps, t)
}

func TestWAIInstructionPausingProcessor(t *testing.T) {
	cpu, ram, nmiLine, irqLine, resetLine, readyLine := createComputerWithControlLines()

	ram.Poke(0xC000, 0xA9) // LDA #FF
	ram.Poke(0xC001, 0xFF)
	ram.Poke(0xC002, 0xCB) // WAI
	ram.Poke(0xC003, 0xEA) // NOP
	ram.Poke(0xC004, 0xEA) // NOP

	steps := []addressModeTestDataWithControlLines{
		{addressModeTestData{0xC000, 0xA9, true, 0x00, 0x00, 0x00, "S", 0xC001}, false, false, false, false},
		{addressModeTestData{0xC001, 0xFF, true, 0xFF, 0x00, 0x00, "", 0xC002}, false, false, false, false},
		{addressModeTestData{0xC002, 0xCB, true, 0xFF, 0x00, 0x00, "S", 0xC003}, false, false, false, false},
		{addressModeTestData{0xC003, 0xEA, true, 0xFF, 0x00, 0x00, "", 0xC003}, false, false, false, false},
		{addressModeTestData{0xC003, 0xEA, true, 0xFF, 0x00, 0x00, "r", 0xC003}, false, false, false, false},
		{addressModeTestData{0xC003, 0xEA, true, 0xFF, 0x00, 0x00, "r", 0xC003}, false, false, false, false},
		{addressModeTestData{0xC003, 0xEA, true, 0xFF, 0x00, 0x00, "r", 0xC003}, false, false, false, false},
		{addressModeTestData{0xC003, 0xEA, true, 0xFF, 0x00, 0x00, "S", 0xC004}, true, false, false, false},
		{addressModeTestData{0xC004, 0xEA, true, 0xFF, 0x00, 0x00, "", 0xC004}, false, false, false, false},
	}

	runTestWithInterrupts(cpu, ram, nmiLine, irqLine, resetLine, readyLine, steps, t)
}

func TestSTPInstructionStoppingProcessor(t *testing.T) {
	cpu, ram, nmiLine, irqLine, resetLine, readyLine := createComputerWithControlLines()

	cpu.processorStatusRegister.SetFlag(IrqDisableFlagBit, false)

	ram.Poke(0xC000, 0xA9) // LDA #FF
	ram.Poke(0xC001, 0xFF)
	ram.Poke(0xC002, 0xDB) // STP
	ram.Poke(0xC003, 0xEA) // NOP
	ram.Poke(0xC004, 0xEA) // NOP

	// Processor will not be awake by IRQ nor MMI after STP only reset

	steps := []addressModeTestDataWithControlLines{
		{addressModeTestData{0xC000, 0xA9, true, 0x00, 0x00, 0x00, "S", 0xC001}, false, false, false, false},
		{addressModeTestData{0xC001, 0xFF, true, 0xFF, 0x00, 0x00, "", 0xC002}, false, false, false, false},
		{addressModeTestData{0xC002, 0xDB, true, 0xFF, 0x00, 0x00, "S", 0xC003}, false, false, false, false},
		{addressModeTestData{0xC003, 0xEA, true, 0xFF, 0x00, 0x00, "", 0xC003}, false, false, false, false},
		{addressModeTestData{0xC003, 0xEA, true, 0xFF, 0x00, 0x00, "r", 0xC003}, false, false, false, false},
		{addressModeTestData{0xC003, 0xEA, true, 0xFF, 0x00, 0x00, "r", 0xC003}, true, false, false, false}, // IRQ doesn't affect the processor
		{addressModeTestData{0xC003, 0xEA, true, 0xFF, 0x00, 0x00, "r", 0xC003}, false, true, false, false}, // NMI doesn't affect the processor
		{addressModeTestData{0xC003, 0xEA, true, 0xFF, 0x00, 0x00, "r", 0xC003}, false, false, false, false},
		{addressModeTestData{0xC003, 0xEA, true, 0xFF, 0x00, 0x00, "r", 0xC003}, false, false, true, false}, // Reset is held for 2 cycles
		{addressModeTestData{0xC003, 0xEA, true, 0xFF, 0x00, 0x00, "r", 0xC003}, false, false, true, false},
		{addressModeTestData{0xC003, 0xEA, true, 0xFF, 0x00, 0x00, "", 0xC003}, false, false, false, false},
		{addressModeTestData{0xC003, 0xEA, true, 0xFF, 0x00, 0x00, "", 0xC004}, false, false, false, false},
		{addressModeTestData{0xC004, 0xEA, true, 0xFF, 0x00, 0x00, "", 0xC005}, false, false, false, false},
	}

	runTestWithInterrupts(cpu, ram, nmiLine, irqLine, resetLine, readyLine, steps, t)
}

func TestSetOverflowFlagForcesVStatus(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0xEA) // NOP
	ram.Poke(0xC001, 0xEA) // NOP

	steps := []addressModeTestData{
		{0xC000, 0xEA, true, 0x00, 0x00, 0x00, "S", 0xC001},
		{0xC001, 0xEA, true, 0x00, 0x00, 0x00, "", 0xC001},
	}

	runTest(cpu, ram, steps, t)

	sobLine := buses.CreateStandaloneLine(false)
	cpu.SetOverflow().Connect(sobLine)

	steps2 := []addressModeTestData{
		{0xC001, 0xEA, true, 0x00, 0x00, 0x00, "S", 0xC002},
		{0xC002, 0x00, true, 0x00, 0x00, 0x00, "", 0xC002},
	}

	runTest(cpu, ram, steps2, t)

	if !cpu.processorStatusRegister.Flag(OverflowFlagBit) {
		t.Errorf("V flag expected to be set when SOB line is held low")
	}
}
