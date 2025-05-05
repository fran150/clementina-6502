package computers

import (
	"testing"
	"time"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/stretchr/testify/assert"
)

func TestNewEmulationLoop(t *testing.T) {
	config := &EmulationLoopConfig{
		SkipCycles: 10,
		DisplayFps: 5,
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
			SkipCycles: 10,
			DisplayFps: 5,
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
		time.Sleep(1 * time.Second)
		context.Stop = true

		// Give it time to stop
		time.Sleep(100 * time.Millisecond)

		assert.True(t, tickCalled, "Tick handler should have been called")
		assert.True(t, drawCalled, "Draw handler should have been called")
	})
}

func TestEmulationLoop_GetConfig(t *testing.T) {
	config := &EmulationLoopConfig{
		SkipCycles: 0,
		DisplayFps: 30,
	}

	loop := NewEmulationLoop(config)

	retrievedConfig := loop.GetConfig()
	assert.Equal(t, config, retrievedConfig)
	assert.Equal(t, int64(0), retrievedConfig.SkipCycles)
	assert.Equal(t, 30, retrievedConfig.DisplayFps)
}
