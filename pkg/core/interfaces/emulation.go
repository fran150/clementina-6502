package interfaces

import (
	"github.com/fran150/clementina-6502/pkg/common"
)

// ComputerCore combines emulation and rendering capabilities.
// This represents the core computer functionality.
type ComputerCore interface {
	Ticker

	// GetProgramCounter returns the current program counter value.
	GetProgramCounter() uint16

	// Puts the computer in reset state
	Reset(status bool)
}

// EmulationLoop defines the interface for managing emulation execution.
// This handles the lifecycle and timing of the emulation process.
type EmulationLoop interface {
	// Start begins the emulation loop and returns the execution context.
	// Returns nil if the loop is already running.
	Start() *common.StepContext

	// Stop stops the emulation loop.
	Stop()

	// Pauses execution of the emulation loop
	Pause()

	// Resumes the execution of the emulation loop if it was previously paused
	Resume()

	// IsRunning checks if the emulation loop is currently running.
	IsRunning() bool

	// IsStopping checks if the emulation loop is in the process of stopping.
	IsStopping() bool

	// IsPaused returns if the emulation loop is paused
	IsPaused() bool

	// SetEmulator sets the emulator instance to be used by the loop.
	// Loop must be stopped before calling this function.
	SetEmulator(emulator Emulator)

	// SetPanicHandler sets the panic handler for loop failures.
	SetPanicHandler(handler func(loopType string, panicData any) bool)
}

type Emulator interface {
	Ticker
	Renderer

	Run() (*common.StepContext, error)
	Stop()
	Pause()
	Resume()
	Step()
	Reset()
	UnReset()

	IsRunning() bool
	IsStopping() bool
	IsPaused() bool
	IsStepping() bool
	IsResetting() bool

	GetSpeedController() SpeedController
	GetBreakpointManager() BreakpointManager
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
