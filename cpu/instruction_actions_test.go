package cpu

import (
	"strings"
	"testing"

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

func runInstructionTest(cpu *Cpu65C02S, ram *memory.Ram, cycles uint64) {
	for i := range cycles {
		cpu.Tick(i)
		ram.Tick(i)

		cpu.PostTick(i)
	}
}

func evaluateAccumulator(t *testing.T, cpu *Cpu65C02S, expected uint8) {
	if cpu.accumulatorRegister != expected {
		t.Errorf("Current value of accumulator (%02X) doesnt match the expected value of (%02X)", cpu.accumulatorRegister, 0x0A)
	}
}

// Evaluates if the specified flag is set or not.
// The flag string specifies the value that it will be searched in the testData.flags
// field. If this letter is found then if it's uppercase it will validate if the flag is set
// if it's lowercase it will validate if it's unset.
func evaluateFlag(cycle int, cpu *Cpu65C02S, t *testing.T, flagString string) {
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

func TestActionADC(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0x0010, 0x02)

	ram.Poke(0xC000, 0x69) // ADC #$0A
	ram.Poke(0xC001, 0x0A)
	ram.Poke(0xC002, 0x65) // ADC $10
	ram.Poke(0xC003, 0x10)
	ram.Poke(0xC002, 0x65) // ADC $10
	ram.Poke(0xC003, 0x10)

	runInstructionTest(cpu, ram, 2)
	evaluateAccumulator(t, cpu, 0x0A)
	runInstructionTest(cpu, ram, 3)
	evaluateAccumulator(t, cpu, 0x0C)
}
