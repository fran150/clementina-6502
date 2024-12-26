package buses

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Creates a bus and a bus connector of 8 bits
func createBusConnector() (*Bus[uint8], *BusConnector[uint8]) {
	bus := Create8BitBus()
	connector := CreateBusConnector[uint8]()
	connector.Connect(bus)

	return bus, connector
}

// Tests reading the value of the bus from the connector
func TestReadingFromConnectorReturnsTheBusValue(t *testing.T) {
	bus, connector := createBusConnector()
	bus.Write(0xFF)

	assert.Equal(t, true, connector.isConnected())
	assert.Equal(t, uint8(0xFF), connector.Read())
}

// Reading from a disconnected connector will return 0x00
func TestReadingFromConnectorWhenDisconnectedReturnsZero(t *testing.T) {
	bus, connector := createBusConnector()

	// Disconnect from bus
	connector.Connect(nil)

	bus.Write(0xFF)

	assert.Equal(t, false, connector.isConnected())
	assert.Equal(t, uint8(0x00), connector.Read())
}

// Setting the value on the connector will drive the bus to that value also
func TestWritingToConnectorSetsTheBusValue(t *testing.T) {
	bus, connector := createBusConnector()
	connector.Write(0xFF)

	assert.Equal(t, true, connector.isConnected())
	assert.Equal(t, uint8(0xFF), bus.Read())
}

// Writing to the bus when disconnected does nothing
func TestWritingToConnectorWhenDisconnectedDoesNothing(t *testing.T) {
	bus, connector := createBusConnector()

	// Disconnect from bus
	connector.Connect(nil)

	connector.Write(0xFF)

	assert.Equal(t, false, connector.isConnected())
	assert.Equal(t, uint8(0x00), bus.Read())
}

// GetLine returns a reference to a line of the bus.
// Depending on the type of bus it can be from 0 to 8 or 0 to 15.
// Also the bus line status must change with different values of the bus
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

// Get line of a diconnected bus returns nil
func TestGettingABusLineFromDisconnectedConnectorReturnsNil(t *testing.T) {
	_, connector := createBusConnector()

	// Disconnect from bus
	connector.Connect(nil)

	line := connector.GetLine(0)

	assert.Equal(t, false, connector.isConnected())
	assert.Nil(t, line)
}

// Trying to get an unexistent bus line returns nil
func TestGetInvalidBusLineReturnsNil(t *testing.T) {
	_, connector := createBusConnector()

	line := connector.GetLine(10)

	assert.Equal(t, true, connector.isConnected())
	assert.Nil(t, line)
}
