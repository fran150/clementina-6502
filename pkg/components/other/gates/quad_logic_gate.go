package gates

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
)

// QuadLogicGate defines the interface for a quad logic gate component.
// It provides methods to access the A, B, and Y pins and to execute a tick.
// It is used by various logic gate implementations like NAND, AND, etc.
type QuadLogicGate interface {
	APin(index int) buses.LineConnector
	BPin(index int) buses.LineConnector
	YPin(index int) buses.LineConnector
	Tick(stepContext *common.StepContext)
}
