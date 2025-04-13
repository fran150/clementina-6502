package ui

import (
	"fmt"
	"time"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/computers"
	"github.com/rivo/tview"
)

// SpeedWindow represents a UI component that displays the emulation speed metrics.
// It shows the current execution speed, target speed, and performance statistics.
type SpeedWindow struct {
	text            *tview.TextView
	previousT       int64
	previousC       uint64
	config          *computers.EmulationLoopConfig
	showConfig      bool
	showConfigStart int64
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

func (s *SpeedWindow) IsConfigVisible() bool {
	return s.showConfig
}

func (s *SpeedWindow) Clear() {
	s.text.Clear()
}

func (s *SpeedWindow) Draw(context *common.StepContext) {
	if s.showConfig {
		if context.T-s.showConfigStart > (int64(time.Second) * 3) {
			s.showConfig = false
		}

		fmt.Fprintf(s.text, "[yellow]TGT: %0.8f Mhz", s.config.TargetSpeedMhz)
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

func (d *SpeedWindow) GetDrawArea() tview.Primitive {
	return d.text
}

func (d *SpeedWindow) ShowConfig(context *common.StepContext) {
	d.showConfig = true
	d.showConfigStart = context.T
}
