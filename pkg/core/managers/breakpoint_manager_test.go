package managers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBreakpointManager(t *testing.T) {
	bm := NewDefaultBreakpointManager()

	assert.NotNil(t, bm)
	assert.Equal(t, 0, bm.GetBreakpointCount())
	assert.Empty(t, bm.GetBreakpoints())
}

func TestAddBreakpoint(t *testing.T) {
	bm := NewDefaultBreakpointManager()

	// Add first breakpoint
	bm.AddBreakpoint(0x1234)
	assert.Equal(t, 1, bm.GetBreakpointCount())
	assert.True(t, bm.HasBreakpoint(0x1234))

	// Add second breakpoint
	bm.AddBreakpoint(0x5678)
	assert.Equal(t, 2, bm.GetBreakpointCount())
	assert.True(t, bm.HasBreakpoint(0x5678))

	// Try to add duplicate - should not increase count
	bm.AddBreakpoint(0x1234)
	assert.Equal(t, 2, bm.GetBreakpointCount())
}

func TestRemoveBreakpoint(t *testing.T) {
	bm := NewDefaultBreakpointManager()

	// Add some breakpoints
	bm.AddBreakpoint(0x1234)
	bm.AddBreakpoint(0x5678)
	bm.AddBreakpoint(0x9ABC)

	// Remove existing breakpoint
	bm.RemoveBreakpoint(0x5678)
	assert.Equal(t, 2, bm.GetBreakpointCount())
	assert.False(t, bm.HasBreakpoint(0x5678))
	assert.True(t, bm.HasBreakpoint(0x1234))
	assert.True(t, bm.HasBreakpoint(0x9ABC))

	// Try to remove non-existing breakpoint - should not change anything
	bm.RemoveBreakpoint(0xDEAD)
	assert.Equal(t, 2, bm.GetBreakpointCount())
}

func TestRemoveBreakpointByIndex(t *testing.T) {
	bm := NewDefaultBreakpointManager()

	// Add some breakpoints
	bm.AddBreakpoint(0x1234)
	bm.AddBreakpoint(0x5678)
	bm.AddBreakpoint(0x9ABC)

	// Remove by valid index
	bm.RemoveBreakpointByIndex(1)
	assert.Equal(t, 2, bm.GetBreakpointCount())

	// Try to remove by invalid index - should not change anything
	bm.RemoveBreakpointByIndex(10)
	assert.Equal(t, 2, bm.GetBreakpointCount())

	bm.RemoveBreakpointByIndex(-1)
	assert.Equal(t, 2, bm.GetBreakpointCount())
}

func TestHasBreakpoint(t *testing.T) {
	bm := NewDefaultBreakpointManager()

	// Test with empty manager
	assert.False(t, bm.HasBreakpoint(0x1234))

	// Add breakpoint and test
	bm.AddBreakpoint(0x1234)
	assert.True(t, bm.HasBreakpoint(0x1234))
	assert.False(t, bm.HasBreakpoint(0x5678))
}

func TestGetBreakpoints(t *testing.T) {
	bm := NewDefaultBreakpointManager()

	// Test empty
	breakpoints := bm.GetBreakpoints()
	assert.Empty(t, breakpoints)

	// Add some breakpoints
	bm.AddBreakpoint(0x1234)
	bm.AddBreakpoint(0x5678)

	breakpoints = bm.GetBreakpoints()
	assert.Len(t, breakpoints, 2)
	assert.Contains(t, breakpoints, uint16(0x1234))
	assert.Contains(t, breakpoints, uint16(0x5678))

	// Verify it's a copy (modifying returned slice shouldn't affect manager)
	breakpoints[0] = 0xDEAD
	assert.True(t, bm.HasBreakpoint(0x1234))
	assert.False(t, bm.HasBreakpoint(0xDEAD))
}

func TestClearAllBreakpoints(t *testing.T) {
	bm := NewDefaultBreakpointManager()

	// Add some breakpoints
	bm.AddBreakpoint(0x1234)
	bm.AddBreakpoint(0x5678)
	bm.AddBreakpoint(0x9ABC)
	assert.Equal(t, 3, bm.GetBreakpointCount())

	// Clear all
	bm.ClearAllBreakpoints()
	assert.Equal(t, 0, bm.GetBreakpointCount())
	assert.Empty(t, bm.GetBreakpoints())
	assert.False(t, bm.HasBreakpoint(0x1234))
}
