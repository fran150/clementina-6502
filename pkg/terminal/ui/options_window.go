package ui

import (
	"fmt"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// OptionsWindow represents a UI component that displays and manages menu options.
// It provides a hierarchical menu system with keyboard shortcuts for controlling the emulator.
type OptionsWindow struct {
	text *tview.TextView

	mainMenu []*OptionsWindowMenuOption
	active   *OptionsWindowMenuOption
}

// OptionsWindowMenuOption represents a single menu option in the options window.
// It defines the key binding, description, and actions associated with a menu item.
type OptionsWindowMenuOption struct {
	Key            tcell.Key
	Rune           rune
	KeyName        string
	KeyDescription string
	Action         func(context *common.StepContext)
	BackAction     func(context *common.StepContext)
	DoNotForward   bool

	SubMenu []*OptionsWindowMenuOption

	parent *OptionsWindowMenuOption
}

// NewOptionsWindow creates a new options menu window with the provided menu structure.
// It initializes the UI component and sets up the menu hierarchy.
//
// Parameters:
//   - menu: The top-level menu options to display
//
// Returns:
//   - A pointer to the initialized OptionsWindow
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

// GetActiveMenu returns the currently active menu option.
//
// Returns:
//   - The currently active menu option, or nil if no menu is active
func (d *OptionsWindow) GetActiveMenu() *OptionsWindowMenuOption {
	return d.active
}

// SetActiveMenu sets the specified menu option as the active one.
//
// Parameters:
//   - menu: The menu option to set as active
func (d *OptionsWindow) SetActiveMenu(menu *OptionsWindowMenuOption) {
	d.active = menu
}

// ProcessKey handles keyboard input for the options window.
// It processes key events to navigate menus and execute menu actions.
//
// Parameters:
//   - event: The keyboard event to process
//   - context: The current step context containing system state information
//
// Returns:
//   - The event if it wasn't handled, or nil if the event was consumed
func (d *OptionsWindow) ProcessKey(event *tcell.EventKey, context *common.StepContext) *tcell.EventKey {

	options := d.getActiveOptions()

	if event.Key() == tcell.KeyESC {
		d.GoToPreviousMenu(context)
		return event
	}

	for _, option := range options {
		if (event.Key() == tcell.KeyRune && option.Rune == event.Rune()) ||
			(event.Key() != tcell.KeyRune && option.Key == event.Key()) {
			if option.Action != nil {
				option.Action(context)
			}
			if option.SubMenu != nil {
				d.SetActiveMenu(option)
			}

			if option.DoNotForward {
				return nil
			} else {
				return event
			}
		}
	}

	return event
}

// GoToPreviousMenu navigates one level up in the menu tree.
// Parameters:
//   - context: The current step context containing system state information
func (d *OptionsWindow) GoToPreviousMenu(context *common.StepContext) {
	active := d.GetActiveMenu()
	if active != nil {
		if active.BackAction != nil {
			active.BackAction(context)
		}
		d.SetActiveMenu(active.parent)
	}
}

// Draw updates the options window with the current menu structure.
// It displays the active menu and available options with their key bindings.
//
// Parameters:
//   - context: The current step context containing system state information
func (d *OptionsWindow) Draw(context *common.StepContext) {
	activeMenu := d.GetActiveMenu()
	options := d.getActiveOptions()

	if activeMenu != nil {
		fmt.Fprintf(d.text, " [yellow]%s: ", activeMenu.KeyDescription)
	}

	for _, option := range options {
		fmt.Fprintf(d.text, " [white::r]%s[white:-:-] %s ", option.KeyName, option.KeyDescription)
	}

	if activeMenu != nil {
		fmt.Fprintf(d.text, " [white::r]ESC[white:-:-] Back ")
	}
}

// GetDrawArea returns the primitive that represents this window in the UI.
// This is used by the layout manager to position and render the window.
//
// Returns:
//   - The tview primitive for this window
func (d *OptionsWindow) GetDrawArea() tview.Primitive {
	return d.text
}

// Clear resets the options window, removing all text content.
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
