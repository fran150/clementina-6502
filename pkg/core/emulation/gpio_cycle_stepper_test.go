package emulation

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/common"
)

type testLoopTarget struct {
	events          []string
	paused          bool
	pauseOnPostTick bool
}

func (t *testLoopTarget) PostTick(context *common.StepContext) {
	t.events = append(t.events, "posttick")
	if t.pauseOnPostTick {
		t.paused = true
	}
}

func (t *testLoopTarget) Tick(context *common.StepContext) {
	t.events = append(t.events, "tick")
}

func (t *testLoopTarget) Draw(context *common.StepContext) {}
func (t *testLoopTarget) Pause()                           { t.paused = true }
func (t *testLoopTarget) Resume()                          { t.paused = false }
func (t *testLoopTarget) IsPaused() bool                   { return t.paused }

func TestGPIOCycleStepperTicksThenPostTicksOnNextEdge(t *testing.T) {
	context := common.NewStepContext()
	target := &testLoopTarget{}
	stepper := gpioCycleStepper{}

	stepper.step(&context, target, false)
	if context.Cycle != 0 {
		t.Fatalf("expected first edge to tick without completing a cycle, got cycle %d", context.Cycle)
	}

	stepper.step(&context, target, false)
	if context.Cycle != 1 {
		t.Fatalf("expected second edge to post-tick one cycle, got cycle %d", context.Cycle)
	}

	expected := []string{"tick", "posttick", "tick"}
	if !equalStringSlices(target.events, expected) {
		t.Fatalf("expected events %v, got %v", expected, target.events)
	}
}

func TestGPIOCycleStepperPausePostTicksTickedCycle(t *testing.T) {
	context := common.NewStepContext()
	target := &testLoopTarget{}
	stepper := gpioCycleStepper{}

	stepper.step(&context, target, false)
	stepper.step(&context, target, true)

	if context.Cycle != 1 {
		t.Fatalf("expected pause edge to post-tick one pending cycle, got cycle %d", context.Cycle)
	}

	expected := []string{"tick", "posttick"}
	if !equalStringSlices(target.events, expected) {
		t.Fatalf("expected events %v, got %v", expected, target.events)
	}
}

func TestGPIOCycleStepperDoesNotTickAfterPostTickPausesTarget(t *testing.T) {
	context := common.NewStepContext()
	target := &testLoopTarget{pauseOnPostTick: true}
	stepper := gpioCycleStepper{}

	stepper.step(&context, target, false)
	stepper.step(&context, target, false)

	if context.Cycle != 1 {
		t.Fatalf("expected one completed cycle, got cycle %d", context.Cycle)
	}

	expected := []string{"tick", "posttick"}
	if !equalStringSlices(target.events, expected) {
		t.Fatalf("expected events %v, got %v", expected, target.events)
	}
}

func equalStringSlices(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
