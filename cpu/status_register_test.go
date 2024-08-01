package cpu

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAndSetFlag(t *testing.T) {
	status := StatusRegister(0x80)

	assert.Equal(t, true, status.Flag(NegativeFlagBit))

	status.SetFlag(ZeroFlagBit, true)
	assert.Equal(t, uint8(0x82), uint8(status))
	assert.Equal(t, true, status.Flag(ZeroFlagBit))
	assert.Equal(t, true, status.Flag(NegativeFlagBit))

	status.SetFlag(ZeroFlagBit, false)
	assert.Equal(t, uint8(0x80), uint8(status))
	assert.Equal(t, false, status.Flag(ZeroFlagBit))
	assert.Equal(t, true, status.Flag(NegativeFlagBit))
}
