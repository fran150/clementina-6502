package clementina

import "testing"

// KERN_IRQ ($04E1) is the kernel jump table entry for its IRQ handler, so a
// program's own handler can chain to it (clementina-rom docs/memory-map.md §6).
// KERN_WARMSTART puts the kernel's handler back in IRQ_VEC ($FFFE).

const addrKernIRQ = 0x04E1

// TestKernelIRQChainsAndWarmStartRestoresVector checks that KERN_IRQ jumps to the
// handler the kernel installs at reset, that IRQs routed through it keep KJIFFY
// counting, and that a warm start restores IRQ_VEC.
func TestKernelIRQChainsAndWarmStartRestoresVector(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	// Checkpoints at $0280 (640) and up, as in basic_timing_test.go. Lines stay
	// under 40 characters so the screen editor reads each as one row.
	typeLine(computer, step, "A=PEEK(65534):B=PEEK(65535)")
	typeLine(computer, step, "POKE 640,A:POKE 641,B")
	handler := int(peek(computer, 640)) | int(peek(computer, 641))<<8

	require_eq(t, 0x4C, peek(computer, addrKernIRQ), "KERN_IRQ is a JMP")
	target := int(peek(computer, addrKernIRQ+1)) | int(peek(computer, addrKernIRQ+2))<<8
	if target != handler {
		t.Fatalf("KERN_IRQ jumps to $%04X, IRQ_VEC holds $%04X", target, handler)
	}

	// Route IRQs through KERN_IRQ. Timer 1's IRQ is off while the two vector
	// bytes change ($40/$C0 to VIA_IER at $C00E clear/set it).
	typeLine(computer, step, "POKE 49166,64")
	typeLine(computer, step, "POKE 65534,225:POKE 65535,4")
	typeLine(computer, step, "POKE 49166,192")
	typeLine(computer, step, "A=PEEK(65534):B=PEEK(65535)")
	typeLine(computer, step, "POKE 642,A:POKE 643,B")
	require_eq(t, 0xE1, peek(computer, 642), "IRQ_VEC low byte points at KERN_IRQ")
	require_eq(t, 0x04, peek(computer, 643), "IRQ_VEC high byte points at KERN_IRQ")

	t0 := jiffies(computer)
	editorTickN(computer, step, 600_000) // ~96 ticks at 6250 cycles/tick
	if dt := jiffies(computer) - t0; dt < 80 || dt > 112 {
		t.Fatalf("KJIFFY advanced by %d through KERN_IRQ, expected ~96", dt)
	}

	// SYS 1210 is KERN_WARMSTART ($04BA).
	typeLine(computer, step, "SYS 1210")
	typeLine(computer, step, "A=PEEK(65534):B=PEEK(65535)")
	typeLine(computer, step, "POKE 644,A:POKE 645,B")
	if got := int(peek(computer, 644)) | int(peek(computer, 645))<<8; got != handler {
		t.Fatalf("after KERN_WARMSTART IRQ_VEC holds $%04X, want $%04X", got, handler)
	}
}
