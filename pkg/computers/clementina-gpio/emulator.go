//go:build (linux && arm) || (linux && arm64)

package clementinagpio

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

// clementinaGPIOEmulator implements the core.BaseEmulator interface for GPIO-controlled emulation.
type clementinaGPIOEmulator struct {
	core.BaseEmulator
	speedController   core.SpeedController
	breakpointManager core.BreakpointManager
	computer          *ClementinaGPIOComputer
}

// NewClemetinaGPIOEmulator creates a new GPIO-controlled Clementina 6502 emulator.
// It initializes all necessary components for GPIO-controlled stepping with display updates.
//
// Parameters:
//   - computer: The ClementinaComputer instance to emulate
//   - displayFPS: The frames per second for display updates
//
// Returns:
//   - core.BaseEmulator: The configured GPIO emulator instance
//   - error: Any error that occurred during initialization
func NewClemetinaGPIOEmulator(computer *ClementinaGPIOComputer, displayFPS int) (core.BaseEmulator, error) {
	speedController := controllers.NewSpeedController(1.0) // Dummy speed controller for UI compatibility
	breakPointManager := managers.NewBreakpointManager()
	windowManager := terminal.NewWindowManager()
	navigationManager := managers.NewNavigationManager()

	emulator := &clementinaGPIOEmulator{
		computer:          computer,
		speedController:   speedController,
		breakpointManager: breakPointManager,
	}

	console := newClementinaGPIOEmulatorConsole(clementinaGPIOEmulatorConsoleConfig{
		BaseEmulatorConsoleConfig: terminal.BaseEmulatorConsoleConfig{
			WindowManager:     windowManager,
			NavigationManager: navigationManager,
			InputHandler:      terminal.NewInputHandler(windowManager),
			App:               tview.NewApplication(),
		},
		emulator: emulator,
	})

	loop := emulation.NewGPIOEmulationLoop(emulation.GPIOEmulationLoopConfig{
		DisplayFPS: displayFPS,
		Emulator:   emulator,
		ChipName:   "gpiochip0",
	})

	emulatorConfig := emulation.EmulatorConfig{
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
