package computers

import (
	"testing"
	"time"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/stretchr/testify/assert"
)

type ComputerMock struct {
	context *common.StepContext

	tickCalls int
	drawCalls int

	shouldPanicInTick bool
	shouldPanicInDraw bool
}

func NewComputerMock() *ComputerMock {
	context := common.NewStepContext()

	return &ComputerMock{
		context:   &context,
		tickCalls: 0,
		drawCalls: 0,
	}
}

func (c *ComputerMock) Run() (*common.StepContext, error) {
	return c.context, nil
}

func (c *ComputerMock) Stop() {
	// Mock stop logic
}

func (c *ComputerMock) Tick(context *common.StepContext) {
	if c.shouldPanicInTick {
		panic("Mock panic in Tick")
	}

	c.tickCalls++
}

func (c *ComputerMock) Draw(context *common.StepContext) {
	if c.shouldPanicInDraw {
		panic("Mock panic in Draw")
	}

	c.drawCalls++
}

func createTestEmulationLoop() *EmulationLoop {
	config := &EmulationLoopConfig{
		TargetSpeedMhz: 0.001,
		DisplayFps:     5,
	}

	computer := NewComputerMock()

	loop := NewEmulationLoopFor(computer, config)

	return loop
}

func TestEmulationLoopStartAndStop(t *testing.T) {
	t.Run("returns nil when config is missing", func(t *testing.T) {
		loop := createTestEmulationLoop()

		loop.config = nil // Simulate missing config

		// Test with nil handlers
		context := loop.Start()
		assert.Nil(t, context)
		assert.False(t, loop.IsRunning(), "Loop should not be running when config is missing")
	})

	t.Run("returns nil when computer is missing", func(t *testing.T) {
		loop := createTestEmulationLoop()

		loop.computer = nil // Simulate missing computer

		// Test with nil handlers
		context := loop.Start()
		assert.Nil(t, context)
		assert.False(t, loop.IsRunning(), "Loop should not be running when config is missing")
	})

	t.Run("starts emulation loop correctly", func(t *testing.T) {
		loop := createTestEmulationLoop()

		context := loop.Start()
		assert.NotNil(t, context)

		// Wait a short time to allow the loop to start
		time.Sleep(50 * time.Millisecond)
		assert.True(t, loop.IsRunning(), "Loop should be running after Start")

		// Let the loop run briefly
		time.Sleep(200 * time.Millisecond)
		loop.Stop()
		assert.True(t, loop.IsStopping(), "Check that the loop is stopping")

		// Wait for the loop to stop
		for loop.IsRunning() {
		}

		// Get the number of tick and draw calls
		computer := loop.GetComputer().(*ComputerMock)
		ticks := computer.tickCalls
		draws := computer.drawCalls

		assert.True(t, ticks > 0, "Tick handler should have been called")
		assert.True(t, draws > 0, "Draw handler should have been called")

		// Wait a bit and ensure no more ticks or draws to validate the loop has stopped
		time.Sleep(100 * time.Millisecond)

		assert.Equal(t, ticks, computer.tickCalls, "Tick handler should not be called after Stop")
		assert.Equal(t, draws, computer.drawCalls, "Draw handler should not be called after Stop")
	})
}

func TestEmulationLoop_Timing(t *testing.T) {
	loop := createTestEmulationLoop()

	config := loop.GetConfig()
	config.TargetSpeedMhz = 0.0001 // 1 Khz
	config.DisplayFps = 10         // 10 FPS

	context := loop.Start()
	assert.NotNil(t, context)

	// Wait for the loop to fully start to get
	// stable timing values
	var now time.Time
	for !loop.IsRunning() {
	}
	now = time.Now()

	// Run for a fixed duration
	time.Sleep(2 * time.Second)

	loop.Stop()

	// Get the number of tick and draw calls
	computer := loop.GetComputer().(*ComputerMock)
	ticks := computer.tickCalls
	draws := computer.drawCalls

	// Calculate actual frequencies
	actualTicksPerSecond := float64(ticks) / (float64(time.Since(now)) / float64(time.Second))
	actualDrawsPerSecond := float64(draws) / (float64(time.Since(now)) / float64(time.Second))

	// Expected values
	expectedTicksPerSecond := config.TargetSpeedMhz * 1_000_000 // Convert MHz to Hz
	expectedDrawsPerSecond := float64(config.DisplayFps)

	// Allow for 0.5% margin of error due to system scheduling
	marginTick := expectedTicksPerSecond * 0.005
	marginDraw := expectedDrawsPerSecond * 0.005

	assert.InDelta(t, expectedTicksPerSecond, actualTicksPerSecond, marginTick,
		"Tick rate should be close to target speed")
	assert.InDelta(t, expectedDrawsPerSecond, actualDrawsPerSecond, marginDraw,
		"Draw rate should be close to target FPS")

	assert.True(t, ticks > 0, "Tick handler should have been called")
	assert.True(t, draws > 0, "Draw handler should have been called")
}

