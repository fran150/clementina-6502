package ui

import (
	"fmt"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/rivo/tview"
)

const maxLines = 37

// MemoryWindow represents a UI component that displays the contents of memory.
// It shows a hexadecimal dump of memory contents with navigation capabilities.
type MemoryWindow struct {
	text   *tview.TextView
	memory components.MemoryChip

	start uint16
}

// NewMemoryWindow creates a new memory display window.
// It initializes the UI component and connects it to the provided memory chip.
//
// Parameters:
//   - memory: The memory chip to display
//
// Returns:
//   - A pointer to the initialized MemoryWindow
func NewMemoryWindow(memory components.MemoryChip) *MemoryWindow {
	text := tview.NewTextView()
	text.SetTextAlign(tview.AlignLeft).
		SetScrollable(false).
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle("Memory Explorer")

	return &MemoryWindow{
		memory: memory,
		text:   text,
		start:  0x0000,
	}
}

func (m *MemoryWindow) GetTitle() string {
	return m.text.GetTitle()
}

func (m *MemoryWindow) SetTitle(title string) {
	m.text.SetTitle(title)
}

func (m *MemoryWindow) GetStartAddress() uint16 {
	return m.start
}

func (m *MemoryWindow) Clear() {
	m.text.Clear()
}

func (m *MemoryWindow) ScrollDown(lines uint16) {
	size := uint16(m.memory.Size())

	m.start += lines * 8
	if m.start > size {
		m.start = size - 8
	}
}

func (m *MemoryWindow) ScrollUp(lines uint16) {
	value := int(m.start) - (int(lines) * 8)

	if value < 0 {
		m.start = 0
	} else {
		m.start = uint16(value)
	}
}

func (m *MemoryWindow) Draw(context *common.StepContext) {
	address := m.start

	for range maxLines {
		if address >= uint16(m.memory.Size()) {
			break
		}

		fmt.Fprintf(m.text, "[yellow]%04X:[white]", address)

		for i := range uint16(8) {
			fmt.Fprintf(m.text, " %02X", m.memory.Peek(address+i))
		}

		fmt.Fprint(m.text, "\n")
		address += 8
	}
}

func (d *MemoryWindow) GetDrawArea() tview.Primitive {
	return d.text
}
