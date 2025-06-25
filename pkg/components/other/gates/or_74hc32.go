package gates

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
)

// Total number of OR gates inside this chip
const or74HC32NumGates int = 4

// The 74CH32 chip consists of 4 OR gates. If pin a[X] and b[X] are both high
// pin Y[X] will be low, otherwise will go high
type or74HC32 struct {
	a [or74HC32NumGates]buses.LineConnector // Pin A 1 to 4 (0..3)
	b [or74HC32NumGates]buses.LineConnector // Pin B 1 to 4 (0..3)
	y [or74HC32NumGates]buses.LineConnector // Pin Y 1 to 4 (0..3)
}

// Creates a new 74CH32
func New74HC32() *or74HC32 {
	chip := or74HC32{}

	for i := range or74HC32NumGates {
		chip.a[i] = buses.NewConnectorEnabledHigh()
		chip.b[i] = buses.NewConnectorEnabledHigh()
		chip.y[i] = buses.NewConnectorEnabledHigh()
	}

	return &chip
}

// Returns the connector for the specified pin A
func (gate *or74HC32) APin(index int) buses.LineConnector {
	if index >= 0 && index < or74HC32NumGates {
		return gate.a[index]
	} else {
		return nil
	}
}

// Returns the connector for the specified pin B
func (gate *or74HC32) BPin(index int) buses.LineConnector {
	if index >= 0 && index < or74HC32NumGates {
		return gate.b[index]
	} else {
		return nil
	}
}

// Returns the connector for the specified pin Y
func (gate *or74HC32) YPin(index int) buses.LineConnector {
	if index >= 0 && index < or74HC32NumGates {
		return gate.y[index]
	} else {
		return nil
	}
}

// Executes one emulation step
func (gate *or74HC32) Tick(stepContext *common.StepContext) {
	for i := range or74HC32NumGates {
		value := gate.a[i].Enabled() || gate.b[i].Enabled()
		gate.y[i].SetEnable(value)
	}
}
