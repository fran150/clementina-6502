package managers

import "github.com/fran150/clementina-6502/pkg/core/interfaces"

// DefaultStateManager manages the state of a computer system.
type DefaultStateManager struct {
	state interfaces.ComputerState
}

// NewStateManager creates a new state manager with default state.
//
// Returns:
//   - A pointer to the initialized StateManager
func NewStateManager() *DefaultStateManager {
	return &DefaultStateManager{
		state: interfaces.ComputerState{
			Stopped:   false,
			Paused:    false,
			Stepping:  false,
			Resetting: false,
		},
	}
}

// Pause stops the execution of the computer.
// The computer will remain paused until Resume or Step is called.
func (sm *DefaultStateManager) Pause() {
	sm.state.Paused = true
}

// Resume continues the execution of the computer after being paused.
func (sm *DefaultStateManager) Resume() {
	sm.state.Paused = false
}

// Step signals that the computer should step through one cycle.
// This allows for step-by-step debugging of the computer's operation.
// After executing the step, the flag must be cleared by calling ClearStepping.
func (sm *DefaultStateManager) Step() {
	sm.state.Stepping = true
}

// ClearStepping clears the stepping state of the computer.
// This is typically called after a step has been executed to avoid the computer
// from stepping again unintentionally.
func (sm *DefaultStateManager) ClearStepping() {
	sm.state.Stepping = false
}

// Reset triggers a reset of the computer. To correctly reset a 6502, the reset
// signal must be held for at least 3 clock cycles.
// In real hardware this is equivalent to pressing the reset button.
func (sm *DefaultStateManager) Reset() {
	sm.state.Resetting = true
}

// Unreset clears the resetting state of the computer.
// This should be called after the reset signal has been held for the required
// duration to ensure the computer is ready to resume normal operation.
func (sm *DefaultStateManager) Unreset() {
	sm.state.Resetting = false
}

// IsPaused checks if the computer is currently paused.
func (sm *DefaultStateManager) IsPaused() bool {
	return sm.state.Paused
}

// IsStepping checks if the computer is currently stepping through cycles.
func (sm *DefaultStateManager) IsStepping() bool {
	return sm.state.Stepping
}

// IsResetting checks if the computer is resetting.
func (sm *DefaultStateManager) IsResetting() bool {
	return sm.state.Resetting
}

// Stop stops the computer completely.
func (sm *DefaultStateManager) Stop() {
	sm.state.Stopped = true
}

// IsStopped checks if the computer is currently stopped.
func (sm *DefaultStateManager) IsStopped() bool {
	return sm.state.Stopped
}
