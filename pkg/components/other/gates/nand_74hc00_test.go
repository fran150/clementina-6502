package gates

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/stretchr/testify/assert"
)

func TestAllValuesFor74HC00(t *testing.T) {
	var step common.StepContext

	chip := New74HC00()
	circuit := newLogicGatesTestCircuit(4)
	circuit.wire(chip)

	tests := []logicGatesTestCase{
		{false, false, true},
		{true, false, true},
		{false, true, true},
		{true, true, false},
	}

	for i := range nand74HC00NumGates {
		for _, test := range tests {
			test.test(t, circuit, chip, i, &step)
		}
	}
}

func TestInvalidPinNumberReturnsNilOn74HC00(t *testing.T) {
	chip := New74HC00()

	assert.Nil(t, chip.APin(-1))
	assert.Nil(t, chip.APin(nand74HC00NumGates))
	assert.Nil(t, chip.APin(nand74HC00NumGates+1))

	assert.Nil(t, chip.BPin(-1))
	assert.Nil(t, chip.BPin(nand74HC00NumGates))
	assert.Nil(t, chip.BPin(nand74HC00NumGates+1))

	assert.Nil(t, chip.YPin(-1))
	assert.Nil(t, chip.YPin(nand74HC00NumGates))
	assert.Nil(t, chip.YPin(nand74HC00NumGates+1))
}
