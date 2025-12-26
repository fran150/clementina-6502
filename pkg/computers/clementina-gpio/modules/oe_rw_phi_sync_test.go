package modules

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
)

type clementinaOERWPHISyncTestCircuit struct {
	owRWPHISync *ClementinaOERWPHISync
	cpuRW       buses.Line
}

func newClementinaOERWPHISyncTestCircuit() *clementinaOERWPHISyncTestCircuit {
	circuit := &clementinaOERWPHISyncTestCircuit{
		owRWPHISync: NewClementinaOERWPHISync(),
		cpuRW:       buses.NewStandaloneLine(true),
	}

	circuit.owRWPHISync.CpuRW().Connect(circuit.cpuRW)

	return circuit
}

func TestOERWOutputs(t *testing.T) {
	step := common.NewStepContext()
	circuit := newClementinaOERWPHISyncTestCircuit()

	tests := []struct {
		cpuRWLow   bool
		expectedOE bool
		expectedRW bool
	}{
		{true, false, true},
		{false, true, false},
	}

	for _, test := range tests {
		circuit.cpuRW.Set(test.cpuRWLow)
		circuit.owRWPHISync.Tick(&step)

		if circuit.owRWPHISync.OE().Status() != test.expectedOE {
			t.Errorf("Expected OE to be %v, got %v", test.expectedOE, circuit.owRWPHISync.OE().Status())
		}
		if circuit.owRWPHISync.RW().Status() != test.expectedRW {
			t.Errorf("Expected RW to be %v, got %v", test.expectedRW, circuit.owRWPHISync.RW().Status())
		}
	}
}
