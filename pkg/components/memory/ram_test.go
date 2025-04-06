package memory

import (
	"fmt"
	"os"
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
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
func newTestCircuit(size int) (*Ram, *testCircuit) {
	ram := NewRam(size)
	circuit := &testCircuit{
		addressBus:   buses.New16BitStandaloneBus(),
		dataBus:      buses.New8BitStandaloneBus(),
		writeEnable:  buses.NewStandaloneLine(false),
		outputEnable: buses.NewStandaloneLine(false),
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

// MockFile implements FileReader for testing
type MockFileReader struct {
}

func (m *MockFileReader) Stat() (os.FileInfo, error) {
	return nil, fmt.Errorf("stat error")
}

func (m *MockFileReader) Read(p []byte) (n int, err error) {
	return RAM_SIZE_16K, nil
}

func (m *MockFileReader) Close() error {
	return nil
}

/************************************************************************************
* Test RAM R/W
*************************************************************************************/

// Writes and reads a value from the memory
func TestRamReadWrite(t *testing.T) {
	context := common.NewStepContext()

	ram, circuit := newTestCircuit(RAM_SIZE_64K)
	circuit.Wire(ram)

	// Write Cycle
	circuit.writeEnable.Set(false)
	circuit.addressBus.Write(0x7FFA)
	circuit.dataBus.Write(0xFA)
	ram.Tick(&context)
	context.NextCycle()

	peek := ram.Peek(0x7FFA)

	if peek != 0xFA {
		t.Errorf("Error, expected to have 0xFA in memory 0x7FFA after write cycle but got %x", peek)
	}

	// Clear databus
	circuit.dataBus.Write(0x00)

	// Read Cycle
	circuit.writeEnable.Set(true)
	circuit.addressBus.Write(0x7FFA)
	ram.Tick(&context)
	context.NextCycle()

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
	ram, circuit := newTestCircuit(RAM_SIZE_64K)
	circuit.Wire(ram)

	err := ram.Load(test_directory + "/6502_functional_test.bin")

	if err != nil {
		t.Error(err)
	}
}

// Reading from non existing file fails with error
func TestReadNonExistingFileFails(t *testing.T) {
	ram, circuit := newTestCircuit(RAM_SIZE_1K)
	circuit.Wire(ram)

	err := ram.Load(test_directory + "/non_existing.bin")

	if err == nil {
		t.Error("Reading a non existing file should have failed")
	}
}

// Reading a bin file too big for the memory throws error
func TestReadingFileTooLargeForMemory(t *testing.T) {
	ram, circuit := newTestCircuit(RAM_SIZE_1K)
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
	ram, circuit := newTestCircuit(RAM_SIZE_1K)
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

/************************************************************************************
* Test RAM initialization
*************************************************************************************/

// Tests that NewRamWithLessPins correctly masks address pins
func TestNewRamWithLessPinsMasksAddressCorrectly(t *testing.T) {
	var ram *Ram

	context := common.NewStepContext()

	// Create a RAM with only lower 10 bits active (1024 addresses)
	addressMask := uint16(0x03FF) // Binary: 0000001111111111
	_, circuit := newTestCircuit(RAM_SIZE_2K)
	ram = NewRamWithLessPins(RAM_SIZE_2K, addressMask)
	circuit.Wire(ram)

	// Test writing to an address above the mask
	// Address 0x0400 should wrap to 0x0000 due to mask
	circuit.writeEnable.Set(false)
	circuit.addressBus.Write(0x0400)
	circuit.dataBus.Write(0xAA)
	ram.Tick(&context)
	context.NextCycle()

	// Verify that the value was written to the masked address (0x0000)
	if ram.Peek(0x0000) != 0xAA {
		t.Errorf("Expected value 0xAA at address 0x0000, got %02X", ram.Peek(0x0000))
	}

	// Write to another address above mask
	// Address 0x0401 should wrap to 0x0001
	circuit.addressBus.Write(0x0401)
	circuit.dataBus.Write(0xBB)
	ram.Tick(&context)
	context.NextCycle()

	// Verify that the value was written to the masked address (0x0001)
	if ram.Peek(0x0001) != 0xBB {
		t.Errorf("Expected value 0xBB at address 0x0001, got %02X", ram.Peek(0x0001))
	}

	// Test reading from a masked address
	circuit.writeEnable.Set(true)
	circuit.addressBus.Write(0x0400) // Should read from 0x0000
	circuit.dataBus.Write(0x00)      // Clear data bus
	ram.Tick(&context)
	context.NextCycle()

	value := circuit.dataBus.Read()
	if value != 0xAA {
		t.Errorf("Expected to read 0xAA from masked address 0x0400, got %02X", value)
	}

	// Verify that addresses within mask work normally
	circuit.writeEnable.Set(false)
	circuit.addressBus.Write(0x03FF) // Last valid address
	circuit.dataBus.Write(0xCC)
	ram.Tick(&context)
	context.NextCycle()

	if ram.Peek(0x03FF) != 0xCC {
		t.Errorf("Expected value 0xCC at address 0x03FF, got %02X", ram.Peek(0x03FF))
	}
}

/************************************************************************************
* Test RAM Size Reporting
*************************************************************************************/

// TestRamSizeReporting verifies that the Size method correctly reports the RAM size
func TestRamSizeReporting(t *testing.T) {
	testCases := []struct {
		size     int
		expected int
	}{
		{RAM_SIZE_64K, 65536},
		{RAM_SIZE_32K, 32768},
		{RAM_SIZE_16K, 16384},
		{RAM_SIZE_8K, 8192},
		{RAM_SIZE_4K, 4096},
		{RAM_SIZE_2K, 2048},
		{RAM_SIZE_1K, 1024},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("RAM size %d bytes", tc.size), func(t *testing.T) {
			ram, _ := newTestCircuit(tc.size)
			actualSize := ram.Size()

			if actualSize != tc.expected {
				t.Errorf("RAM size mismatch: got %d bytes, expected %d bytes", actualSize, tc.expected)
			}
		})
	}

	// Test custom size
	customSize := 512
	ram, _ := newTestCircuit(customSize)
	if ram.Size() != customSize {
		t.Errorf("Custom RAM size mismatch: got %d bytes, expected %d bytes", ram.Size(), customSize)
	}
}

/************************************************************************************
* Amazon Q generated tests
*************************************************************************************/
// TestLoadFailsOnReadError verifies that Load returns an error when bufr.Read fails
func TestLoadFailsOnReadError(t *testing.T) {
	ram, circuit := newTestCircuit(RAM_SIZE_1K)
	circuit.Wire(ram)

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test*.bin")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// Create a file but don't write anything to it
	// This will cause Read to return EOF error when trying to read RAM_SIZE_1K bytes
	tmpFile.Close()

	// Attempt to load the empty file
	err = ram.Load(tmpFile.Name())
	if err == nil {
		t.Error("Expected error when bufr.Read fails, but got nil")
	}
}

// Add this test to ram_test.go
func TestLoadFailsOnStatError(t *testing.T) {
	ram, circuit := newTestCircuit(RAM_SIZE_1K)
	circuit.Wire(ram)

	mockFile := &MockFileReader{}

	err := ram.loadFromReader(mockFile, nil)
	if err == nil {
		t.Error("Expected error when Stat fails, but got nil")
	}
	if err.Error() != "stat error" {
		t.Errorf("Expected 'stat error', got '%v'", err)
	}
}
