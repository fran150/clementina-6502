package tests

import (
	"fmt"
	"testing"

	"github.com/fran150/clementina6502/buses"
	"github.com/fran150/clementina6502/cpu"
	"github.com/fran150/clementina6502/memory"
)

/******************************************************************************************************
* Configuration
*******************************************************************************************************/

// Number of times a jump or branch instruction is repeated before detecting error condition.
const maxRepeats int = 10

// Expected procgram counter at the end of execution
// https://github.com/Klaus2m5/6502_65C02_functional_tests/blob/master/bin_files/6502_functional_test.lst#L13377
const successPcValue uint16 = 0x346B

// Max number of cycles allowed to copmlete execution
const maxAllowedCycles uint64 = 100_000_000

/******************************************************************************************************
* Support functions
*******************************************************************************************************/

// Creates a CPU connected to a RAM memory
func CreateComputer() (*cpu.Cpu65C02S, *memory.Ram) {
	addressBus := buses.CreateBus[uint16]()
	dataBus := buses.CreateBus[uint8]()

	alwaysHighLine := buses.CreateStandaloneLine(true)
	alwaysLowLine := buses.CreateStandaloneLine(false)

	writeEnableLine := buses.CreateStandaloneLine(true)

	memoryLockLine := buses.CreateStandaloneLine(false)
	syncLine := buses.CreateStandaloneLine(false)
	vectorPullLine := buses.CreateStandaloneLine(false)

	ram := memory.CreateRam()
	ram.Connect(addressBus, dataBus, writeEnableLine, alwaysLowLine, alwaysLowLine)

	processor := cpu.CreateCPU()
	processor.ConnectAddressBus(addressBus)
	processor.ConnectDataBus(dataBus)

	processor.BusEnable().Connect(alwaysHighLine)
	processor.ReadWrite().Connect(writeEnableLine)
	processor.MemoryLock().Connect(memoryLockLine)
	processor.Sync().Connect(syncLine)
	processor.Ready().Connect(alwaysHighLine)
	processor.VectorPull().Connect(vectorPullLine)
	processor.SetOverflow().Connect(alwaysHighLine)
	processor.Reset().Connect(alwaysHighLine)

	processor.InterruptRequest().Connect(alwaysHighLine)
	processor.NonMaskableInterrupt().Connect(alwaysHighLine)

	return processor, ram
}

// In case of error the test code does not fail but is usually trapped in a repeating loop
// Detecting when the code is "stuck" is necessary to determine success / failure of the tests

// This variable will be used to track repeating opcodes
var previousOpCode cpu.OpCode

// This variable will count how many times the opcode was repeated
var repeats int = 0

// Error trap opcodes are usually JMP * instructions or branch instructions BNE, BCS, etc (address mode relative)
func isTrapOpCode(processor *cpu.Cpu65C02S) bool {
	return processor.GetCurrentInstruction().Mnemonic() == cpu.JMP || processor.GetCurrentAddressMode().Name() == cpu.AddressModeRelative
}

// Counts how many times a "trap" opcode (JMP or branches) is being repeated.
func verifyAndCountRepeats(processor *cpu.Cpu65C02S) bool {
	if previousOpCode == processor.GetCurrentInstruction().OpCode() && isTrapOpCode(processor) {
		repeats++
	} else {
		previousOpCode = processor.GetCurrentInstruction().OpCode()
		repeats = 0
	}

	// If we reach the maximum number of repeats we can consider that the code is "trapped"
	// This signals either an error condition or the end of the tests
	return repeats >= maxRepeats
}

// Validates the status of the processor when the test finish and fails the test if required.
func showFinishCondition(processor *cpu.Cpu65C02S, cycles uint64, t *testing.T) {
	// If processor is trapped in SUCCESS_PC_VALUE, this means that the tests were completed successfullly
	// Otherwise this is an error condition and must fail the tests
	if processor.GetProgramCounter() != successPcValue {
		t.Errorf("Possible ERROR trap found with PC in %04X", processor.GetProgramCounter())
	}

	// If the execution was cancelled due to exceeding the number of allowed cycles then fail the tests
	if cycles >= maxAllowedCycles {
		t.Errorf("Maximum limit of %v cycles was reached, typical execution is 96,241,272", cycles)
	}

	// Show number of elapsed cycles
	fmt.Printf("Functional Tests execution completed in %v cycles\n", cycles)
}

/******************************************************************************************************
* TESTS
*******************************************************************************************************/

// This function runs the 6502/65C02 functional tests created by Klaus2m5.
// See https://github.com/Klaus2m5/6502_65C02_functional_tests
// In particular we use the compiled functional test provided in the "bin_files" folder as is.
func TestProcessorFunctional(t *testing.T) {
	processor, ram := CreateComputer()

	// Loads Klaus2m5 functional tests. See repository mentioned above for reference
	ram.Load("../tests/6502_functional_test.bin")

	// Functional Tests starts at $0400
	processor.ForceProgramCounter(0x0400)

	// Initialize opcode repeats counter
	previousOpCode = cpu.OpCode(0)
	repeats = 0

	// Will count the cycles required to complete execution
	var cycles uint64 = 0

	for cycles < maxAllowedCycles {

		// Exeutes the CPU cycles
		processor.Tick(cycles)
		ram.Tick(cycles)
		processor.PostTick(cycles)

		// Verfies and count repeated "trap" opcodes. If this functions returns true
		// it means that the code is trapped and the tests are either in error or finished
		if verifyAndCountRepeats(processor) {
			break
		}

		// Count number of cycles
		cycles++
	}

	// Mark the test as failed or success depending on the exit conditions
	showFinishCondition(processor, cycles, t)
}
