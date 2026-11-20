package buses

// BusLine represents a single line from a bus. Buses are groups of lines that allow
// chips to communicate between each other. In 6502 architectures,
// data buses typically have 8 lines and address buses have 16 lines.
type BusLine[T uint8 | uint16] struct {
	busLineNumber uint8
	mask          T
	bus           Bus[T]
	// standalone is bus when it is a *StandaloneBus, so the hot path reads
	// and writes its value directly instead of through the Bus interface.
	standalone *StandaloneBus[T]
}

// Creates a reference to the specified line of the bus. Any change in the
// bus value will be reflected in the status high or low of the line
func newBusLine[T uint8 | uint16](bus Bus[T], busLineNumber uint8) *BusLine[T] {
	standalone, _ := bus.(*StandaloneBus[T])
	return &BusLine[T]{
		bus:           bus,
		busLineNumber: busLineNumber,
		mask:          T(1) << busLineNumber,
		standalone:    standalone,
	}
}

// Returns if the line is high (true) or low (false). For example in an 8 bit bus
// any value in where bit 7 is 1 will cause the line of the same number to go high (true)
func (line *BusLine[T]) Status() bool {
	if line.standalone != nil {
		return line.standalone.value&line.mask != 0
	}
	return line.bus.Read()&line.mask != 0
}

// Sets the status of the line. This will update the value represented in the bus.
// For example, setting the value of line 7 in an 8 bit bus that was in 0x0F will change
// the value to 0x8F.
func (line *BusLine[T]) Set(value bool) {
	if sb := line.standalone; sb != nil {
		if value {
			sb.value |= line.mask
		} else {
			sb.value &^= line.mask
		}
		return
	}
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
