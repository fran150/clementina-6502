package computers

import (
	"time"

	"github.com/fran150/clementina6502/pkg/components/common"
)

type ExecutorConfig struct {
	TargetSpeedMhz float64
	DisplayFps     int
}

type Executor struct {
	computer Computer
	config   *ExecutorConfig
}

func CreateExecutor(computer Computer, config *ExecutorConfig) *Executor {
	return &Executor{
		computer: computer,
		config:   config,
	}
}

func (e *Executor) Run() common.StepContext {
	context := common.CreateStepContext()

	go e.runComputerLoop(&context)
	go e.runDisplayLoop(&context)

	e.computer.RunEventLoop()

	context.Stop = true

	return context
}

func (e *Executor) runComputerLoop(context *common.StepContext) {
	targetNano := int64(float64(time.Microsecond) / e.config.TargetSpeedMhz)
	var lastExecuted int64 = context.T + targetNano

	for !context.Stop {
		if (context.T - lastExecuted) > targetNano {
			lastExecuted = context.T

			e.computer.Step(context)
			context.NextCycle()
		}

		context.SkipCycle()
	}
}

func (e *Executor) runDisplayLoop(context *common.StepContext) {
	targetNano := int64(int(time.Second) / e.config.DisplayFps)
	var lastExecuted int64 = context.T + targetNano

	for !context.Stop {
		if (context.T - lastExecuted) > targetNano {
			lastExecuted = context.T
			e.computer.UpdateDisplay(context)
		}
	}
}
