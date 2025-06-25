package gates

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/stretchr/testify/assert"
)

func TestAllValuesFor74HC32(t *testing.T) {
	var step common.StepContext

	chip := New74HC32()
	circuit := newLogicGatesTestCircuit(4)
	circuit.wire(chip)

	tests := []logicGatesTestCase{
		{false, false, false},
		{true, false, true},
		{false, true, true},
		{true, true, true},
	}

	for i := range or74HC32NumGates {
		for _, test := range tests {
			test.test(t, circuit, chip, i, &step)
		}
	}
}

func TestInvalidPinNumberReturnsNilOn74HC32(t *testing.T) {
	chip := New74HC32()

	assert.Nil(t, chip.APin(-1))
	assert.Nil(t, chip.APin(or74HC32NumGates))
	assert.Nil(t, chip.APin(or74HC32NumGates+1))

	assert.Nil(t, chip.BPin(-1))
	assert.Nil(t, chip.BPin(or74HC32NumGates))
	assert.Nil(t, chip.BPin(or74HC32NumGates+1))

	assert.Nil(t, chip.YPin(-1))
	assert.Nil(t, chip.YPin(or74HC32NumGates))
	assert.Nil(t, chip.YPin(or74HC32NumGates+1))
}
