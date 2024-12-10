package buses

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Tests basic functions of the standalone line
func TestCreateSetAndToggleLine(t *testing.T) {
	line := CreateStandaloneLine(false)

	assert.Equal(t, false, line.Status())

	line.Set(true)
	assert.Equal(t, true, line.Status())

	line.Set(false)
	assert.Equal(t, false, line.Status())

	line.Toggle()
	assert.Equal(t, true, line.Status())

	line.Toggle()
	assert.Equal(t, false, line.Status())
}
