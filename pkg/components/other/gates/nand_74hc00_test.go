package gates

import (
	"testing"

	"github.com/fran150/clementina6502/pkg/components/buses"
	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/stretchr/testify/assert"
)

type testCircuit struct {
	a [numberOfGates]buses.Line
	b [numberOfGates]buses.Line
	y [numberOfGates]buses.Line
}

func newTestCircuit() *testCircuit {
	return &testCircuit{
		a: [numberOfGates]buses.Line{
			buses.NewStandaloneLine(false),
			buses.NewStandaloneLine(false),
			buses.NewStandaloneLine(false),
			buses.NewStandaloneLine(false),
		},
		b: [numberOfGates]buses.Line{
			buses.NewStandaloneLine(false),
			buses.NewStandaloneLine(false),
			buses.NewStandaloneLine(false),
			buses.NewStandaloneLine(false),
		},
		y: [numberOfGates]buses.Line{
			buses.NewStandaloneLine(false),
			buses.NewStandaloneLine(false),
			buses.NewStandaloneLine(false),
			buses.NewStandaloneLine(false),
		},
	}
}

type testCases struct {
	a bool
	b bool
	y bool
}

func (testCase *testCases) test(t *testing.T, circuit *testCircuit, chip *Nand74HC00, index int, step *common.StepContext) {
	circuit.a[index].Set(testCase.a)
	circuit.b[index].Set(testCase.b)

	chip.Tick(step)

	assert.Equal(t, testCase.y, circuit.y[index].Status())
}

func (circuit *testCircuit) wire(chip *Nand74HC00) {
	for i := range numberOfGates {
		chip.APin(i).Connect(circuit.a[i])
		chip.BPin(i).Connect(circuit.b[i])
		chip.YPin(i).Connect(circuit.y[i])
	}
}

func TestAllValuesForGates(t *testing.T) {
	var step common.StepContext

	chip := New74HC00()
	circuit := newTestCircuit()
	circuit.wire(chip)

	tests := []testCases{
		{false, false, true},
		{true, false, true},
		{false, true, true},
		{true, true, false},
	}

	for i := range numberOfGates {
		for _, test := range tests {
			test.test(t, circuit, chip, i, &step)
		}
	}
}

func TestInvalidPinNumberReturnsNil(t *testing.T) {
	chip := New74HC00()

	assert.Nil(t, chip.APin(-1))
	assert.Nil(t, chip.APin(numberOfGates))
	assert.Nil(t, chip.APin(numberOfGates+1))

	assert.Nil(t, chip.BPin(-1))
	assert.Nil(t, chip.BPin(numberOfGates))
	assert.Nil(t, chip.BPin(numberOfGates+1))

	assert.Nil(t, chip.YPin(-1))
	assert.Nil(t, chip.YPin(numberOfGates))
	assert.Nil(t, chip.YPin(numberOfGates+1))

}
