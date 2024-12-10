package buses

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Creates a line and a connector for testing
func createConnectorLowAndLine() (*ConnectorEnabledLow, Line) {
	connector := CreateConnectorEnabledLow()
	line := CreateStandaloneLine(false)

	connector.Connect(line)

	return connector, line
}

// This type of connector is enabled when line is low.
func TestConnectorIsEnabledWhenLineIsLow(t *testing.T) {
	connector, line := createConnectorLowAndLine()

	line.Set(false)
	assert.Equal(t, true, connector.Enabled())

	line.Set(true)
	assert.Equal(t, false, connector.Enabled())
}

// Enabling this connector sets the line low in this type
func TestLineIsSetLowWhenConnectorIsEnabled(t *testing.T) {
	connector, line := createConnectorLowAndLine()

	connector.SetEnable(true)
	assert.Equal(t, false, line.Status())

	connector.SetEnable(false)
	assert.Equal(t, true, line.Status())
}

// Regardless of line status connector shows as not enabled when disconnected
func TestConnectorLowIsNotEnabledWhenDisconnected(t *testing.T) {
	connector, line := createConnectorLowAndLine()

	// Disconnect from line
	connector.Connect(nil)

	line.Set(false)
	assert.Equal(t, false, connector.Enabled())

	line.Set(true)
	assert.Equal(t, false, connector.Enabled())
}

// Changing the connector value does not affect the line when disconnected
func TestLineIsNotAffectedWhenConnectorLowDisconnected(t *testing.T) {
	connector, line := createConnectorLowAndLine()

	// Disconnect from line
	connector.Connect(nil)

	connector.SetEnable(true)
	assert.Equal(t, false, line.Status())

	connector.SetEnable(false)
	assert.Equal(t, false, line.Status())
}

// Tests the function that return the line to which the connector is attached
func TestGetLineReturnsTheConnectedLineForConnectorLow(t *testing.T) {
	connector, line := createConnectorLowAndLine()

	reference := connector.GetLine()

	assert.Equal(t, line, reference)

	connector.Connect(nil)

	assert.Nil(t, connector.GetLine())
}
