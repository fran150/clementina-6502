package interfaces

import "github.com/fran150/clementina-6502/pkg/common"

// ComputerCore combines emulation and rendering capabilities.
// This represents the core computer functionality.
type ComputerCore interface {
	Ticker

	// GetProgramCounter returns the current program counter value.
	GetProgramCounter() uint16
}

// EmulationLoop defines the interface for managing emulation execution.
// This handles the lifecycle and timing of the emulation process.
type EmulationLoop interface {
	// Start begins the emulation loop and returns the execution context.
	// Returns nil if the loop is already running.
	Start() *common.StepContext

	// Stop stops the emulation loop.
	Stop()

	// IsRunning checks if the emulation loop is currently running.
	IsRunning() bool

	// IsStopping checks if the emulation loop is in the process of stopping.
	IsStopping() bool

	// SetPanicHandler sets the panic handler for loop failures.
	SetPanicHandler(handler func(loopType string, panicData any) bool)
}

// Ticker defines the core emulation logic interface.
// This represents the pure emulation functionality without lifecycle concerns.
type Ticker interface {
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
