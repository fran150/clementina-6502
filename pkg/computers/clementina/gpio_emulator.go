//go:build (linux && arm) || (linux && arm64)

package clementina

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
	computer          *ClementinaComputer
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
func NewClemetinaGPIOEmulator(computer *ClementinaComputer, displayFPS int, chipName string) (core.BaseEmulator, error) {
	speedController := controllers.NewSpeedController(1.0) // Dummy speed controller for UI compatibility
	breakPointManager := managers.NewBreakpointManager()
	windowManager := terminal.NewWindowManager()
	navigationManager := managers.NewNavigationManager()

	emulator := &clementinaGPIOEmulator{
		computer:          computer,
		speedController:   speedController,
		breakpointManager: breakPointManager,
	}

	// Cast to *clementinaEmulator for console compatibility
	regularEmulator := (*clementinaEmulator)(emulator)

	console := newClementinaEmulatorConsole(clementinaEmulatorConsoleConfig{
		BaseEmulatorConsoleConfig: terminal.BaseEmulatorConsoleConfig{
			WindowManager:     windowManager,
			NavigationManager: navigationManager,
			InputHandler:      terminal.NewInputHandler(windowManager),
			App:               tview.NewApplication(),
		},
		emulator: regularEmulator,
	})

	loop := emulation.NewGPIOEmulationLoop(emulation.GPIOEmulationLoopConfig{
		DisplayFPS: displayFPS,
		Emulator:   emulator,
		ChipName:   chipName,
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
