// Package computers provides computer system implementations and emulation control.
package computers

import (
	"github.com/fran150/clementina-6502/pkg/common"
)

// Emulator defines the core emulation logic interface.
// This represents the pure emulation functionality without lifecycle concerns.
type Emulator interface {
	// Tick processes one clock cycle of the computer system.
	// This includes updating all components like CPU, memory, and peripherals.
	//
	// Parameters:
	//   - context: The current step context for the emulation cycle
	Tick(context *common.StepContext)
}

// Renderer defines the display rendering interface.
// This handles the visual representation of the computer state.
type Renderer interface {
	// Draw updates the visual representation of the computer state.
	// This is called separately from Tick to allow for different update rates.
	//
	// Parameters:
	//   - context: The current step context for the emulation cycle
	Draw(context *common.StepContext)
}

// ComputerCore combines emulation and rendering capabilities.
// This represents the core computer functionality.
type ComputerCore interface {
	Emulator
	Renderer
}

// SpeedController defines the interface for managing emulation speed.
type SpeedController interface {
	// SpeedUp increases the emulation speed.
	SpeedUp()

	// SpeedDown decreases the emulation speed.
	SpeedDown()

	// GetTargetSpeed returns the current target speed in MHz.
	GetTargetSpeed() float64

	// SetTargetSpeed sets the target speed in MHz.
	SetTargetSpeed(speedMhz float64)

	// GetTargetSpeedPtr returns a pointer to the current target speed in MHz.
	GetTargetSpeedPtr() *float64
}
