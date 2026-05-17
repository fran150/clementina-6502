package emulation

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/fran150/clementina-6502/pkg/common"
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
