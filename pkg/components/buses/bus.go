package buses

// Bus represents an interface for electrical buses that can handle 8-bit or 16-bit values.
// It provides methods to access individual bus lines and read/write values to the bus.
type Bus[T uint16 | uint8] interface {
	// GetBusLine returns a pointer to the specified bus line
	// number: the index of the bus line to retrieve (0-based)
	// returns: pointer to the bus line, or nil if the line number is invalid
	GetBusLine(number uint8) *BusLine[T]

	// Write sets the current value on the bus
	// value: the value to write to the bus
	Write(value T)

	// Read retrieves the current value from the bus
	// returns: the current value on the bus
	Read() T
}

// StandaloneBus implements a physical electrical bus with multiple lines/traces
// that can represent either 8-bit or 16-bit values. It maintains the current
// value and references to individual bus lines.
type StandaloneBus[T uint16 | uint8] struct {
	value    T             // Current value of the bus represented as a number
	busLines []*BusLine[T] // References to all bus lines
}

// New8BitStandaloneBus creates and initializes a new 8-bit bus with 8 individual lines.
// It's typically used for data buses in 8-bit systems.
// returns: a Bus interface implementation for 8-bit operations
func New8BitStandaloneBus() Bus[uint8] {
	bus := StandaloneBus[uint8]{
		value:    0x00,
		busLines: make([]*BusLine[uint8], 8),
	}

	for i := range len(bus.busLines) {
		bus.busLines[i] = newBusLine(&bus, uint8(i))
	}

	return &bus
}

// New16BitStandaloneBus creates and initializes a new 16-bit bus with 16 individual lines.
// It's typically used for address buses in 8-bit systems with 16-bit addressing.
// returns: a Bus interface implementation for 16-bit operations
func New16BitStandaloneBus() Bus[uint16] {
	bus := StandaloneBus[uint16]{
		value:    0x00,
		busLines: make([]*BusLine[uint16], 16),
	}

	for i := range len(bus.busLines) {
		bus.busLines[i] = newBusLine(&bus, uint8(i))
	}

	return &bus
}

// GetBusLine returns a pointer to the specified bus line
// number: the index of the bus line to retrieve (0-based)
// returns: pointer to the bus line, or nil if the line number is out of range
func (bus *StandaloneBus[T]) GetBusLine(number uint8) *BusLine[T] {
	if number < uint8(len(bus.busLines)) {
		return bus.busLines[number]
	} else {
		return nil
	}
}

// Write updates the current value stored in the bus
// value: the new value to store in the bus
func (bus *StandaloneBus[T]) Write(value T) {
	bus.value = value
}

// Read returns the current value stored in the bus
// returns: the current value on the bus
func (bus *StandaloneBus[T]) Read() T {
	return bus.value
}
