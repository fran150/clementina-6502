package mia

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// contextTestCircuit is a normal-mode MIA with two stepping descriptors over
// known bytes: $10 reads $20000 onward and $11 reads $21000 onward.
func contextTestCircuit(t *testing.T) *emulatedMiaTestCircuit {
	t.Helper()

	circuit := silentAudioCircuit(t)
	chip := circuit.chip
	for i := uint32(0); i < 16; i++ {
		chip.memory[0x20000+i] = uint8(0xA0 + i)
		chip.memory[0x21000+i] = uint8(0xB0 + i)
	}
	stepping := uint8(1<<miaIndexFlagReadStep | 1<<miaIndexFlagWriteStep)
	chip.indexes[0x10] = miaIndex{currentAddr: 0x20000, defaultAddr: 0x20000, limitAddr: 0x20010, step: 1, flags: stepping}
	chip.indexes[0x11] = miaIndex{currentAddr: 0x21000, defaultAddr: 0x21000, limitAddr: 0x21010, step: 1, flags: stepping}

	return circuit
}

// TestEmulatedMiaContextRestoresInterruptedState plays out an interrupt that
// arrives mid-stream: the handler pushes, re-points both windows, moves the
// interrupted code's descriptor and stages its own command parameters, then
// pops. The interrupted code must find everything as it left it.
func TestEmulatedMiaContextRestoresInterruptedState(t *testing.T) {
	circuit := contextTestCircuit(t)
	chip := circuit.chip

	circuit.write(miaRegIdxASelector, 0x10)
	circuit.write(miaRegIdxBSelector, 0x11)
	circuit.write(miaRegCfgSelector, 0x02)
	circuit.write(miaRegCmdParam1, 0x01)
	circuit.write(miaRegCmdParam2, 0x02)
	circuit.write(miaRegCmdParam3, 0x03)
	assert.Equal(t, uint8(0xA0), circuit.read(miaRegIdxAPort))

	// The interrupt handler.
	circuit.write(miaRegContext, miaCtxPush)
	circuit.write(miaRegIdxASelector, 0x11)
	circuit.write(miaRegIdxBSelector, 0x10)
	for i := 0; i < 5; i++ {
		circuit.read(miaRegIdxBPort) // moves descriptor $10 on
	}
	circuit.write(miaRegCfgSelector, 0x05)
	circuit.write(miaRegCmdParam1, 0x09)
	circuit.write(miaRegCmdParam2, 0x09)
	circuit.write(miaRegCmdParam3, 0x09)
	circuit.write(miaRegContext, miaCtxPop)

	assert.Equal(t, uint8(0x10), chip.readRegister(miaRegIdxASelector))
	assert.Equal(t, uint8(0x11), chip.readRegister(miaRegIdxBSelector))
	assert.Equal(t, uint8(0x02), chip.readRegister(miaRegCfgSelector))
	assert.Equal(t, [3]uint8{0x01, 0x02, 0x03}, [3]uint8{
		chip.readRegister(miaRegCmdParam1), chip.readRegister(miaRegCmdParam2), chip.readRegister(miaRegCmdParam3),
	})
	assert.Equal(t, uint32(0x20001), chip.indexes[0x10].currentAddr, "the descriptor the handler moved is back where it was")
	assert.Equal(t, uint8(0xA1), circuit.read(miaRegIdxAPort), "window A resumes with the next byte")
	assert.Equal(t, uint8(0xB0), circuit.read(miaRegIdxBPort), "window B's port was reloaded")
	assert.Equal(t, chip.getCfg(0x02), chip.readRegister(miaRegCfgPort), "the CFG port was reloaded")
	assert.Zero(t, chip.ctxDepth)
}

// TestEmulatedMiaContextNestsAndReportsMisuse verifies pushes nest up to
// miaCtxDepth, a push beyond it and a pop with nothing saved each report an
// error without changing the stack, and values other than PUSH and POP do
// nothing but read back.
func TestEmulatedMiaContextNestsAndReportsMisuse(t *testing.T) {
	circuit := contextTestCircuit(t)
	chip := circuit.chip

	circuit.write(miaRegContext, miaCtxPop)
	assert.Equal(t, miaErrorCtxUnderflow, circuit.read(miaRegErrorLSB))

	for i := 0; i < miaCtxDepth; i++ {
		circuit.write(miaRegIdxASelector, uint8(0x10+i))
		circuit.write(miaRegContext, miaCtxPush)
	}
	assert.Equal(t, miaCtxDepth, chip.ctxDepth)
	circuit.write(miaRegContext, miaCtxPush)
	assert.Equal(t, miaErrorCtxOverflow, circuit.read(miaRegErrorLSB))
	assert.Equal(t, miaCtxDepth, chip.ctxDepth)

	circuit.write(miaRegContext, 0x7F)
	assert.Equal(t, miaCtxDepth, chip.ctxDepth, "other values do nothing")
	assert.Equal(t, uint8(0x7F), circuit.read(miaRegContext), "reads return the last value written")

	for i := miaCtxDepth - 1; i >= 0; i-- {
		circuit.write(miaRegContext, miaCtxPop)
		assert.Equal(t, uint8(0x10+i), chip.readRegister(miaRegIdxASelector), "pops come back in reverse order")
	}
	assert.Zero(t, circuit.read(miaRegErrorLSB), "no further errors")
}
