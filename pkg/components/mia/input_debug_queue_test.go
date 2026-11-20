package mia

import "testing"

// A debugger types launch commands with DebugQueueInput while the video
// client's Wi-Fi input owns the text FIFO; those bytes must still arrive.
func TestDebugQueueInputReachesFIFOInWifiMode(t *testing.T) {
	c := NewEmulatedMia().(*emulated_mia)
	c.mu.Lock()
	c.input.mode = miaInputModeWifi
	c.mu.Unlock()
	c.DebugQueueInput('R')
	c.mu.Lock()
	c.inputConsoleByte('X') // a console byte is still ignored outside console mode
	c.mu.Unlock()
	if got := c.inputFifoCount(c.input.textHead, c.input.textTail); got != 1 {
		t.Fatalf("FIFO holds %d bytes, want 1", got)
	}
}
