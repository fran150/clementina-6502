package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/fran150/clementina6502/pkg/components/buses"
	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/fran150/clementina6502/pkg/components/cpu"
	"github.com/fran150/clementina6502/pkg/components/memory"
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
	addressBus := buses.Create16BitStandaloneBus()
	dataBus := buses.Create8BitStandaloneBus()

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

	processor := cpu.CreateCPU()
	processor.AddressBus().Connect(addressBus)
	processor.DataBus().Connect(dataBus)

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
func showFinishCondition(processor *cpu.Cpu65C02S, context common.StepContext, b *testing.B, elapsed time.Duration) {
	// If processor is trapped in SUCCESS_PC_VALUE, this means that the tests were completed successfullly
	// Otherwise this is an error condition and must fail the tests
	if processor.GetProgramCounter() != successPcValue {
		b.Errorf("Possible ERROR trap found with PC in %04X", processor.GetProgramCounter())
	}

	// If the execution was cancelled due to exceeding the number of allowed cycles then fail the tests
	if context.Cycle >= maxAllowedCycles {
		b.Errorf("Maximum limit of %v cycles was reached, typical execution is 96,241,272", context.Cycle)
	}

	// Show number of elapsed cycles
	fmt.Printf("Functional Tests execution completed in %v cycles\n", context.Cycle)

	total := (float64(context.Cycle) / elapsed.Seconds()) / 1_000_000

	fmt.Printf("Processor ran at %v MHZ\n", total)
}

/******************************************************************************************************
* TESTS
*******************************************************************************************************/

// This function runs the 6502/65C02 functional tests created by Klaus2m5.
// See https://github.com/Klaus2m5/6502_65C02_functional_tests
// In particular we use the compiled functional test provided in the "bin_files" folder as is.
func BenchmarkProcessor(b *testing.B) {
	processor, ram := CreateComputer()

	// Loads Klaus2m5 functional tests. See repository mentioned above for reference
	ram.Load("../tests/6502_functional_test.bin")

	// Functional Tests starts at $0400
	processor.ForceProgramCounter(0x0400)

	// Initialize opcode repeats counter
	previousOpCode = cpu.OpCode(0)
	repeats = 0

	// Will count the cycles required to complete execution
	context := common.CreateStepContext()
	var start = time.Now()

	for i := 0; i < b.N; i++ {
		for context.Cycle < maxAllowedCycles {
			// Exeutes the CPU cycles
			processor.Tick(context)
			ram.Tick(context)
			processor.PostTick(context)

			// Verfies and count repeated "trap" opcodes. If this functions returns true
			// it means that the code is trapped and the tests are either in error or finished
			if verifyAndCountRepeats(processor) {
				break
			}

			// Count number of cycles
			context.Next()
		}
	}

	// Measure the elapsed time
	elapsed := time.Since(start)

	// Mark the test as failed or success depending on the exit conditions
	showFinishCondition(processor, context, b, elapsed)
}
