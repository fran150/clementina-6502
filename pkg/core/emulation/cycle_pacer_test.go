package emulation

import (
	"testing"
	"time"
)

// countTicks returns how many cycles the pacer runs back to back at one instant.
func countTicks(pacer *cyclePacer, now int64, nanosPerCycle float64) int {
	count := 0
	for pacer.tick(now, nanosPerCycle) {
		count++
	}
	return count
}

func TestCyclePacerRunsOneCyclePerPeriod(t *testing.T) {
	pacer := cyclePacer{}
	pacer.reset(0)

	if !pacer.tick(0, 1000) {
		t.Fatal("expected the first cycle to be due immediately")
	}
	if pacer.tick(999, 1000) {
		t.Fatal("expected no cycle before one period passed")
	}
	if !pacer.tick(1000, 1000) {
		t.Fatal("expected a cycle once one period passed")
	}
}

func TestCyclePacerCatchesUpLateCycles(t *testing.T) {
	pacer := cyclePacer{}
	pacer.reset(0)

	// Cycles were due at 0, 1000, ... 5000 ns.
	if got := countTicks(&pacer, 5000, 1000); got != 6 {
		t.Fatalf("expected 6 cycles to catch up, got %d", got)
	}
}

func TestCyclePacerKeepsFractionalPeriods(t *testing.T) {
	pacer := cyclePacer{}
	pacer.reset(0)

	// 1.2 MHz is 833.33 ns per cycle: 12000 periods fit in 10 ms, plus the cycle at 0.
	// Rounding the period down to 833 ns would run 12006.
	if got := countTicks(&pacer, 10_000_100, 1000.0/1.2); got != 12001 {
		t.Fatalf("expected 12001 cycles in 10 ms at 1.2 MHz, got %d", got)
	}
}

func TestCyclePacerDropsCyclesMissedByMoreThanMaxLag(t *testing.T) {
	pacer := cyclePacer{}
	pacer.reset(0)

	if got := countTicks(&pacer, int64(pacerMaxLag)+5000, 1000); got != 1 {
		t.Fatalf("expected the missed cycles to be dropped, got %d cycles", got)
	}
}

func TestCyclePacerResetForgetsMissedCycles(t *testing.T) {
	pacer := cyclePacer{}
	pacer.reset(0)
	pacer.reset(5000)

	if got := countTicks(&pacer, 5000, 1000); got != 1 {
		t.Fatalf("expected one cycle after reset, got %d", got)
	}
}

func TestCyclePacerRunsUnthrottledWithZeroPeriod(t *testing.T) {
	pacer := cyclePacer{}
	pacer.reset(0)

	for range 3 {
		if !pacer.tick(0, 0) {
			t.Fatal("expected every check to run a cycle with a zero period")
		}
	}
}

func TestCyclePacerWaitSleepsAtLeastOneSlice(t *testing.T) {
	pacer := cyclePacer{}
	pacer.reset(0)
	pacer.tick(0, 1000)

	if got := pacer.wait(500); got != pacerSleepSlice {
		t.Fatalf("expected a %v sleep when the next cycle is 500 ns away, got %v", pacerSleepSlice, got)
	}

	pacer.reset(0)
	pacer.tick(0, float64(5*time.Millisecond))

	if got := pacer.wait(0); got != 5*time.Millisecond {
		t.Fatalf("expected to sleep until the next cycle, got %v", got)
	}
}
