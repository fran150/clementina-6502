package memory

import (
	"bufio"
	"fmt"
	"os"

	"github.com/fran150/clementina6502/pkg/buses"
)

// Constants for multiple typical memory sizes
const (
	RAM_SIZE_64K int = 65536 // 64K memory
	RAM_SIZE_32K int = 32768 // 32K memory
	RAM_SIZE_16K int = 16384 // 16K memory
	RAM_SIZE_8K  int = 8192  // 8K memory
	RAM_SIZE_4K  int = 4096  // 4K memory
	RAM_SIZE_2K  int = 2048  // 2K memory
	RAM_SIZE_1K  int = 1024  // 1K memory
)

// Represents a RAM chip.
type Ram struct {
	values       []uint8
	addressBus   *buses.BusConnector[uint16]
	dataBus      *buses.BusConnector[uint8]
	writeEnable  *buses.ConnectorEnabledLow
	chipSelect   *buses.ConnectorEnabledLow
	outputEnable *buses.ConnectorEnabledLow
}

// Creates a RAM chip
func CreateRam(size int) *Ram {
	return &Ram{
		values:       make([]uint8, size),
		addressBus:   buses.CreateBusConnector[uint16](),
		dataBus:      buses.CreateBusConnector[uint8](),
		writeEnable:  buses.CreateConnectorEnabledLow(),
		chipSelect:   buses.CreateConnectorEnabledLow(),
		outputEnable: buses.CreateConnectorEnabledLow(),
	}
}

/************************************************************************************
* Getters / Setters
*************************************************************************************/

// Connector to the address bus. The value of the address bus indicates where
// the value should be written or read from.
func (ram *Ram) AddressBus() *buses.BusConnector[uint16] {
	return ram.addressBus
}

// Connector to the data bus. The value of the data bus indicates the value read from
// the memory or the value to be written depending on the operation.
func (ram *Ram) DataBus() *buses.BusConnector[uint8] {
	return ram.dataBus
}

// Connector to the R/W line. When low it indicates that it's a write operation, when
// high it indicates it's a read operation
func (ram *Ram) WriteEnable() *buses.ConnectorEnabledLow {
	return ram.writeEnable
}

// Connector to the chip select line. When low it indicates that this chip is selected and
// must respond to requests
func (ram *Ram) ChipSelect() *buses.ConnectorEnabledLow {
	return ram.chipSelect
}

// Connector to the output enable line. The chip will only set the requested value in the
// data bus if this line is low.
func (ram *Ram) OutputEnable() *buses.ConnectorEnabledLow {
	return ram.outputEnable
}

/************************************************************************************
* Utility functions
*************************************************************************************/

// Returns the value on the specified address
func (ram *Ram) Peek(address uint16) uint8 {
	return ram.values[address]
}

// Sets the value on the specified address
func (ram *Ram) Poke(address uint16, value uint8) {
	ram.values[address] = value
}

// Loads a bin file into memory. Not used memory is always initialized in 0.
// File is loaded starting from 0x0000 address
func (ram *Ram) Load(binFilePath string) error {
	file, err := os.Open(binFilePath)

	if err != nil {
		return err
	}
	defer file.Close()

	stats, statsErr := file.Stat()
	if statsErr != nil {
		return statsErr
	}

	var size int64 = stats.Size()

	if size <= int64(len(ram.values)) {
		bufr := bufio.NewReader(file)
		bufr.Read(ram.values[:])
	} else {
		return fmt.Errorf("the file %s is too large for this ram memory (file size: %v, ram size: %v)", binFilePath, size, len(ram.values))
	}

	return nil
}

/************************************************************************************
* Internal functions
*************************************************************************************/

// Gets the data in the address on the address bus and puts it on the data bus.
func (ram *Ram) read() {
	ram.dataBus.Write(ram.values[ram.addressBus.Read()])
}

// Writes the data in the bus to the address specified in the address bus.
func (ram *Ram) write() {
	ram.values[ram.addressBus.Read()] = ram.dataBus.Read()
}

/************************************************************************************
 * Timer Tick
*************************************************************************************/

// Executes one emulation step
func (ram *Ram) Tick(t uint64) {
	if ram.chipSelect.Enabled() {
		if ram.writeEnable.Enabled() {
			ram.write()
		} else if ram.outputEnable.Enabled() && !ram.writeEnable.Enabled() {
			ram.read()
		}
	}
}
