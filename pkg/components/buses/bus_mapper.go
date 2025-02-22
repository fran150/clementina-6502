package buses

type MappedBus[T uint8 | uint16, S uint8 | uint16] struct {
	mapTo     func(value T) S
	mapFrom   func(value S) T
	targetBus Bus[S]
	busLines  []*BusLine[T]
}

func (bus *MappedBus[T, S]) Write(value T) {
	bus.targetBus.Write(bus.mapTo(value))
}

func (bus *MappedBus[T, S]) Read() T {
	return bus.mapFrom(bus.targetBus.Read())
}

func (bus *MappedBus[T, S]) GetBusLine(number uint8) *BusLine[T] {
	if number < uint8(len(bus.busLines)) {
		return bus.busLines[number]
	} else {
		return nil
	}
}

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
