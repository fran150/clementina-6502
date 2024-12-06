package buses

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func createBusConnector() (*Bus[uint8], *BusConnector[uint8]) {
	bus := Create8BitBus()
	connector := CreateBusConnector[uint8]()
	connector.Connect(bus)

	return bus, connector
}

func TestReadingFromConnectorReturnsTheBusValue(t *testing.T) {
	bus, connector := createBusConnector()
	bus.Write(0xFF)

	assert.Equal(t, true, connector.isConnected())
	assert.Equal(t, uint8(0xFF), connector.Read())
}

func TestReadingFromConnectorWhenDisconnectedReturnsZero(t *testing.T) {
	bus, connector := createBusConnector()

	// Disconnect from bus
	connector.Connect(nil)

	bus.Write(0xFF)

	assert.Equal(t, false, connector.isConnected())
	assert.Equal(t, uint8(0x00), connector.Read())
}

func TestWritingToConnectorSetsTheBusValue(t *testing.T) {
	bus, connector := createBusConnector()
	connector.Write(0xFF)

	assert.Equal(t, true, connector.isConnected())
	assert.Equal(t, uint8(0xFF), bus.Read())
}

func TestWritingToConnectorWhenDisconnectedDoesNothing(t *testing.T) {
	bus, connector := createBusConnector()

	// Disconnect from bus
	connector.Connect(nil)

	connector.Write(0xFF)

	assert.Equal(t, false, connector.isConnected())
	assert.Equal(t, uint8(0x00), bus.Read())
}

func TestGetLineReturnsReferenceToLine(t *testing.T) {
	bus, connector := createBusConnector()

	var lines [8]Line

	for i := range uint8(8) {
		lines[i] = connector.GetLine(i)
	}

	// Writing 0xAA to the bus makes the lines toggle
	// starting with 0 from bit 0
	bus.Write(0xAA)
	expected := false
	for i := range uint(8) {
		assert.Equal(t, expected, lines[i].Status())
		expected = !expected
	}

	// Writing 0x55 to the bus updates the lines to toggle
	// starting with 1 from bit 0
	bus.Write(0x55)
	expected = true
	for i := range uint(8) {
		assert.Equal(t, expected, lines[i].Status())
		expected = !expected
	}
}

func TestGettingABusLineFromDisconnectedConnectorReturnsNil(t *testing.T) {
	_, connector := createBusConnector()

	// Disconnect from bus
	connector.Connect(nil)

	line := connector.GetLine(0)

	assert.Equal(t, false, connector.isConnected())
	assert.Nil(t, line)
}

func TestGetInvalidBusLineReturnsNil(t *testing.T) {
	_, connector := createBusConnector()

	line := connector.GetLine(10)

	assert.Equal(t, true, connector.isConnected())
	assert.Nil(t, line)
}
