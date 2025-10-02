package computers

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/core/interfaces"
	"github.com/fran150/clementina-6502/pkg/core/managers"
)

// ComputerSystem orchestrates all computer components using composition.
// It provides a clean interface for controlling computer execution while
// maintaining separation of concerns between different components.
type ComputerSystem struct {
	core            interfaces.ComputerCore
	loop            *EmulationLoop
	stateManager    *managers.StateManager
	speedController interfaces.SpeedController

	context *common.StepContext
}

// NewComputerSystem creates a new computer system with the specified components.
//
// Parameters:
//   - core: The computer core that handles emulation and rendering
//   - speedController: The speed controller for managing emulation speed
//   - config: Configuration for the emulation loop
//
// Returns:
//   - A pointer to the initialized ComputerSystem
func NewComputerSystem(core interfaces.ComputerCore, speedController interfaces.SpeedController, config *EmulationLoopConfig) *ComputerSystem {
	stateManager := managers.NewStateManager()
	loop := NewEmulationLoop(core, core, speedController, config)

	return &ComputerSystem{
		core:            core,
		loop:            loop,
		stateManager:    stateManager,
		speedController: speedController,
	}
}

// Start begins computer execution and returns the execution context.
func (cs *ComputerSystem) Start() (*common.StepContext, error) {
	cs.context = cs.loop.Start()
	return cs.context, nil
}

// Stop stops the computer system execution.
func (cs *ComputerSystem) Stop() {
	cs.loop.Stop()
}

// Pause stops the execution of the computer.
func (cs *ComputerSystem) Pause() {
	cs.stateManager.Pause()
}

// Resume continues the execution of the computer after being paused.
func (cs *ComputerSystem) Resume() {
	cs.stateManager.Resume()
}

// Reset triggers a reset of the computer.
func (cs *ComputerSystem) Reset() {
	cs.stateManager.Reset()
}

// IsRunning checks if the computer is currently running.
func (cs *ComputerSystem) IsRunning() bool {
	return cs.loop.IsRunning()
}

// IsPaused checks if the computer is currently paused.
func (cs *ComputerSystem) IsPaused() bool {
	return cs.stateManager.IsPaused()
}

// SpeedUp increases the emulation speed.
func (cs *ComputerSystem) SpeedUp() {
	cs.speedController.SpeedUp()
}

// SpeedDown decreases the emulation speed.
func (cs *ComputerSystem) SpeedDown() {
	cs.speedController.SpeedDown()
}

// GetTargetSpeed returns the current target speed in MHz.
func (cs *ComputerSystem) GetTargetSpeed() float64 {
	return cs.speedController.GetTargetSpeed()
}

// Step signals that the computer should step through one cycle.
func (cs *ComputerSystem) Step() {
	cs.stateManager.Step()
}

// ClearStepping clears the stepping state of the computer.
func (cs *ComputerSystem) ClearStepping() {
	cs.stateManager.ClearStepping()
}

// IsStepping checks if the computer is currently stepping through cycles.
func (cs *ComputerSystem) IsStepping() bool {
	return cs.stateManager.IsStepping()
}

// IsResetting checks if the computer is resetting.
func (cs *ComputerSystem) IsResetting() bool {
	return cs.stateManager.IsResetting()
}

// Unreset clears the resetting state of the computer.
func (cs *ComputerSystem) Unreset() {
	cs.stateManager.Unreset()
}

// GetStateManager returns the state manager for direct access if needed.
func (cs *ComputerSystem) GetStateManager() *managers.StateManager {
	return cs.stateManager
}

// GetSpeedController returns the speed controller for direct access if needed.
func (cs *ComputerSystem) GetSpeedController() interfaces.SpeedController {
	return cs.speedController
}

// GetTargetSpeedPtr returns a pointer to the current target speed in MHz.
func (cs *ComputerSystem) GetTargetSpeedPtr() *float64 {
	return cs.speedController.GetTargetSpeedPtr()
}

// GetEmulationLoop returns the emulation loop for direct access if needed.
func (cs *ComputerSystem) GetEmulationLoop() *EmulationLoop {
	return cs.loop
}
