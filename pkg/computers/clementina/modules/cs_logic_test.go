package modules

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/stretchr/testify/assert"
)

type csLogicTestCircuit struct {
	addressBus buses.Bus[uint16]

	picoHiRAME buses.Line
	ioOE       buses.Bus[uint8]
	exRAME     buses.Line
	hiRAME     buses.Line

	csLogic *ClementinaCSLogic
}

func newCSLogicTestCircuit() *csLogicTestCircuit {
	circuit := &csLogicTestCircuit{
		addressBus: buses.New16BitStandaloneBus(),
		csLogic:    NewClementinaCSLogic(),
	}

	for i := uint8(15); i >= 10; i-- {
		circuit.csLogic.A1(int(i - 10)).Connect(circuit.addressBus.GetBusLine(i))
	}

	circuit.picoHiRAME = buses.NewStandaloneLine(false)

	circuit.ioOE = buses.New8BitStandaloneBus()
	circuit.exRAME = buses.NewStandaloneLine(false)
	circuit.hiRAME = buses.NewStandaloneLine(false)

	circuit.csLogic.PicoHiRAME().Connect(circuit.picoHiRAME)

	circuit.csLogic.IOOE().Connect(circuit.ioOE)
	circuit.csLogic.ExRAME().Connect(circuit.exRAME)
	circuit.csLogic.HiRAME().Connect(circuit.hiRAME)

	return circuit
}

type csLogicTestCase struct {
	inputValue uint16
	picoHiRAME bool

	expectedIOOE   uint8
	expectedExRAME bool
	expectedHiRAME bool
}

func (tc *csLogicTestCase) test(t *testing.T, circuit *csLogicTestCircuit, step *common.StepContext) {
	circuit.addressBus.Write(tc.inputValue)
	circuit.picoHiRAME.Set(tc.picoHiRAME)

	circuit.csLogic.Tick(step)

	if circuit.ioOE.Read() != tc.expectedIOOE {
		t.Errorf("For adddess $%04X expected IOOE to be %02X, got %02X", tc.inputValue, tc.expectedIOOE, circuit.ioOE.Read())
	}

	if circuit.exRAME.Status() != tc.expectedExRAME {
		t.Errorf("For adddess $%04X expected ExRAME to be %t, got %t", tc.inputValue, tc.expectedExRAME, circuit.exRAME.Status())
	}

	if circuit.hiRAME.Status() != tc.expectedHiRAME {
		t.Errorf("For adddess $%04X expected HiRAME to be %t, got %t", tc.inputValue, tc.expectedHiRAME, circuit.hiRAME.Status())
	}
}

func TestClementinaCSLogicMemoryMapBaseMem(t *testing.T) {
	step := common.NewStepContext()
	circuit := newCSLogicTestCircuit()

	for i := uint16(0x0000); i < 0x8000; i++ {
		csLogicTestCase := csLogicTestCase{
			inputValue:     i,
			picoHiRAME:     false,
			expectedIOOE:   0xFF,
			expectedExRAME: true,
			expectedHiRAME: true,
		}

		csLogicTestCase.test(t, circuit, &step)
	}
}

func TestClementinaCSLogicMemoryMapExtendedMem(t *testing.T) {
	step := common.NewStepContext()
	circuit := newCSLogicTestCircuit()

	for i := uint16(0x8000); i < 0xC000; i++ {
		csLogicTestCase := csLogicTestCase{
			inputValue:     i,
			picoHiRAME:     false,
			expectedIOOE:   0xFF,
			expectedExRAME: false,
			expectedHiRAME: true,
		}

		csLogicTestCase.test(t, circuit, &step)
	}
}

func TestClementinaCSLogicMemoryMapIOMem(t *testing.T) {
	step := common.NewStepContext()
	circuit := newCSLogicTestCircuit()

	io := uint8(1)
	base := uint16(0xC000)

	for range 8 {
		for i := base; i < (base + 1023); i++ {
			csLogicTestCase := csLogicTestCase{
				inputValue:     i,
				picoHiRAME:     false,
				expectedIOOE:   ^io,
				expectedExRAME: true,
				expectedHiRAME: true,
			}

			csLogicTestCase.test(t, circuit, &step)
		}

		io <<= 1
		base += 1024
	}
}

func TestClementinaCSLogicMemoryMapHiMem(t *testing.T) {
	step := common.NewStepContext()
	circuit := newCSLogicTestCircuit()

	// HiRAM is enabled for addresses from 0xE000 to 0xFFFF when pico is enabling
	// with picoHiRAME low.
	// The for checks != 0x0000 as after 0xFFFF the address wraps around to 0x0000
	for i := uint16(0xE000); i != 0x0000; i++ {
		csLogicTestCase := csLogicTestCase{
			inputValue:     i,
			picoHiRAME:     false,
			expectedIOOE:   0xFF,
			expectedExRAME: true,
			expectedHiRAME: false,
		}

		csLogicTestCase.test(t, circuit, &step)
	}

	// When pico is not enabling the HiRAM chip is disabled
	for i := uint16(0xE000); i != 0x0000; i++ {
		csLogicTestCase := csLogicTestCase{
			inputValue:     i,
			picoHiRAME:     true,
			expectedIOOE:   0xFF,
			expectedExRAME: true,
			expectedHiRAME: true,
		}

		csLogicTestCase.test(t, circuit, &step)
	}
}

func TestClementinaCSLogicAccessingIncorrectAddressLineReturnsNull(t *testing.T) {
	circuit := newCSLogicTestCircuit()

	assert.Nil(t, circuit.csLogic.A1(8))
}
