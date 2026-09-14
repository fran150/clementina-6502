package clementina

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/stretchr/testify/require"
)

// Exercise CPU bus writes through the actual VIA and SRAM wiring, without a ROM.
func TestClementinaVIABankSwitch(t *testing.T) {
	c, err := NewClementinaComputer()
	require.NoError(t, err)
	step := common.NewStepContext()
	write := func(address uint16, value uint8) {
		c.circuit.addressBus.Write(address)
		c.circuit.dataBus.Write(value)
		c.circuit.cpuRW.Set(false)
		c.chips.csLogic.Tick(&step)
		c.chips.oeRWSync.Tick(&step)
		c.chips.via.Tick(&step)
		c.chips.exram.Tick(&step)
		c.circuit.cpuRW.Set(true)
		step.NextCycle()
	}
	write(0xC003, 0x1F) // DDRA: PA0-PA4 drive the bank address.
	for bank := uint8(0); bank < 32; bank++ {
		write(0xC001, bank)
		write(0x8000, bank+1)
		write(0xBFFF, bank+65)
	}
	for bank := uint8(0); bank < 32; bank++ {
		require.Equal(t, bank+1, c.chips.exram.Peek(uint32(bank)*0x4000), "bank %d start", bank)
		require.Equal(t, bank+65, c.chips.exram.Peek(uint32(bank)*0x4000+0x3FFF), "bank %d end", bank)
		write(0xC001, bank)
		for _, address := range []uint16{0x8000, 0xBFFF} {
			c.circuit.addressBus.Write(address)
			c.chips.csLogic.Tick(&step)
			c.chips.oeRWSync.Tick(&step)
			c.chips.exram.Tick(&step)
			expected := bank + 1
			if address == 0xBFFF {
				expected = bank + 65
			}
			require.Equal(t, expected, c.circuit.dataBus.Read(), "bank %d address %04X", bank, address)
		}
	}
}
