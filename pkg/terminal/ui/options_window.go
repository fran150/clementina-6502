package ui

import (
	"fmt"

	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type OptionsWindow struct {
	text *tview.TextView

	mainMenu []*OptionsWindowMenuOption
	active   *OptionsWindowMenuOption
}

type OptionsWindowMenuOption struct {
	Key            rune
	KeyName        string
	KeyDescription string
	Action         func(context *common.StepContext)

	SubMenu []*OptionsWindowMenuOption

	parent *OptionsWindowMenuOption
}

func NewOptionsWindow(menu []*OptionsWindowMenuOption) *OptionsWindow {
	text := tview.NewTextView()
	text.SetBorder(true)
	text.SetDynamicColors(true)

	setParents(nil, menu)

	return &OptionsWindow{
		text:     text,
		mainMenu: menu,
		active:   nil,
	}
}

func (d *OptionsWindow) GetActiveMenu() *OptionsWindowMenuOption {
	return d.active
}

func (d *OptionsWindow) SetActiveMenu(menu *OptionsWindowMenuOption) {
	d.active = menu
}

func (d *OptionsWindow) ProcessKey(event *tcell.EventKey, context *common.StepContext) *tcell.EventKey {
	options := d.getActiveOptions()

	for _, option := range options {
		if option.Key == event.Rune() {
			if option.SubMenu != nil {
				d.SetActiveMenu(option)
			} else if option.Action != nil {
				option.Action(context)
			}
		} else if event.Key() == tcell.KeyEsc {
			d.SetActiveMenu(nil)
		}
	}

	return event
}

func (d *OptionsWindow) Draw(context *common.StepContext) {
	activeMenu := d.GetActiveMenu()
	options := d.getActiveOptions()

	for _, option := range options {
		fmt.Fprintf(d.text, " [white::r]%s[white:-:-] %s ", option.KeyName, option.KeyDescription)
	}

	if activeMenu != nil {
		fmt.Fprintf(d.text, " [white::r]ESC[white:-:-] Back ")
	}
}

func (d *OptionsWindow) GetDrawArea() *tview.TextView {
	return d.text
}

func (d *OptionsWindow) Clear() {
	d.text.Clear()
}

func (d *OptionsWindow) getActiveOptions() []*OptionsWindowMenuOption {
	activeMenu := d.GetActiveMenu()

	if activeMenu == nil {
		return d.mainMenu
	} else {
		return activeMenu.SubMenu
	}
}

func setParents(parent *OptionsWindowMenuOption, menu []*OptionsWindowMenuOption) {
	if menu == nil {
		return
	}

	for _, option := range menu {
		option.parent = parent
		setParents(option, option.SubMenu)
	}
}
