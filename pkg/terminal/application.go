package terminal

import (
	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/fran150/clementina6502/pkg/computers"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ApplicationConfig struct {
	computers.EmulationLoopConfig
}

type Application struct {
	tvApplication *tview.Application
	computer      Computer
	executor      *computers.EmulationLoop
	config        *ApplicationConfig
}

func NewApplication(computer Computer) *Application {
	config := ApplicationConfig{
		computers.EmulationLoopConfig{
			TargetSpeedMhz: 1.05,
			DisplayFps:     10,
		},
	}

	return &Application{
		tvApplication: tview.NewApplication(),
		computer:      computer,
		executor:      computers.NewEmulationLoop(&config.EmulationLoopConfig),
		config:        &config,
	}
}

func (a *Application) Run() *common.StepContext {
	a.computer.Init(a.tvApplication, a.config)

	context := a.executor.Start(computers.EmulationLoopHandlers{
		Tick: func(context *common.StepContext) {
			a.computer.Tick(context)

			if context.Stop {
				a.tvApplication.Stop()
			}
		},
		Draw: func(context *common.StepContext) {
			a.computer.Draw(context)
			a.tvApplication.Draw()
		},
	})

	a.tvApplication.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		value := a.computer.KeyPressed(event, context)

		if context.Stop {
			a.tvApplication.Stop()
		}

		return value
	})

	if err := a.tvApplication.Run(); err != nil {
		panic(err)
	}

	a.executor.Stop()

	return context
}
