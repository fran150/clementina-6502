package emulation

import (
	"fmt"
	"sync/atomic"
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

// EmulationLoopConfig contains settings that control the display refresh rate.
type EmulationLoopConfig struct {
	SpeedController core.SpeedController
	DisplayFPS      int
	RefreshNanos    int64
	Emulator        LoopTarget
}

// emulationLoop manages the timing and execution of the emulation cycle.
// It ensures the emulation runs at the specified speed and handles the
// separation between processing cycles and display updates.
type emulationLoop struct {
	config       *EmulationLoopConfig
	panicHandler func(loopType string, panicData any) bool

	tickLoopRunning atomic.Bool
	drawLoopRunning atomic.Bool
	stop            atomic.Bool
	pause           atomic.Bool
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

func newEmulationLoop(config EmulationLoopConfig) *emulationLoop {
	if config.RefreshNanos <= 0 {
		config.RefreshNanos = 15 * 1_000_000
	}

	if config.DisplayFPS <= 0 {
		config.DisplayFPS = 10
	}

	loop := &emulationLoop{
		config: &config,
	}
	loop.stop.Store(true)

	return loop
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
func NewEmulationLoop(config EmulationLoopConfig) core.EmulationLoop {
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
func (e *emulationLoop) SetPanicHandler(handler func(loopType string, panicData any) bool) {
	e.panicHandler = handler
}

/************************************************************************************
* Getters
*************************************************************************************/

// IsRunning checks if the emulation loop is currently running.
// Returns true if the loop is not stopped.
func (e *emulationLoop) IsRunning() bool {
	return e.drawLoopRunning.Load() || e.tickLoopRunning.Load()
}

// IsPaused checks if the emulation loop is currently paused.
// Returns true if the loop is paused.
func (e *emulationLoop) IsPaused() bool {
	return e.pause.Load()
}

// IsStopping checks if the emulation loop is in the process of stopping.
func (e *emulationLoop) IsStopping() bool {
	return e.stop.Load() && e.IsRunning()
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
func (e *emulationLoop) Start() (*common.StepContext, error) {
	if !e.IsRunning() && e.config.Emulator != nil {
		context := common.NewStepContext()

		e.pause.Store(false)
		e.stop.Store(false)
		e.tickLoopRunning.Store(true)
		e.drawLoopRunning.Store(true)

		go e.executeLoop(&context)

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
func (e *emulationLoop) Stop() {
	e.stop.Store(true)
}

// Resume resumes the emulation loop execution after it has been paused.
// This allows the tick loop to continue processing CPU cycles.
func (e *emulationLoop) Resume() {
	e.pause.Store(false)
}

// Pause pauses the emulation loop execution.
// The tick loop will stop processing CPU cycles but the draw loop continues.
func (e *emulationLoop) Pause() {
	e.pause.Store(true)
}

/************************************************************************************
* Loops
*************************************************************************************/

// executeLoop runs the main emulation loop that processes CPU cycles.
//
// Parameters:
//   - context: The step context for emulation state
func (e *emulationLoop) executeLoop(context *common.StepContext) {
	defer func() {
		e.tickLoopRunning.Store(false)
		e.drawLoopRunning.Store(false)
		if r := recover(); r != nil {
			e.handlePanic("Loop", r)
		}
	}()

	var lastTPSExecuted, targetTPSNano int64
	var lastSpeedCheck int64
	drawInterval := time.Second / time.Duration(e.config.DisplayFPS)
	nextDraw := time.Now().Add(drawInterval)

	e.tickLoopRunning.Store(true)
	e.drawLoopRunning.Store(true)

	for !e.stop.Load() {
		if time.Now().After(nextDraw) {
			e.config.Emulator.Draw(context)
			nextDraw = nextDraw.Add(drawInterval)
			if time.Now().After(nextDraw) {
				nextDraw = time.Now().Add(drawInterval)
			}
		}

		if (context.T - lastSpeedCheck) > e.config.RefreshNanos {
			targetTPSNano = int64(e.config.SpeedController.GetNanosPerCycle())
			lastSpeedCheck = context.T
		}

		if (context.T-lastTPSExecuted) > targetTPSNano && !e.pause.Load() {
			lastTPSExecuted = context.T
			e.config.Emulator.Tick(context)
			context.NextCycle()
		} else {
			context.SkipCycle()
		}
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
func (e *emulationLoop) handlePanic(loopType string, r any) {
	e.Stop()
	if e.panicHandler != nil {
		if !e.panicHandler(loopType, r) {
			panic(r)
		}
	}
}
