package mia

const (
	miaClockSnapshotOffset = 0x11078
	miaClockTicksPerDay    = 5184000
	miaCmdClockSnapshot    = 0x55
	miaCmdClockSetTI       = 0x56
)

type miaTimingState struct {
	epochNs     int64
	tiEpochNs   int64
	tiBaseTicks uint32
}

func (c *emulated_mia) timingReset() {
	c.timing = miaTimingState{epochNs: c.nowNs, tiEpochNs: c.nowNs}
}
func (c *emulated_mia) timingSetTI(ticks uint32) {
	if ticks >= miaClockTicksPerDay {
		return
	}
	c.timing.tiEpochNs = c.nowNs
	c.timing.tiBaseTicks = ticks
}
func (c *emulated_mia) timingSnapshot() {
	// Use elapsed wall time (StepContext.T), never PHI2 cycles.
	elapsed := c.nowNs - c.timing.epochNs
	if elapsed < 0 {
		elapsed = 0
	}
	ms := uint32(elapsed / 1_000_000)
	tiElapsed := c.nowNs - c.timing.tiEpochNs
	if tiElapsed < 0 {
		tiElapsed = 0
	}
	ticks := (uint64(c.timing.tiBaseTicks) + uint64(tiElapsed/1_000_000_000)*60 + uint64(tiElapsed%1_000_000_000)*60/1_000_000_000) % miaClockTicksPerDay
	for i := 0; i < 4; i++ {
		c.memory[miaClockSnapshotOffset+i] = byte(ms >> (8 * i))
	}
	for i := 0; i < 3; i++ {
		c.memory[miaClockSnapshotOffset+4+i] = byte(ticks >> (8 * i))
	}
	c.memory[miaClockSnapshotOffset+7] = 1
}
