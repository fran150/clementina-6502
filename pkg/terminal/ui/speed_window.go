package ui

import (
	"fmt"
	"time"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/computers"
	"github.com/rivo/tview"
)

var beginning = time.Now()

// SpeedWindow represents a UI component that displays the emulation speed metrics.
// It shows the current execution speed, target speed, and performance statistics.
type SpeedWindow struct {
	text            *tview.TextView
	previousT       int64
	previousC       uint64
	config          *computers.EmulationLoopConfig
	showConfig      bool
	showConfigStart int64
	currentSpeed    float64
}

// NewSpeedWindow creates a new emulation speed display window.
// It initializes the UI component and connects it to the provided emulation configuration.
//
// Parameters:
//   - config: The emulation loop configuration to monitor
//
// Returns:
//   - A pointer to the initialized SpeedWindow
func NewSpeedWindow(config *computers.EmulationLoopConfig) *SpeedWindow {
	text := tview.NewTextView()
	text.SetTextAlign(tview.AlignCenter).
		SetScrollable(false).
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle("Speed")

	return &SpeedWindow{
		text:            text,
		config:          config,
		showConfig:      false,
		showConfigStart: 0,
	}
}

// IsConfigVisible returns whether the configuration display is currently visible.
//
// Returns:
//   - true if the configuration is being displayed, false otherwise
func (s *SpeedWindow) IsConfigVisible() bool {
	return s.showConfig
}

// Clear resets the speed window, removing all text content.
func (s *SpeedWindow) Clear() {
	s.text.Clear()
}

// Draw updates the speed window with the current emulation speed metrics.
// It displays either the configuration or the current performance statistics.
//
// Parameters:
//   - context: The current step context containing timing information
func (s *SpeedWindow) Draw(context *common.StepContext) {
	currentTime := int64(time.Since(beginning))

	if s.previousT != 0 {
		cycles := context.Cycle - s.previousC
		elapsedMicro := (float64(currentTime) - float64(s.previousT)) / float64(time.Microsecond)

		s.currentSpeed = (float64(cycles) / elapsedMicro)

		if s.showConfig {
			if currentTime-s.showConfigStart > (int64(time.Second) * 3) {
				s.showConfig = false
			}

			fmt.Fprintf(s.text, "[white]%0.4f [yellow]DLY: %07d", s.currentSpeed, s.config.SkipCycles)
		} else {
			fmt.Fprintf(s.text, "[white]%0.8f Mhz", s.currentSpeed)
		}

	}

	s.previousT = currentTime
	s.previousC = context.Cycle
}

// GetDrawArea returns the primitive that represents this window in the UI.
// This is used by the layout manager to position and render the window.
//
// Returns:
//   - The tview primitive for this window
func (d *SpeedWindow) GetDrawArea() tview.Primitive {
	return d.text
}

// ShowConfig displays the emulation configuration in the speed window.
// The configuration will be shown for a few seconds before returning to speed display.
//
// Parameters:
//   - context: The current step context containing timing information
func (d *SpeedWindow) ShowConfig(context *common.StepContext) {
	d.showConfig = true
	d.showConfigStart = int64(time.Since(beginning))
}
