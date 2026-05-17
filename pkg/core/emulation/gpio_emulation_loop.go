//go:build (linux && arm) || (linux && arm64)

package emulation

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/core"
)

// GPIOEmulationLoopConfig contains settings for GPIO-controlled emulation.
type GPIOEmulationLoopConfig struct {
	DisplayFPS int
	Emulator   LoopTarget
	ChipName   string
}

// gpioEmulationLoop manages GPIO-controlled emulation execution.
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

// NewGPIOEmulationLoop creates a new GPIO-controlled emulation loop.
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

// SetPanicHandler sets the panic handler function.
func (g *gpioEmulationLoop) SetPanicHandler(handler func(loopType string, panicData any) bool) {
	g.panicHandler = handler
}

// IsRunning checks if the emulation loop is currently running.
func (g *gpioEmulationLoop) IsRunning() bool {
	return g.drawLoopRunning.Load() || g.gpioLoopRunning.Load()
}

// IsPaused checks if the emulation loop is currently paused.
func (g *gpioEmulationLoop) IsPaused() bool {
	return g.pause.Load()
}

// IsStopping checks if the emulation loop is in the process of stopping.
func (g *gpioEmulationLoop) IsStopping() bool {
	return g.stop.Load() && g.IsRunning()
}

// Start begins the GPIO-controlled emulation loop.
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

// Stop signals the emulation loop to stop execution.
func (g *gpioEmulationLoop) Stop() {
	g.stop.Store(true)
}

// Resume resumes the emulation loop execution.
func (g *gpioEmulationLoop) Resume() {
	g.pause.Store(false)
}

// Pause pauses the emulation loop execution.
func (g *gpioEmulationLoop) Pause() {
	g.pause.Store(true)
}

// executeGPIOLoop runs the GPIO-controlled emulation loop.
func (g *gpioEmulationLoop) executeGPIOLoop(context *common.StepContext) {
	defer func() {
		g.gpioLoopRunning.Store(false)
		if r := recover(); r != nil {
			g.handlePanic("GPIO", r)
		}
	}()

	g.gpioLoopRunning.Store(true)
	var lastState int

	for !g.stop.Load() {
		if !g.pause.Load() {
			currentState, err := g.gpioController.Phi2().Value()
			if err != nil {
				log.Printf("Error reading GPIO: %v", err)
				continue
			}

			// Step on falling edge (1 -> 0)
			if lastState == 1 && currentState == 0 {
				g.tickStep(context)
			} else {
				g.skipStep(context)
			}

			lastState = currentState
		}
	}
}

func (g *gpioEmulationLoop) tickStep(context *common.StepContext) {
	g.stepMu.Lock()
	defer g.stepMu.Unlock()

	g.config.Emulator.Tick(context)
	context.NextCycle()
}

func (g *gpioEmulationLoop) skipStep(context *common.StepContext) {
	g.stepMu.Lock()
	defer g.stepMu.Unlock()

	context.SkipCycle()
}

// executeDraw runs the display update loop.
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

func (g *gpioEmulationLoop) drawStep(context *common.StepContext) {
	g.stepMu.Lock()
	defer g.stepMu.Unlock()

	g.config.Emulator.Draw(context)
}

// handlePanic triggers the execution of a handler before panicking.
func (g *gpioEmulationLoop) handlePanic(loopType string, r any) {
	g.Stop()
	if g.panicHandler != nil {
		if !g.panicHandler(loopType, r) {
			panic(r)
		}
	}
}
