package emulation

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/core/interfaces"
)

type EmulatorConfig struct {
	SpeedController   interfaces.SpeedController
	StateManager      interfaces.StateManager
	BreakpointManager interfaces.BreakpointManager
}

type Emulator struct {
	context *common.StepContext

	computer interfaces.ComputerCore
	loop     *EmulationLoop
	console  interfaces.EmulationConsole

	speedController   interfaces.SpeedController
	stateManager      interfaces.StateManager
	breakpointManager interfaces.BreakpointManager
}

// Run starts the emulation loop and runs the console application.
func (e *Emulator) Run() (*common.StepContext, error) {
	e.context = e.loop.Start()

	if err := e.console.Run(); err != nil {
		e.loop.Stop()
		return e.context, err
	}

	return e.context, nil
}

// Stop stops computer execution and finishes the console application.
func (e *Emulator) Stop() {
	e.loop.Stop()
	e.console.Stop()
}

func (e *Emulator) Tick() {
	if !e.stateManager.IsPaused() || e.stateManager.IsStepping() {
		e.computer.Tick(e.context)
	}

	// Clear stepping state
	if e.stateManager.IsStepping() {
		e.stateManager.ClearStepping()
	}

	if e.breakpointManager.HasBreakpoint(e.computer.GetProgramCounter() - 1) {
		e.stateManager.Pause()
	}

	// TODO: I'm getting 7.29 Mhz without calling this function. That is the upper limit.
	// I get 5.9 Mhz when commenting ticker.Tick call in the console.Tick method. If I comment only
	// the contents of the ticket.Tick but leave the call I get 5.5. This means that is 0.4 Mhz lost in the
	// overhead of that call (could it be the interface transformation?)
	// I was getting:
	// 2.9 mhz with the window manager implemeantion of returning a copy of the maps.
	// 3.6 mhz using go's new enumerators functions
	// 4.1 returning a function for each iteration loop
	// 4.3 returning the map reference directly (but this means that it can be altered directly)
	e.console.Tick(e.context)
}

// GetSpeedController returns the speed controller interface.
func (e *Emulator) GetSpeedController() interfaces.SpeedController {
	return e.speedController
}

// GetStateManager returns the state manager interface.
func (e *Emulator) GetStateManager() interfaces.StateManager {
	return e.stateManager
}

// GetBreakpointManager returns the breakpoint manager interface.
func (e *Emulator) GetBreakpointManager() interfaces.BreakpointManager {
	return e.breakpointManager
}

// loopConfig := &emulation.EmulationLoopConfig{
// 	DisplayFps: config.DisplayFps,
// }

// computer.loop = emulation.NewEmulationLoop(computer, computer.speedController, loopConfig)

// computer.loop.SetPanicHandler(func(loopType string, panicData any) bool {
// 	fmt.Fprintf(os.Stderr, "%s panic: %v\n", loopType, panicData)
// 	computer.Stop()
// 	return false
// })
