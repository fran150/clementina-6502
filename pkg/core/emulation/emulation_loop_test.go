package emulation

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/core"
)

type testSpeedController struct{}

func (testSpeedController) SpeedUp()                        {}
func (testSpeedController) SpeedDown()                      {}
func (testSpeedController) GetTargetSpeed() float64         { return 1 }
func (testSpeedController) SetTargetSpeed(speedMhz float64) {}
func (testSpeedController) GetNanosPerCycle() float64       { return 0 }

type serializedLoopTarget struct {
	active  atomic.Bool
	overlap atomic.Bool
	ticks   atomic.Int64
	draws   atomic.Int64
}

func (t *serializedLoopTarget) Tick(context *common.StepContext) {
	t.enter()
	defer t.exit()

	t.ticks.Add(1)
	time.Sleep(time.Millisecond)
}

func (t *serializedLoopTarget) PostTick(context *common.StepContext) {
	t.enter()
	defer t.exit()

	time.Sleep(time.Millisecond)
}

func (t *serializedLoopTarget) Draw(context *common.StepContext) {
	t.enter()
	defer t.exit()

	t.draws.Add(1)
	time.Sleep(time.Millisecond)
}

func (t *serializedLoopTarget) Pause()         {}
func (t *serializedLoopTarget) Resume()        {}
func (t *serializedLoopTarget) IsPaused() bool { return false }

func (t *serializedLoopTarget) enter() {
	if !t.active.CompareAndSwap(false, true) {
		t.overlap.Store(true)
	}
}

func (t *serializedLoopTarget) exit() {
	t.active.Store(false)
}

func TestEmulationLoopSerializesTickAndDraw(t *testing.T) {
	target := &serializedLoopTarget{}
	loop := NewEmulationLoop(EmulationLoopConfig{
		SpeedController: testSpeedController{},
		DisplayFPS:      1000,
		RefreshNanos:    1,
		Emulator:        target,
	})

	if _, err := loop.Start(); err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}

	if !loop.IsRunning() {
		t.Fatal("expected loop to report running immediately after start")
	}

	time.Sleep(25 * time.Millisecond)
	loop.Stop()

	deadline := time.Now().Add(time.Second)
	for loop.IsRunning() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}

	if loop.IsRunning() {
		t.Fatal("loop did not stop before deadline")
	}

	if target.overlap.Load() {
		t.Fatal("tick and draw overlapped")
	}

	if target.ticks.Load() == 0 {
		t.Fatal("expected at least one tick")
	}

	if target.draws.Load() == 0 {
		t.Fatal("expected at least one draw")
	}
}

type pacedSpeedController struct {
	nanosPerCycle float64
}

func (pacedSpeedController) SpeedUp()                        {}
func (pacedSpeedController) SpeedDown()                      {}
func (s pacedSpeedController) GetTargetSpeed() float64       { return 1000 / s.nanosPerCycle }
func (pacedSpeedController) SetTargetSpeed(speedMhz float64) {}
func (s pacedSpeedController) GetNanosPerCycle() float64     { return s.nanosPerCycle }

type countingLoopTarget struct {
	ticks atomic.Uint64
}

func (t *countingLoopTarget) Tick(context *common.StepContext)     { t.ticks.Add(1) }
func (t *countingLoopTarget) PostTick(context *common.StepContext) {}
func (t *countingLoopTarget) Draw(context *common.StepContext)     {}
func (t *countingLoopTarget) Pause()                               {}
func (t *countingLoopTarget) Resume()                              {}
func (t *countingLoopTarget) IsPaused() bool                       { return false }

// startPacedLoop runs a loop at 0.5 MHz (2000 ns per cycle) and stops it when the test ends.
func startPacedLoop(t *testing.T) (core.EmulationLoop, *countingLoopTarget) {
	t.Helper()

	target := &countingLoopTarget{}
	loop := NewEmulationLoop(EmulationLoopConfig{
		SpeedController: pacedSpeedController{nanosPerCycle: 2000},
		DisplayFPS:      15,
		Emulator:        target,
	})

	if _, err := loop.Start(); err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}
	t.Cleanup(loop.Stop)

	return loop, target
}

func TestEmulationLoopRunsAtTargetSpeed(t *testing.T) {
	_, target := startPacedLoop(t)

	time.Sleep(100 * time.Millisecond)
	c0, t0 := target.ticks.Load(), time.Now()
	time.Sleep(500 * time.Millisecond)
	c1, t1 := target.ticks.Load(), time.Now()

	mhz := float64(c1-c0) / t1.Sub(t0).Seconds() / 1_000_000
	if mhz < 0.475 || mhz > 0.525 {
		t.Fatalf("expected about 0.5 MHz, got %0.3f MHz", mhz)
	}
}

func TestEmulationLoopResumeDoesNotReplayPausedTime(t *testing.T) {
	loop, target := startPacedLoop(t)

	time.Sleep(50 * time.Millisecond)
	loop.Pause()
	time.Sleep(10 * time.Millisecond)
	paused := target.ticks.Load()
	time.Sleep(200 * time.Millisecond)

	if target.ticks.Load() != paused {
		t.Fatal("expected no cycles while paused")
	}

	loop.Resume()
	time.Sleep(50 * time.Millisecond)

	// 50 ms at 0.5 MHz is 25000 cycles; replaying the 200 ms pause would add 100000.
	if ran := target.ticks.Load() - paused; ran == 0 || ran > 40000 {
		t.Fatalf("expected about 25000 cycles after resume, got %d", ran)
	}
}
