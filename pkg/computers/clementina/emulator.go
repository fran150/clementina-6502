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

// clementinaEmulator implements the core.BaseEmulator interface for the Clementina 6502 emulator.
// It manages the emulation of a Clementina computer system, including speed control,
// breakpoint management, and the underlying computer hardware simulation.
type clementinaEmulator struct {
	core.BaseEmulator
	speedController   core.SpeedController
	breakpointManager core.BreakpointManager
	computer          *ClementinaComputer
}

// NewClemetinaEmulator creates a new instance of the Clementina 6502 emulator.
// It initializes all necessary components including speed control, breakpoint management,
// console interface, and emulation loop with the specified configuration.

// Parameters:
//   - computer: The ClementinaComputer instance to emulate
//   - speed: The emulation speed multiplier (1.0 = real hardware speed)
//   - displayFPS: The frames per second for display updates
//
// Returns:
//   - core.BaseEmulator: The configured emulator instance
//   - error: Any error that occurred during initialization
func NewClemetinaEmulator(computer *ClementinaComputer, speed float64, displayFPS int) (core.BaseEmulator, error) {
	speedController := controllers.NewSpeedController(speed)
	breakPointManager := managers.NewBreakpointManager()
	windowManager := terminal.NewWindowManager()
	navigationManager := managers.NewNavigationManager()

	emulator := &clementinaEmulator{
		computer:          computer,
		speedController:   speedController,
		breakpointManager: breakPointManager,
	}

	console := newClementinaEmulatorConsole(clementinaEmulatorConsoleConfig{
		BaseEmulatorConsoleConfig: terminal.BaseEmulatorConsoleConfig{
			WindowManager:     windowManager,
			NavigationManager: navigationManager,
			InputHandler:      terminal.NewInputHandler(windowManager),
			App:               tview.NewApplication(),
		},
		emulator: emulator,
	})

	loop := emulation.NewEmulationLoop(emulation.EmulationLoopConfig{
		SpeedController: speedController,
		DisplayFPS:      displayFPS,
		Emulator:        emulator,
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
