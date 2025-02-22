package ui

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

type OptionsWindow struct {
	text *tview.TextView
}

type OptionsWindowConfig struct {
	KeyName        string
	KeyDescription string
}

func CreateOptionsWindow(options []OptionsWindowConfig) *OptionsWindow {
	text := tview.NewTextView()
	text.SetBorder(true)
	text.SetDynamicColors(true)

	sb := strings.Builder{}

	for _, option := range options {
		fmt.Fprintf(&sb, " [white::r]%s[white:-:-] %s ", option.KeyName, option.KeyDescription)
	}

	text.SetText(sb.String())

	return &OptionsWindow{
		text: text,
	}
}

func (d *OptionsWindow) GetDrawArea() *tview.TextView {
	return d.text
}
