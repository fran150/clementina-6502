package buses

// Provides a connection point from a chip to a bus.
// It can have 8 lines, which allows for values from 0 - 255 (represented by uint8) or
// it can have 16 lines which allows for values from 0 - 65535 (represented by unit16)
// This can be used in chip emulations as the interface between the chip and the bus.
// Typically in 6502 architecture address buses uses 16 lines and data buses uses 8 lines.
type BusConnector[T uint8 | uint16] struct {
	bus Bus[T]
}

// Creates and returns a connector of the specified type
func CreateBusConnector[T uint8 | uint16]() *BusConnector[T] {
	return &BusConnector[T]{}
}

// Connects to the specified bus
func (connector *BusConnector[T]) Connect(bus Bus[T]) {
	connector.bus = bus
}

// Reads the value currently present in the bus. If the bus is disconnected
// then it returns 0
func (connector *BusConnector[T]) Read() T {
	if connector.bus != nil {
		return connector.bus.Read()
	}

	return 0x00
}

// Writes the specified value to the connected bus (if any)
func (connector *BusConnector[T]) Write(value T) {
	if connector.bus != nil {
		connector.bus.Write(value)
	}
}

// Returns the specified line number of the bus. For example, getting line 7 will return
// a line that will be high when bus has a value that has a 1 in that bit like 0xF0
func (connector *BusConnector[T]) GetLine(lineNumber uint8) Line {
	if connector.bus != nil {
		return connector.bus.GetBusLine(lineNumber)
	}

	return nil
}

// Returns if the connector is actually connected to a bus
func (connector *BusConnector[T]) IsConnected() bool {
	return connector.bus != nil
}
