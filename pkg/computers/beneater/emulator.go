package beneater

import (
	"fmt"
	"os"

	"github.com/fran150/clementina-6502/pkg/core/controllers"
	"github.com/fran150/clementina-6502/pkg/core/emulation"
	"github.com/fran150/clementina-6502/pkg/core/interfaces"
	"github.com/fran150/clementina-6502/pkg/core/managers"
)

func NewBenEaterEmulation(computer *BenEaterComputer, speed float64, displayFPS int) (interfaces.Emulator, error) {
	speedController := controllers.NewSpeedController(speed)
	breakPointManager := managers.NewDefaultBreakpointManager()

	console := NewBenEaterEmulationConsole(computer)

	loopConfig := &emulation.DefaultEmulationLoopConfig{
		SpeedController: speedController,
		DisplayFPS:      displayFPS,
	}
	loop := emulation.NewEmulationLoop(*loopConfig)

	emulatorConfig := emulation.DefaultEmulatorConfig{
		Computer:          computer,
		Console:           console,
		Loop:              loop,
		SpeedController:   speedController,
		BreakpointManager: breakPointManager,
	}

	emulator := emulation.NewDefaultEmulator(emulatorConfig)

	loop.SetPanicHandler(func(loopType string, panicData any) bool {
		fmt.Fprintf(os.Stderr, "%s panic: %v\n", loopType, panicData)
		emulator.Stop()
		return false
	})

	return emulator, nil
}
