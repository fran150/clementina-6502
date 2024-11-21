package buses

// Electrical buses. Buses typically have 8 or 16 lines. So they can be
// used to represent uint8 or uint16 addresses.
type Bus[T uint16 | uint8] struct {
	value    T
	busLines [16]*BusLine[T]
}

func CreateBus[T uint16 | uint8]() *Bus[T] {
	bus := Bus[T]{
		value:    0x00,
		busLines: [16]*BusLine[T]{},
	}

	for i := range uint8(16) {
		bus.busLines[i] = createBusLine(&bus, i)
	}

	return &bus
}

// Returns a line for each bus line
func (bus *Bus[T]) GetBusLine(number uint8) *BusLine[T] {
	return bus.busLines[number]
}

// Writes a number to the bus.
func (bus *Bus[T]) Write(value T) {
	bus.value = value
}

// Reads a value from the bus.
func (bus *Bus[T]) Read() T {
	return bus.value
}
