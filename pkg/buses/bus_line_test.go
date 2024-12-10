package buses

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A bus line is a reference to 1 of the lines or traces that forms a bus.
// The line will react to changes in bus values
func TestChangingBusValueUpdatesLineStatus(t *testing.T) {
	bus, connector := createBusConnector()

	// Gets line for the most significant bit
	line := connector.GetLine(7)

	// With a value of bus in all 1s, status of the line should be high
	bus.Write(0xFF)
	assert.Equal(t, true, line.Status())

	// Changing the value of the line to 0x7F, now bit 7 is 0 so line should be low
	bus.Write(0x7F)
	assert.Equal(t, false, line.Status())
}

// A bus line is a reference to 1 of the lines or traces that forms a bus.
// The bus value will be changes when the status of the line is changed.
func TestChangingLineStatusUpdatesBusValue(t *testing.T) {
	bus, connector := createBusConnector()

	// Gets line for the most significant bit
	line := connector.GetLine(7)

	// Set Bus value to all 1s except for bit 7
	bus.Write(0x7F)

	// Setting the line high should bring the bus value to all 1s or 0xFF
	line.Set(true)
	assert.Equal(t, true, line.Status())
	assert.Equal(t, uint8(0xFF), bus.Read())

	// Setting the line low should reset bit 7 of the bus value
	line.Set(false)
	assert.Equal(t, false, line.Status())
	assert.Equal(t, uint8(0x7F), bus.Read())
}

// Toggling the status of a bus line updates the bus value
func TestTogglingLineStatusUpdatesBusValue(t *testing.T) {
	bus, connector := createBusConnector()

	// Gets line for the most significant bit
	line := connector.GetLine(7)

	// Set Bus value to all 1s except for bit 7
	bus.Write(0x7F)

	// Setting the line high should bring the bus value to all 1s or 0xFF
	line.Toggle()
	assert.Equal(t, uint8(0xFF), bus.Read())

	// Setting the line low should reset bit 7 of the bus value
	line.Toggle()
	assert.Equal(t, uint8(0x7F), bus.Read())
}
