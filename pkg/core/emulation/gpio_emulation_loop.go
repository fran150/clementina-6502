//go:build (linux && arm) || (linux && arm64)

package emulation

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/core"
)

/*******************************************************************************************
* Structs definition
********************************************************************************************/

// GPIOEmulationLoopConfig contains the settings for GPIO-controlled emulation.
type GPIOEmulationLoopConfig struct {
	// DisplayFPS sets how often the draw loop refreshes the UI.
	DisplayFPS int

	// Emulator is the target controlled by the external PHI2 clock.
	Emulator LoopTarget

	// ChipName is the Linux GPIO chip name used to access the Raspberry Pi pins.
	ChipName string

	// RefreshDisplay populates the UI from emulator state. It is run while holding
	// the step lock so it sees a consistent cycle, but it must be fast (no terminal
	// I/O). When nil, the loop falls back to Emulator.Draw under the lock.
	RefreshDisplay func(context *common.StepContext)

	// FlushDisplay writes the populated UI to the terminal. It is run WITHOUT the
	// step lock so the slow terminal write never blocks the clock-response loop.
	FlushDisplay func()
}

// gpioEmulationLoop manages emulation driven by an external GPIO clock.
// The GPIO loop advances Tick/PostTick phases from PHI2 falling edges while the
// draw loop refreshes the UI on a fixed timer. Both loops share one StepContext,
// so stepMu serializes access to the emulator state.
type gpioEmulationLoop struct {
	config          *GPIOEmulationLoopConfig
	panicHandler    func(loopType string, panicData any) bool
	gpioLoopRunning atomic.Bool
	drawLoopRunning atomic.Bool
	stop            atomic.Bool
	pause           atomic.Bool
	stepMu          sync.Mutex

	gpioController *common.GPIOController
}

/*******************************************************************************************
* Constructor
********************************************************************************************/

// NewGPIOEmulationLoop creates a GPIO-controlled emulation loop.
// It opens the configured GPIO controller immediately because this loop cannot run
// without access to the physical PHI2, bus, and control lines.
//
// Parameters:
//   - config: The GPIO emulation loop settings
//
// Returns:
//   - The initialized emulation loop
func NewGPIOEmulationLoop(config GPIOEmulationLoopConfig) core.EmulationLoop {
	if config.DisplayFPS <= 0 {
		config.DisplayFPS = 10
	}

	gpioController, err := common.GetGPIOController(config.ChipName)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize GPIO interface: %v", err))
	}

	loop := &gpioEmulationLoop{
		config:         &config,
		gpioController: gpioController,
	}
	loop.stop.Store(true)

	return loop
}

/*******************************************************************************************
* EmulationLoop Interface methods
********************************************************************************************/

// SetPanicHandler sets the panic handler used by the GPIO and draw goroutines.
//
// Parameters:
//   - handler: The function called before a loop panic is re-raised
func (g *gpioEmulationLoop) SetPanicHandler(handler func(loopType string, panicData any) bool) {
	g.panicHandler = handler
}

// IsRunning returns true when either the GPIO loop or draw loop is running.
//
// Returns:
//   - true if at least one loop goroutine is active
func (g *gpioEmulationLoop) IsRunning() bool {
	return g.drawLoopRunning.Load() || g.gpioLoopRunning.Load()
}

// IsPaused returns true when the GPIO loop is paused.
//
// Returns:
//   - true if the loop is paused
func (g *gpioEmulationLoop) IsPaused() bool {
	return g.pause.Load()
}

// IsStopping returns true when a stop was requested while a loop is still running.
//
// Returns:
//   - true if the loop is stopping
func (g *gpioEmulationLoop) IsStopping() bool {
	return g.stop.Load() && g.IsRunning()
}

// Start begins the GPIO-controlled emulation loop.
// It starts one goroutine for external PHI2 edges and one goroutine for drawing.
//
// Returns:
//   - The shared step context used by the loop goroutines
//   - An error if the loop is already running or no emulator is configured
func (g *gpioEmulationLoop) Start() (*common.StepContext, error) {
	if !g.IsRunning() && g.config.Emulator != nil {
		context := common.NewStepContext()

		g.pause.Store(false)
		g.stop.Store(false)
		g.gpioLoopRunning.Store(true)
		g.drawLoopRunning.Store(true)

		go g.executeGPIOLoop(&context)
		go g.executeDraw(&context)

		return &context, nil
	}

	var err error
	if g.IsRunning() {
		err = fmt.Errorf("cannot start again while loop is running")
	} else {
		err = fmt.Errorf("cannot start as emulator is not set")
	}

	return nil, err
}

// Stop signals the GPIO and draw loops to stop.
func (g *gpioEmulationLoop) Stop() {
	g.stop.Store(true)
}

// Resume allows the GPIO loop to start new cycles again.
func (g *gpioEmulationLoop) Resume() {
	g.pause.Store(false)
}

