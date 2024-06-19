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

	cpu.yRegister = 5

	ram.Poke(0xFFFC, 0xB9) // LDA $02FF,x
	ram.Poke(0xFFFD, 0xFF)
	ram.Poke(0xFFFE, 0x02)
	ram.Poke(0xFFFF, 0xEA) // NOOP
	ram.Poke(0x0000, 0xA1) // LDA ($B0, X)
	ram.Poke(0x0001, 0xB0)

	ram.Poke(0x00B0, 0x10)
	ram.Poke(0x00B1, 0x20)

	ram.Poke(0x02FF+5, 0x77)

	ram.Poke(0x2010, 0xDD)

	for range 10 {
		cpu.Tick(100)
		ram.Tick(100)

		cpu.PostTick(100)
	}
}
