package buses

// MappedBus represents a bus that can map values between buses.
// It can convert between uint8 and uint16 values when reading from or writing to the target bus.
// Can be used for any partially connected buses for example connecting the upper 4 lines of a 8 line bus
// to the lower 4 lines of another 8 line bus
type MappedBus[T uint8 | uint16, S uint8 | uint16] struct {
	mapTo     func(value T) S // Function to map values when writing to target bus
	mapFrom   func(value S) T // Function to map values when reading from target bus
	targetBus Bus[S]          // The underlying bus being mapped to/from
	busLines  []*BusLine[T]   // Individual bus lines that can be accessed
}

// Write the specified value ot the bus, this also writes the transformed value
// to the target bus
func (bus *MappedBus[T, S]) Write(value T) {
	bus.targetBus.Write(bus.mapTo(value))
}

// Read retrieves a value from the target bus and applies the specified transformation
func (bus *MappedBus[T, S]) Read() T {
	return bus.mapFrom(bus.targetBus.Read())
}

// GetBusLine returns a pointer to the specified bus line.
// Returns nil if the bus line number is out of range.
func (bus *MappedBus[T, S]) GetBusLine(number uint8) *BusLine[T] {
	if number < uint8(len(bus.busLines)) {
		return bus.busLines[number]
	} else {
		return nil
	}
}

// Creates a new 8-bit mapped bus that converts between uint8 and type S.
// Parameters:
//   - target: The target bus to map to/from
//   - mapTo: Function to convert uint8 values to type S when writing
//   - mapFrom: Function to convert type S values to uint8 when reading
func New8BitMappedBus[S uint8 | uint16](target Bus[S], mapTo func(value uint8) S, mapFrom func(value S) uint8) *MappedBus[uint8, S] {
	bus := MappedBus[uint8, S]{
		targetBus: target,
		mapTo:     mapTo,
		mapFrom:   mapFrom,
		busLines:  make([]*BusLine[uint8], 8),
	}

	for i := range len(bus.busLines) {
		bus.busLines[i] = newBusLine(&bus, uint8(i))
	}

	return &bus
}

// Creates a new 16-bit mapped bus that converts between uint16 and type S.
// Parameters:
//   - target: The target bus to map to/from
//   - mapTo: Function to convert uint16 values to type S when writing
//   - mapFrom: Function to convert type S values to uint16 when reading
func New16BitMappedBus[S uint8 | uint16](target Bus[S], mapTo func(value uint16) S, mapFrom func(value S) uint16) *MappedBus[uint16, S] {
	bus := MappedBus[uint16, S]{
		targetBus: target,
		mapTo:     mapTo,
		mapFrom:   mapFrom,
		busLines:  make([]*BusLine[uint16], 16),
	}

	for i := range len(bus.busLines) {
		bus.busLines[i] = newBusLine(&bus, uint8(i))
	}

	return &bus
}
