package emulation

import "github.com/fran150/clementina-6502/pkg/common"

// gpioCycleStepper tracks the split GPIO clock cycle.
// The GPIO loop receives one event per external rising PHI2 edge. A full emulator cycle
// needs two phases: Tick drives the buses, and PostTick lets components consume the values
// that were driven. pendingPostTick records that Tick already ran and the next GPIO edge
// must finish that same cycle before starting a new one.
type gpioCycleStepper struct {
	pendingPostTick bool
}

// step advances the emulator by the phase required at the current GPIO edge.
// When there is a pending PostTick, it always completes first so the emulator does not
// leave buses half-driven if the user pauses between the two phases. A new Tick starts only
// when the loop is not paused after that completion.
//
// Parameters:
//   - context: The current step context
//   - target: The emulator target controlled by the GPIO loop
//   - paused: true when the loop was paused before this GPIO edge
func (s *gpioCycleStepper) step(context *common.StepContext, target LoopTarget, paused bool) {
	if s.pendingPostTick {
		target.PostTick(context)
		context.NextCycle()
		s.pendingPostTick = false
		paused = paused || target.IsPaused()
	}

	if !paused {
		target.Tick(context)
		s.pendingPostTick = true
	}
}
