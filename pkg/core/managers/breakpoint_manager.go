package managers

import (
	"slices"

	"github.com/fran150/clementina-6502/pkg/core"
)

// breakpointManager manages breakpoints for debugging purposes.
// It provides functionality to add, remove, and check breakpoints at specific addresses.
type breakpointManager struct {
	breakpoints []uint16
}

// newBreakpointManager creates a new breakpoint manager.
//
// Returns:
//   - A pointer to the initialized BreakpointManager
func newBreakpointManager() *breakpointManager {
	return &breakpointManager{
		breakpoints: make([]uint16, 0),
	}
}

// NewBreakpointManager creates a new breakpoint manager.
//
// Returns:
//   - A pointer to the initialized BreakpointManager
func NewBreakpointManager() core.BreakpointManager {
	return newBreakpointManager()
}

// AddBreakpoint adds a breakpoint at the specified address.
// If a breakpoint already exists at the address, it won't be added again.
//
// Parameters:
//   - address: The address where the breakpoint should be set
func (bm *breakpointManager) AddBreakpoint(address uint16) {
	if !bm.HasBreakpoint(address) {
		bm.breakpoints = append(bm.breakpoints, address)
	}
}

// RemoveBreakpoint removes a breakpoint at the specified address.
// If no breakpoint exists at the address, this method has no effect.
//
// Parameters:
//   - address: The address where the breakpoint should be removed
func (bm *breakpointManager) RemoveBreakpoint(address uint16) {
	for i, bp := range bm.breakpoints {
		if bp == address {
			bm.breakpoints = slices.Delete(bm.breakpoints, i, i+1)
			break
		}
	}
}

// RemoveBreakpointByIndex removes a breakpoint at the specified index.
// If the index is out of bounds, this method has no effect.
//
// Parameters:
//   - index: The index of the breakpoint to remove
func (bm *breakpointManager) RemoveBreakpointByIndex(index int) {
	if index >= 0 && index < len(bm.breakpoints) {
		bm.breakpoints = slices.Delete(bm.breakpoints, index, index+1)
	}
}

// HasBreakpoint checks if a breakpoint exists at the specified address.
//
// Parameters:
//   - address: The address to check
//
// Returns:
//   - true if a breakpoint exists at the address, false otherwise
func (bm *breakpointManager) HasBreakpoint(address uint16) bool {
	return slices.Contains(bm.breakpoints, address)
}

// GetBreakpoints returns a copy of all breakpoint addresses.
//
// Returns:
//   - A slice containing all breakpoint addresses
func (bm *breakpointManager) GetBreakpoints() []uint16 {
	result := make([]uint16, len(bm.breakpoints))
	copy(result, bm.breakpoints)
	return result
}

// GetBreakpointCount returns the number of active breakpoints.
//
// Returns:
//   - The number of breakpoints currently set
func (bm *breakpointManager) GetBreakpointCount() int {
	return len(bm.breakpoints)
}

// ClearAllBreakpoints removes all breakpoints.
func (bm *breakpointManager) ClearAllBreakpoints() {
	bm.breakpoints = bm.breakpoints[:0]
}
