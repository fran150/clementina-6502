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
	// Creates a nanoseconds ticker based on the target speed in mhz
	ticker := time.NewTicker(time.Duration(1000/e.config.TargetSpeedMhz) * time.Nanosecond)

	for range ticker.C {
		e.computer.Step(context)

		if context.Stop {
			ticker.Stop()
			break
		}

		context.Next()
	}
}

func (e *Executor) runDisplayLoop(context *common.StepContext) {
	ticker := time.NewTicker(time.Duration(1000/e.config.DisplayFps) * time.Millisecond)

	for range ticker.C {
		e.computer.UpdateDisplay(context)

		if context.Stop {
			ticker.Stop()
			break
		}
	}
}
