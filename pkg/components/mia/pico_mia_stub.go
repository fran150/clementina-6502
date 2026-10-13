//go:build !((linux && arm) || (linux && arm64))

package mia

import (
	"fmt"

	"github.com/fran150/clementina-6502/pkg/components"
)

// NewPicoMia returns an error: the real-MIA GPIO bridge only builds for Linux on ARM.
func NewPicoMia(chipName string) (components.MiaChip, error) {
	return nil, fmt.Errorf("the Pico MIA GPIO bridge is only supported on Linux systems (Raspberry Pi)")
}
