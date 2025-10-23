package ui

import (
	"fmt"
	"time"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/core"
	"github.com/rivo/tview"
)

// SpeedWindow represents a UI component that displays the emulation speed metrics.
// It shows the current execution speed, target speed, and performance statistics.
type SpeedWindow struct {
	text            *tview.TextView
	previousT       int64
	previousC       uint64
	speedController core.SpeedController
	showConfig      bool
	showConfigStart int64
}

// NewSpeedWindow creates a new emulation speed display window.
// It initializes the UI component and connects it to the provided speed controller.
//
// Parameters:
//   - speedController: The speed controller interface for managing emulation speed
//
// Returns:
//   - A pointer to the initialized SpeedWindow
func NewSpeedWindow(speedController core.SpeedController) *SpeedWindow {
	text := tview.NewTextView()
	text.SetTextAlign(tview.AlignCenter).
		SetScrollable(false).
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle("Speed")

	return &SpeedWindow{
		text:            text,
		speedController: speedController,
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
	if s.showConfig {
		if s.showConfigStart == 0 {
			s.showConfigStart = context.T
		}

		if context.T-s.showConfigStart > (int64(time.Second) * 3) {
			s.showConfig = false
			s.showConfigStart = 0
		}

		fmt.Fprintf(s.text, "[yellow]TGT: %0.8f Mhz", s.speedController.GetTargetSpeed())
	} else {
		if s.previousT != 0 {
			cycles := context.Cycle - s.previousC
			elapsedMicro := (float64(context.T) - float64(s.previousT)) / float64(time.Microsecond)

			mhz := (float64(cycles) / elapsedMicro)

			fmt.Fprintf(s.text, "[white]%0.8f Mhz", mhz)
		}

		s.previousT = context.T
		s.previousC = context.Cycle
	}
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
func (d *SpeedWindow) ShowConfig() {
	d.showConfig = true
}
