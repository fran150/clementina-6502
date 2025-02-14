package beneater

import (
	"fmt"
	"time"

	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/rivo/tview"
)

type speedWindow struct {
	text      *tview.TextView
	previousT int64
	previousC uint64
}

func createSpeedWindow() *speedWindow {
	text := tview.NewTextView()
	text.SetTextAlign(tview.AlignCenter).
		SetScrollable(false).
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle("Speed")

	return &speedWindow{
		text: text,
	}
}

func (s *speedWindow) Clear() {
	s.text.Clear()
}

func (s *speedWindow) Draw(context *common.StepContext) {
	if s.previousT != 0 {
		cycles := context.Cycle - s.previousC
		elapsedMicro := (context.T - s.previousT) / int64(time.Microsecond)

		mhz := (float64(cycles) / float64(elapsedMicro))

		fmt.Fprintf(s.text, "%0.8f Mhz", mhz)
	}

	s.previousT = context.T
	s.previousC = context.Cycle
}
