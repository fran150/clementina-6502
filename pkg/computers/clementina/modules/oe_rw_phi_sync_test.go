package modules

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/fran150/clementina-6502/pkg/components/other/gates"
)

type clementinaOERWPHISyncTestCircuit struct {
	owRWPHISync *ClementinaOERWPHISync
	cpuRW       buses.Line
	phi2        buses.Line
}

func newClementinaOERWPHISyncTestCircuit() *clementinaOERWPHISyncTestCircuit {
	circuit := &clementinaOERWPHISyncTestCircuit{
		owRWPHISync: NewClementinaOERWPHISync(gates.NewNand74AC00()),
		cpuRW:       buses.NewStandaloneLine(true),
		phi2:        buses.NewStandaloneLine(true),
	}

	circuit.owRWPHISync.CpuRW().Connect(circuit.cpuRW)
	circuit.owRWPHISync.PHI2().Connect(circuit.phi2)

	return circuit
}

func TestOERWOutputs(t *testing.T) {
	step := common.NewStepContext()
	circuit := newClementinaOERWPHISyncTestCircuit()

	tests := []struct {
		rwb        bool
		phi2       bool
		expectedOE bool
		expectedWE bool
	}{
		{true, true, false, true},  // read: ~OE low
		{false, true, true, false}, // write while PHI2 is high: ~WE low
		{false, false, true, true}, // write while PHI2 is low: no strobe
		{true, false, false, true}, // read while PHI2 is low: ~OE stays low
	}

	for _, test := range tests {
		circuit.cpuRW.Set(test.rwb)
		circuit.phi2.Set(test.phi2)
		circuit.owRWPHISync.Tick(&step)

		if circuit.owRWPHISync.OE().Status() != test.expectedOE {
			t.Errorf("RWB %v PHI2 %v: expected ~OE to be %v, got %v", test.rwb, test.phi2, test.expectedOE, circuit.owRWPHISync.OE().Status())
		}
		if circuit.owRWPHISync.WE().Status() != test.expectedWE {
			t.Errorf("RWB %v PHI2 %v: expected ~WE to be %v, got %v", test.rwb, test.phi2, test.expectedWE, circuit.owRWPHISync.WE().Status())
		}
	}
}
