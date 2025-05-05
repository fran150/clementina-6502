// Package computers provides computer system implementations and emulation control.
package computers

import (
	"time"

	"github.com/fran150/clementina-6502/pkg/common"
)

// EmulationLoopConfig contains settings that control the emulation speed and display refresh rate.
type EmulationLoopConfig struct {
	SkipCycles int64

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

	skippedCycles int64
}

// NewEmulationLoop creates a new emulation loop with the specified configuration.
func NewEmulationLoop(config *EmulationLoopConfig) *EmulationLoop {
	// TODO: This might not be the best way to validate this
	if config.SkipCycles < 0 {
		config.SkipCycles = 0
	}

	return &EmulationLoop{
		config:        config,
		skippedCycles: 0,
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
	for !context.Stop {
		if e.config.SkipCycles > 0 && e.skippedCycles < e.config.SkipCycles && context.Cycle != 0 {
			e.skippedCycles++
			continue
		}

		e.skippedCycles = 0
		handlers.Tick(context)
		context.NextCycle()
	}
}

// This function is called to update the display at the specified FPS rate.
// By calling hanldlers.Draw
func (e *EmulationLoop) executeDraw(context *common.StepContext, handlers EmulationLoopHandlers) {
	frameTime := 1000 / e.config.DisplayFps
	ticker := time.NewTicker(time.Duration(frameTime) * time.Millisecond)
	defer ticker.Stop()

	for !context.Stop {
		<-ticker.C
		handlers.Draw(context)
	}
}
