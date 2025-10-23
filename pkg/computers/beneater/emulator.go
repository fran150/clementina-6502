package beneater

import (
	"fmt"
	"os"

	"github.com/fran150/clementina-6502/pkg/computers"
	"github.com/fran150/clementina-6502/pkg/core"
	"github.com/fran150/clementina-6502/pkg/core/controllers"
	"github.com/fran150/clementina-6502/pkg/core/emulation"
	"github.com/fran150/clementina-6502/pkg/core/managers"
	"github.com/fran150/clementina-6502/pkg/terminal"
	"github.com/rivo/tview"
)

type benEaterEmulator struct {
	core.BaseEmulator
	speedController   core.SpeedController
	breakpointManager core.BreakpointManager
	computer          *BenEaterComputer
}

func NewBenEaterEmulator(computer *BenEaterComputer, speed float64, displayFPS int) (core.BaseEmulator, error) {
	speedController := controllers.NewSpeedController(speed)
	breakPointManager := managers.NewBreakpointManager()
	windowManager := terminal.NewDefaultWindowManager()
	navigationManager := managers.NewNavigationManager()

	emulator := &benEaterEmulator{
		computer:          computer,
		speedController:   speedController,
		breakpointManager: breakPointManager,
	}

	console := newBenEaterEmulatorConsole(benEaterEmulatorConsoleConfig{
		BaseTerminalEmulatorConsoleConfig: computers.BaseTerminalEmulatorConsoleConfig{
			WindowManager:     windowManager,
			NavigationManager: navigationManager,
			InputHandler:      terminal.NewDefaultInputHandler(windowManager),
			App:               tview.NewApplication(),
		},
		emulator: emulator,
	})

	loop := emulation.NewEmulationLoop(emulation.DefaultEmulationLoopConfig{
		SpeedController: speedController,
		DisplayFPS:      displayFPS,
		Emulator:        emulator,
	})

	emulatorConfig := emulation.DefaultEmulatorConfig{
		Computer:          computer,
		Console:           console,
		Loop:              loop,
		SpeedController:   speedController,
		BreakpointManager: breakPointManager,
	}

	emulator.BaseEmulator = emulation.NewBaseEmulator(emulatorConfig)

	loop.SetPanicHandler(func(loopType string, panicData any) bool {
		fmt.Fprintf(os.Stderr, "%s panic: %v\n", loopType, panicData)
		emulator.Stop()
		return false
	})

	return emulator, nil
}
