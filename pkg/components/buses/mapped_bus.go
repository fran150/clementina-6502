package buses

// MappedBus represents a bus that can map values between buses.
// It can convert between uint8 and uint16 values when reading from or writing to the target bus.
// Can be used for any partially connected buses for example connecting the upper 4 lines of a 8 line bus
// to the lower 4 lines of another 8 line bus
type MappedBus[T uint8 | uint16, S uint8 | uint16] struct {
	mapToSource   func(value T, current []S) []S
	mapFromSource func(value []S) T
	sourceBuses   []Bus[S]      // The underlying bus being mapped to/from
	busLines      []*BusLine[T] // Individual bus lines that can be accessed
}

// Write the specified value ot the bus, this also writes the transformed value
// to the target bus
func (bus *MappedBus[T, S]) Write(value T) {
	current := make([]S, len(bus.sourceBuses))

	for i, source := range bus.sourceBuses {
		current[i] = source.Read()
	}

	sourceValues := bus.mapToSource(value, current)

	for i, source := range bus.sourceBuses {
		source.Write(sourceValues[i])
	}
}

// Read retrieves a value from the target bus and applies the specified transformation
func (bus *MappedBus[T, S]) Read() T {
	values := make([]S, len(bus.sourceBuses))

	for i, source := range bus.sourceBuses {
		values[i] = source.Read()
	}

	return bus.mapFromSource(values)
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
func New8BitMappedBus[S uint8 | uint16](sources []Bus[S], mapToSource func(uint8, []S) []S, mapFromSource func([]S) uint8) *MappedBus[uint8, S] {
	bus := MappedBus[uint8, S]{
		sourceBuses:   sources,
		mapToSource:   mapToSource,
		mapFromSource: mapFromSource,
		busLines:      make([]*BusLine[uint8], 8),
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
func New16BitMappedBus[S uint8 | uint16](sources []Bus[S], mapToSource func(uint16, []S) []S, mapFromSource func([]S) uint16) *MappedBus[uint16, S] {
	bus := MappedBus[uint16, S]{
		sourceBuses:   sources,
		mapToSource:   mapToSource,
		mapFromSource: mapFromSource,
		busLines:      make([]*BusLine[uint16], 16),
	}

	for i := range len(bus.busLines) {
		bus.busLines[i] = newBusLine(&bus, uint8(i))
	}

	return &bus
}
