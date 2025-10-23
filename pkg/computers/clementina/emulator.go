package clementina

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

func NewClemetinaEmulator(computer *ClementinaComputer, speed float64, displayFPS int) (core.BaseEmulator, error) {
	speedController := controllers.NewSpeedController(speed)
	breakPointManager := managers.NewBreakpointManager()
	windowManager := terminal.NewDefaultWindowManager()
	navigationManager := managers.NewNavigationManager()

	console := NewClementinaEmulatorConsole(ClementinaEmulatorConsoleConfig{
		BaseTerminalEmulatorConsoleConfig: computers.BaseTerminalEmulatorConsoleConfig{
			WindowManager:     windowManager,
			NavigationManager: navigationManager,
			InputHandler:      terminal.NewDefaultInputHandler(windowManager),
			App:               tview.NewApplication(),
		},
		Computer: computer,
	})

	loop := emulation.NewEmulationLoop(emulation.DefaultEmulationLoopConfig{
		SpeedController: speedController,
		DisplayFPS:      displayFPS,
	})

	emulator := emulation.NewBaseEmulator(emulation.DefaultEmulatorConfig{
		Computer:          computer,
		Console:           console,
		Loop:              loop,
		SpeedController:   speedController,
		BreakpointManager: breakPointManager,
	})

	loop.SetPanicHandler(func(loopType string, panicData any) bool {
		fmt.Fprintf(os.Stderr, "%s panic: %v\n", loopType, panicData)
		emulator.Stop()
		return false
	})

	return emulator, nil
}
