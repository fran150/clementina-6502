package mia

import "testing"

func TestPicoDataBusDirectionForCycle(t *testing.T) {
	tests := []struct {
		name         string
		miaSelected  bool
		writeEnabled bool
		expected     picoDataBusDirection
	}{
		{
			name:         "unselected keeps gpio input high impedance",
			miaSelected:  false,
			writeEnabled: false,
			expected:     picoDataBusHighZ,
		},
		{
			name:         "selected cpu read samples pico driven data",
			miaSelected:  true,
			writeEnabled: false,
			expected:     picoDataBusInput,
		},
		{
			name:         "selected cpu write drives data for pico",
			miaSelected:  true,
			writeEnabled: true,
			expected:     picoDataBusOutput,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := picoDataBusDirectionForCycle(test.miaSelected, test.writeEnabled)

			if actual != test.expected {
				t.Fatalf("expected %v, got %v", test.expected, actual)
			}
		})
	}
}
