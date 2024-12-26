package buses

// Electrical buses. Buses typically have 8 or 16 lines or traces. They can be
// used to represent uint8 or uint16 addresses.
type Bus[T uint16 | uint8] struct {
	value    T             // Current value of the bus represented as a number
	busLines []*BusLine[T] // References to all bus lines
}

// Creates a 8 bit bus
func Create8BitBus() *Bus[uint8] {
	bus := Bus[uint8]{
		value:    0x00,
		busLines: make([]*BusLine[uint8], 8),
	}

	for i := range len(bus.busLines) {
		bus.busLines[i] = createBusLine(&bus, uint8(i))
	}

	return &bus
}

// Creates a 16 bits bus
func Create16BitBus() *Bus[uint16] {
	bus := Bus[uint16]{
		value:    0x00,
		busLines: make([]*BusLine[uint16], 16),
	}

	for i := range len(bus.busLines) {
		bus.busLines[i] = createBusLine(&bus, uint8(i))
	}

	return &bus
}

// Returns a line for each bus line
func (bus *Bus[T]) GetBusLine(number uint8) *BusLine[T] {
	if number < uint8(len(bus.busLines)) {
		return bus.busLines[number]
	} else {
		return nil
	}
}

// Writes a number to the bus.
func (bus *Bus[T]) Write(value T) {
	bus.value = value
}

// Reads a value from the bus.
func (bus *Bus[T]) Read() T {
	return bus.value
}
