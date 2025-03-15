package computers

import (
	"testing"
	"time"

	"github.com/fran150/clementina6502/pkg/common"
	"github.com/stretchr/testify/assert"
)

func TestNewEmulationLoop(t *testing.T) {
	config := &EmulationLoopConfig{
		TargetSpeedMhz: 1.0,
		DisplayFps:     60,
	}

	loop := NewEmulationLoop(config)

	assert.NotNil(t, loop)
	assert.Equal(t, config, loop.GetConfig())
}

func TestEmulationLoop_Start(t *testing.T) {
	t.Run("returns nil when handlers are missing", func(t *testing.T) {
		loop := NewEmulationLoop(&EmulationLoopConfig{})

		// Test with nil handlers
		context := loop.Start(EmulationLoopHandlers{})
		assert.Nil(t, context)

		// Test with only Tick handler
		context = loop.Start(EmulationLoopHandlers{
			Tick: func(context *common.StepContext) {},
		})
		assert.Nil(t, context)

		// Test with only Draw handler
		context = loop.Start(EmulationLoopHandlers{
			Draw: func(context *common.StepContext) {},
		})
		assert.Nil(t, context)
	})

	t.Run("starts emulation loop with valid handlers", func(t *testing.T) {
		loop := NewEmulationLoop(&EmulationLoopConfig{
			TargetSpeedMhz: 1.0,
			DisplayFps:     60,
		})

		tickCalled := false
		drawCalled := false

		handlers := EmulationLoopHandlers{
			Tick: func(context *common.StepContext) {
				tickCalled = true
			},
			Draw: func(context *common.StepContext) {
				drawCalled = true
			},
		}

		context := loop.Start(handlers)
		assert.NotNil(t, context)

		// Let the loop run briefly
		time.Sleep(100 * time.Millisecond)
		context.Stop = true

		// Give it time to stop
		time.Sleep(10 * time.Millisecond)

		assert.True(t, tickCalled, "Tick handler should have been called")
		assert.True(t, drawCalled, "Draw handler should have been called")
	})
}

func TestEmulationLoop_Timing(t *testing.T) {
	t.Run("respects target speed and FPS", func(t *testing.T) {
		config := &EmulationLoopConfig{
			TargetSpeedMhz: 1.0, // 1MHz
			DisplayFps:     60,  // 60 FPS
		}

		loop := NewEmulationLoop(config)

		tickCount := 0
		drawCount := 0
		var firstTickTime, lastTickTime, firstDrawTime, lastDrawTime int64

		handlers := EmulationLoopHandlers{
			Tick: func(context *common.StepContext) {
				if firstTickTime == 0 {
					firstTickTime = context.T
				}
				lastTickTime = context.T
				tickCount++
			},
			Draw: func(context *common.StepContext) {
				if firstDrawTime == 0 {
					firstDrawTime = context.T
				}
				lastDrawTime = context.T
				drawCount++
			},
		}

		context := loop.Start(handlers)
		assert.NotNil(t, context)

		// Run for a fixed duration
		time.Sleep(100 * time.Millisecond)
		context.Stop = true

		// Calculate actual rates
		tickDuration := lastTickTime - firstTickTime
		drawDuration := lastDrawTime - firstDrawTime

		// Calculate actual frequencies
		actualTicksPerSecond := float64(tickCount) / (float64(tickDuration) / float64(time.Second))
		actualDrawsPerSecond := float64(drawCount) / (float64(drawDuration) / float64(time.Second))

		// Expected values
		expectedTicksPerSecond := config.TargetSpeedMhz * 1_000_000 // Convert MHz to Hz
		expectedDrawsPerSecond := float64(config.DisplayFps)

		// Allow for 20% margin of error due to system scheduling
		marginTick := expectedTicksPerSecond * 0.2
		marginDraw := expectedDrawsPerSecond * 0.2

		assert.InDelta(t, expectedTicksPerSecond, actualTicksPerSecond, marginTick,
			"Tick rate should be close to target speed")
		assert.InDelta(t, expectedDrawsPerSecond, actualDrawsPerSecond, marginDraw,
			"Draw rate should be close to target FPS")
	})
}

func TestEmulationLoop_GetConfig(t *testing.T) {
	config := &EmulationLoopConfig{
		TargetSpeedMhz: 2.0,
		DisplayFps:     30,
	}

	loop := NewEmulationLoop(config)

	retrievedConfig := loop.GetConfig()
	assert.Equal(t, config, retrievedConfig)
	assert.Equal(t, 2.0, retrievedConfig.TargetSpeedMhz)
	assert.Equal(t, 30, retrievedConfig.DisplayFps)
}
