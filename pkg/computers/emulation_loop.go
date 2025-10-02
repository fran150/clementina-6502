package computers

import (
	"time"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/core/controllers"
	"github.com/fran150/clementina-6502/pkg/core/interfaces"
)

// EmulationLoopConfig contains settings that control the display refresh rate.
type EmulationLoopConfig struct {
	// DisplayFps specifies the target frames per second for display updates
	DisplayFps int
}

// EmulationLoop manages the timing and execution of the emulation cycle.
// It ensures the emulation runs at the specified speed and handles the
// separation between processing cycles and display updates.
type EmulationLoop struct {
	emulator        interfaces.Emulator
	renderer        interfaces.Renderer
	speedController interfaces.SpeedController
	config          *EmulationLoopConfig

	panicHandler func(loopType string, panicData any) bool

	tickLoopRunning bool
	drawLoopRunning bool

	stop bool
}

// NewEmulationLoop creates a new emulation loop with the specified components.
//
// Parameters:
//   - emulator: The emulator to run
//   - renderer: The renderer for display updates
//   - speedController: The speed controller for timing
//   - config: Configuration settings for the emulation loop
//
// Returns:
//   - A pointer to the initialized EmulationLoop
func NewEmulationLoop(emulator interfaces.Emulator, renderer interfaces.Renderer, speedController interfaces.SpeedController, config *EmulationLoopConfig) *EmulationLoop {
	return &EmulationLoop{
		emulator:        emulator,
		renderer:        renderer,
		speedController: speedController,
		config:          config,
		panicHandler:    nil,
		stop:            true,
		tickLoopRunning: false,
		drawLoopRunning: false,
	}
}

// GetConfig returns the current emulation loop configuration.
// This includes target speed and display refresh rate settings.
func (e *EmulationLoop) GetConfig() *EmulationLoopConfig {
	return e.config
}

// SetPanicHandler sets the panic handler. This function will be called for cleanup if
// any of the emulation loops fail.
//
// Parameters:
//   - handler: Function to handle panics, returns true if panic should be suppressed
func (e *EmulationLoop) SetPanicHandler(handler func(loopType string, panicData any) bool) {
	e.panicHandler = handler
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
	context := common.NewStepContext()

	e.stop = false

	go e.executeLoop(&context)
	go e.executeDraw(&context)

	return &context
}

// Stop stops the emulation loop.
func (e *EmulationLoop) Stop() {
	e.stop = true
}

// handlePanic triggers the execution of a handler before panicking.
//
// Parameters:
//   - loopType: The type of loop that panicked (for logging purposes)
//   - r: The recovered panic value
func (e *EmulationLoop) handlePanic(loopType string, r any) {
	e.Stop()
	if e.panicHandler != nil {
		if !e.panicHandler(loopType, r) {
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
	var lastSpeedCheck uint64
	var currentSpeed float64 = e.speedController.GetTargetSpeed()

	e.tickLoopRunning = true

	// Try to cast to DefaultSpeedController for performance optimization
	var defaultSpeedController *controllers.DefaultSpeedController
	if dsc, ok := e.speedController.(*controllers.DefaultSpeedController); ok {
		defaultSpeedController = dsc
	}

	for !e.stop {
		// Only check speed every 1000 cycles to reduce overhead
		if (context.Cycle - lastSpeedCheck) > 1000 {
			if defaultSpeedController != nil {
				// Use cached nanoseconds per cycle for better performance
				targetTPSNano = int64(defaultSpeedController.GetNanosPerCycle())
			} else {
				// Fallback to interface method
				currentSpeed = e.speedController.GetTargetSpeed()
				targetTPSNano = int64(float64(time.Microsecond) / currentSpeed)
			}
			lastSpeedCheck = context.Cycle
		}

		if (context.T - lastTPSExecuted) > targetTPSNano {
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

	ticker := time.NewTicker(time.Second / time.Duration(e.config.DisplayFps))
	defer ticker.Stop()

	e.drawLoopRunning = true

	for !e.stop {
		<-ticker.C
		e.renderer.Draw(context)
	}
}
