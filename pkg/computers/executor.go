package computers

import (
	"time"

	"github.com/fran150/clementina6502/pkg/components/common"
)

type RateExecutorConfig struct {
	TargetSpeedMhz float64
	DisplayFps     int
}

type RateExecutorHandlers struct {
	Tick func(context *common.StepContext)
	Draw func(context *common.StepContext)
}

type RateExecutor struct {
	config  *RateExecutorConfig
	context *common.StepContext
}

func CreateExecutor(config *RateExecutorConfig) *RateExecutor {
	return &RateExecutor{
		config: config,
	}
}

func (e *RateExecutor) GetConfig() *RateExecutorConfig {
	return e.config
}

func (e *RateExecutor) Start(handlers RateExecutorHandlers) *common.StepContext {
	if handlers.Tick == nil || handlers.Draw == nil {
		return nil
	}

	context := common.CreateStepContext()
	e.context = &context

	go e.executeLoop(e.context, handlers)

	return e.context
}

func (e *RateExecutor) Stop() {
	if e.context == nil {
		return
	}

	e.context.Stop = true
}

func (e *RateExecutor) executeLoop(context *common.StepContext, handlers RateExecutorHandlers) {
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
