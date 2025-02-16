package common

import (
	"time"
)

// An object pass to all Tick functions that contains
// one emulation step data
type StepContext struct {
	Cycle uint64 // Current cycle number
	T     int64  // Time of this execution in nanoseconds
	Stop  bool   // Return true to stop the execution
}

var beginning = time.Now()

func CreateStepContext() StepContext {
	return StepContext{
		Cycle: 0,
		T:     now(),
		Stop:  false,
	}
}

func (context *StepContext) SkipCycle() {
	context.T = now()
}

func (context *StepContext) NextCycle() {
	context.Cycle++
	context.T = now()
}

func now() int64 {
	return int64(time.Since(beginning))
}
