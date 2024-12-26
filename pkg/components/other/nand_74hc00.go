package nand

import (
	"github.com/fran150/clementina6502/pkg/components/buses"
	"github.com/fran150/clementina6502/pkg/components/common"
)

// Total number of NAND gates inside this chip
const numberOfGates int = 4

// The 74CH00 chip consists of 4 NAND gates. If pin a[X] and b[X] are both high
// pin Y[X] will be low, otherwise will go high
type Nand74HC00 struct {
	a [numberOfGates]buses.LineConnector // Pin A 1 to 4 (0..3)
	b [numberOfGates]buses.LineConnector // Pin B 1 to 4 (0..3)
	y [numberOfGates]buses.LineConnector // Pin Y 1 to 4 (0..3)
}

// Creates a new 74CH00
func Create74HC00() *Nand74HC00 {
	chip := Nand74HC00{}

	for i := range numberOfGates {
		chip.a[i] = buses.CreateConnectorEnabledHigh()
		chip.b[i] = buses.CreateConnectorEnabledHigh()
		chip.y[i] = buses.CreateConnectorEnabledHigh()
	}

	return &chip
}

// Returns the connector for the specified pin A
func (gate *Nand74HC00) APin(index int) buses.LineConnector {
	if index >= 0 && index < numberOfGates {
		return gate.a[index]
	} else {
		return nil
	}
}

// Returns the connector for the specified pin B
func (gate *Nand74HC00) BPin(index int) buses.LineConnector {
	if index >= 0 && index < numberOfGates {
		return gate.b[index]
	} else {
		return nil
	}
}

// Returns the connector for the specified pin Y
func (gate *Nand74HC00) YPin(index int) buses.LineConnector {
	if index >= 0 && index < numberOfGates {
		return gate.y[index]
	} else {
		return nil
	}
}

// Executes one emulation step
func (gate *Nand74HC00) Tick(stepContext common.StepContext) {
	for i := range numberOfGates {
		value := !(gate.a[i].Enabled() && gate.b[i].Enabled())
		gate.y[i].SetEnable(value)
	}
}
