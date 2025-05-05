package ui

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/computers"
	"github.com/stretchr/testify/assert"
)

func TestNewSpeedWindow(t *testing.T) {
	config := &computers.EmulationLoopConfig{
		SkipCycles: 1.0,
		DisplayFps: 10,
	}

	sw := NewSpeedWindow(config)

	assert.NotNil(t, sw)
	assert.NotNil(t, sw.text)
	assert.Equal(t, int64(0), sw.previousT)
	assert.Equal(t, uint64(0), sw.previousC)
}

func TestSpeedWindow_Clear(t *testing.T) {
	config := &computers.EmulationLoopConfig{
		SkipCycles: 1.0,
		DisplayFps: 10,
	}

	sw := NewSpeedWindow(config)
	// Set some initial values
	sw.previousT = 1000
	sw.previousC = 2000

	sw.Clear()
	// Clear should only clear the text view, not reset the previous values
	assert.Equal(t, int64(1000), sw.previousT)
	assert.Equal(t, uint64(2000), sw.previousC)
}

func TestSpeedWindow_GetDrawArea(t *testing.T) {
	config := &computers.EmulationLoopConfig{
		SkipCycles: 1.0,
		DisplayFps: 10,
	}

	sw := NewSpeedWindow(config)
	primitive := sw.GetDrawArea()

	assert.NotNil(t, primitive)
	assert.Equal(t, sw.text, primitive)
}
