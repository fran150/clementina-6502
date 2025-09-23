package computers

import (
	"github.com/fran150/clementina-6502/pkg/common"
)

// BaseComputer is the base structure for a computer emulation based on the 6502
// architecture used by Ben Eater's 6502 computer and the Clementina 6502 project.
// It provides methods to control the emulation loop, pause, step, reset, and manage
// the emulation speed.
type BaseComputer struct {
	pause       bool // Indicates if the computer is currently paused
	step        bool // Indicates if the computer is stepping through cycles
	isResetting bool // Indicates if the computer is in the process of resetting

	loop *EmulationLoop
}

// NewBaseComputer creates a new instance of BaseComputer with the provided
// emulation loop and tview application.
//
// Parameters:
//   - loop: The emulation loop to manage computer execution
//   - tvApp: The tview application for the console interface
//
// Returns:
//   - A pointer to the initialized BaseComputer
func NewBaseComputer(loop *EmulationLoop) *BaseComputer {
	return &BaseComputer{
		pause:       false,
		step:        false,
		isResetting: false,
		loop:        loop,
	}
}

// Run starts the emulation loop
func (c *BaseComputer) Run() *common.StepContext {
	context := c.loop.Start()

	return context
}

// Stop stops computer execution
func (c *BaseComputer) Stop() {
	c.loop.Stop()
}

// Pause stops the execution of the computer.
// The computer will remain paused until Resume or Step is called.
func (c *BaseComputer) Pause() {
	c.pause = true
}

// Resume continues the execution of the computer after being paused.
func (c *BaseComputer) Resume() {
	c.pause = false
}

// Step signals that the computer should step through one cycle.
// This allows for step-by-step debugging of the computer's operation.
// After executing the step, the flag must be cleared by calling ClearStepping.
func (c *BaseComputer) Step() {
	c.step = true
}

// ClearStepping clears the stepping state of the computer.
// This is typically called after a step has been executed to avoid the computer
// from stepping again unintentionally.
func (c *BaseComputer) ClearStepping() {
	c.step = false
}

// Reset triggers a reset of the computer. To correctly reset a 6502, the reset
// signal must be held for at least 3 clock cycles.
// In real hardware this is equivalent to pressing the reset button.
func (c *BaseComputer) Reset() {
	c.isResetting = true
}

// Unreset clears the resetting state of the computer.
// This should be called after the reset signal has been held for the required
// duration to ensure the computer is ready to resume normal operation.
func (c *BaseComputer) Unreset() {
	c.isResetting = false
}

// SpeedUp increases the emulation speed of the computer.
// It uses a non-linear scale for speeds below 0.5 MHz and a linear scale above.
func (c *BaseComputer) SpeedUp() {
	config := c.loop.GetConfig()
	currentSpeed := config.TargetSpeedMhz

	if currentSpeed < 0.5 {
		// Non-linear increase below 0.5 MHz
		// Increase by 20% of current speed
		increase := currentSpeed * 0.2
		if increase < 0.000001 {
			// Ensure minimum increase to avoid tiny increments
			increase = 0.000001
		}
		config.TargetSpeedMhz += increase
	} else {
		// Linear increase above 0.5 MHz
		config.TargetSpeedMhz += 0.1
	}
}

// SpeedDown decreases the emulation speed of the computer.
// It uses a linear scale for speeds above 0.5 MHz and a non-linear scale below,
// ensuring the speed never goes below a minimum threshold.
func (c *BaseComputer) SpeedDown() {
	config := c.loop.GetConfig()
	currentSpeed := config.TargetSpeedMhz

	if currentSpeed > 0.5 {
		// Linear reduction above 0.5 MHz
		config.TargetSpeedMhz -= 0.1
	} else {
		// Non-linear reduction below 0.5 MHz to avoid reaching 0
		// This will reduce by a fraction of the current speed
		reduction := currentSpeed * 0.2
		if reduction < 0.000001 {
			// Ensure minimum reduction to avoid tiny decrements
			reduction = 0.000001
		}
		config.TargetSpeedMhz -= reduction
	}
}

// Loop returns the current emulation loop.
func (c *BaseComputer) Loop() *EmulationLoop {
	return c.loop
}

// IsPaused checks if the computer is currently paused.
func (c *BaseComputer) IsPaused() bool {
	return c.pause
}

// IsStepping checks if the computer is currently stepping through cycles.
func (c *BaseComputer) IsStepping() bool {
	return c.step
}

// IsResetting checks if the computer is resetting.
func (c *BaseComputer) IsResetting() bool {
	return c.isResetting
}
