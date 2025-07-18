// Package common provides shared utilities and types used across the emulator.

/*
StepContext represents the execution context for a single step in the emulation cycle.
It tracks important metrics and control flags that are used across various components
during emulation.

The StepContext is passed to all Tick functions throughout the emulator to maintain
consistent timing and cycle counting across components.

Usage:

	context := NewStepContext()
	// Pass to component Tick functions
	component.Tick(&context)
*/
package common

import "time"

// StepContext holds the state for a single emulation step.
// It is passed to all Tick functions to maintain synchronization
// and timing across the emulated system.
type StepContext struct {
	// Cycle represents the current cycle number in the emulation.
	// It starts at 0 and increments with each cycle.
	Cycle uint64

	// CycleT represents the current time in nanoseconds of the last time a cycle
	// was executed. Components can compare the current value of T with CycleT
	// to check how much time passed between the last time a cycle was executed
	CycleT int64

	// T represents the current time in nanoseconds since the emulation started.
	// This is used for timing and synchronization purposes.
	// Components can store previous cycle T and compare it with the current one
	// to calculate the time passed between both cycles.
	T int64

	// Stop is a control flag that can be set to true to halt the emulation.
	// Components should check this flag and respect it by stopping their execution.
	Stop bool
}

// beginning stores the timestamp when the emulation started.
// It is used as a reference point for all timing calculations.
var beginning = time.Now()

// NewStepContext creates and initializes a new StepContext with default values.
// The Cycle is set to 0, T is set to the current time since emulation start,
// and Stop is set to false.
func NewStepContext() StepContext {
	return StepContext{
		Cycle:  0,
		CycleT: 0,
		T:      0,
		Stop:   false,
	}
}

// SkipCycle updates the timing information without incrementing the cycle counter.
// This is used by the emulation when skipping emulation cycles. It shouldn't be called by any components
// as it used directly by the EmulationLoop in the computers package.
func (context *StepContext) SkipCycle() {
	context.T = now()
}

// NextCycle advances the emulation by one cycle and updates the timing information.
// This is used by the emulation when completing an emulation cycle. It shouldn't be called by any components
// as it used directly by the EmulationLoop in the computers package.
func (context *StepContext) NextCycle() {
	context.Cycle++
	context.T = now()
	context.CycleT = context.T
}

// now returns the number of nanoseconds that have elapsed since the emulation started.
// It is used internally to maintain accurate timing information.
func now() int64 {
	return int64(time.Since(beginning))
}
