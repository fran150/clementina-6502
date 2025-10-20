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

// BreakpointManager defines the contract for managing breakpoints in debugging.
type BreakpointManager interface {
	// AddBreakpoint adds a breakpoint at the specified address.
	// If a breakpoint already exists at the address, it won't be added again.
	AddBreakpoint(address uint16)

	// RemoveBreakpoint removes a breakpoint at the specified address.
	// If no breakpoint exists at the address, this method has no effect.
	RemoveBreakpoint(address uint16)

	// RemoveBreakpointByIndex removes a breakpoint at the specified index.
	// If the index is out of bounds, this method has no effect.
	RemoveBreakpointByIndex(index int)

	// HasBreakpoint checks if a breakpoint exists at the specified address.
	HasBreakpoint(address uint16) bool

	// GetBreakpoints returns a copy of all breakpoint addresses.
	GetBreakpoints() []uint16

	// GetBreakpointCount returns the number of active breakpoints.
	GetBreakpointCount() int

	// ClearAllBreakpoints removes all breakpoints.
	ClearAllBreakpoints()
}
