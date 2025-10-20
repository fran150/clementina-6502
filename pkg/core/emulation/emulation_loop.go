package emulation

import (
	"time"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/core/interfaces"
)

// EmulationLoopConfig contains settings that control the display refresh rate.
type EmulationLoopConfig struct {
	SpeedController interfaces.SpeedController
	PanicHandler    func(loopType string, panicData any) bool
	DisplayFPS      int
	RefreshNanos    int64
}

// EmulationLoop manages the timing and execution of the emulation cycle.
// It ensures the emulation runs at the specified speed and handles the
// separation between processing cycles and display updates.
type EmulationLoop struct {
	emulator interfaces.Emulator

	config *EmulationLoopConfig

	tickLoopRunning bool
	drawLoopRunning bool
	stop            bool
	pause           bool
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
func NewEmulationLoop(config EmulationLoopConfig) *EmulationLoop {
	if config.RefreshNanos <= 0 {
		config.RefreshNanos = 5 * 1_000_000
	}

	if config.DisplayFPS <= 0 {
		config.DisplayFPS = 10
	}

	return &EmulationLoop{
		config:          &config,
		tickLoopRunning: false,
		drawLoopRunning: false,
		stop:            true,
	}
}

func (e *EmulationLoop) SetEmulator(emulator interfaces.Emulator) {
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
func (e *EmulationLoop) SetPanicHandler(handler func(loopType string, panicData any) bool) {
	e.config.PanicHandler = handler
}

// IsPaused checks if the emulation loop is currently paused.
// Returns true if the loop is paused.
func (e *EmulationLoop) IsPaused() bool {
	return e.pause
}

// IsRunning checks if the emulation loop is currently running.
// Returns true if the loop is not stopped.
func (e *EmulationLoop) IsRunning() bool {
	return e.drawLoopRunning || e.tickLoopRunning
}

// IsStopping checks if the emulation loop is in the process of stopping.
func (e *EmulationLoop) IsStopping() bool {
	return e.stop && e.IsRunning()
}

// Start begins the emulation loop with the provided handlers.
// It runs the emulation at the configured speed and manages the timing
// for both CPU cycles and display updates.
//
// Returns:
//   - A StepContext that can be used to control and monitor the emulation
func (e *EmulationLoop) Start() *common.StepContext {
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

// Stop stops the emulation loop.
func (e *EmulationLoop) Stop() {
	e.stop = true
}

func (e *EmulationLoop) Resume() {
	e.pause = false
}

func (e *EmulationLoop) Pause() {
	e.pause = true
}

// handlePanic triggers the execution of a handler before panicking.
//
// Parameters:
//   - loopType: The type of loop that panicked (for logging purposes)
//   - r: The recovered panic value
func (e *EmulationLoop) handlePanic(loopType string, r any) {
	e.Stop()
	if e.config.PanicHandler != nil {
		if !e.config.PanicHandler(loopType, r) {
			panic(r)
		}
	}
}

// executeLoop runs the main emulation loop that processes CPU cycles.
//
// Parameters:
//   - context: The step context for emulation state
func (e *EmulationLoop) executeLoop(context *common.StepContext) {
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
func (e *EmulationLoop) executeDraw(context *common.StepContext) {
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
