package emulation

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/core/interfaces"
)

type EmulatorConfig struct {
	Computer          interfaces.ComputerCore
	Console           interfaces.EmulationConsole
	Loop              interfaces.EmulationLoop
	SpeedController   interfaces.SpeedController
	BreakpointManager interfaces.BreakpointManager
}

type DefaultEmulator struct {
	computer          interfaces.ComputerCore
	console           interfaces.EmulationConsole
	loop              interfaces.EmulationLoop
	speedController   interfaces.SpeedController
	breakpointManager interfaces.BreakpointManager

	stepping  bool
	resetting bool
}

func NewDefaultEmulator(config *EmulatorConfig) *DefaultEmulator {
	emulator := &DefaultEmulator{
		computer:          config.Computer,
		console:           config.Console,
		loop:              config.Loop,
		speedController:   config.SpeedController,
		breakpointManager: config.BreakpointManager,
	}

	emulator.loop.SetEmulator(emulator)
	emulator.console.SetEmulator(emulator)

	return emulator
}

// Run starts the emulation loop and runs the console application.
func (e *DefaultEmulator) Run() (*common.StepContext, error) {
	context := e.loop.Start()

	if err := e.console.Run(); err != nil {
		e.loop.Stop()
		return context, err
	}

	return context, nil
}

// Stop stops computer execution and finishes the console application.
func (e *DefaultEmulator) Stop() {
	e.loop.Stop()
	e.console.Stop()
}

func (e *DefaultEmulator) Pause() {
	e.loop.Pause()
}

func (e *DefaultEmulator) Resume() {
	e.loop.Resume()
}

func (e *DefaultEmulator) Step() {
	if e.IsPaused() {
		e.stepping = true
		e.Resume()
	}
}

func (e *DefaultEmulator) Reset() {
	e.resetting = true
	e.computer.Reset(true)
}

func (e *DefaultEmulator) UnReset() {
	e.resetting = false
	e.computer.Reset(false)
}

func (e *DefaultEmulator) IsPaused() bool {
	return e.loop.IsPaused()
}

func (e *DefaultEmulator) IsResetting() bool {
	return e.resetting
}

func (e *DefaultEmulator) IsStepping() bool {
	return e.stepping
}

func (e *DefaultEmulator) IsStopping() bool {
	return e.loop.IsStopping()
}

func (e *DefaultEmulator) Tick(context *common.StepContext) {
	e.computer.Tick(context)

	// Clear stepping state
	if e.IsStepping() {
		e.Pause()
		e.stepping = false
	}

	if e.breakpointManager.HasBreakpoint(e.computer.GetProgramCounter() - 1) {
		e.Pause()
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
	e.console.Tick(context)
}

func (e *DefaultEmulator) Draw(context *common.StepContext) {
	e.console.Draw(context)
}

func (e *DefaultEmulator) GetSpeedController() interfaces.SpeedController {
	return e.speedController
}

func (e *DefaultEmulator) GetBreakpointManager() interfaces.BreakpointManager {
	return e.breakpointManager
}

// loopConfig := &emulation.EmulationLoopConfig{
// 	DisplayFps: config.DisplayFps,
// }

// computer.loop = emulation.NewEmulationLoop(computer, computer.speedController, loopConfig)
