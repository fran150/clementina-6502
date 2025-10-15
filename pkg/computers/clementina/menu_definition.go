package clementina

import (
	"github.com/fran150/clementina-6502/pkg/core/emulation"
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/gdamore/tcell/v2"
)

// createMenuOptions creates the main menu structure for the Clementina computer console.
// It defines all available menu options including emulation controls, view options, and quit functionality.
//
// Parameters:
//   - computer: The ClementinaComputer instance to control
//   - console: The console instance for UI operations
//
// Returns:
//   - A slice of menu options for the options window
func createMenuOptions(console *console, emulatorConfig *emulation.EmulatorConfig) []*ui.OptionsWindowMenuOption {
	return []*ui.OptionsWindowMenuOption{
		{
			Rune:           'e',
			KeyName:        "E",
			KeyDescription: "Emulation",
			SubMenu: []*ui.OptionsWindowMenuOption{
				{
					Rune:           'r',
					KeyName:        "R",
					KeyDescription: "Reset",
					Action:         emulatorConfig.StateManager.Reset,
				},
				{
					Rune:           'e',
					KeyName:        "E",
					KeyDescription: "Execution",
					SubMenu: []*ui.OptionsWindowMenuOption{
						{
							Rune:           'p',
							KeyName:        "P",
							KeyDescription: "Pause",
							Action:         emulatorConfig.StateManager.Pause,
						},
						{
							Rune:           'r',
							KeyName:        "R",
							KeyDescription: "Resume",
							Action:         emulatorConfig.StateManager.Resume,
						},
						{
							Rune:           's',
							KeyName:        "S",
							KeyDescription: "Step",
							Action:         emulatorConfig.StateManager.Step,
						},
						{
							Rune:           'b',
							KeyName:        "B",
							KeyDescription: "Breakpoints",
							Action:         console.SetBreakpointConfigMode,
							BackAction:     console.ReturnToPreviousWindow,
							SubMenu: []*ui.OptionsWindowMenuOption{
								{
									Rune:           'r',
									KeyName:        "R",
									KeyDescription: "Remove Selected Breakpoint",
									Action: func() {
										console.RemoveSelectedItem()
									},
								},
							},
						},
					},
				},
				{
					Rune:           's',
					KeyName:        "S",
					KeyDescription: "Speed",
					SubMenu: []*ui.OptionsWindowMenuOption{
						{
							Key:            tcell.KeyUp,
							KeyName:        "Up",
							KeyDescription: "Speed Up",
							Action: func() {
								console.ShowEmulationSpeed()
								emulatorConfig.SpeedController.SpeedUp()
							},
							DoNotForward: true,
						},
						{
							Key:            tcell.KeyDown,
							KeyName:        "Dn",
							KeyDescription: "Speed Down",
							Action: func() {
								console.ShowEmulationSpeed()
								emulatorConfig.SpeedController.SpeedDown()
							},
							DoNotForward: true,
						},
					},
				},
			},
		},
		{
			Rune:           'v',
			KeyName:        "V",
			KeyDescription: "View",
			SubMenu: []*ui.OptionsWindowMenuOption{
				{
					Key:            tcell.KeyF1,
					KeyName:        "F1",
					KeyDescription: "CPU",
					Action: func() {
						console.ShowWindow("cpu")
					},
				},
				{
					Key:            tcell.KeyF2,
					KeyName:        "F2",
					KeyDescription: "VIA",
					Action: func() {
						console.ShowWindow("via")
					},
				},
				{
					Key:            tcell.KeyF3,
					KeyName:        "F3",
					KeyDescription: "Base RAM",
					Action: func() {
						console.ShowWindow("baseram")
					},
					SubMenu: createMemoryWindowSubMenu(console),
				},
				{
					Key:            tcell.KeyF4,
					KeyName:        "F4",
					KeyDescription: "Ext. RAM",
					Action: func() {
						console.ShowWindow("exram")
					},
					SubMenu: createMemoryWindowSubMenu(console),
				},
				{
					Key:            tcell.KeyF5,
					KeyName:        "F5",
					KeyDescription: "Hi RAM",
					Action: func() {
						console.ShowWindow("hiram")
					},
					SubMenu: createMemoryWindowSubMenu(console),
				},
				{
					Key:            tcell.KeyF6,
					KeyName:        "F6",
					KeyDescription: "Buses",
					Action: func() {
						console.ShowWindow("bus")
					},
				},
			},
		},
		{
			Rune:           'q',
			KeyName:        "Q",
			KeyDescription: "Quit",
			Action:         emulatorConfig.StateManager.Stop,
		},
	}
}

// createMemoryWindowSubMenu creates navigation options for memory windows.
// It provides scrolling functionality and go-to navigation for memory views.
//
// Parameters:
//   - console: The console instance for navigation operations
//
// Returns:
//   - A slice of menu options for memory window navigation
func createMemoryWindowSubMenu(console *console) []*ui.OptionsWindowMenuOption {
	return []*ui.OptionsWindowMenuOption{
		{
			Key:            tcell.KeyUp,
			KeyName:        "Up",
			KeyDescription: "Scroll Up",
			Action: func() {
				console.ScrollUp(1)
			},
			DoNotForward: true,
		},
		{
			Key:            tcell.KeyDown,
			KeyName:        "Dn",
			KeyDescription: "Scroll Down",
			Action: func() {
				console.ScrollDown(1)
			},
			DoNotForward: true,
		},
		{
			Key:            tcell.KeyPgUp,
			KeyName:        "Pg Up",
			KeyDescription: "S. Up Fast",
			Action: func() {
				console.ScrollUp(64)
			},
		},
		{
			Key:            tcell.KeyPgDn,
			KeyName:        "Pg Dn",
			KeyDescription: "S. Down Fast",
			Action: func() {
				console.ScrollDown(64)
			},
		},
		{
			Rune:           'g',
			KeyName:        "G",
			KeyDescription: "Go To",
			Action:         console.ShowGotoForm,
			BackAction:     console.ReturnToPreviousWindow,
			SubMenu:        []*ui.OptionsWindowMenuOption{},
		},
	}
}
