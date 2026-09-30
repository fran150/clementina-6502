// Package modules contains hardware emulation modules for the Clementina 6502 computer.
package modules

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/fran150/clementina-6502/pkg/components/buses"
)

// ClementinaOERWPHISync emulates the "OE / RW / PHI2 Sync" sheet of the schematic: the memory
// read and write strobes, made from gates A and B of U9, the 74AC00 shared with the CS logic sheet.
//
//   - U9A: ~OE = NAND(RWB, RWB), low while the CPU reads.
//   - U9B: ~WE = NAND(PHI2, ~OE), low while the CPU writes and PHI2 is high. One gate after PHI2,
//     so ~WE ends within the CPU's address and data hold time.
//
// The emulator runs each bus cycle as a whole and doesn't generate PHI2, so the computer ties the
// PHI2 input high.
type ClementinaOERWPHISync struct {
	cpuRW buses.LineConnector // RWB, to both inputs of U9A
	phi2  buses.LineConnector // PHI2, U9B pin 5

	nand components.LogicGateArray // U9 (74AC00), gates A and B; shared with the CS logic sheet

	oe buses.Line // ~OE: U9A output, also U9B pin 4
	we buses.Line // ~WE: U9B output
}

// NewClementinaOERWPHISync creates a new instance of the ClementinaOERWPHISync module.
//
// Parameters:
//   - nand: U9, the 74AC00 whose gates A and B this sheet uses. Gates C and D belong to the
//     CS logic sheet, which receives the same chip.
func NewClementinaOERWPHISync(nand components.LogicGateArray) *ClementinaOERWPHISync {
	sync := &ClementinaOERWPHISync{
		nand: nand,
		oe:   buses.NewStandaloneLine(true),
		we:   buses.NewStandaloneLine(true),
	}

	sync.cpuRW = newNet(nand.APin(0), nand.BPin(0))
	sync.phi2 = nand.BPin(1)

	nand.YPin(0).Connect(sync.oe)
	nand.APin(1).Connect(sync.oe)
	nand.YPin(1).Connect(sync.we)

	return sync
}

// CpuRW returns the LineConnector for the CPU's RWB line input to the sync module.
func (sync *ClementinaOERWPHISync) CpuRW() buses.LineConnector {
	return sync.cpuRW
}

// PHI2 returns the LineConnector for the PHI2 clock input that qualifies ~WE.
func (sync *ClementinaOERWPHISync) PHI2() buses.LineConnector {
	return sync.phi2
}

// OE returns the Output Enable (~OE) line controlled by the sync module.
func (sync *ClementinaOERWPHISync) OE() buses.Line {
	return sync.oe
}

// WE returns the Write Enable (~WE) line controlled by the sync module.
func (sync *ClementinaOERWPHISync) WE() buses.Line {
	return sync.we
}

// Tick updates ~OE and ~WE from RWB and PHI2. The chip evaluates its gates in order, so ~WE
// sees this step's ~OE. It should be called once per emulation step.
//
// Parameters:
//   - stepContext: The current step context for the emulation cycle
func (sync *ClementinaOERWPHISync) Tick(stepContext *common.StepContext) {
	sync.nand.Tick(stepContext)
}
