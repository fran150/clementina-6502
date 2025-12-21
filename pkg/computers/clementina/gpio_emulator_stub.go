//go:build !((linux && arm) || (linux && arm64))

package clementina

import (
	"fmt"

	"github.com/fran150/clementina-6502/pkg/core"
)

// NewClemetinaGPIOEmulator returns an error on non-Linux systems.
func NewClemetinaGPIOEmulator(computer *ClementinaComputer, displayFPS int) (core.BaseEmulator, error) {
	return nil, fmt.Errorf("GPIO emulation is only supported on Linux systems (Raspberry Pi)")
}