// Pause prevents the GPIO loop from starting a new cycle.
// A pending PostTick is still allowed to complete so the emulator does not stop
// halfway through an external bus transaction.
func (g *gpioEmulationLoop) Pause() {
	g.pause.Store(true)
}

/*******************************************************************************************
* Execution methods
********************************************************************************************/

// executeGPIOLoop runs the GPIO-controlled emulation loop.
// The external PHI2 signal owns the cycle pacing. This loop polls PHI2 and advances
// the emulator only on a falling edge. When paused, it stops polling unless a Tick
// already ran and the matching PostTick still needs to complete.
//
// Parameters:
//   - context: The shared step context
func (g *gpioEmulationLoop) executeGPIOLoop(context *common.StepContext) {
	// Pin this goroutine to a dedicated OS thread. The external clock has no
	// wait-state handshake, so the Pico samples the bus a fixed time after each
	// edge; pinning keeps the Go scheduler from migrating or descheduling the
	// response loop mid-cycle, which would show up as missed edges and force a
	// slower clock. For best results also isolate a core (isolcpus / taskset).
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	defer func() {
		g.gpioLoopRunning.Store(false)
		if r := recover(); r != nil {
			g.handlePanic("GPIO", r)
		}
	}()

	g.gpioLoopRunning.Store(true)
	var lastState int
	stepper := gpioCycleStepper{}

	for !g.stop.Load() {
		if !g.pause.Load() || stepper.pendingPostTick {
			currentState, err := g.gpioController.Phi2().Value()
			if err != nil {
				log.Printf("Error reading GPIO: %v", err)
				continue
			}

			// External PHI2 falling edge is the emulator phase boundary.
			if lastState == 1 && currentState == 0 {
				g.tickStep(context, &stepper)
			} else {
				g.skipStep(context)
			}

			lastState = currentState
		}
	}
}

// tickStep advances the emulator on a GPIO clock edge.
// stepMu keeps Tick/PostTick from racing the draw loop while both goroutines use the
// same emulator state and step context.
//
// Parameters:
//   - context: The shared step context
//   - stepper: The GPIO cycle stepper that tracks the pending cycle phase
func (g *gpioEmulationLoop) tickStep(context *common.StepContext, stepper *gpioCycleStepper) {
	g.stepMu.Lock()
	defer g.stepMu.Unlock()

	stepper.step(context, g.config.Emulator, g.pause.Load())
}

// skipStep updates timing when polling did not find a GPIO clock edge.
// This keeps the current wall-clock time fresh without incrementing the emulated cycle.
//
// This runs on the vast majority of poll iterations, so it deliberately does NOT take
// stepMu: SkipCycle only refreshes a wall-clock timestamp used for display, and taking
// the lock here would make the hot polling path contend with drawing on every sample.
//
// Parameters:
//   - context: The shared step context
func (g *gpioEmulationLoop) skipStep(context *common.StepContext) {
	context.SkipCycle()
}

// executeDraw runs the display update loop.
// Drawing is timer-driven rather than GPIO-driven so the UI can refresh even when the
// external computer clock is slow or paused.
//
// Parameters:
//   - context: The shared step context
func (g *gpioEmulationLoop) executeDraw(context *common.StepContext) {
	defer func() {
		g.drawLoopRunning.Store(false)
		if r := recover(); r != nil {
			g.handlePanic("Draw", r)
		}
	}()

	ticker := time.NewTicker(time.Second / time.Duration(g.config.DisplayFPS))
	defer ticker.Stop()

	g.drawLoopRunning.Store(true)

	for !g.stop.Load() {
		<-ticker.C
		if g.stop.Load() {
			return
		}
		g.drawStep(context)
	}
}

// drawStep draws the emulator output while holding the shared step lock.
//
// Parameters:
//   - context: The shared step context
func (g *gpioEmulationLoop) drawStep(context *common.StepContext) {
	// Fast path: split the draw so only the state-reading refresh holds stepMu,
	// while the slow terminal flush runs unlocked. This is what keeps a frame
	// render from freezing the clock-response loop (and capping the clock rate).
	if g.config.RefreshDisplay != nil {
		g.stepMu.Lock()
		g.config.RefreshDisplay(context)
		g.stepMu.Unlock()

		if g.config.FlushDisplay != nil {
			g.config.FlushDisplay()
		}
		return
	}

	// Fallback: no split renderer wired, draw entirely under the lock.
	g.stepMu.Lock()
	defer g.stepMu.Unlock()

	g.config.Emulator.Draw(context)
}

/*******************************************************************************************
* Miscellaneous functions
********************************************************************************************/

// handlePanic stops the loop and calls the configured panic handler.
//
// Parameters:
//   - loopType: The loop where the panic happened
//   - r: The recovered panic value
func (g *gpioEmulationLoop) handlePanic(loopType string, r any) {
	g.Stop()
	if g.panicHandler != nil {
		if !g.panicHandler(loopType, r) {
			panic(r)
		}
	}
}
