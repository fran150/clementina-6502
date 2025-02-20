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
	Tick          func(*common.StepContext)
	UpdateDisplay func(*common.StepContext)
}

type RateExecutor struct {
	config   *RateExecutorConfig
	handlers *RateExecutorHandlers
}

func CreateExecutor(handlers *RateExecutorHandlers, config *RateExecutorConfig) *RateExecutor {
	return &RateExecutor{
		handlers: handlers,
		config:   config,
	}
}

func (e *RateExecutor) GetHandlers() *RateExecutorHandlers {
	return e.handlers
}

func (e *RateExecutor) GetConfig() *RateExecutorConfig {
	return e.config
}

func (e *RateExecutor) Start() *common.StepContext {
	context := common.CreateStepContext()

	go e.executeLoop(&context)

	return &context
}

func (e *RateExecutor) Stop(context *common.StepContext) {
	context.Stop = true
}

func (e *RateExecutor) executeLoop(context *common.StepContext) {
	targetFPSNano := int64(int(time.Second) / e.config.DisplayFps)
	targetTPSNano := int64(float64(time.Microsecond) / e.config.TargetSpeedMhz)

	var lastTPSExecuted int64 = context.T + targetTPSNano
	var lastFPSExecuted int64 = context.T + targetFPSNano

	for !context.Stop {
		if (context.T - lastTPSExecuted) > targetTPSNano {
			lastTPSExecuted = context.T

			e.handlers.Tick(context)
			context.NextCycle()
		}

		if (context.T - lastFPSExecuted) > targetFPSNano {
			lastFPSExecuted = context.T
			e.handlers.UpdateDisplay(context)
		}

		context.SkipCycle()
	}
}
