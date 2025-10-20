package emulation

import (
	"time"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/core/interfaces"
)

// DefaultEmulationLoopConfig contains settings that control the display refresh rate.
type DefaultEmulationLoopConfig struct {
	SpeedController interfaces.SpeedController
	DisplayFPS      int
	RefreshNanos    int64
}

// DefaultEmulationLoop manages the timing and execution of the emulation cycle.
// It ensures the emulation runs at the specified speed and handles the
// separation between processing cycles and display updates.
type DefaultEmulationLoop struct {
	emulator interfaces.Emulator

	config       *DefaultEmulationLoopConfig
	panicHandler func(loopType string, panicData any) bool

	tickLoopRunning bool
	drawLoopRunning bool
	stop            bool
	pause           bool
}

/************************************************************************************
* Constructor
*************************************************************************************/

// NewEmulationLoop creates a new emulation loop with the specified components.
//
// Parameters:
//   - computer: The computer to run
//   - speedController: The speed controller for timing
//   - config: Configuration settings for the emulation loop
//
// Returns:
//   - A pointer to the initialized EmulationLoop
func NewEmulationLoop(config DefaultEmulationLoopConfig) *DefaultEmulationLoop {
	if config.RefreshNanos <= 0 {
		config.RefreshNanos = 15 * 1_000_000
	}

	if config.DisplayFPS <= 0 {
		config.DisplayFPS = 10
	}

	return &DefaultEmulationLoop{
		config:          &config,
		tickLoopRunning: false,
		drawLoopRunning: false,
		stop:            true,
	}
}

/************************************************************************************
* Setters
*************************************************************************************/

// SetEmulator sets the emulator instance for the emulation loop.
// This can only be called when the loop is not running.
//
// Parameters:
//   - emulator: The emulator instance to set
//
// Panics if called while the emulation loop is running.
func (e *DefaultEmulationLoop) SetEmulator(emulator interfaces.Emulator) {
	if !e.IsRunning() {
		e.emulator = emulator
	} else {
		panic("Cannot change emulator object while emulator loop is running")
	}

}

// SetPanicHandler sets the panic handler. This function will be called for cleanup if
// any of the emulation loops fail.
//
// Parameters:
//   - handler: Function to handle panics, returns true if panic should be suppressed
func (e *DefaultEmulationLoop) SetPanicHandler(handler func(loopType string, panicData any) bool) {
	e.panicHandler = handler
}

/************************************************************************************
* Getters
*************************************************************************************/

// IsRunning checks if the emulation loop is currently running.
// Returns true if the loop is not stopped.
func (e *DefaultEmulationLoop) IsRunning() bool {
	return e.drawLoopRunning || e.tickLoopRunning
}

// IsPaused checks if the emulation loop is currently paused.
// Returns true if the loop is paused.
func (e *DefaultEmulationLoop) IsPaused() bool {
	return e.pause
}

// IsStopping checks if the emulation loop is in the process of stopping.
func (e *DefaultEmulationLoop) IsStopping() bool {
	return e.stop && e.IsRunning()
}

/************************************************************************************
* State management
*************************************************************************************/

// Start begins the emulation loop with the provided handlers.
// It runs the emulation at the configured speed and manages the timing
// for both CPU cycles and display updates.
//
// Returns:
//   - A StepContext that can be used to control and monitor the emulation
func (e *DefaultEmulationLoop) Start() *common.StepContext {
	if !e.IsRunning() && e.emulator != nil {
		context := common.NewStepContext()

		e.pause = false
		e.stop = false

		go e.executeLoop(&context)
		go e.executeDraw(&context)

		return &context
	} else {
		return nil
	}
}

// Stop signals the emulation loop to stop execution.
// This will cause both the tick and draw loops to exit gracefully.
func (e *DefaultEmulationLoop) Stop() {
	e.stop = true
}

// Resume resumes the emulation loop execution after it has been paused.
// This allows the tick loop to continue processing CPU cycles.
func (e *DefaultEmulationLoop) Resume() {
	e.pause = false
}

// Pause pauses the emulation loop execution.
// The tick loop will stop processing CPU cycles but the draw loop continues.
func (e *DefaultEmulationLoop) Pause() {
	e.pause = true
}

/************************************************************************************
* Loops
*************************************************************************************/

// executeLoop runs the main emulation loop that processes CPU cycles.
//
// Parameters:
//   - context: The step context for emulation state
func (e *DefaultEmulationLoop) executeLoop(context *common.StepContext) {
	defer func() {
		e.tickLoopRunning = false
		if r := recover(); r != nil {
			e.handlePanic("Loop", r)
		}
	}()

	var lastTPSExecuted, targetTPSNano int64
	var lastSpeedCheck int64

	e.tickLoopRunning = true

	for !e.stop {
		if (context.T - lastSpeedCheck) > e.config.RefreshNanos {
			targetTPSNano = int64(e.config.SpeedController.GetNanosPerCycle())
			lastSpeedCheck = context.T
		}

		if (context.T-lastTPSExecuted) > targetTPSNano && !e.pause {
			lastTPSExecuted = context.T
			e.emulator.Tick(context)
			context.NextCycle()
		} else {
			context.SkipCycle()
		}
	}
}

// executeDraw runs the display update loop at the configured frame rate.
//
// Parameters:
//   - context: The step context for emulation state
func (e *DefaultEmulationLoop) executeDraw(context *common.StepContext) {
	defer func() {
		e.drawLoopRunning = false
		if r := recover(); r != nil {
			e.handlePanic("Draw", r)
		}
	}()

	ticker := time.NewTicker(time.Second / time.Duration(e.config.DisplayFPS))
	defer ticker.Stop()

	e.drawLoopRunning = true

	for !e.stop {
		<-ticker.C
		e.emulator.Draw(context)
	}
}

/************************************************************************************
* Loop error management
*************************************************************************************/

// handlePanic triggers the execution of a handler before panicking.
//
// Parameters:
//   - loopType: The type of loop that panicked (for logging purposes)
//   - r: The recovered panic value
func (e *DefaultEmulationLoop) handlePanic(loopType string, r any) {
	e.Stop()
	if e.panicHandler != nil {
		if !e.panicHandler(loopType, r) {
			panic(r)
		}
	}
}
