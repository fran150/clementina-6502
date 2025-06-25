package gates

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
)

// Total number of AND gates inside this chip
const and74HC08NumGates int = 4

// The 74CH08 chip consists of 4 AND gates. If pin a[X] and b[X] are both high
// pin Y[X] will be high, otherwise will go low
type And74HC08 struct {
	a [and74HC08NumGates]buses.LineConnector // Pin A 1 to 4 (0..3)
	b [and74HC08NumGates]buses.LineConnector // Pin B 1 to 4 (0..3)
	y [and74HC08NumGates]buses.LineConnector // Pin Y 1 to 4 (0..3)
}

// Creates a new 74CH08
func New74HC08() *And74HC08 {
	chip := And74HC08{}

	for i := range and74HC08NumGates {
		chip.a[i] = buses.NewConnectorEnabledHigh()
		chip.b[i] = buses.NewConnectorEnabledHigh()
		chip.y[i] = buses.NewConnectorEnabledHigh()
	}

	return &chip
}

// Returns the connector for the specified pin A
func (gate *And74HC08) APin(index int) buses.LineConnector {
	if index >= 0 && index < and74HC08NumGates {
		return gate.a[index]
	} else {
		return nil
	}
}

// Returns the connector for the specified pin B
func (gate *And74HC08) BPin(index int) buses.LineConnector {
	if index >= 0 && index < and74HC08NumGates {
		return gate.b[index]
	} else {
		return nil
	}
}

// Returns the connector for the specified pin Y
func (gate *And74HC08) YPin(index int) buses.LineConnector {
	if index >= 0 && index < and74HC08NumGates {
		return gate.y[index]
	} else {
		return nil
	}
}

// Executes one emulation step
func (gate *And74HC08) Tick(stepContext *common.StepContext) {
	for i := range and74HC08NumGates {
		value := gate.a[i].Enabled() && gate.b[i].Enabled()
		gate.y[i].SetEnable(value)
	}
}
