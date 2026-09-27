package mia

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	rectMapBase   = 0x20000 // a 64-byte-wide map, byte (row, col) = row*64 + col
	rectMapWidth  = 64
	rectTableBase = 0x30000
)

// copyRectCircuit is a normal-mode MIA whose RAM holds a 64-wide row-major map
// with rows 0-3 filled.
func copyRectCircuit(t *testing.T) *emulatedMiaTestCircuit {
	t.Helper()

	circuit := newEmulatedMiaTestCircuit()
	circuit.chip.state = miaStateNormal
	for i := uint32(0); i < 4*rectMapWidth; i++ {
		circuit.chip.memory[rectMapBase+i] = uint8(i)
	}
	return circuit
}

// copyRect points index $20 at width bytes of the source with the given step
// and index $21 at the destination, then triggers COPY_RECT for rows rows.
func copyRect(circuit *emulatedMiaTestCircuit, src, width, srcStep, dst, dstStep uint32, rows uint8) {
	chip := circuit.chip
	chip.indexes[0x20] = miaIndex{currentAddr: src, limitAddr: src + width, step: uint16(srcStep)}
	chip.indexes[0x21] = miaIndex{currentAddr: dst, step: uint16(dstStep)}
	circuit.write(miaRegCmdParam1, 0x20)
	circuit.write(miaRegCmdParam2, 0x21)
	circuit.write(miaRegCmdParam3, rows)
	circuit.write(miaRegCmdTrigger, 0x11)
}

// TestEmulatedMiaCopyRectCopiesABlockOfAMap copies a 5x3 block at column 10,
// row 1 of the map into a packed buffer, in one command, without moving
// either index.
func TestEmulatedMiaCopyRectCopiesABlockOfAMap(t *testing.T) {
	circuit := copyRectCircuit(t)
	chip := circuit.chip

	copyRect(circuit, rectMapBase+1*rectMapWidth+10, 5, rectMapWidth, rectTableBase, 5, 3)

	want := []uint8{74, 75, 76, 77, 78, 138, 139, 140, 141, 142, 202, 203, 204, 205, 206}
	assert.Equal(t, want, chip.memory[rectTableBase:rectTableBase+15])
	assert.Zero(t, chip.memory[rectTableBase+15], "nothing past the rectangle")
	assert.Equal(t, uint32(rectMapBase+1*rectMapWidth+10), chip.indexes[0x20].currentAddr, "indexes do not move")
	assert.Equal(t, miaIRQCommand, chip.irqStatus()&miaIRQCommand)
	assert.Zero(t, chip.status()&miaStatusDMARunning)
}

// TestEmulatedMiaCopyRectCopiesAColumnIntoATable copies one map column into a
// 40-wide table column - what a scroll does as a new column comes into view.
func TestEmulatedMiaCopyRectCopiesAColumnIntoATable(t *testing.T) {
	circuit := copyRectCircuit(t)
	chip := circuit.chip

	copyRect(circuit, rectMapBase+7, 1, rectMapWidth, rectTableBase+39, 40, 4)

	for row := uint32(0); row < 4; row++ {
		assert.Equal(t, uint8(row*rectMapWidth+7), chip.memory[rectTableBase+row*40+39], "row %d", row)
	}
	assert.Zero(t, chip.memory[rectTableBase+38], "neighbouring cells untouched")
}

// TestEmulatedMiaCopyRectRepeatsARowWithSourceStepZero fills a rectangle by
// repeating one row, the way BASIC's MFILL builds rectangle fills.
func TestEmulatedMiaCopyRectRepeatsARowWithSourceStepZero(t *testing.T) {
	circuit := copyRectCircuit(t)
	chip := circuit.chip

	copyRect(circuit, rectMapBase, 3, 0, rectTableBase, 8, 3)

	for row := uint32(0); row < 3; row++ {
		assert.Equal(t, []uint8{0, 1, 2}, chip.memory[rectTableBase+row*8:rectTableBase+row*8+3], "row %d", row)
	}
}

// TestEmulatedMiaCopyRectRejectsRectanglesOutsideRAM verifies a rectangle
// whose last row would run past the top of MIA RAM fails before copying
// anything, and an empty row reports a zero size.
func TestEmulatedMiaCopyRectRejectsRectanglesOutsideRAM(t *testing.T) {
	circuit := copyRectCircuit(t)
	chip := circuit.chip

	copyRect(circuit, rectMapBase, 4, rectMapWidth, miaRAMSize-100, 64, 3)
	assert.Equal(t, miaErrorDMATargetOverflow, circuit.read(miaRegErrorLSB))
	assert.Zero(t, chip.memory[miaRAMSize-100], "nothing was copied")

	copyRect(circuit, rectMapBase, 0, rectMapWidth, rectTableBase, 40, 3)
	assert.Equal(t, miaErrorDMASizeZero, circuit.read(miaRegErrorLSB))
}
