package beneater

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

type optionsWindow struct {
	text *tview.TextView
}

type options struct {
	keyName        string
	keyDescription string
}

func createOptionsWindow(options []options) *optionsWindow {
	text := tview.NewTextView()
	text.SetBorder(true)
	text.SetDynamicColors(true)

	sb := strings.Builder{}

	for _, option := range options {
		fmt.Fprintf(&sb, " [white::r]%s[white:-:-] %s ", option.keyName, option.keyDescription)
	}

	text.SetText(sb.String())

	return &optionsWindow{
		text: text,
	}
}
