package buses

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func createConnectorHighAndLine() (*ConnectorEnabledHigh, Line) {
	connector := CreateConnectorEnabledHigh()
	line := CreateStandaloneLine(false)

	connector.Connect(line)

	return connector, line
}

func TestConnectorIsEnabledWhenLineIsHigh(t *testing.T) {
	connector, line := createConnectorHighAndLine()

	line.Set(false)
	assert.Equal(t, false, connector.Enabled())

	line.Set(true)
	assert.Equal(t, true, connector.Enabled())
}

func TestLineIsSetHighWhenConnectorIsEnabled(t *testing.T) {
	connector, line := createConnectorHighAndLine()

	connector.SetEnable(true)
	assert.Equal(t, true, line.Status())

	connector.SetEnable(false)
	assert.Equal(t, false, line.Status())
}

func TestConnectorIsNotEnabledWhenDisconnected(t *testing.T) {
	connector, line := createConnectorHighAndLine()

	// Disconnect from line
	connector.Connect(nil)

	line.Set(false)
	assert.Equal(t, false, connector.Enabled())

	line.Set(true)
	assert.Equal(t, false, connector.Enabled())
}

func TestLineIsNotAffectedWhenConnectorHighDisconnected(t *testing.T) {
	connector, line := createConnectorHighAndLine()

	// Disconnect from line
	connector.Connect(nil)

	connector.SetEnable(true)
	assert.Equal(t, false, line.Status())

	connector.SetEnable(false)
	assert.Equal(t, false, line.Status())
}

func TestGetLineReturnsTheConnectedLineForConnectorHigh(t *testing.T) {
	connector, line := createConnectorHighAndLine()

	reference := connector.GetLine()

	assert.Equal(t, line, reference)

	connector.Connect(nil)

	assert.Nil(t, connector.GetLine())
}
