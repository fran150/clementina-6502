package beneater

import (
	"fmt"
	"os"

	"github.com/fran150/clementina-6502/pkg/core"
	"github.com/fran150/clementina-6502/pkg/core/controllers"
	"github.com/fran150/clementina-6502/pkg/core/emulation"
	"github.com/fran150/clementina-6502/pkg/core/managers"
	"github.com/fran150/clementina-6502/pkg/terminal"
	"github.com/rivo/tview"
)

// benEaterEmulator implements the core.BaseEmulator interface for Ben Eater's 6502 computer.
// It provides emulation capabilities with speed control and breakpoint management.
type benEaterEmulator struct {
	core.BaseEmulator
	speedController   core.SpeedController
	breakpointManager core.BreakpointManager
	computer          *BenEaterComputer
}

// NewBenEaterEmulator creates a new Ben Eater 6502 computer emulator instance.
// It initializes all necessary components including speed control, breakpoint management,
// console interface, and emulation loop with the specified parameters.
//
// Parameters:
//   - computer: The BenEaterComputer instance to emulate
//   - speed: The emulation speed multiplier (1.0 = real hardware speed)
//   - displayFPS: The frames per second for display updates
//
// Returns:
//   - core.BaseEmulator: The configured emulator instance
//   - error: Any error that occurred during initialization
func NewBenEaterEmulator(computer *BenEaterComputer, speed float64, displayFPS int) (core.BaseEmulator, error) {
	speedController := controllers.NewSpeedController(speed)
	breakPointManager := managers.NewBreakpointManager()
	windowManager := terminal.NewWindowManager()
	navigationManager := managers.NewNavigationManager()

	emulator := &benEaterEmulator{
		computer:          computer,
		speedController:   speedController,
		breakpointManager: breakPointManager,
	}

	console := newBenEaterEmulatorConsole(benEaterEmulatorConsoleConfig{
		EmulatorConsoleConfig: terminal.EmulatorConsoleConfig{
			WindowManager:     windowManager,
			NavigationManager: navigationManager,
			InputHandler:      terminal.NewInputHandler(windowManager),
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