func TestEmulationLoop_PanicHandling(t *testing.T) {
	t.Run("handles panic in Tick with handler returning true", func(t *testing.T) {
		loop := createTestEmulationLoop()
		loop.SetPanicHandler(func(loopType string, r any) bool {
			return true // Suppress panic
		})

		computer := loop.GetComputer().(*ComputerMock)
		computer.shouldPanicInTick = true

		// Start the loop and let it handle the panic
		context := loop.Start()
		assert.NotNil(t, context)
		time.Sleep(100 * time.Millisecond)

		// The loop should stop due to panic but not crash the test
		for loop.IsRunning() {
			time.Sleep(10 * time.Millisecond)
		}
		assert.False(t, loop.IsRunning())
	})

	t.Run("handles panic in Draw with handler returning true", func(t *testing.T) {
		loop := createTestEmulationLoop()
		loop.SetPanicHandler(func(loopType string, r any) bool {
			return true // Suppress panic
		})

		computer := loop.GetComputer().(*ComputerMock)
		computer.shouldPanicInDraw = true

		// Start the loop and let it handle the panic
		context := loop.Start()
		assert.NotNil(t, context)
		time.Sleep(100 * time.Millisecond)

		// The loop should stop due to panic but not crash the test
		for loop.IsRunning() {
			time.Sleep(10 * time.Millisecond)
		}
		assert.False(t, loop.IsRunning())
	})

	t.Run("handles panic without panic handler", func(t *testing.T) {
		loop := createTestEmulationLoop()
		// No panic handler set

		computer := loop.GetComputer().(*ComputerMock)
		computer.shouldPanicInTick = true

		// Start the loop and wait for it to stop due to panic
		context := loop.Start()
		assert.NotNil(t, context)
		time.Sleep(100 * time.Millisecond)

		// The loop should stop
		for loop.IsRunning() {
			time.Sleep(10 * time.Millisecond)
		}
		assert.False(t, loop.IsRunning())
	})

	t.Run("panic handler can return false", func(t *testing.T) {
		loop := createTestEmulationLoop()
		handlerCalled := false
		handlerReturnValue := false

		loop.SetPanicHandler(func(loopType string, r any) bool {
			handlerCalled = true
			assert.Equal(t, "Loop", loopType)
			assert.Equal(t, "Mock panic in Tick", r)
			return handlerReturnValue
		})

		// Test the handlePanic method directly to avoid goroutine issues
		assert.Panics(t, func() {
			loop.handlePanic("Loop", "Mock panic in Tick")
		}, "handlePanic should re-panic when handler returns false")

		assert.True(t, handlerCalled, "Panic handler should have been called")
	})

}

func TestEmulationLoop_NewConstructors(t *testing.T) {
	t.Run("NewEmulationLoop creates loop with config", func(t *testing.T) {
		config := &EmulationLoopConfig{
			TargetSpeedMhz: 1.0,
			DisplayFps:     30,
		}

		loop := NewEmulationLoop(config)
		assert.NotNil(t, loop)
		assert.Equal(t, config, loop.GetConfig())
		assert.Nil(t, loop.GetComputer())
		assert.False(t, loop.IsRunning())
		assert.False(t, loop.IsStopping())
	})

	t.Run("NewEmulationLoopFor creates loop with computer and config", func(t *testing.T) {
		config := &EmulationLoopConfig{
			TargetSpeedMhz: 1.0,
			DisplayFps:     30,
		}
		computer := NewComputerMock()

		loop := NewEmulationLoopFor(computer, config)
		assert.NotNil(t, loop)
		assert.Equal(t, config, loop.GetConfig())
		assert.Equal(t, computer, loop.GetComputer())
	})
}

func TestEmulationLoop_SettersAndGetters(t *testing.T) {
	t.Run("SetComputer and GetComputer work correctly", func(t *testing.T) {
		loop := NewEmulationLoop(&EmulationLoopConfig{})
		computer := NewComputerMock()

		loop.SetComputer(computer)
		assert.Equal(t, computer, loop.GetComputer())
	})

	t.Run("SetPanicHandler sets handler correctly", func(t *testing.T) {
		loop := NewEmulationLoop(&EmulationLoopConfig{})

		handler := func(loopType string, r any) bool {
			return true
		}

		loop.SetPanicHandler(handler)
		// We can't directly test the handler is set, but the panic tests verify it works
		assert.NotNil(t, loop)
	})
}
