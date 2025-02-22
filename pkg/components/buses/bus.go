package buses

type Bus[T uint16 | uint8] interface {
	GetBusLine(number uint8) *BusLine[T]
	Write(value T)
	Read() T
}

// Electrical buses. Buses typically have 8 or 16 lines or traces. They can be
// used to represent uint8 or uint16 addresses.
type StandaloneBus[T uint16 | uint8] struct {
	value    T             // Current value of the bus represented as a number
	busLines []*BusLine[T] // References to all bus lines
}

// Creates a 8 bit bus
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

// Creates a 16 bits bus
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

// Returns a line for each bus line
func (bus *StandaloneBus[T]) GetBusLine(number uint8) *BusLine[T] {
	if number < uint8(len(bus.busLines)) {
		return bus.busLines[number]
	} else {
		return nil
	}
}

// Writes a number to the bus.
func (bus *StandaloneBus[T]) Write(value T) {
	bus.value = value
}

// Reads a value from the bus.
func (bus *StandaloneBus[T]) Read() T {
	return bus.value
}
