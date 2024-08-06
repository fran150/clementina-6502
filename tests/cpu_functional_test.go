package tests

import (
	"testing"

	"github.com/fran150/clementina6502/buses"
	"github.com/fran150/clementina6502/cpu"
	"github.com/fran150/clementina6502/memory"
)

func TestProcessorFunctional(t *testing.T) {
	addressBus := buses.CreateBus[uint16]()
	dataBus := buses.CreateBus[uint8]()

	alwaysHighLine := buses.CreateStandaloneLine(true)
	alwaysLowLine := buses.CreateStandaloneLine(false)

	writeEnableLine := buses.CreateStandaloneLine(true)

	memoryLockLine := buses.CreateStandaloneLine(false)
	syncLine := buses.CreateStandaloneLine(false)
	vectorPullLine := buses.CreateStandaloneLine(false)

	ram := memory.CreateRam()
	ram.Connect(addressBus, dataBus, writeEnableLine, alwaysLowLine, alwaysLowLine)

	processor := cpu.CreateCPU()
	processor.ConnectAddressBus(addressBus)
	processor.ConnectDataBus(dataBus)

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

	processor.ForceProgramCounter(0x0400)

	ram.Load("../tests/6502_functional_test.bin")

	var time uint64 = 0
	for {
		processor.Tick(time)
		ram.Tick(time)

		processor.PostTick(time)

		processor.ShowProcessorStatus()

		time++
	}
}
