//go:build (linux && arm) || (linux && arm64)

package emulation

import (
	"fmt"
	"log"
	"time"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/core"
)

const chipName = "gpiochip0"

// GPIOEmulationLoopConfig contains settings for GPIO-controlled emulation.
type GPIOEmulationLoopConfig struct {
	DisplayFPS int
	Emulator   LoopTarget
}

// gpioEmulationLoop manages GPIO-controlled emulation execution.
type gpioEmulationLoop struct {
	config          *GPIOEmulationLoopConfig
	panicHandler    func(loopType string, panicData any) bool
	gpioLoopRunning bool
	drawLoopRunning bool
	stop            bool
	pause           bool

	gpioController *common.GPIOController
}

// NewGPIOEmulationLoop creates a new GPIO-controlled emulation loop.
func NewGPIOEmulationLoop(config GPIOEmulationLoopConfig) core.EmulationLoop {
	if config.DisplayFPS <= 0 {
		config.DisplayFPS = 10
	}

	gpioController, err := common.GetGPIOInterface(chipName)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize GPIO interface: %v", err))
	}

	return &gpioEmulationLoop{
		config:          &config,
		gpioLoopRunning: false,
		drawLoopRunning: false,
		stop:            true,
		gpioController:  gpioController,
	}
}

// SetPanicHandler sets the panic handler function.
func (g *gpioEmulationLoop) SetPanicHandler(handler func(loopType string, panicData any) bool) {
	g.panicHandler = handler
}

// IsRunning checks if the emulation loop is currently running.
func (g *gpioEmulationLoop) IsRunning() bool {
	return g.drawLoopRunning || g.gpioLoopRunning
}

// IsPaused checks if the emulation loop is currently paused.
func (g *gpioEmulationLoop) IsPaused() bool {
	return g.pause
}

// IsStopping checks if the emulation loop is in the process of stopping.
func (g *gpioEmulationLoop) IsStopping() bool {
	return g.stop && g.IsRunning()
}

// Start begins the GPIO-controlled emulation loop.
func (g *gpioEmulationLoop) Start() (*common.StepContext, error) {
	if !g.IsRunning() && g.config.Emulator != nil {
		context := common.NewStepContext()

		g.pause = false
		g.stop = false

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
	g.stop = true
}

// Resume resumes the emulation loop execution.
func (g *gpioEmulationLoop) Resume() {
	g.pause = false
}

// Pause pauses the emulation loop execution.
func (g *gpioEmulationLoop) Pause() {
	g.pause = true
}

// executeGPIOLoop runs the GPIO-controlled emulation loop.
func (g *gpioEmulationLoop) executeGPIOLoop(context *common.StepContext) {
	defer func() {
		g.gpioLoopRunning = false
		if r := recover(); r != nil {
			g.handlePanic("GPIO", r)
		}
	}()

	g.gpioLoopRunning = true
	var lastState int

	for !g.stop {
		if !g.pause {
			currentState, err := g.gpioController.Clock().Value()
			if err != nil {
				log.Printf("Error reading GPIO: %v", err)
				continue
			}

			// Step on rising edge (0 -> 1)
			if lastState == 0 && currentState == 1 {
				g.config.Emulator.Tick(context)
				context.NextCycle()
			} else {
				context.SkipCycle()
			}

			lastState = currentState
		}

		time.Sleep(100 * time.Microsecond)
	}
}

// executeDraw runs the display update loop.
func (g *gpioEmulationLoop) executeDraw(context *common.StepContext) {
	defer func() {
		g.drawLoopRunning = false
		if r := recover(); r != nil {
			g.handlePanic("Draw", r)
		}
	}()

	ticker := time.NewTicker(time.Second / time.Duration(g.config.DisplayFPS))
	defer ticker.Stop()

	g.drawLoopRunning = true

	for !g.stop {
		<-ticker.C
		g.config.Emulator.Draw(context)
	}
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
