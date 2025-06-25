package gates

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/stretchr/testify/assert"
)

type logicGatesTestCircuit struct {
	size int
	a    []buses.Line
	b    []buses.Line
	y    []buses.Line
}

func newLogicGatesTestCircuit(gatesNum int) *logicGatesTestCircuit {
	circuit := &logicGatesTestCircuit{}

	circuit.size = gatesNum

	circuit.a = make([]buses.Line, gatesNum)
	circuit.b = make([]buses.Line, gatesNum)
	circuit.y = make([]buses.Line, gatesNum)

	for i := range gatesNum {
		circuit.a[i] = buses.NewStandaloneLine(false)
		circuit.b[i] = buses.NewStandaloneLine(false)
		circuit.y[i] = buses.NewStandaloneLine(false)
	}

	return circuit
}

func (circuit *logicGatesTestCircuit) wire(chip QuadLogicGate) {
	for i := range circuit.size {
		chip.APin(i).Connect(circuit.a[i])
		chip.BPin(i).Connect(circuit.b[i])
		chip.YPin(i).Connect(circuit.y[i])
	}
}

type logicGatesTestCase struct {
	a bool
	b bool
	y bool
}

func (testCase *logicGatesTestCase) test(t *testing.T, circuit *logicGatesTestCircuit, chip QuadLogicGate, index int, step *common.StepContext) {
	circuit.a[index].Set(testCase.a)
	circuit.b[index].Set(testCase.b)

	chip.Tick(step)

	assert.Equal(t, testCase.y, circuit.y[index].Status())
}
