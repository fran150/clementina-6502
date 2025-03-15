package terminal

import (
	"testing"
	"time"

	"github.com/fran150/clementina6502/pkg/common"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

// MockComputer implements the Computer interface for testing
type MockComputer struct {
	initCalled bool
	tickCalled bool
	drawCalled bool
	keyPressed bool
	shouldStop bool
	config     *ApplicationConfig
	app        *tview.Application
}

func NewMockComputer() *MockComputer {
	return &MockComputer{}
}

func (m *MockComputer) Init(app *tview.Application, config *ApplicationConfig) {
	m.initCalled = true
	m.app = app
	m.config = config
}

func (m *MockComputer) Tick(context *common.StepContext) {
	m.tickCalled = true
	if m.shouldStop {
		context.Stop = true
	}
}

func (m *MockComputer) Draw(context *common.StepContext) {
	m.drawCalled = true
}

func (m *MockComputer) KeyPressed(event *tcell.EventKey, context *common.StepContext) *tcell.EventKey {
	m.keyPressed = true
	if m.shouldStop {
		context.Stop = true
	}
	return event
}

func TestApplicationRun(t *testing.T) {
	mockComputer := NewMockComputer()
	app := NewApplication(mockComputer)

	// Create a simulation screen
	screen := tcell.NewSimulationScreen("")
	err := screen.Init()
	if err != nil {
		t.Fatal(err)
	}

	// Don't defer screen.Fini() here since tview will handle it

	// Set the screen for the tview application
	app.tvApp.SetScreen(screen)

	// Create channels for synchronization
	done := make(chan struct{})
	started := make(chan struct{})

	// Run the application in a goroutine
	go func() {
		close(started)
		context := app.Run()
		assert.NotNil(t, context)
		close(done)
	}()

	// Wait for the application to start
	<-started
	time.Sleep(100 * time.Millisecond)

	// Queue updates in the main thread
	go func() {
		// Verify that Init was called
		assert.True(t, mockComputer.initCalled)

		// Simulate a key press
		screen.InjectKey(tcell.KeyRune, 'a', tcell.ModNone)

		// Wait a bit for the key press to be processed
		time.Sleep(100 * time.Millisecond)
		assert.True(t, mockComputer.keyPressed)

		// Stop the application
		mockComputer.shouldStop = true
		app.tvApp.Stop()
	}()

	// Wait for the application to stop with timeout
	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out")
	}

	// Verify that all expected methods were called
	assert.True(t, mockComputer.tickCalled)
	assert.True(t, mockComputer.drawCalled)
}
