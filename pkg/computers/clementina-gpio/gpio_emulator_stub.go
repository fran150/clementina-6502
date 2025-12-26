//go:build !((linux && arm) || (linux && arm64))

package clementinagpio

import (
	"fmt"

	"github.com/fran150/clementina-6502/pkg/core"
)

type ClementinaGPIOComputer struct {
}

func NewClementinaGPIOComputer() (*ClementinaGPIOComputer, error) {
	return &ClementinaGPIOComputer{}, nil
}

// NewClemetinaGPIOEmulator returns an error on non-Linux systems.
func NewClemetinaGPIOEmulator(computer *ClementinaGPIOComputer, displayFPS int) (core.BaseEmulator, error) {
	return nil, fmt.Errorf("GPIO emulation is only supported on Linux systems (Raspberry Pi)")
}
