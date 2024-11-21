package buses

import "math"

type BusLine[T uint8 | uint16] struct {
	busLineNumber uint8
	bus           *Bus[T]
}

func createBusLine[T uint8 | uint16](bus *Bus[T], busLineNumber uint8) *BusLine[T] {
	return &BusLine[T]{
		bus:           bus,
		busLineNumber: busLineNumber,
	}
}

func (line *BusLine[T]) ConnectBus(bus *Bus[T]) {
	line.bus = bus
}

func (line *BusLine[T]) Status() bool {
	return line.bus.Read()&T(math.Pow(2, float64(line.busLineNumber))) > 0
}

func (line *BusLine[T]) Set(value bool) {
	busValue := uint16(line.bus.Read())
	mask := uint16(math.Pow(2, float64(line.busLineNumber)))

	if value {
		busValue = (busValue | (0x00 + mask))
	} else {
		busValue = (busValue & (0xFFFF - mask))
	}

	line.bus.Write(T(busValue))
}

func (line *BusLine[T]) Toggle() {
	line.Set(!line.Status())
}
