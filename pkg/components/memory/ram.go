// Package memory provides components for handling memory operations in the 6502 emulator
package memory

import (
	"fmt"
	"os"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
)

// FileReader interface abstracts file operations for loading binary data into memory.
// This allows for easier testing by mocking file operations.
type FileReader interface {
	Stat() (os.FileInfo, error)
	Read(p []byte) (n int, err error)
	Close() error
}

// Memory size constants representing common RAM configurations
const (
	RAM_SIZE_64K int = 65536 // 64K memory
	RAM_SIZE_32K int = 32768 // 32K memory
	RAM_SIZE_16K int = 16384 // 16K memory
	RAM_SIZE_8K  int = 8192  // 8K memory
	RAM_SIZE_4K  int = 4096  // 4K memory
	RAM_SIZE_2K  int = 2048  // 2K memory
	RAM_SIZE_1K  int = 1024  // 1K memory
)

// useAllPins represents a mask where all address pins are active
const useAllPins uint16 = 0xFFFF

// Ram represents a RAM chip emulation with standard control signals.
// It implements typical RAM chip functionality including read/write operations,
// chip select, and output enable controls.
type Ram struct {
	values          []uint8                     // Memory contents
	addressBus      *buses.BusConnector[uint16] // Connection to the address bus
	dataBus         *buses.BusConnector[uint8]  // Connection to the data bus
	writeEnable     *buses.ConnectorEnabledLow  // Write Enable signal (active low)
	chipSelect      *buses.ConnectorEnabledLow  // Chip Select signal (active low)
	outputEnable    *buses.ConnectorEnabledLow  // Output Enable signal (active low)
	addressPinsMask uint16                      // Mask for active address pins
}

// NewRam creates a new RAM chip with the specified size in bytes.
// It initializes all the necessary bus connectors and control signals.
func NewRam(size int) *Ram {
	return &Ram{
		values:       make([]uint8, size),
		addressBus:   buses.NewBusConnector[uint16](),
		dataBus:      buses.NewBusConnector[uint8](),
		writeEnable:  buses.NewConnectorEnabledLow(),
		chipSelect:   buses.NewConnectorEnabledLow(),
		outputEnable: buses.NewConnectorEnabledLow(),

		addressPinsMask: useAllPins,
	}
}

// NewRamWithLessPins creates a new RAM chip with the specified size and address pin mask.
// The address pin mask allows emulation of chips with fewer address pins by masking
// unused address lines. This is useful for emulating smaller memory chips or memory-mapped I/O.
func NewRamWithLessPins(size int, addressPinsMask uint16) *Ram {
	return &Ram{
		values:       make([]uint8, size),
		addressBus:   buses.NewBusConnector[uint16](),
		dataBus:      buses.NewBusConnector[uint8](),
		writeEnable:  buses.NewConnectorEnabledLow(),
		chipSelect:   buses.NewConnectorEnabledLow(),
		outputEnable: buses.NewConnectorEnabledLow(),

		addressPinsMask: addressPinsMask,
	}
}

/************************************************************************************
* Getters / Setters
*************************************************************************************/

// AddressBus returns the connector to the address bus.
// The address bus determines the memory location for read/write operations.
func (ram *Ram) AddressBus() *buses.BusConnector[uint16] {
	return ram.addressBus
}

// DataBus returns the connector to the data bus.
// The data bus carries the value being read from or written to memory.
func (ram *Ram) DataBus() *buses.BusConnector[uint8] {
	return ram.dataBus
}

// WriteEnable returns the write enable signal connector.
// When low, indicates a write operation; when high, indicates a read operation.
func (ram *Ram) WriteEnable() *buses.ConnectorEnabledLow {
	return ram.writeEnable
}

// ChipSelect returns the chip select signal connector.
// When low, indicates this chip is selected and should respond to operations.
func (ram *Ram) ChipSelect() *buses.ConnectorEnabledLow {
	return ram.chipSelect
}

// OutputEnable returns the output enable signal connector.
// When low, allows the chip to put data on the data bus during read operations.
func (ram *Ram) OutputEnable() *buses.ConnectorEnabledLow {
	return ram.outputEnable
}

/************************************************************************************
* Utility functions
*************************************************************************************/

// Peek returns the value at the specified memory address without
// going through the normal bus operations.
func (ram *Ram) Peek(address uint16) uint8 {
	return ram.values[address]
}

// PeekRange returns a slice of memory values between startAddress and endAddress.
// Useful for debugging and memory dumps.
func (ram *Ram) PeekRange(startAddress uint16, endAddress uint16) []uint8 {
	return ram.values[startAddress:endAddress]
}

// Poke writes a value directly to the specified memory address without
// going through the normal bus operations.
func (ram *Ram) Poke(address uint16, value uint8) {
	ram.values[address] = value
}

// Load reads a binary file into memory starting at address 0x0000.
// Returns an error if the file is too large for the available memory
// or if there are any I/O errors.
func (ram *Ram) Load(binFilePath string) error {
	return ram.loadFromReader(os.Open(binFilePath))
}

// Size returns the total size of the RAM chip in bytes.
func (ram *Ram) Size() int {
	return len(ram.values)
}

/************************************************************************************
* Internal functions
*************************************************************************************/

// getAddress returns the current address from the address bus
// masked with the active address pins mask.
func (ram *Ram) getAddress() uint16 {
	return ram.addressBus.Read() & ram.addressPinsMask
}

// read gets the data from the current address and puts it on the data bus.
func (ram *Ram) read() {
	ram.dataBus.Write(ram.values[ram.getAddress()])
}

// write stores the data from the data bus to the current address.
func (ram *Ram) write() {
	ram.values[ram.getAddress()] = ram.dataBus.Read()
}

// LoadFromReader reads binary data from a reader into memory
func (ram *Ram) loadFromReader(file FileReader, err error) error {
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
		if _, err := file.Read(ram.values[:]); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("the file is too large for this ram memory (file size: %v, ram size: %v)", size, len(ram.values))
	}

	return nil
}

/************************************************************************************
 * Timer Tick
*************************************************************************************/

// Tick performs one emulation step, handling memory operations based on
// the current state of the control signals (chip select, output enable, and write enable).
func (ram *Ram) Tick(context *common.StepContext) {
	cs := ram.chipSelect.Enabled()
	oe := ram.outputEnable.Enabled()
	writeEnable := ram.writeEnable.Enabled()

	if cs {
		if writeEnable {
			ram.write()
		} else if oe && !writeEnable {
			ram.read()
		}
	}
}
