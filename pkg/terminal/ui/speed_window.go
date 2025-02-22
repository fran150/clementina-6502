package ui

import (
	"fmt"
	"time"

	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/rivo/tview"
)

type SpeedWindow struct {
	text      *tview.TextView
	previousT int64
	previousC uint64
}

func NewSpeedWindow() *SpeedWindow {
	text := tview.NewTextView()
	text.SetTextAlign(tview.AlignCenter).
		SetScrollable(false).
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle("Speed")

	return &SpeedWindow{
		text: text,
	}
}

func (s *SpeedWindow) Clear() {
	s.text.Clear()
}

func (s *SpeedWindow) Draw(context *common.StepContext) {
	if s.previousT != 0 {
		cycles := context.Cycle - s.previousC
		elapsedMicro := (context.T - s.previousT) / int64(time.Microsecond)

		mhz := (float64(cycles) / float64(elapsedMicro))

		fmt.Fprintf(s.text, "%0.8f Mhz", mhz)
	}

	s.previousT = context.T
	s.previousC = context.Cycle
}

func (d *SpeedWindow) GetDrawArea() *tview.TextView {
	return d.text
}
