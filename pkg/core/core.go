package core

import "github.com/fran150/clementina-6502/pkg/common"

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

// EmulationConsole represents the console interface for the emulator.
// This is not the main display but the window that allows the user to interact with the emulator.
type EmulationConsole interface {
	// Ticker is used to update the internal state of console's components after each emulation frame
	Ticker
	// Renderer is called to draw the console.
	Renderer

	// Run starts the emulation console and returns an error if startup fails.
	Run() error

	// Stop gracefully shuts down the emulation console.
	Stop()

	// SetEmulator assigns an emulator instance to this console.
	SetEmulator(emulator Emulator)
}

// SpeedController defines a mechanism for managing emulation speed.
type SpeedController interface {
	// SpeedUp increases the emulation speed.
	SpeedUp()

	// SpeedDown decreases the emulation speed.
	SpeedDown()

	// GetTargetSpeed returns the current target speed in MHz.
	GetTargetSpeed() float64

	// SetTargetSpeed sets the target speed in MHz.
	SetTargetSpeed(speedMhz float64)

	// GetNanosPerCycle returns the nanoseconds per cycle for the current speed.
	GetNanosPerCycle() float64
}

// This is the core computer to be emulaterd. It will typically contain the list of componentes
// that form the computer and their wiring
type ComputerCore interface {
	Ticker

	// GetProgramCounter returns the current program counter value.
	GetProgramCounter() uint16

	// Allows to set the computer in reset state (similar to pressing reset button)
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

// Emulator defines the main emulator interface
type Emulator interface {
	Ticker
	Renderer

	// Run starts the emulator and returns the execution context.
	// Returns an error if the emulator cannot be started.
	Run() (*common.StepContext, error)

	// Stop stops the emulator execution.
	// After signaling the stop, it takes some time to stop the loop, some extra emulation steps or frames might be rendered.
	// Check IsStopping and IsRunning functions to get the status.
	Stop()

	// Pause pauses the emulator execution.
	Pause()

	// Resume resumes the emulator execution if it was previously paused.
	Resume()

	// Step executes a single emulation step and then pauses.
	Step()

	// Reset puts the emulated computer into reset state.
	Reset()

	// UnReset releases the emulated computer from reset state.
	UnReset()

	// IsRunning checks if the emulator is currently running.
	IsRunning() bool

	// IsStopping checks if the emulator is in the process of stopping.
	IsStopping() bool

	// IsPaused returns true if the emulator is paused.
	IsPaused() bool

	// IsStepping returns true if the emulator is executing a single emulation step and will pause when finished.
	IsStepping() bool

	// IsResetting returns true if the emulated computer is in reset state.
	IsResetting() bool

	// GetComputer returns the computer core instance.
	GetComputer() ComputerCore

	// GetConsole returns the emulation console instance.
	GetConsole() EmulationConsole

	// GetLoop returns the emulation loop instance.
	GetLoop() EmulationLoop

	// GetSpeedController returns the speed controller for managing emulation timing.
	GetSpeedController() SpeedController

	// GetBreakpointManager returns the breakpoint manager for debugging functionality.
	GetBreakpointManager() BreakpointManager
}

// NavigationManager defines the interface for managing window navigation.
type NavigationManager interface {
	// NavigateTo switches to the specified window.
	NavigateTo(key string)

	// GoBack returns to the previous window.
	GoBack()

	// GetCurrent returns the currently active window key.
	GetCurrent() string

	// PushToHistory adds the current window to history and navigates to new window.
	PushToHistory(key string)
}

// BreakpointManager defines the contract for managing breakpoints for computer debugging.
type BreakpointManager interface {
	// AddBreakpoint adds a breakpoint at the specified address.
	// If a breakpoint already exists at the address, it won't be added again.
	AddBreakpoint(address uint16)

	// RemoveBreakpoint removes a breakpoint at the specified address.
	// If no breakpoint exists at the address, this method has no effect.
	RemoveBreakpoint(address uint16)

	// RemoveBreakpointByIndex removes a breakpoint at the specified index.
	// If the index is out of bounds, this method has no effect.
	RemoveBreakpointByIndex(index int)

	// HasBreakpoint checks if a breakpoint exists at the specified address.
	HasBreakpoint(address uint16) bool

	// GetBreakpoints returns a copy of all breakpoint addresses.
	GetBreakpoints() []uint16

	// GetBreakpointCount returns the number of active breakpoints.
	GetBreakpointCount() int

	// ClearAllBreakpoints removes all breakpoints.
	ClearAllBreakpoints()
}
