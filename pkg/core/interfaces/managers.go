package interfaces

// NavigationManager defines the interface for managing window navigation.
type NavigationManager interface {
	// NavigateTo switches to the specified window.
	NavigateTo(key string)

	// GoBack returns to the previous window.
	GoBack()

	// GetCurrent returns the currently active window key.
	GetCurrent() string

	// PushToHistory adds the current window to history and navigates to new window.
	PushToHistory(key string)
}

// ComputerState represents the current state of a computer system.
type ComputerState struct {
	Paused    bool // Indicates if the computer is currently paused
	Stepping  bool // Indicates if the computer is stepping through cycles
	Resetting bool // Indicates if the computer is in the process of resetting
}

// StateManager defines the contract for managing computer state.
type StateManager interface {
	// Pause stops the execution of the computer.
	// The computer will remain paused until Resume or Step is called.
	Pause()

	// Resume continues the execution of the computer after being paused.
	Resume()

	// Step signals that the computer should step through one cycle.
	// This allows for step-by-step debugging of the computer's operation.
	// After executing the step, the flag must be cleared by calling ClearStepping.
	Step()

	// ClearStepping clears the stepping state of the computer.
	// This is typically called after a step has been executed to avoid the computer
	// from stepping again unintentionally.
	ClearStepping()

	// Reset triggers a reset of the computer. To correctly reset a 6502, the reset
	// signal must be held for at least 3 clock cycles.
	// In real hardware this is equivalent to pressing the reset button.
	Reset()

	// Unreset clears the resetting state of the computer.
	// This should be called after the reset signal has been held for the required
	// duration to ensure the computer is ready to resume normal operation.
	Unreset()

	// IsPaused checks if the computer is currently paused.
	IsPaused() bool

	// IsStepping checks if the computer is currently stepping through cycles.
	IsStepping() bool

	// IsResetting checks if the computer is resetting.
	IsResetting() bool

	// GetState returns a copy of the current computer state.
	GetState() ComputerState
}
