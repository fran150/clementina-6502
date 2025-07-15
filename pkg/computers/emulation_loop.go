// Package computers provides computer system implementations and emulation control.
package computers

import (
	"time"

	"github.com/fran150/clementina-6502/pkg/common"
)

// EmulationLoopConfig contains settings that control the emulation speed and display refresh rate.
type EmulationLoopConfig struct {
	// TargetSpeedMhz specifies the target CPU speed in MHz
	TargetSpeedMhz float64

	// DisplayFps specifies the target frames per second for display updates
	DisplayFps int
}

// EmulationLoopHandlers contains callback functions for the main emulation loop.
type EmulationLoopHandlers struct {
	// Tick is called for each CPU cycle to update the emulated system state
	Tick func(context *common.StepContext)

	// Draw is called at the specified FPS rate to update the display
	Draw func(context *common.StepContext)
}

// EmulationLoop manages the timing and execution of the emulation cycle.
// It ensures the emulation runs at the specified speed and handles the
// separation between processing cycles and display updates.
type EmulationLoop struct {
	config  *EmulationLoopConfig
	context *common.StepContext
}

// NewEmulationLoop creates a new emulation loop with the specified configuration.
func NewEmulationLoop(config *EmulationLoopConfig) *EmulationLoop {
	return &EmulationLoop{
		config: config,
	}
}

// GetConfig returns the current emulation loop configuration.
// This includes target speed and display refresh rate settings.
func (e *EmulationLoop) GetConfig() *EmulationLoopConfig {
	return e.config
}

// Start begins the emulation loop with the provided handlers.
// It runs the emulation at the configured speed and manages the timing
// for both CPU cycles and display updates.
//
// Parameters:
//   - handlers: The callback functions to use during emulation
//
// Returns:
//   - A StepContext that can be used to control and monitor the emulation,
//     or nil if required handlers are missing
func (e *EmulationLoop) Start(handlers EmulationLoopHandlers) *common.StepContext {
	if handlers.Tick == nil || handlers.Draw == nil {
		return nil
	}

	context := common.NewStepContext()
	e.context = &context

	go e.executeLoop(e.context, handlers)
	go e.executeDraw(e.context, handlers)

	return e.context
}

func (e *EmulationLoop) executeLoop(context *common.StepContext, handlers EmulationLoopHandlers) {
	var lastTPSExecuted, targetTPSNano int64

	for !context.Stop {
		targetTPSNano = int64(float64(time.Microsecond) / e.config.TargetSpeedMhz)

		if (context.T - lastTPSExecuted) > targetTPSNano {
			lastTPSExecuted = context.T
			handlers.Tick(context)
			context.NextCycle()
		} else {
			context.SkipCycle()
		}
	}
}

// This function is called to update the display at the specified FPS rate.
// By calling hanldlers.Draw
func (e *EmulationLoop) executeDraw(context *common.StepContext, handlers EmulationLoopHandlers) {
	ticker := time.NewTicker(time.Second / time.Duration(e.config.DisplayFps))
	defer ticker.Stop()

	for !context.Stop {
		<-ticker.C
		handlers.Draw(context)
	}
}
