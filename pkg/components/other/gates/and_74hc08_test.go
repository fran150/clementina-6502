package gates

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/stretchr/testify/assert"
)

func TestAllValuesFor74HC08(t *testing.T) {
	var step common.StepContext

	chip := New74HC08()
	circuit := newLogicGatesTestCircuit(4)
	circuit.wire(chip)

	tests := []logicGatesTestCase{
		{false, false, false},
		{true, false, false},
		{false, true, false},
		{true, true, true},
	}

	for i := range and74HC08NumGates {
		for _, test := range tests {
			test.test(t, circuit, chip, i, &step)
		}
	}
}

func TestInvalidPinNumberReturnsNilOn74HC08(t *testing.T) {
	chip := New74HC08()

	assert.Nil(t, chip.APin(-1))
	assert.Nil(t, chip.APin(and74HC08NumGates))
	assert.Nil(t, chip.APin(and74HC08NumGates+1))

	assert.Nil(t, chip.BPin(-1))
	assert.Nil(t, chip.BPin(and74HC08NumGates))
	assert.Nil(t, chip.BPin(and74HC08NumGates+1))

	assert.Nil(t, chip.YPin(-1))
	assert.Nil(t, chip.YPin(and74HC08NumGates))
	assert.Nil(t, chip.YPin(and74HC08NumGates+1))
}
