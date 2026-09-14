package clementina

import "testing"

// KJIFFY ($00F7) is the kernel's free-running tick counter, driven by the VIA
// Timer-1 IRQ. Split out of the old play_test.go (PLAY itself is retired -
// see track_band_test.go) since this test is unrelated to PLAY.

const addrKJiffy = 0x00F7

func jiffies(computer *ClementinaComputer) int {
	return int(peek(computer, addrKJiffy)) | int(peek(computer, addrKJiffy+1))<<8
}

// TestKernelJiffyClockAdvances checks the VIA Timer-1 IRQ increments KJIFFY.
func TestKernelJiffyClockAdvances(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	t0 := jiffies(computer)
	editorTickN(computer, step, 600_000) // ~96 ticks at 6250 cycles/tick
	dt := jiffies(computer) - t0
	if dt < 80 || dt > 112 {
		t.Fatalf("KJIFFY advanced by %d over 600k cycles, expected ~96", dt)
	}
}
