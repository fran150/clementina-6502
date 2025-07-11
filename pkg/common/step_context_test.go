package common

import (
	"testing"
	"time"
)

func TestNewStepContext(t *testing.T) {
	context := NewStepContext()

	if context.Cycle != 0 {
		t.Errorf("Expected Cycle to be 0, got %d", context.Cycle)
	}

	if context.Stop != false {
		t.Errorf("Expected Stop to be false, got %t", context.Stop)
	}

	if context.B.IsZero() {
		t.Error("Expected B to be set to current time")
	}

	if context.T.IsZero() {
		t.Error("Expected T to be set to current time")
	}

	if !context.B.Equal(context.T) {
		t.Error("Expected B and T to be equal on initialization")
	}
}

func TestNextCycle(t *testing.T) {
	context := NewStepContext()
	initialCycle := context.Cycle
	initialTime := context.T

	time.Sleep(1 * time.Millisecond)
	context.NextCycle()

	if context.Cycle != initialCycle+1 {
		t.Errorf("Expected Cycle to increment from %d to %d, got %d", initialCycle, initialCycle+1, context.Cycle)
	}

	if !context.T.After(initialTime) {
		t.Error("Expected T to be updated to a later time after NextCycle")
	}
}

func TestMultipleNextCycle(t *testing.T) {
	context := NewStepContext()

	for i := uint64(1); i <= 5; i++ {
		context.NextCycle()
		if context.Cycle != i {
			t.Errorf("Expected Cycle to be %d, got %d", i, context.Cycle)
		}
	}
}

func TestStopFlag(t *testing.T) {
	context := NewStepContext()
	
	context.Stop = true
	if !context.Stop {
		t.Error("Expected Stop flag to be settable to true")
	}

	context.Stop = false
	if context.Stop {
		t.Error("Expected Stop flag to be settable to false")
	}
}