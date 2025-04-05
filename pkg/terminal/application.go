package terminal

import (
	"github.com/fran150/clementina6502/pkg/common"
	"github.com/fran150/clementina6502/pkg/computers"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ApplicationConfig struct {
	computers.EmulationLoopConfig
}

type Application struct {
	tvApp    *tview.Application
	computer Computer
	executor *computers.EmulationLoop
	config   *ApplicationConfig
}

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
