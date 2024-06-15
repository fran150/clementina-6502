package buses

import "math"

// Electrical buses. Buses typically have 8 or 16 lines. So they can be
// used to represent uint8 or uint16 addresses.
type Bus[T uint16 | uint8] struct {
	value T
}

// Returns true if the status of a particular bus line is high.
// Line 0 is the least significative
func (bus *Bus[T]) LineStatus(number T) bool {
	return bus.value&T(math.Pow(2, float64(number))) > 0
}

// Writes a number to the bus.
func (bus *Bus[T]) Write(value T) {
	bus.value = value
}

// Reads a value from the bus.
func (bus *Bus[T]) Read() T {
	return bus.value
}
