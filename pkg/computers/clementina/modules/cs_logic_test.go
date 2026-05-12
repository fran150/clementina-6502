package modules

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/stretchr/testify/assert"
)

type csLogicTestCircuit struct {
	addressBus buses.Bus[uint16]

	csLogic *ClementinaCSLogic
}

func newCSLogicTestCircuit() *csLogicTestCircuit {
	// Creates the test circuit logic that contains a bus and the CS Logic circuit.
	circuit := &csLogicTestCircuit{
		addressBus: buses.New16BitStandaloneBus(),
		csLogic:    NewClementinaCSLogic(),
	}

	// Connects all CS logic address line to individual lines on the bus.
	for i := uint8(15); i >= 10; i-- {
		circuit.csLogic.A1(int(i - 10)).Connect(circuit.addressBus.GetBusLine(i))
	}

	return circuit
}

// Structure to store test cases
type csLogicTestCase struct {
	inputValue uint16

	expectedIOCS    uint8
	expectedExRAMCS bool
	expectedMiaCS   bool
}

// Runs the current test case
func (tc *csLogicTestCase) test(t *testing.T, circuit *csLogicTestCircuit, step *common.StepContext) {
	circuit.addressBus.Write(tc.inputValue)

	circuit.csLogic.Tick(step)

	if circuit.csLogic.IOCS().Read() != tc.expectedIOCS {
		t.Errorf("For adddess $%04X expected IOCS to be %02X, got %02X", tc.inputValue, tc.expectedIOCS, circuit.csLogic.IOCS().Read())
	}

	if circuit.csLogic.ExRAMCS().Status() != tc.expectedExRAMCS {
		t.Errorf("For adddess $%04X expected ExRAMCS to be %t, got %t", tc.inputValue, tc.expectedExRAMCS, circuit.csLogic.ExRAMCS().Status())
	}

	if circuit.csLogic.MiaCS().Status() != tc.expectedMiaCS {
		t.Errorf("For adddess $%04X expected MIACS to be %t, got %t", tc.inputValue, tc.expectedMiaCS, circuit.csLogic.MiaCS().Status())
	}
}

func TestClementinaCSLogicMemoryMapBaseMem(t *testing.T) {
	step := common.NewStepContext()
	circuit := newCSLogicTestCircuit()

	// $0000-$7FFF: base RAM range. IOCS stays inactive, ExRAMCS stays inactive,
	// and active-high MIASEL stays low.
	for i := uint16(0x0000); i < 0x8000; i++ {
		csLogicTestCase := csLogicTestCase{
			inputValue:      i,
			expectedIOCS:    0xFF,
			expectedExRAMCS: true,
			expectedMiaCS:   false,
		}

		csLogicTestCase.test(t, circuit, &step)
	}
}

func TestClementinaCSLogicMemoryMapExtendedMem(t *testing.T) {
	step := common.NewStepContext()
	circuit := newCSLogicTestCircuit()

	// $8000-$BFFF: extended RAM range. ExRAMCS is asserted low, while IOCS
	// stays inactive and active-high MIASEL stays low.
	for i := uint16(0x8000); i < 0xC000; i++ {
		csLogicTestCase := csLogicTestCase{
			inputValue:      i,
			expectedIOCS:    0xFF,
			expectedExRAMCS: false,
			expectedMiaCS:   false,
		}

		csLogicTestCase.test(t, circuit, &step)
	}
}

func TestClementinaCSLogicMemoryMapIOMem(t *testing.T) {
	step := common.NewStepContext()
	circuit := newCSLogicTestCircuit()

	io := uint8(1)
	base := uint16(0xC000)

	// $C000-$DFFF: I/O range split into eight 1 KB slots. Exactly one active-low
	// IOCS line is asserted per slot; ExRAMCS and active-high MIASEL stay inactive.
	for range 8 {
		for i := base; i < (base + 1023); i++ {
			csLogicTestCase := csLogicTestCase{
				inputValue:      i,
				expectedIOCS:    ^io,
				expectedExRAMCS: true,
				expectedMiaCS:   false,
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

	// $E000-$FFFF: MIA range. Active-high MIASEL is asserted, while IOCS and
	// ExRAMCS stay inactive.
	// The for checks != 0x0000 as after 0xFFFF the address wraps around to 0x0000
	for i := uint16(0xE000); i != 0x0000; i++ {
		csLogicTestCase := csLogicTestCase{
			inputValue:      i,
			expectedIOCS:    0xFF,
			expectedExRAMCS: true,
			expectedMiaCS:   true,
		}

		csLogicTestCase.test(t, circuit, &step)
	}
}

func TestClementinaCSLogicAccessingIncorrectAddressLineReturnsNull(t *testing.T) {
	circuit := newCSLogicTestCircuit()

	// Only A10-A15 are exposed by this module; out-of-range address line indexes
	// should not return a connector.
	assert.Nil(t, circuit.csLogic.A1(8))
}
