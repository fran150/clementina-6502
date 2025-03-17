package beneater

import (
	"github.com/fran150/clementina6502/pkg/common"
	"github.com/fran150/clementina6502/pkg/terminal/ui"
	"github.com/gdamore/tcell/v2"
)

func createMenuOptions(computer *BenEaterComputer, console *console) []*ui.OptionsWindowMenuOption {
	return []*ui.OptionsWindowMenuOption{
		{
			Rune:           'r',
			KeyName:        "R",
			KeyDescription: "Reset",
			Action:         computer.Reset,
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
					Action: func(context *common.StepContext) {
						if breakpointForm := GetWindow[ui.BreakPointForm](console, "breakpoint"); breakpointForm != nil {
							breakpointForm.RemoveSelectedItem(context)
						}
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
					Rune:           '=',
					KeyName:        "+",
					KeyDescription: "Speed Up",
					Action:         computer.SpeedUp,
				},
				{
					Rune:           '-',
					KeyName:        "-",
					KeyDescription: "Speed Down",
					Action:         computer.SpeedDown,
				},
			},
		},
		{
			Rune:           'w',
			KeyName:        "W",
			KeyDescription: "Windows",
			SubMenu: []*ui.OptionsWindowMenuOption{
				{
					Key:            tcell.KeyF1,
					KeyName:        "F1",
					KeyDescription: "CPU",
					Action: func(context *common.StepContext) {
						console.ShowWindow("cpu", context)
					},
				},
				{
					Key:            tcell.KeyF2,
					KeyName:        "F2",
					KeyDescription: "VIA",
					Action: func(context *common.StepContext) {
						console.ShowWindow("via", context)
					},
				},
				{
					Key:            tcell.KeyF3,
					KeyName:        "F3",
					KeyDescription: "ACIA",
					Action: func(context *common.StepContext) {
						console.ShowWindow("acia", context)
					},
				},
				{
					Key:            tcell.KeyF4,
					KeyName:        "F4",
					KeyDescription: "LCD",
					Action: func(context *common.StepContext) {
						console.ShowWindow("lcd_controller", context)
					},
				},
				{
					Key:            tcell.KeyF5,
					KeyName:        "F5",
					KeyDescription: "ROM",
					Action: func(context *common.StepContext) {
						console.ShowWindow("rom", context)
					},
					SubMenu: createMemoryWindowSubMenu(console),
				},
				{
					Key:            tcell.KeyF6,
					KeyName:        "F6",
					KeyDescription: "RAM",
					Action: func(context *common.StepContext) {
						console.ShowWindow("ram", context)
					},
					SubMenu: createMemoryWindowSubMenu(console),
				},
				{
					Key:            tcell.KeyF7,
					KeyName:        "F7",
					KeyDescription: "Buses",
					Action: func(context *common.StepContext) {
						console.ShowWindow("bus", context)
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
		{
			Rune:           'p',
			KeyName:        "P",
			KeyDescription: "Pause",
			Action:         computer.Pause,
		},
		{
			Rune:           'o',
			KeyName:        "O",
			KeyDescription: "Resume",
			Action:         computer.Resume,
		},
		{
			Rune:           'i',
			KeyName:        "I",
			KeyDescription: "Step",
			Action:         computer.Step,
		},
	}
}

// Helper function to create memory window navigation submenu
func createMemoryWindowSubMenu(console *console) []*ui.OptionsWindowMenuOption {
	return []*ui.OptionsWindowMenuOption{
		{
			Rune:           'w',
			KeyName:        "W",
			KeyDescription: "Scroll Up",
			Action: func(context *common.StepContext) {
				console.ScrollUp(context, 1)
			},
		},
		{
			Rune:           's',
			KeyName:        "S",
			KeyDescription: "Scroll Down",
			Action: func(context *common.StepContext) {
				console.ScrollDown(context, 1)
			},
		},
		{
			Rune:           'e',
			KeyName:        "E",
			KeyDescription: "Scroll Up Fast",
			Action: func(context *common.StepContext) {
				console.ScrollUp(context, 20)
			},
		},
		{
			Rune:           'd',
			KeyName:        "D",
			KeyDescription: "Scroll Down Fast",
			Action: func(context *common.StepContext) {
				console.ScrollDown(context, 20)
			},
		},
	}
}
