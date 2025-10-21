package clementina

import (
	"github.com/fran150/clementina-6502/pkg/core/interfaces"
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
func createMenuOptions(console *ClementinaComputerConsole, emulator interfaces.Emulator) []*ui.OptionsWindowMenuOption {
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
					Action: func(option *ui.OptionsWindowMenuOption) {
						if emulator.IsResetting() {
							option.KeyDescription = "Reset"
							emulator.UnReset()
						} else {
							option.KeyDescription = "Release Reset"
							emulator.Reset()
						}
					},
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
							Action: func(option *ui.OptionsWindowMenuOption) {
								emulator.Pause()
							},
						},
						{
							Rune:           'r',
							KeyName:        "R",
							KeyDescription: "Resume",
							Action: func(option *ui.OptionsWindowMenuOption) {
								emulator.Resume()
							},
						},
						{
							Rune:           's',
							KeyName:        "S",
							KeyDescription: "Step",
							Action: func(option *ui.OptionsWindowMenuOption) {
								emulator.Step()
							},
						},
						{
							Rune:           'b',
							KeyName:        "B",
							KeyDescription: "Breakpoints",
							Action: func(option *ui.OptionsWindowMenuOption) {
								console.SetBreakpointConfigMode()
							},
							BackAction: func(option *ui.OptionsWindowMenuOption) {
								console.ReturnToPreviousWindow()
							},
							SubMenu: []*ui.OptionsWindowMenuOption{
								{
									Rune:           'r',
									KeyName:        "R",
									KeyDescription: "Remove Selected Breakpoint",
									Action: func(option *ui.OptionsWindowMenuOption) {
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
							Action: func(option *ui.OptionsWindowMenuOption) {
								console.ShowEmulationSpeed()
								emulator.GetSpeedController().SpeedUp()
							},
							DoNotForward: true,
						},
						{
							Key:            tcell.KeyDown,
							KeyName:        "Dn",
							KeyDescription: "Speed Down",
							Action: func(option *ui.OptionsWindowMenuOption) {
								console.ShowEmulationSpeed()
								emulator.GetSpeedController().SpeedDown()
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
					Action: func(option *ui.OptionsWindowMenuOption) {
						console.ShowWindow("cpu")
					},
				},
				{
					Key:            tcell.KeyF2,
					KeyName:        "F2",
					KeyDescription: "VIA",
					Action: func(option *ui.OptionsWindowMenuOption) {
						console.ShowWindow("via")
					},
				},
				{
					Key:            tcell.KeyF3,
					KeyName:        "F3",
					KeyDescription: "Base RAM",
					Action: func(option *ui.OptionsWindowMenuOption) {
						console.ShowWindow("baseram")
					},
					SubMenu: createMemoryWindowSubMenu(console),
				},
				{
					Key:            tcell.KeyF4,
					KeyName:        "F4",
					KeyDescription: "Ext. RAM",
					Action: func(option *ui.OptionsWindowMenuOption) {
						console.ShowWindow("exram")
					},
					SubMenu: createMemoryWindowSubMenu(console),
				},
				{
					Key:            tcell.KeyF5,
					KeyName:        "F5",
					KeyDescription: "Hi RAM",
					Action: func(option *ui.OptionsWindowMenuOption) {
						console.ShowWindow("hiram")
					},
					SubMenu: createMemoryWindowSubMenu(console),
				},
				{
					Key:            tcell.KeyF6,
					KeyName:        "F6",
					KeyDescription: "Buses",
					Action: func(option *ui.OptionsWindowMenuOption) {
						console.ShowWindow("bus")
					},
				},
			},
		},
		{
			Rune:           'q',
			KeyName:        "Q",
			KeyDescription: "Quit",
			Action: func(option *ui.OptionsWindowMenuOption) {
				emulator.Stop()
			},
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
func createMemoryWindowSubMenu(console *ClementinaComputerConsole) []*ui.OptionsWindowMenuOption {
	return []*ui.OptionsWindowMenuOption{
		{
			Key:            tcell.KeyUp,
			KeyName:        "Up",
			KeyDescription: "Scroll Up",
			Action: func(option *ui.OptionsWindowMenuOption) {
				console.ScrollUp(1)
			},
			DoNotForward: true,
		},
		{
			Key:            tcell.KeyDown,
			KeyName:        "Dn",
			KeyDescription: "Scroll Down",
			Action: func(option *ui.OptionsWindowMenuOption) {
				console.ScrollDown(1)
			},
			DoNotForward: true,
		},
		{
			Key:            tcell.KeyPgUp,
			KeyName:        "Pg Up",
			KeyDescription: "S. Up Fast",
			Action: func(option *ui.OptionsWindowMenuOption) {
				console.ScrollUp(64)
			},
		},
		{
			Key:            tcell.KeyPgDn,
			KeyName:        "Pg Dn",
			KeyDescription: "S. Down Fast",
			Action: func(option *ui.OptionsWindowMenuOption) {
				console.ScrollDown(64)
			},
		},
		{
			Rune:           'g',
			KeyName:        "G",
			KeyDescription: "Go To",
			Action: func(option *ui.OptionsWindowMenuOption) {
				console.ShowGotoForm()
			},
			BackAction: func(option *ui.OptionsWindowMenuOption) {
				console.ReturnToPreviousWindow()
			},
			SubMenu: []*ui.OptionsWindowMenuOption{},
		},
	}
}
