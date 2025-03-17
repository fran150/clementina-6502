package ui

import (
	"fmt"

	"github.com/fran150/clementina6502/pkg/common"
	"github.com/fran150/clementina6502/pkg/components/buses"
	"github.com/rivo/tview"
)

// BusInfo holds information about how to display a specific bus
type BusInfo struct {
	name     string
	bus      any // will hold either Bus[uint8] or Bus[uint16]
	bitWidth int // 8 or 16
}

type BusWindow struct {
	text     *tview.TextView
	busInfos []BusInfo
}

func NewBusWindow() *BusWindow {
	text := tview.NewTextView()
	text.SetScrollable(false).
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle("Bus Status")

	return &BusWindow{
		text:     text,
		busInfos: make([]BusInfo, 0),
	}
}

// AddBus8 adds an 8-bit bus to the window
func (w *BusWindow) AddBus8(name string, bus buses.Bus[uint8]) {
	w.busInfos = append(w.busInfos, BusInfo{
		name:     name,
		bus:      bus,
		bitWidth: 8,
	})
}

// AddBus16 adds a 16-bit bus to the window
func (w *BusWindow) AddBus16(name string, bus buses.Bus[uint16]) {
	w.busInfos = append(w.busInfos, BusInfo{
		name:     name,
		bus:      bus,
		bitWidth: 16,
	})
}

func (w *BusWindow) Clear() {
	w.text.Clear()
}

func drawBusLine(value uint16, bitPosition int) string {
	if value&(1<<bitPosition) != 0 {
		return "[green]━━━"
	}
	return "[red]───"
}

func (w *BusWindow) drawBusStatus(name string, value uint16, bitWidth int) {
	var format string
	if bitWidth == 8 {
		format = "[yellow]%s: [white]$%02X\n"
	} else {
		format = "[yellow]%s: [white]$%04X\n"
	}
	fmt.Fprintf(w.text, format, name, value)

	// Draw bit numbers with proper padding to align with bus lines
	fmt.Fprint(w.text, "[blue] ")
	for i := bitWidth - 1; i >= 0; i-- {
		if i >= 10 {
			fmt.Fprintf(w.text, "%2d  ", i) // Two digits + separator space
		} else {
			fmt.Fprintf(w.text, " %d  ", i) // One digit + separator space
		}
	}
	fmt.Fprint(w.text, "\n")

	// Draw top rail
	fmt.Fprint(w.text, "[white]┏")
	for i := 0; i < bitWidth; i++ {
		if i < bitWidth-1 {
			fmt.Fprint(w.text, "━━━┳")
		} else {
			fmt.Fprint(w.text, "━━━┓")
		}
	}
	fmt.Fprint(w.text, "\n")

	// Draw bus lines
	fmt.Fprint(w.text, "[white]┃")
	for i := bitWidth - 1; i >= 0; i-- {
		fmt.Fprint(w.text, drawBusLine(value, i))
		if i > 0 {
			fmt.Fprint(w.text, "[white]┃")
		} else {
			fmt.Fprint(w.text, "[white]┃")
		}
	}
	fmt.Fprint(w.text, "\n")

	// Draw bottom rail
	fmt.Fprint(w.text, "[white]┗")
	for i := 0; i < bitWidth; i++ {
		if i < bitWidth-1 {
			fmt.Fprint(w.text, "━━━┻")
		} else {
			fmt.Fprint(w.text, "━━━┛")
		}
	}
	fmt.Fprint(w.text, "\n\n")
}

func (w *BusWindow) Draw(context *common.StepContext) {
	w.Clear()

	for _, busInfo := range w.busInfos {
		var value uint16
		switch bus := busInfo.bus.(type) {
		case buses.Bus[uint8]:
			value = uint16(bus.Read())
		case buses.Bus[uint16]:
			value = bus.Read()
		}
		w.drawBusStatus(busInfo.name, value, busInfo.bitWidth)
	}
}

func (w *BusWindow) GetDrawArea() tview.Primitive {
	return w.text
}
