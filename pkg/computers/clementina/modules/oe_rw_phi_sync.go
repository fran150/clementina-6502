// Package modules contains hardware emulation modules for the Clementina 6502 computer.
package modules

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
)

// ClementinaOERWPHISync emulates the synchronization logic for Output Enable (OE) and Read/Write (RW)
// signals with the clock PHI line. The PHI clock line is not emulated.
type ClementinaOERWPHISync struct {
	cpuRW buses.LineConnector

	oe buses.Line
	rw buses.Line
}

// NewClementinaOERWPHISync creates a new instance of the ClementinaOERWPHISync module.
func NewClementinaOERWPHISync() *ClementinaOERWPHISync {
	return &ClementinaOERWPHISync{
		cpuRW: buses.NewConnectorEnabledLow(),
		oe:    buses.NewStandaloneLine(true),
		rw:    buses.NewStandaloneLine(true),
	}
}

// CpuRW returns the LineConnector representing the CPU's RW line input to the sync module.
func (sync *ClementinaOERWPHISync) CpuRW() buses.LineConnector {
	return sync.cpuRW
}

// OE returns the Output Enable (OE) line controlled by the sync module.
func (sync *ClementinaOERWPHISync) OE() buses.Line {
	return sync.oe
}

// RW returns the Read/Write (RW) line controlled by the sync module.
func (sync *ClementinaOERWPHISync) RW() buses.Line {
	return sync.rw
}

// Tick updates the OE and RW lines based on the current state of the CPU's RW line.
// It should be called once per emulation step.
func (sync *ClementinaOERWPHISync) Tick(stepContext *common.StepContext) {
	cpuRWLow := sync.cpuRW.Enabled()

	// Connector is enabled low, so if cpu RW line is enable then RW output should
	// be low and OE should be high and vice versa
	sync.rw.Set(!cpuRWLow)
	sync.oe.Set(cpuRWLow)
}
