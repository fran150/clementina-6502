package cpu

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddressModeDataGetters(t *testing.T) {
	addressModeDataSet := CreateAddressModesSet()
	absoluteAddressModeData := addressModeDataSet.GetByName(AddressModeAbsolute)

	assert.Equal(t, AddressModeAbsolute, absoluteAddressModeData.Name())
	assert.Equal(t, "a", absoluteAddressModeData.Text())
	assert.Equal(t, "$%04X", absoluteAddressModeData.Format())
	assert.Equal(t, 4, absoluteAddressModeData.Cycles())
	assert.Equal(t, uint8(3), absoluteAddressModeData.MemSize())
}
