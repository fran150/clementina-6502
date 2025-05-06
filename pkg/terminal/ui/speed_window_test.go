package ui

import (
	"fmt"
	"testing"
	"time"

	"github.com/fran150/clementina-6502/pkg/common"
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

func TestSpeedWindowDraw(t *testing.T) {
	config := &computers.EmulationLoopConfig{
		SkipCycles: 0,
		DisplayFps: 10,
	}

	// Create a new SpeedWindow instance
	context := common.NewStepContext()
	sw := NewSpeedWindow(config)

	// Initialize by drawing first frame in zero
	sw.Draw(&context)

	// Simulate ~1 Mhz speed
	time.Sleep(1 * time.Second)
	context.Cycle = 1000000
	sw.Draw(&context)

	// Validate it's showing the speed
	text := sw.text.GetText(false)
	assert.Equal(t, fmt.Sprintf("[white]%0.8f Mhz", sw.currentSpeed), text)
	assert.InDelta(t, 1.0, sw.currentSpeed, 0.10)

	// Clear and call the show config command
	sw.Clear()
	sw.ShowConfig(&context)
	assert.True(t, sw.IsConfigVisible())

	// Simulate a frame and validate that now the test is showing the config along with current speed
	sw.Draw(&context)
	text = sw.text.GetText(false)
	assert.Equal(t, fmt.Sprintf("[white]%0.4f [yellow]DLY: %07d", sw.currentSpeed, sw.config.SkipCycles), text)

	// Wait for 3 seconds to hide the config
	time.Sleep(3 * time.Second)
	sw.Draw(&context)
	assert.False(t, sw.IsConfigVisible())

	// Clear and draw again with the config hidden
	sw.Clear()
	time.Sleep(100 * time.Millisecond)
	sw.Draw(&context)

	// Validate that the text is showing the speed again
	text = sw.text.GetText(false)
	assert.Equal(t, fmt.Sprintf("[white]%0.8f Mhz", sw.currentSpeed), text)
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
