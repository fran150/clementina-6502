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
}

// NewGPIOEmulationLoop creates a stub that returns an error on non-Linux systems.
func NewGPIOEmulationLoop(config GPIOEmulationLoopConfig) core.EmulationLoop {
	return &stubGPIOLoop{}
}

type stubGPIOLoop struct{}

func (s *stubGPIOLoop) SetPanicHandler(handler func(loopType string, panicData any) bool) {}
func (s *stubGPIOLoop) IsRunning() bool                                                   { return false }
func (s *stubGPIOLoop) IsPaused() bool                                                    { return false }
func (s *stubGPIOLoop) IsStopping() bool                                                  { return false }
func (s *stubGPIOLoop) Stop()                                                             {}
func (s *stubGPIOLoop) Resume()                                                           {}
func (s *stubGPIOLoop) Pause()                                                            {}

func (s *stubGPIOLoop) Start() (*common.StepContext, error) {
	return nil, fmt.Errorf("GPIO emulation is only supported on Linux systems")
}
