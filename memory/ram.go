package memory

import (
	"bufio"
	"fmt"
	"os"

	"github.com/fran150/clementina6502/buses"
)

type Ram struct {
	values       [0xFFFF + 1]uint8
	addressBus   *buses.BusConnector[uint16]
	dataBus      *buses.BusConnector[uint8]
	writeEnable  buses.ConnectorEnabledLow
	chipSelect   buses.ConnectorEnabledLow
	outputEnable buses.ConnectorEnabledLow
}

func CreateRam() *Ram {
	return &Ram{
		values:       [0xFFFF + 1]uint8{},
		addressBus:   buses.CreateBusConnector[uint16](),
		dataBus:      buses.CreateBusConnector[uint8](),
		writeEnable:  *buses.CreateConnectorEnabledLow(),
		chipSelect:   *buses.CreateConnectorEnabledLow(),
		outputEnable: *buses.CreateConnectorEnabledLow(),
	}
}

func (ram *Ram) Connect(addressBus *buses.Bus[uint16], dataBus *buses.Bus[uint8], writeEnable buses.Line, chipSelect buses.Line, outputEnable buses.Line) {
	ram.addressBus.Connect(addressBus)
	ram.dataBus.Connect(dataBus)
	ram.writeEnable.Connect(writeEnable)
	ram.chipSelect.Connect(chipSelect)
	ram.outputEnable.Connect(outputEnable)
}

func (ram *Ram) Peek(address uint16) uint8 {
	return ram.values[address]
}

func (ram *Ram) Poke(address uint16, data uint8) {
	ram.values[address] = data
}

func (ram *Ram) read() {
	ram.dataBus.Write(ram.values[ram.addressBus.Read()])
}

func (ram *Ram) write() {
	ram.values[ram.addressBus.Read()] = ram.dataBus.Read()
}

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

// TODO: Add handling for disconnected lines

func (ram *Ram) Tick(t uint64) {
	if ram.chipSelect.Enabled() {
		if ram.writeEnable.Enabled() {
			ram.write()
		} else if ram.outputEnable.Enabled() && !ram.writeEnable.Enabled() {
			ram.read()
		}
	}
}
