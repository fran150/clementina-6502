package terminal

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/computers"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ApplicationConfig holds configuration parameters for the terminal application.
// It extends the EmulationLoopConfig with terminal-specific settings.
type ApplicationConfig struct {
	computers.EmulationLoopConfig
}

// Application represents the main terminal application that manages the UI and emulation.
// It coordinates the terminal UI, computer emulation, and execution loop.
type Application struct {
	tvApp    *tview.Application
	computer Computer
	executor *computers.EmulationLoop
	config   *ApplicationConfig
}

// NewApplication creates a new terminal application with the provided computer and configuration.
// If no configuration is provided, default values are used.
//
// Parameters:
//   - computer: The computer system to emulate
//   - config: Optional configuration parameters (nil for defaults)
//
// Returns:
//   - A pointer to the initialized Application
func NewApplication(computer Computer, config *ApplicationConfig) *Application {
	if config == nil {
		config = &ApplicationConfig{
			computers.EmulationLoopConfig{
				TargetSpeedMhz: 1.05,
				DisplayFps:     10,
			},
		}
	}

	return &Application{
		tvApp:    tview.NewApplication(),
		computer: computer,
		executor: computers.NewEmulationLoop(&config.EmulationLoopConfig),
		config:   config,
	}
}

func (a *Application) Run() (*common.StepContext, error) {
	a.computer.Init(a.tvApp, a.config)

	context := a.executor.Start(computers.EmulationLoopHandlers{
		Tick: func(context *common.StepContext) {
			a.computer.Tick(context)

			if context.Stop {
				a.tvApp.Stop()
			}
		},
		Draw: func(context *common.StepContext) {
			a.computer.Draw(context)
			a.tvApp.Draw()
		},
	})

	a.tvApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		value := a.computer.KeyPressed(event, context)

		if context.Stop {
			a.tvApp.Stop()
		}

		return value
	})

	if err := a.tvApp.Run(); err != nil {
		return nil, err
	}

	return context, nil
}
