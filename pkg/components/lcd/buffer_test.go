package lcd

import "testing"

func TestNewLcdBuffer(t *testing.T) {
	buf := newLcdBuffer()

	if !buf.is8BitMode {
		t.Error("New buffer should be in 8-bit mode by default")
	}
	if buf.value != 0x00 {
		t.Error("New buffer should have value 0x00")
	}
	if buf.index != BUFFER_EMPTY_INDEX {
		t.Error("New buffer should have empty index")
	}
}

func TestWriteReadNibbles(t *testing.T) {
	buf := newLcdBuffer()

	// Test MSNibble
	buf.writeMSNibble(0xF0)
	if buf.readMSNibble() != 0xF0 {
		t.Errorf("Expected MSNibble 0xF0, got %#02x", buf.readMSNibble())
	}

	// Test LSNibble
	buf.writeLSNibble(0xF0)
	if buf.readLSNibble() != 0xF0 {
		t.Errorf("Expected LSNibble 0xF0, got %#02x", buf.readLSNibble())
	}
}

func TestPushPull8BitMode(t *testing.T) {
	buf := newLcdBuffer()

	// Test pushing in 8-bit mode
	buf.push(0xAA)
	if !buf.isFull() {
		t.Error("Buffer should be full after push in 8-bit mode")
	}
	if buf.value != 0xAA {
		t.Errorf("Expected value 0xAA, got %#02x", buf.value)
	}

	// Test pulling in 8-bit mode
	value := buf.pull()
	if value != 0xAA {
		t.Errorf("Expected pulled value 0xAA, got %#02x", value)
	}
	if !buf.isEmpty() {
		t.Error("Buffer should be empty after pull in 8-bit mode")
	}
}

func TestPushPull4BitMode(t *testing.T) {
	buf := newLcdBuffer()
	buf.is8BitMode = false

	// Test pushing in 4-bit mode (two operations)
	buf.push(0xA0) // First nibble
	if buf.isFull() {
		t.Error("Buffer should not be full after first nibble")
	}

	buf.push(0x40) // Second nibble
	if !buf.isFull() {
		t.Error("Buffer should be full after second nibble")
	}

	// Test pulling in 4-bit mode
	value := buf.pull() // First nibble
	if value != 0xA0 {
		t.Errorf("Expected first pulled value 0xA0, got %#02x", value)
	}

	value = buf.pull() // Second nibble
	if value != 0x40 {
		t.Errorf("Expected second pulled value 0x40, got %#02x", value)
	}
}

func TestLoadAndFlush(t *testing.T) {
	buf := newLcdBuffer()

	// Test load
	buf.load(0xFF)
	if !buf.isFull() {
		t.Error("Buffer should be full after load")
	}
	if buf.value != 0xFF {
		t.Errorf("Expected loaded value 0xFF, got %#02x", buf.value)
	}

	// Test flush
	buf.flush()
	if !buf.isEmpty() {
		t.Error("Buffer should be empty after flush")
	}
	if buf.value != 0x00 {
		t.Errorf("Expected flushed value 0x00, got %#02x", buf.value)
	}
}

func TestBufferStateChecks(t *testing.T) {
	buf := newLcdBuffer()

	if !buf.isEmpty() {
		t.Error("New buffer should be empty")
	}

	buf.push(0xFF)
	if !buf.isFull() {
		t.Error("Buffer should be full after push")
	}

	buf.pull()
	if !buf.isEmpty() {
		t.Error("Buffer should be empty after pull")
	}
}

func TestPushWhenFull(t *testing.T) {
	buf := newLcdBuffer()

	// Fill the buffer
	buf.push(0xAA)
	originalValue := buf.value

	// Try to push when full
	buf.push(0xFF)
	if buf.value != originalValue {
		t.Error("Push on full buffer should not change the value")
	}
}

func TestPullWhenEmpty(t *testing.T) {
	buf := newLcdBuffer()

	value := buf.pull()
	if value != 0x00 {
		t.Errorf("Pull on empty buffer should return 0x00, got %#02x", value)
	}
}
