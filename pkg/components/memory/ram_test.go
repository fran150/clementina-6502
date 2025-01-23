package memory

import (
	"testing"

	"github.com/fran150/clementina6502/pkg/components/buses"
	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/stretchr/testify/assert"
)

const test_directory string = "../../../tests/"

// Circuit to test the RAM
type testCircuit struct {
	addressBus   buses.Bus[uint16]
	dataBus      buses.Bus[uint8]
	writeEnable  *buses.StandaloneLine
	outputEnable *buses.StandaloneLine
}

// Creates the test circuit and the ram
func createTestCircuit(size int) (*Ram, *testCircuit) {
	ram := CreateRam(size)
	circuit := &testCircuit{
		addressBus:   buses.Create16BitStandaloneBus(),
		dataBus:      buses.Create8BitStandaloneBus(),
		writeEnable:  buses.CreateStandaloneLine(false),
		outputEnable: buses.CreateStandaloneLine(false),
	}

	return ram, circuit
}

// Connects the circuit to the RAM memory
func (circuit *testCircuit) Wire(ram *Ram) {
	ram.AddressBus().Connect(circuit.addressBus)
	ram.DataBus().Connect(circuit.dataBus)
	ram.WriteEnable().Connect(circuit.writeEnable)
	ram.OutputEnable().Connect(circuit.outputEnable)

	ram.ChipSelect().Connect(circuit.addressBus.GetBusLine(15))
}

/************************************************************************************
* Test RAM R/W
*************************************************************************************/

// Writes and reads a value from the memory
func TestRamReadWrite(t *testing.T) {
	context := common.CreateStepContext()

	ram, circuit := createTestCircuit(RAM_SIZE_64K)
	circuit.Wire(ram)

	// Write Cycle
	circuit.writeEnable.Set(false)
	circuit.addressBus.Write(0x7FFA)
	circuit.dataBus.Write(0xFA)
	ram.Tick(context)
	context.Next()

	peek := ram.Peek(0x7FFA)

	if peek != 0xFA {
		t.Errorf("Error, expected to have 0xFA in memory 0x7FFA after write cycle but got %x", peek)
	}

	// Clear databus
	circuit.dataBus.Write(0x00)

	// Read Cycle
	circuit.writeEnable.Set(true)
	circuit.addressBus.Write(0x7FFA)
	ram.Tick(context)
	context.Next()

	value := circuit.dataBus.Read()

	if value != 0xFA {
		t.Errorf("Error, expected to read 0xFA in databus after read cycle but got %x", value)
	}

	// Clear databus
	circuit.dataBus.Write(0x00)
}

/************************************************************************************
* Test of method to read .bin files
*************************************************************************************/

// Loads a bin file into the memory
func TestLoadBinFile(t *testing.T) {
	ram, circuit := createTestCircuit(RAM_SIZE_64K)
	circuit.Wire(ram)

	err := ram.Load(test_directory + "/6502_functional_test.bin")

	if err != nil {
		t.Error(err)
	}
}

// Reading from non existing file fails with error
func TestReadNonExistingFileFails(t *testing.T) {
	ram, circuit := createTestCircuit(RAM_SIZE_1K)
	circuit.Wire(ram)

	err := ram.Load(test_directory + "/non_existing.bin")

	if err == nil {
		t.Error("Reading a non existing file should have failed")
	}
}

// Reading a bin file too big for the memory throws error
func TestReadingFileTooLargeForMemory(t *testing.T) {
	ram, circuit := createTestCircuit(RAM_SIZE_1K)
	circuit.Wire(ram)

	err := ram.Load(test_directory + "/6502_functional_test.bin")

	if err == nil {
		t.Error("Reading an existing file should have failed")
	}
}

/************************************************************************************
* Test other public methods
*************************************************************************************/

// Tests peek and poke
func TestPeekAndPokeReadsAndWritesValuesDirectly(t *testing.T) {
	ram, circuit := createTestCircuit(RAM_SIZE_1K)
	circuit.Wire(ram)

	for i := range RAM_SIZE_1K {
		ram.Poke(uint16(i), uint8(i))
	}

	for i := range RAM_SIZE_1K {
		value := ram.Peek(uint16(i))

		assert.Equal(t, uint8(i), value)
	}

	// Test PeekRange
	const size = 100
	values := ram.PeekRange(0, size)
	for i := range size {
		assert.Equal(t, uint8(i), values[i])
	}

}
