package mia

import (
	"encoding/binary"
	"testing"
)

func TestTimingClockProtocol(t *testing.T) {
	c := newEmulatedMiaTestCircuit().chip
	c.nowNs = 1_000_000_000
	c.timingReset()
	snap := func() (uint32, uint32) {
		c.executeCommand(miaCmdClockSnapshot, [3]byte{})
		p := c.memory[miaClockSnapshotOffset:]
		if p[7] != 1 {
			t.Fatal("version")
		}
		return binary.LittleEndian.Uint32(p), uint32(p[4]) | uint32(p[5])<<8 | uint32(p[6])<<16
	}
	for _, hz := range []uint32{100_000, 1_000_000, 8_000_000} {
		c.appliedPhi2Hz = hz
		c.nowNs = 2_500_000_000
		ms, ti := snap()
		if ms != 1500 || ti != 90 {
			t.Fatalf("clock depends on CPU speed %d: %d,%d", hz, ms, ti)
		}
	}
	before := append([]byte(nil), c.memory[miaClockSnapshotOffset:miaClockSnapshotOffset+8]...)
	c.nowNs += 1_000_000_000
	c.inputService()
	for i, b := range before {
		if c.memory[miaClockSnapshotOffset+i] != b {
			t.Fatal("snapshot changed without command")
		}
	}
	c.executeCommand(miaCmdClockSetTI, [3]byte{0xff, 0x19, 0x4f}) // last tick of day
	_, ti := snap()
	if ti != 5183999 {
		t.Fatal("24-bit set")
	}
	c.nowNs += 16_666_666
	_, ti = snap()
	if ti != 5183999 {
		t.Fatal("early TI tick")
	}
	c.nowNs++
	ms, ti := snap()
	if ti != 0 || ms < 2500 {
		t.Fatal("midnight wrap or uptime reset")
	}
	c.executeCommand(miaCmdClockSetTI, [3]byte{0, 0x1a, 0x4f})
	_, ti = snap()
	if ti != 0 {
		t.Fatal("invalid set accepted")
	}
	c.nowNs = c.timing.epochNs + (int64(1)<<32)*1_000_000 + 7_000_000
	ms, _ = snap()
	if ms != 7 {
		t.Fatalf("millisecond wrap: %d", ms)
	}
	c.timingReset()
	ms, ti = snap()
	if ms != 0 || ti != 0 {
		t.Fatal("runtime reset")
	}
}
