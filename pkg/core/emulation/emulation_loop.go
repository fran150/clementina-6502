package emulation

import (
	"fmt"
	"time"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/core"
)

// LoopTarget defines the interface that an emulation target must implement
// to be used with the emulation loop. It combines the ability to be paused,
// execute ticks, and render output.
type LoopTarget interface {
	core.Pausable
	core.Ticker
	core.Renderer
}

// DefaultEmulationLoopConfig contains settings that control the display refresh rate.
type DefaultEmulationLoopConfig struct {
	SpeedController core.SpeedController
	DisplayFPS      int
	RefreshNanos    int64
	Emulator        LoopTarget
}

// defaultEmulationLoop manages the timing and execution of the emulation cycle.
// It ensures the emulation runs at the specified speed and handles the
// separation between processing cycles and display updates.
type defaultEmulationLoop struct {
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
// newEmulationLoop creates a new defaultEmulationLoop instance with the provided configuration.
// It sets default values for RefreshNanos (15ms) and DisplayFPS (10) if they are not positive.
//
// Parameters:
//   - config: Configuration settings for the emulation loop
//
// Returns:
//   - A pointer to the initialized defaultEmulationLoop

func newEmulationLoop(config DefaultEmulationLoopConfig) *defaultEmulationLoop {
	if config.RefreshNanos <= 0 {
		config.RefreshNanos = 15 * 1_000_000
	}

	if config.DisplayFPS <= 0 {
		config.DisplayFPS = 10
	}

	return &defaultEmulationLoop{
		config:          &config,
		tickLoopRunning: false,
		drawLoopRunning: false,
		stop:            true,
	}
}

// NewEmulationLoop creates a new emulation loop with the specified components.
//
// Parameters:
//   - computer: The computer to run
//   - speedController: The speed controller for timing
//   - config: Configuration settings for the emulation loop
//
// Returns:
//   - A pointer to the initialized EmulationLoop
func NewEmulationLoop(config DefaultEmulationLoopConfig) core.EmulationLoop {
	return newEmulationLoop(config)
}

/************************************************************************************
* Setters
*************************************************************************************/

// SetPanicHandler sets the panic handler. This function will be called for cleanup if
// any of the emulation loops fail.
//
// Parameters:
//   - handler: Function to handle panics, returns true if panic should be suppressed
func (e *defaultEmulationLoop) SetPanicHandler(handler func(loopType string, panicData any) bool) {
	e.panicHandler = handler
}

/************************************************************************************
* Getters
*************************************************************************************/

// IsRunning checks if the emulation loop is currently running.
// Returns true if the loop is not stopped.
func (e *defaultEmulationLoop) IsRunning() bool {
	return e.drawLoopRunning || e.tickLoopRunning
}

// IsPaused checks if the emulation loop is currently paused.
// Returns true if the loop is paused.
func (e *defaultEmulationLoop) IsPaused() bool {
	return e.pause
}

// IsStopping checks if the emulation loop is in the process of stopping.
func (e *defaultEmulationLoop) IsStopping() bool {
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
func (e *defaultEmulationLoop) Start() (*common.StepContext, error) {
	if !e.IsRunning() && e.config.Emulator != nil {
		context := common.NewStepContext()

		e.pause = false
		e.stop = false

		go e.executeLoop(&context)
		go e.executeDraw(&context)

		return &context, nil
	} else {
		var err error

		if e.IsRunning() {
			err = fmt.Errorf("cannot start again while loop is running")
		} else {
			err = fmt.Errorf("cannot start as emulator is not set")
		}

		return nil, err
	}
}

// Stop signals the emulation loop to stop execution.
// This will cause both the tick and draw loops to exit gracefully.
func (e *defaultEmulationLoop) Stop() {
	e.stop = true
}

// Resume resumes the emulation loop execution after it has been paused.
// This allows the tick loop to continue processing CPU cycles.
func (e *defaultEmulationLoop) Resume() {
	e.pause = false
}

// Pause pauses the emulation loop execution.
// The tick loop will stop processing CPU cycles but the draw loop continues.
func (e *defaultEmulationLoop) Pause() {
	e.pause = true
}

/************************************************************************************
* Loops
*************************************************************************************/

// executeLoop runs the main emulation loop that processes CPU cycles.
//
// Parameters:
//   - context: The step context for emulation state
func (e *defaultEmulationLoop) executeLoop(context *common.StepContext) {
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
			e.config.Emulator.Tick(context)
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
func (e *defaultEmulationLoop) executeDraw(context *common.StepContext) {
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
		e.config.Emulator.Draw(context)
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
func (e *defaultEmulationLoop) handlePanic(loopType string, r any) {
	e.Stop()
	if e.panicHandler != nil {
		if !e.panicHandler(loopType, r) {
			panic(r)
		}
	}
}
