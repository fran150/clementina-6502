package mia

import "testing"

func TestInputRepeatConfiguration(t *testing.T) {
	c := newEmulatedMiaTestCircuit().chip
	c.state = miaStateNormal
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.inputKeyRepeats(79) || c.inputKeyRepeats(4) {
		t.Fatal("defaults")
	}
	c.executeCommand(0x53, [3]byte{25, 0, 0})
	c.executeCommand(0x52, [3]byte{232, 3, 0})
	if c.input.repeatDelayNs != 1_000_000_000 || c.input.repeatIntervalNs != 25_000_000 {
		t.Fatal("timing commands")
	}
	c.memory[miaInputKeyboardBitmapOffset+9] = 128
	c.nowNs = 0
	c.inputArmRepeat(79, 29)
	c.nowNs = 999_999_999
	c.inputRepeatService()
	if c.registers[miaRegInputCharCount] != 0 {
		t.Fatal("early repeat")
	}
	c.nowNs++
	c.inputRepeatService()
	if c.registers[miaRegInputCharCount] != 1 {
		t.Fatal("missing repeat")
	}
	c.executeCommand(0x54, [3]byte{79, 0, 0})
	if c.inputKeyRepeats(79) || c.input.repeatUsage != 0 {
		t.Fatal("per-key disable")
	}
	c.executeCommand(0x54, [3]byte{4, 1, 0})
	if !c.inputKeyRepeats(4) {
		t.Fatal("per-key enable")
	}
	c.executeCommand(0x52, [3]byte{})
	if c.inputKeyRepeats(4) {
		t.Fatal("global disable")
	}
	c.inputResetRuntimeState()
	if !c.inputKeyRepeats(79) || c.inputKeyRepeats(4) {
		t.Fatal("reset defaults")
	}
}
