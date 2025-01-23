package buses

// Represents a Line from a bus. Buses are group of lines that allows
// chips to communicate between each other. In 6502 architectures
// data buses tipically have 8 lines and address bus have 16 lines.
type BusLine[T uint8 | uint16] struct {
	busLineNumber uint8
	bus           Bus[T]
}

// Creates a reference to the specified line of the bus. Any change in the
// bus value will be reflected in the status high or low of the line
func createBusLine[T uint8 | uint16](bus Bus[T], busLineNumber uint8) *BusLine[T] {
	return &BusLine[T]{
		bus:           bus,
		busLineNumber: busLineNumber,
	}
}

// Returns if the line is high (true) or low (false). For example in an 8 bit bus
// any value in where bit 7 is 1 will cause the line of the same number to go high (true)
func (line *BusLine[T]) Status() bool {
	value := 1 << line.busLineNumber
	return line.bus.Read()&T(value) > 0
}

// Sets the status of the line. This will update the value represented in the bus.
// For example, setting the value of line 7 in an 8 bit bus that was in 0x0F will change
// the value to 0x8F.
func (line *BusLine[T]) Set(value bool) {
	busValue := uint16(line.bus.Read())
	mask := uint16(1 << line.busLineNumber)

	if value {
		busValue = (busValue | (0x00 + mask))
	} else {
		busValue = (busValue & (0xFFFF - mask))
	}

	line.bus.Write(T(busValue))
}

// Toggle the status of the line. This will update the value represented in the bus.
// For example, toggling the value of line 7 in an 8 bit bus that had 0xFF as value will change
// the bus value to 0x7F.
func (line *BusLine[T]) Toggle() {
	line.Set(!line.Status())
}
