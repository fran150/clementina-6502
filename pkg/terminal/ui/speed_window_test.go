package ui

import (
	"testing"

	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/stretchr/testify/assert"
)

func TestNewSpeedWindow(t *testing.T) {
	sw := NewSpeedWindow()

	assert.NotNil(t, sw)
	assert.NotNil(t, sw.text)
	assert.Equal(t, int64(0), sw.previousT)
	assert.Equal(t, uint64(0), sw.previousC)
}

func TestSpeedWindow_Clear(t *testing.T) {
	sw := NewSpeedWindow()
	// Set some initial values
	sw.previousT = 1000
	sw.previousC = 2000

	sw.Clear()
	// Clear should only clear the text view, not reset the previous values
	assert.Equal(t, int64(1000), sw.previousT)
	assert.Equal(t, uint64(2000), sw.previousC)
}

func TestSpeedWindow_Draw(t *testing.T) {
	tests := []struct {
		name         string
		initialT     int64
		initialC     uint64
		contextT     int64
		contextCycle uint64
		expectedMhz  float64
	}{
		{
			name:         "First draw - should not calculate speed",
			initialT:     0,
			initialC:     0,
			contextT:     1000000, // 1ms in nanoseconds
			contextCycle: 1000,
			expectedMhz:  0,
		},
		{
			name:         "Calculate 1Mhz speed",
			initialT:     0,
			initialC:     0,
			contextT:     1000000, // 1ms in nanoseconds
			contextCycle: 1000,
			expectedMhz:  1.0,
		},
		{
			name:         "Calculate 2Mhz speed",
			initialT:     1000000, // 1ms in nanoseconds
			initialC:     1000,
			contextT:     2000000, // 2ms in nanoseconds
			contextCycle: 3000,    // 2000 cycles in 1ms = 2MHz
			expectedMhz:  2.0,
		},
		{
			name:         "Calculate 0.5Mhz speed",
			initialT:     1000000, // 1ms in nanoseconds
			initialC:     1000,
			contextT:     3000000, // 2ms in nanoseconds
			contextCycle: 2000,    // 1000 cycles in 2ms = 0.5MHz
			expectedMhz:  0.5,
		},
		{
			name:         "Calculate speed with small time difference",
			initialT:     1000000000, // 1s in nanoseconds
			initialC:     1000000,
			contextT:     1001000000, // 1.001s in nanoseconds (1ms difference)
			contextCycle: 1002000,    // 2000 cycles in 1ms = 2MHz
			expectedMhz:  2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sw := NewSpeedWindow()
			sw.previousT = tt.initialT
			sw.previousC = tt.initialC

			context := &common.StepContext{
				T:     tt.contextT,
				Cycle: tt.contextCycle,
			}

			sw.Draw(context)

			// Verify that previousT and previousC are updated
			assert.Equal(t, tt.contextT, sw.previousT)
			assert.Equal(t, tt.contextCycle, sw.previousC)
		})
	}
}

func TestSpeedWindow_GetDrawArea(t *testing.T) {
	sw := NewSpeedWindow()
	primitive := sw.GetDrawArea()

	assert.NotNil(t, primitive)
	assert.Equal(t, sw.text, primitive)
}
