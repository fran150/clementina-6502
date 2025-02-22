package cpu

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAndSetFlag(t *testing.T) {
	// Creates new status register but 0x80 is transformed in
	// 0xB0 as unused (U) and break (B) flags are always set
	status := NewStatusRegister(0x80)

	assert.Equal(t, true, status.Flag(NegativeFlagBit))

	status.SetFlag(ZeroFlagBit, true)
	assert.Equal(t, uint8(0xB2), uint8(status))
	assert.Equal(t, true, status.Flag(ZeroFlagBit))
	assert.Equal(t, true, status.Flag(NegativeFlagBit))

	status.SetFlag(ZeroFlagBit, false)
	assert.Equal(t, uint8(0xB0), uint8(status))
	assert.Equal(t, false, status.Flag(ZeroFlagBit))
	assert.Equal(t, true, status.Flag(NegativeFlagBit))
}
