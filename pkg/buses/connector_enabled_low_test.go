package buses

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func createConnectorLowAndLine() (*ConnectorEnabledLow, Line) {
	connector := CreateConnectorEnabledLow()
	line := CreateStandaloneLine(false)

	connector.Connect(line)

	return connector, line
}

func TestConnectorIsEnabledWhenLineIsLow(t *testing.T) {
	connector, line := createConnectorLowAndLine()

	line.Set(false)
	assert.Equal(t, true, connector.Enabled())

	line.Set(true)
	assert.Equal(t, false, connector.Enabled())
}

func TestLineIsSetLowWhenConnectorIsEnabled(t *testing.T) {
	connector, line := createConnectorLowAndLine()

	connector.SetEnable(true)
	assert.Equal(t, false, line.Status())

	connector.SetEnable(false)
	assert.Equal(t, true, line.Status())
}

func TestConnectorLowIsNotEnabledWhenDisconnected(t *testing.T) {
	connector, line := createConnectorLowAndLine()

	// Disconnect from line
	connector.Connect(nil)

	line.Set(false)
	assert.Equal(t, false, connector.Enabled())

	line.Set(true)
	assert.Equal(t, false, connector.Enabled())
}

func TestLineIsNotAffectedWhenConnectorLowDisconnected(t *testing.T) {
	connector, line := createConnectorLowAndLine()

	// Disconnect from line
	connector.Connect(nil)

	connector.SetEnable(true)
	assert.Equal(t, false, line.Status())

	connector.SetEnable(false)
	assert.Equal(t, false, line.Status())
}

func TestGetLineReturnsTheConnectedLineForConnectorLow(t *testing.T) {
	connector, line := createConnectorLowAndLine()

	reference := connector.GetLine()

	assert.Equal(t, line, reference)

	connector.Connect(nil)

	assert.Nil(t, connector.GetLine())
}
