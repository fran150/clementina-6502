package computers

// ComputerState represents the current state of a computer system.
type ComputerState struct {
	paused    bool // Indicates if the computer is currently paused
	stepping  bool // Indicates if the computer is stepping through cycles
	resetting bool // Indicates if the computer is in the process of resetting
}

// StateManager manages the state of a computer system.
type StateManager struct {
	state *ComputerState
}

// NewStateManager creates a new state manager with default state.
//
// Returns:
//   - A pointer to the initialized StateManager
func NewStateManager() *StateManager {
	return &StateManager{
		state: &ComputerState{
			paused:    false,
			stepping:  false,
			resetting: false,
		},
	}
}

// Pause stops the execution of the computer.
// The computer will remain paused until Resume or Step is called.
func (sm *StateManager) Pause() {
	sm.state.paused = true
}

// Resume continues the execution of the computer after being paused.
func (sm *StateManager) Resume() {
	sm.state.paused = false
}

// Step signals that the computer should step through one cycle.
// This allows for step-by-step debugging of the computer's operation.
// After executing the step, the flag must be cleared by calling ClearStepping.
func (sm *StateManager) Step() {
	sm.state.stepping = true
}

// ClearStepping clears the stepping state of the computer.
// This is typically called after a step has been executed to avoid the computer
// from stepping again unintentionally.
func (sm *StateManager) ClearStepping() {
	sm.state.stepping = false
}

// Reset triggers a reset of the computer. To correctly reset a 6502, the reset
// signal must be held for at least 3 clock cycles.
// In real hardware this is equivalent to pressing the reset button.
func (sm *StateManager) Reset() {
	sm.state.resetting = true
}

// Unreset clears the resetting state of the computer.
// This should be called after the reset signal has been held for the required
// duration to ensure the computer is ready to resume normal operation.
func (sm *StateManager) Unreset() {
	sm.state.resetting = false
}

// IsPaused checks if the computer is currently paused.
func (sm *StateManager) IsPaused() bool {
	return sm.state.paused
}

// IsStepping checks if the computer is currently stepping through cycles.
func (sm *StateManager) IsStepping() bool {
	return sm.state.stepping
}

// IsResetting checks if the computer is resetting.
func (sm *StateManager) IsResetting() bool {
	return sm.state.resetting
}

// GetState returns a copy of the current computer state.
func (sm *StateManager) GetState() ComputerState {
	return *sm.state
}
