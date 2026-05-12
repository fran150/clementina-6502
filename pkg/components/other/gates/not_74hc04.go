package gates

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/fran150/clementina-6502/pkg/components/buses"
)

// Total number of OR gates inside this chip
const or74HC04NumGates int = 6

// The 74CH04 chip consists of 6 inverters. Pin y[X] will have the inverse output of a[X].
// For example if a[X] goes high then y[X] will go low. And if a[X] goes low, y[X] will go high
type not74HC04 struct {
	a [or74HC04NumGates]buses.LineConnector // Pin A 0 to 5 (0..5)
	y [or74HC04NumGates]buses.LineConnector // Pin Y 0 to 5 (0..5)
}

// NewOr74HC32 creates a new 74HC32 quad OR gate chip
func NewNot74HC04() components.InverterArray {
	return newNot74HC04()
}

// Creates a new 74CH04
func newNot74HC04() *not74HC04 {
	chip := not74HC04{}

	for i := range or74HC32NumGates {
		chip.a[i] = buses.NewConnectorEnabledHigh()
		chip.y[i] = buses.NewConnectorEnabledHigh()
	}

	return &chip
}

// Returns the connector for the specified pin A
func (gate *not74HC04) APin(index int) buses.LineConnector {
	if index >= 0 && index < or74HC32NumGates {
		return gate.a[index]
	} else {
		return nil
	}
}

// Returns the connector for the specified pin Y
func (gate *not74HC04) YPin(index int) buses.LineConnector {
	if index >= 0 && index < or74HC32NumGates {
		return gate.y[index]
	} else {
		return nil
	}
}

// Executes one emulation step
func (gate *not74HC04) Tick(stepContext *common.StepContext) {
	for i := range or74HC32NumGates {
		gate.y[i].SetEnable(!gate.a[i].Enabled())
	}
}
