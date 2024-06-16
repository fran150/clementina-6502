package cpu

import (
	"testing"

	"github.com/fran150/clementina6502/buses"
	"github.com/fran150/clementina6502/memory"
)

func TestCpuReadOpCodeCycle(t *testing.T) {
	addressBus := buses.CreateBus[uint16]()
	dataBus := buses.CreateBus[uint8]()

	alwaysHighLine := buses.CreateStandaloneLine(true)
	alwaysLowLine := buses.CreateStandaloneLine(false)

	writeEnableLine := buses.CreateStandaloneLine(true)

	ram := memory.CreateRam()
	ram.Connect(addressBus, dataBus, writeEnableLine, alwaysLowLine, alwaysLowLine)

	cpu := CreateCPU()
	cpu.ConnectAddressBus(addressBus)
	cpu.ConnectDataBus(dataBus)

	cpu.BusEnable().Connect(alwaysHighLine)
	cpu.ReadWrite().Connect(writeEnableLine)

	ram.Poke(0xFFFC, 0xA9)
	ram.Poke(0xFFFD, 0xFF)
	ram.Poke(0xFFFE, 0xA9)
	ram.Poke(0xFFFF, 0xAA)

	cpu.Tick(100)
	ram.Tick(100)

	cpu.PostTick(100)

	cpu.Tick(100)
	ram.Tick(100)

	cpu.PostTick(100)

	cpu.Tick(100)
	ram.Tick(100)

	cpu.PostTick(100)

	cpu.Tick(100)
	ram.Tick(100)

	cpu.PostTick(100)

}
