package buses

type BusConnector[T uint8 | uint16] struct {
	bus *Bus[T]
}

func CreateBusConnector[T uint8 | uint16]() *BusConnector[T] {
	return &BusConnector[T]{}
}

func (connector *BusConnector[T]) Connect(bus *Bus[T]) {
	connector.bus = bus
}

func (connector *BusConnector[T]) Read() T {
	return connector.bus.Read()
}

func (connector *BusConnector[T]) Write(value T) {
	connector.bus.Write(value)
}
