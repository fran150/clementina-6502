package beneater

import (
	"github.com/fran150/clementina-6502/pkg/terminal/ui"
	"github.com/gdamore/tcell/v2"
)

// createMenuOptions creates the main menu structure for the Ben Eater computer console.
// It defines all available menu options including emulation controls, view options, and quit functionality.
//
// Parameters:
//   - computer: The BenEaterComputer instance to control
//   - console: The console instance for UI operations
//
// Returns:
//   - A slice of menu options for the options window
func createMenuOptions(computer *BenEaterComputer, console *console) []*ui.OptionsWindowMenuOption {
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
					Action:         computer.Reset,
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
							Action:         computer.Pause,
						},
						{
							Rune:           'r',
							KeyName:        "R",
							KeyDescription: "Resume",
							Action:         computer.Resume,
						},
						{
							Rune:           's',
							KeyName:        "S",
							KeyDescription: "Step",
							Action:         computer.Step,
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
										if breakpointController := console.GetConsole().GetBreakpointController("breakpoint"); breakpointController != nil {
											breakpointController.GetWindow().RemoveSelectedItem()
										}
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
								computer.SpeedUp()
							},
							DoNotForward: true,
						},
						{
							Key:            tcell.KeyDown,
							KeyName:        "Dn",
							KeyDescription: "Speed Down",
							Action: func() {
								console.ShowEmulationSpeed()
								computer.SpeedDown()
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
					KeyDescription: "ACIA",
					Action: func() {
						console.ShowWindow("acia")
					},
				},
				{
					Key:            tcell.KeyF4,
					KeyName:        "F4",
					KeyDescription: "LCD",
					Action: func() {
						console.ShowWindow("lcd_controller")
					},
				},
				{
					Key:            tcell.KeyF5,
					KeyName:        "F5",
					KeyDescription: "ROM",
					Action: func() {
						console.ShowWindow("rom")
					},
					SubMenu: createMemoryWindowSubMenu(console),
				},
				{
					Key:            tcell.KeyF6,
					KeyName:        "F6",
					KeyDescription: "RAM",
					Action: func() {
						console.ShowWindow("ram")
					},
					SubMenu: createMemoryWindowSubMenu(console),
				},
				{
					Key:            tcell.KeyF7,
					KeyName:        "F7",
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
			Action:         computer.Stop,
		},
	}
}

// createMemoryWindowSubMenu creates navigation options for memory windows.
// It provides scrolling functionality for ROM and RAM memory views.
//
// Parameters:
//   - console: The console instance for scroll operations
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
			KeyDescription: "Scroll Up Fast",
			Action: func() {
				console.ScrollUp(20)
			},
		},
		{
			Key:            tcell.KeyPgDn,
			KeyName:        "Pg Dn",
			KeyDescription: "Scroll Down Fast",
			Action: func() {
				console.ScrollDown(20)
			},
		},
	}
}
