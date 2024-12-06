package memory

import (
	"testing"

	"github.com/fran150/clementina6502/pkg/buses"
)

func TestRamReadWrite(t *testing.T) {
	addressBus := buses.Create16BitBus()
	dataBus := buses.Create8BitBus()

	ramWriteEnableLine := buses.CreateStandaloneLine(true)
	alwaysLowLine := buses.CreateStandaloneLine(false)

	ram := CreateRam()
	ram.Connect(addressBus, dataBus, ramWriteEnableLine, addressBus.GetBusLine(15), alwaysLowLine)

	// Write Cycle
	ramWriteEnableLine.Set(false)
	addressBus.Write(0x7FFA)
	dataBus.Write(0xFA)
	ram.Tick(100)

	peek := ram.Peek(0x7FFA)

	if peek != 0xFA {
		t.Errorf("Error, expected to have 0xFA in memory 0x7FFA after write cycle but got %x", peek)
	}

	// Clear databus
	dataBus.Write(0x00)

	// Read Cycle
	ramWriteEnableLine.Set(true)
	addressBus.Write(0x7FFA)
	ram.Tick(100)
	value := dataBus.Read()

	if value != 0xFA {
		t.Errorf("Error, expected to read 0xFA in databus after read cycle but got %x", value)
	}

	// Clear databus
	dataBus.Write(0x00)
}

func TestReadBinFile(t *testing.T) {
	addressBus := buses.Create16BitBus()
	dataBus := buses.Create8BitBus()

	ramWriteEnableLine := buses.CreateStandaloneLine(true)
	alwaysLowLine := buses.CreateStandaloneLine(false)

	ram := CreateRam()
	ram.Connect(addressBus, dataBus, ramWriteEnableLine, addressBus.GetBusLine(15), alwaysLowLine)

	ram.Load("../tests/6502_functional_test.bin")
}
