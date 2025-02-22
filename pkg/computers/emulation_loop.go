package computers

import (
	"time"

	"github.com/fran150/clementina6502/pkg/components/common"
)

type EmulationLoopConfig struct {
	TargetSpeedMhz float64
	DisplayFps     int
}

type EmulationLoopHandlers struct {
	Tick func(context *common.StepContext)
	Draw func(context *common.StepContext)
}

type EmulationLoop struct {
	config  *EmulationLoopConfig
	context *common.StepContext
}

func NewEmulationLoop(config *EmulationLoopConfig) *EmulationLoop {
	return &EmulationLoop{
		config: config,
	}
}

func (e *EmulationLoop) GetConfig() *EmulationLoopConfig {
	return e.config
}

func (e *EmulationLoop) Start(handlers EmulationLoopHandlers) *common.StepContext {
	if handlers.Tick == nil || handlers.Draw == nil {
		return nil
	}

	context := common.NewStepContext()
	e.context = &context

	go e.executeLoop(e.context, handlers)

	return e.context
}

func (e *EmulationLoop) Stop() {
	if e.context == nil {
		return
	}

	e.context.Stop = true
}

func (e *EmulationLoop) executeLoop(context *common.StepContext, handlers EmulationLoopHandlers) {
	var lastFPSExecuted, lastTPSExecuted, targetFPSNano, targetTPSNano int64

	for !context.Stop {
		targetFPSNano = int64(int(time.Second) / e.config.DisplayFps)
		targetTPSNano = int64(float64(time.Microsecond) / e.config.TargetSpeedMhz)

		if (context.T - lastTPSExecuted) > targetTPSNano {
			lastTPSExecuted = context.T

			handlers.Tick(context)
			context.NextCycle()
		}

		if (context.T - lastFPSExecuted) > targetFPSNano {
			lastFPSExecuted = context.T
			handlers.Draw(context)
		}

		context.SkipCycle()
	}
}
