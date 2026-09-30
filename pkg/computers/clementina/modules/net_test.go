package modules

import (
	"testing"

	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/stretchr/testify/assert"
)

func TestNetConnectsEveryPin(t *testing.T) {
	high := buses.NewConnectorEnabledHigh()
	low := buses.NewConnectorEnabledLow()
	line := buses.NewStandaloneLine(false)

	n := newNet(high, low)
	n.Connect(line)

	assert.Equal(t, line, high.GetLine())
	assert.Equal(t, line, low.GetLine())
	assert.Equal(t, line, n.GetLine())

	line.Set(true)
	assert.True(t, n.Enabled())
	assert.True(t, high.Enabled())
	assert.False(t, low.Enabled())

	n.SetEnable(false)
	assert.False(t, line.Status())
}
