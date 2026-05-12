//go:build !((linux && arm) || (linux && arm64))

package mia

import (
	"fmt"

	"github.com/fran150/clementina-6502/pkg/components"
)

func NewPicoMia(chipName string) (components.MiaChip, error) {
	return nil, fmt.Errorf("Pico MIA GPIO bridge is only supported on Linux systems (Raspberry Pi)")
}
