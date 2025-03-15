package common

import (
	"testing"
	"time"
)

func TestNewStepContext(t *testing.T) {
	ctx := NewStepContext()

	if ctx.Cycle != 0 {
		t.Errorf("Expected initial Cycle to be 0, got %d", ctx.Cycle)
	}

	if ctx.Stop {
		t.Error("Expected initial Stop to be false")
	}

	if ctx.T <= 0 {
		t.Errorf("Expected T to be greater than 0, got %d", ctx.T)
	}
}

func TestStepContext_NextCycle(t *testing.T) {
	ctx := NewStepContext()
	initialCycle := ctx.Cycle
	initialT := ctx.T

	// Wait a tiny bit to ensure time difference
	time.Sleep(time.Nanosecond)

	ctx.NextCycle()

	if ctx.Cycle != initialCycle+1 {
		t.Errorf("Expected Cycle to increment by 1, got %d", ctx.Cycle)
	}

	if ctx.T <= initialT {
		t.Errorf("Expected T to increase, initial: %d, current: %d", initialT, ctx.T)
	}
}

func TestStepContext_SkipCycle(t *testing.T) {
	ctx := NewStepContext()
	initialCycle := ctx.Cycle
	initialT := ctx.T

	// Wait a tiny bit to ensure time difference
	time.Sleep(time.Nanosecond)

	ctx.SkipCycle()

	if ctx.Cycle != initialCycle {
		t.Errorf("Expected Cycle to remain unchanged, got %d", ctx.Cycle)
	}

	if ctx.T <= initialT {
		t.Errorf("Expected T to increase, initial: %d, current: %d", initialT, ctx.T)
	}
}

func TestStepContext_TimeProgression(t *testing.T) {
	ctx := NewStepContext()
	var times []int64

	// Collect multiple timestamps
	for i := 0; i < 3; i++ {
		times = append(times, ctx.T)
		time.Sleep(time.Millisecond) // Wait to ensure time difference
		ctx.NextCycle()
	}

	// Verify timestamps are monotonically increasing
	for i := 1; i < len(times); i++ {
		if times[i] <= times[i-1] {
			t.Errorf("Time should strictly increase: time[%d]=%d, time[%d]=%d",
				i-1, times[i-1], i, times[i])
		}
	}
}

func TestStepContext_Stop(t *testing.T) {
	ctx := NewStepContext()

	if ctx.Stop {
		t.Error("Stop should be initially false")
	}

	ctx.Stop = true

	if !ctx.Stop {
		t.Error("Stop should be settable to true")
	}
}
