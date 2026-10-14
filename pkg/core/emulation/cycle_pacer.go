package emulation

import "time"

// pacerSleepSlice is how long the loop sleeps once it is ahead of schedule. The cycles
// that fall due meanwhile then run back to back, so the computer advances in bursts of
// about this length. Go's timers resolve well below it on Linux, macOS and Windows.
const pacerSleepSlice = time.Millisecond

// pacerMaxLag is how far behind schedule the pacer may fall before it gives up on the
// missed cycles. It is longer than a sleep overshoot or a display refresh, so normal
// running never loses cycles, but short enough that a stall (or a host too slow for the
// target speed) does not replay a long burst afterwards.
const pacerMaxLag = 20 * time.Millisecond

// cyclePacer schedules emulated cycles against wall time.
// Each cycle falls due one cycle period after the previous one was due, not after it
// actually ran, so a late cycle is made up by running the next ones back to back and the
// average speed matches the target whenever the host can keep up.
type cyclePacer struct {
	// due is when the next cycle is due, in nanoseconds since the emulation started.
	// It is a float so fractional cycle periods (833.3 ns at 1.2 MHz) do not drift.
	due float64
}

// reset makes the next cycle due now and forgets any missed cycles. The loop calls it
// while paused so resuming or stepping does not replay the paused time.
//
// Parameters:
//   - now: The current time in nanoseconds since the emulation started
func (p *cyclePacer) reset(now int64) {
	p.due = float64(now)
}

// tick reports whether a cycle is due and, if so, schedules the next one.
//
// Parameters:
//   - now: The current time in nanoseconds since the emulation started
//   - nanosPerCycle: The target cycle period; 0 runs as fast as the host allows
//
// Returns:
//   - true when the loop should run a cycle now
func (p *cyclePacer) tick(now int64, nanosPerCycle float64) bool {
	current := float64(now)
	if current < p.due {
		return false
	}

	if current-p.due > float64(pacerMaxLag) {
		p.due = current
	}

	p.due += nanosPerCycle
	return true
}

// wait returns how long the loop should sleep when no cycle is due. It sleeps at least
// pacerSleepSlice, so cycles run in bursts instead of one sleep per cycle.
//
// Parameters:
//   - now: The current time in nanoseconds since the emulation started
//
// Returns:
//   - The time to sleep before checking the schedule again
func (p *cyclePacer) wait(now int64) time.Duration {
	return max(time.Duration(p.due-float64(now)), pacerSleepSlice)
}
