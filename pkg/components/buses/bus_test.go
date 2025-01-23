package buses

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Tests reading, writing and checking different lines from a 8 bit bus
func Test8BitBus(t *testing.T) {
	type lineTest struct {
		value    uint8
		expected [8]bool
	}

	var tests = []lineTest{
		// Note that the lines are reversed (starting with the least significant),
		// if read as binary must be read right to left.
		// Lines         0     1     2     3     4     5     6     7
		{0xFF, [8]bool{true, true, true, true, true, true, true, true}},
		{0x00, [8]bool{false, false, false, false, false, false, false, false}},
		{0x55, [8]bool{true, false, true, false, true, false, true, false}},
		{0xAA, [8]bool{false, true, false, true, false, true, false, true}},
		{0x11, [8]bool{true, false, false, false, true, false, false, false}},
	}

	bus := Create8BitStandaloneBus()

	for _, test := range tests {
		// Tests writing to the bus
		bus.Write(test.value)

		for i := range uint8(8) {
			got := bus.GetBusLine(i)

			if got.Status() != test.expected[i] {
				t.Errorf("For %x, line %v expected %v, got %v", test.value, i, test.expected[i], got)
			}
		}

		// Test reading from the bus
		onBus := bus.Read()
		t.Logf("Read binary value %b", onBus)
		assert.Equal(t, onBus, test.value, "The number read from the bus doesn't match the test value")
	}
}

// Tests reading, writing and checking different lines from a 16 bit bus
func Test16BitBus(t *testing.T) {
	type lineTest struct {
		value    uint16
		expected [16]bool
	}

	var tests = []lineTest{
		// Note that the lines are reversed (starting with the least significant),
		// if read as binary must be read right to left.
		// Lines            0     1     2     3     4     5     6     7     8     9     A     B     C     D     E     F
		{0xFFFF, [16]bool{true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true}},
		{0x0000, [16]bool{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false}},
		{0x5555, [16]bool{true, false, true, false, true, false, true, false, true, false, true, false, true, false, true, false}},
		{0xAAAA, [16]bool{false, true, false, true, false, true, false, true, false, true, false, true, false, true, false, true}},
		{0x1111, [16]bool{true, false, false, false, true, false, false, false, true, false, false, false, true, false, false, false}},
	}

	bus := Create16BitStandaloneBus()

	for _, test := range tests {
		// Tests writing to the bus
		bus.Write(test.value)

		for i := range uint8(8) {
			got := bus.GetBusLine(i)

			if got.Status() != test.expected[i] {
				t.Errorf("For %x, line %v expected %v, got %v", test.value, i, test.expected[i], got)
			}
		}

		// Test reading from the bus
		onBus := bus.Read()
		t.Logf("Read binary value %b", onBus)
		assert.Equal(t, onBus, test.value, "The number read from the bus doesn't match the test value")
	}
}
