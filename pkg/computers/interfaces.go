// Package computers provides computer system implementations and emulation control.
package computers

import (
	"github.com/fran150/clementina-6502/pkg/common"
)

// Computer defines the interface for emulated computer systems.
// Any implementation must provide methods for processing clock cycles
// and updating the display.
type Computer interface {
	// Run starts the computer system and returns the execution context.
	Run() (*common.StepContext, error)

	// Stop stops the computer system execution.
	Stop()

	// Tick processes one clock cycle of the computer system.
	// This includes updating all components like CPU, memory, and peripherals.
	//
	// Parameters:
	//   - context: The current step context for the emulation cycle
	Tick(context *common.StepContext)

	// Draw updates the visual representation of the computer state.
	// This is called separately from Tick to allow for different update rates.
	//
	// Parameters:
	//   - context: The current step context for the emulation cycle
	Draw(context *common.StepContext)
}
