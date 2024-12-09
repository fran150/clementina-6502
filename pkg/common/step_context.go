package common

import "time"

// An object pass to all Tick functions that contains
// one emulation step data
type StepContext struct {
	Cycle uint64    // Current cycle number
	T     time.Time // Time of this execution
}
