//go:build !((linux && arm) || (linux && arm64))

package emulation

import (
	"fmt"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/core"
)

// GPIOEmulationLoopConfig contains settings for GPIO-controlled emulation.
type GPIOEmulationLoopConfig struct {
	DisplayFPS int
	Emulator   LoopTarget
	ChipName   string
}

// NewGPIOEmulationLoop creates a stub that returns an error on non-Linux systems.
func NewGPIOEmulationLoop(config GPIOEmulationLoopConfig) core.EmulationLoop {
	return &stubGPIOLoop{}
}

type stubGPIOLoop struct{}

// SetPanicHandler does nothing; the stub never runs.
func (s *stubGPIOLoop) SetPanicHandler(handler func(loopType string, panicData any) bool) {}

// IsRunning always reports false.
func (s *stubGPIOLoop) IsRunning() bool { return false }

// IsPaused always reports false.
func (s *stubGPIOLoop) IsPaused() bool { return false }

// IsStopping always reports false.
func (s *stubGPIOLoop) IsStopping() bool { return false }

// Stop does nothing.
func (s *stubGPIOLoop) Stop() {}

// Resume does nothing.
func (s *stubGPIOLoop) Resume() {}

// Pause does nothing.
func (s *stubGPIOLoop) Pause() {}

// Start returns an error: GPIO emulation only builds for Linux on ARM.
func (s *stubGPIOLoop) Start() (*common.StepContext, error) {
	return nil, fmt.Errorf("GPIO emulation is only supported on Linux systems")
}
