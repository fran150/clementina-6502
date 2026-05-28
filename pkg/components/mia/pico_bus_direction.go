package mia

// Kept outside pico_mia.go so this pure logic can be tested without Linux GPIO support.
type picoDataBusDirection uint8

const (
	// MIA is not selected, keep Pi data pins released.
	picoDataBusHighZ picoDataBusDirection = iota
	// CPU reads from MIA, Pico drives data and Pi samples it.
	picoDataBusInput
	// CPU writes to MIA, Pi drives data for Pico to sample.
	picoDataBusOutput
)

// picoDataBusDirectionForCycle returns the Pi data bus mode for the current MIA cycle.
func picoDataBusDirectionForCycle(miaSelected bool, writeEnabled bool) picoDataBusDirection {
	if !miaSelected {
		return picoDataBusHighZ
	}

	if miaSelected && writeEnabled {
		return picoDataBusOutput
	}

	return picoDataBusInput
}
