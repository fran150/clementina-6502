package emulation

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/core"
)

// EmulatorConfig holds the configuration for a DefaultEmulator instance.
// It contains all the necessary components required to run the emulation.
type EmulatorConfig struct {
	Computer          core.ComputerCore
	Console           core.EmulationConsole
	Loop              core.EmulationLoop
	SpeedController   core.SpeedController
	BreakpointManager core.BreakpointManager
}

// baseEmulator is the main emulator implementation that orchestrates the execution
// of a 6502 computer system. It manages the emulation loop, console interface,
// speed control, and breakpoint functionality.
type baseEmulator struct {
	config *EmulatorConfig

	stepping  bool
	resetting bool
}

/************************************************************************************
* Constructor
*************************************************************************************/
// newBaseEmulator creates a new defaultBaseEmulator instance with the provided configuration.
// It initializes the emulator with default state values (stepping and resetting set to false).
// This is an internal constructor function used by NewBaseEmulator.
func newBaseEmulator(config EmulatorConfig) *baseEmulator {
	emulator := &baseEmulator{
		config:    &config,
		stepping:  false,
		resetting: false,
	}

	return emulator
}

// NewBaseEmulator creates a new DefaultEmulator instance with the provided configuration.
// It initializes the emulator with default state values and sets up the bidirectional
// references between the emulator and its loop and console components.
func NewBaseEmulator(config EmulatorConfig) core.BaseEmulator {
	return newBaseEmulator(config)
}

/************************************************************************************
* State Management
*************************************************************************************/

// Start starts the emulator by initializing the emulation loop and console.
// It returns the step context from the loop and any error that occurred during console startup.
// If the console fails to start, the loop is stopped and the error is returned.
func (e *baseEmulator) Start() (*common.StepContext, error) {
	context, err := e.config.Loop.Start()
	if err != nil {
		return nil, err
	}

	if err := e.config.Console.Run(); err != nil {
		e.config.Loop.Stop()
		return context, err
	}

	return context, nil
}

// Stop terminates the emulator by stopping both the emulation loop and console.
// This method should be called to cleanly shut down the emulator and release resources.
func (e *baseEmulator) Stop() {
	e.config.Loop.Stop()
	e.config.Console.Stop()
}

// Pause pauses the emulation loop, stopping the execution of the computer system.
// The emulator can be resumed later using the Resume method.
func (e *baseEmulator) Pause() {
	e.config.Loop.Pause()
}

// Resume resumes the emulation loop after it has been paused.
// This continues the execution of the computer system from where it was paused.
func (e *baseEmulator) Resume() {
	e.config.Loop.Resume()
}

// Step executes a single step of the emulation if the emulator is currently paused.
// After executing one step, the emulator will automatically pause again.
// If the emulator is not paused, this method has no effect.
func (e *baseEmulator) Step() {
	if e.IsPaused() {
		e.stepping = true
		e.Resume()
	}
}

// Reset initiates a reset of the computer system by setting the resetting flag
// and calling the computer's Reset method with true to begin the reset process.
func (e *baseEmulator) Reset() {
	e.resetting = true
	e.config.Computer.Reset(true)
}

// UnReset completes the reset process by clearing the resetting flag
// and calling the computer's Reset method with false to finish the reset.
func (e *baseEmulator) UnReset() {
	e.resetting = false
	e.config.Computer.Reset(false)
}

/************************************************************************************
* State Getters
*************************************************************************************/

// IsRunning returns true if the emulation loop is currently running.
// This indicates that the emulator is actively executing the computer system.
func (e *baseEmulator) IsRunning() bool {
	return e.config.Loop.IsRunning()
}

// IsStopping returns true if the emulation loop is in the process of stopping.
// This indicates that a stop operation has been initiated but not yet completed.
func (e *baseEmulator) IsStopping() bool {
	return e.config.Loop.IsStopping()
}

// IsPaused returns true if the emulation loop is currently paused.
// When paused, the computer system execution is temporarily halted but can be resumed.
func (e *baseEmulator) IsPaused() bool {
	return e.config.Loop.IsPaused()
}

// IsStepping returns true if the emulator is currently in stepping mode.
// Stepping mode allows for single-step execution of the computer system.
func (e *baseEmulator) IsStepping() bool {
	return e.stepping
}

// IsResetting returns true if the computer system is currently being reset.
// This indicates that a reset operation is in progress.
func (e *baseEmulator) IsResetting() bool {
	return e.resetting
}

/************************************************************************************
* Loop methods
*************************************************************************************/

// Tick executes one emulation cycle by advancing the computer system by one step
// and updating the console. It handles stepping mode by automatically pausing
// after a single step, and checks for breakpoints to pause execution when hit.
// The method also updates the console with the current execution context.
func (e *baseEmulator) Tick(context *common.StepContext) {
	e.config.Computer.Tick(context)

	// Clear stepping state
	if e.IsStepping() {
		e.Pause()
		e.stepping = false
	}

	if e.config.BreakpointManager.HasBreakpoint(e.config.Computer.GetProgramCounter() - 1) {
		e.Pause()
	}

	e.config.Console.Tick(context)
}

// Draw renders the current state of the emulation by delegating to the console's
// draw method. This is typically called to update the visual representation
// of the computer system's current state.
func (e *baseEmulator) Draw(context *common.StepContext) {
	e.config.Console.Draw(context)
}
