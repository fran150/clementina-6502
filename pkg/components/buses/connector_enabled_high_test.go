package buses

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Creates a line and a connector for testing
func newConnectorHighAndLine() (*ConnectorEnabledHigh, Line) {
	connector := NewConnectorEnabledHigh()
	line := NewStandaloneLine(false)

	connector.Connect(line)

	return connector, line
}

// This type of connector is enabled when line is high.
func TestConnectorIsEnabledWhenLineIsHigh(t *testing.T) {
	connector, line := newConnectorHighAndLine()

	line.Set(false)
	assert.Equal(t, false, connector.Enabled())

	line.Set(true)
	assert.Equal(t, true, connector.Enabled())
}

// Enabling this connector sets the line high in this type
func TestLineIsSetHighWhenConnectorIsEnabled(t *testing.T) {
	connector, line := newConnectorHighAndLine()

	connector.SetEnable(true)
	assert.Equal(t, true, line.Status())

	connector.SetEnable(false)
	assert.Equal(t, false, line.Status())
}

// Regardless of line status connector shows as not enabled when disconnected
func TestConnectorIsNotEnabledWhenDisconnected(t *testing.T) {
	connector, line := newConnectorHighAndLine()

	// Disconnect from line
	connector.Connect(nil)

	line.Set(false)
	assert.Equal(t, false, connector.Enabled())

	line.Set(true)
	assert.Equal(t, false, connector.Enabled())
}

// Changing the connector value does not affect the line when disconnected
func TestLineIsNotAffectedWhenConnectorHighDisconnected(t *testing.T) {
	connector, line := newConnectorHighAndLine()

	// Disconnect from line
	connector.Connect(nil)

	connector.SetEnable(true)
	assert.Equal(t, false, line.Status())

	connector.SetEnable(false)
	assert.Equal(t, false, line.Status())
}

// Tests the function that return the line to which the connector is attached
func TestGetLineReturnsTheConnectedLineForConnectorHigh(t *testing.T) {
	connector, line := newConnectorHighAndLine()

	reference := connector.GetLine()

	assert.Equal(t, line, reference)

	connector.Connect(nil)

	assert.Nil(t, connector.GetLine())
}
