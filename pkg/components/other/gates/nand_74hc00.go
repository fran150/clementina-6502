package gates

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/fran150/clementina-6502/pkg/components/buses"
)

// Total number of NAND gates inside this chip
const nand74HC00NumGates int = 4

// The 74CH00 chip consists of 4 NAND gates. If pin a[X] and b[X] are both high
// pin Y[X] will be low, otherwise will go high
type nand74HC00 struct {
	a [nand74HC00NumGates]buses.LineConnector // Pin A 1 to 4 (0..3)
	b [nand74HC00NumGates]buses.LineConnector // Pin B 1 to 4 (0..3)
	y [nand74HC00NumGates]buses.LineConnector // Pin Y 1 to 4 (0..3)
}

func NewNand74HC00() components.QuadLogicGate {
	return newNand74HC00()
}

// Creates a new 74CH00
func newNand74HC00() *nand74HC00 {
	chip := nand74HC00{}

	for i := range nand74HC00NumGates {
		chip.a[i] = buses.NewConnectorEnabledHigh()
		chip.b[i] = buses.NewConnectorEnabledHigh()
		chip.y[i] = buses.NewConnectorEnabledHigh()
	}

	return &chip
}

// Returns the connector for the specified pin A
func (gate *nand74HC00) APin(index int) buses.LineConnector {
	if index >= 0 && index < nand74HC00NumGates {
		return gate.a[index]
	} else {
		return nil
	}
}

// Returns the connector for the specified pin B
func (gate *nand74HC00) BPin(index int) buses.LineConnector {
	if index >= 0 && index < nand74HC00NumGates {
		return gate.b[index]
	} else {
		return nil
	}
}

// Returns the connector for the specified pin Y
func (gate *nand74HC00) YPin(index int) buses.LineConnector {
	if index >= 0 && index < nand74HC00NumGates {
		return gate.y[index]
	} else {
		return nil
	}
}

// Executes one emulation step
func (gate *nand74HC00) Tick(stepContext *common.StepContext) {
	for i := range nand74HC00NumGates {
		value := !gate.a[i].Enabled() || !gate.b[i].Enabled()
		gate.y[i].SetEnable(value)
	}
}
