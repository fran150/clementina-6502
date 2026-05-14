package mia

type miaIndex struct {
	currentAddr uint32
	defaultAddr uint32
	limitAddr   uint32
	step        uint16
	flags       uint8
}

type miaIndexWindow uint8

const (
	miaIndexWindowA miaIndexWindow = iota
	miaIndexWindowB
)

const (
	miaIndexFlagReadStep  uint8 = 0
	miaIndexFlagWriteStep uint8 = 1
	miaIndexFlagStepDir   uint8 = 2
	miaIndexFlagWrap      uint8 = 3
	miaIndexFlagWrapIRQ   uint8 = 4
)

// indexRead returns the byte pointed to by the selected MIA memory index.
func (c *emulated_mia) indexRead(indexID uint8) uint8 {
	return c.memory[c.memoryOffset(c.indexes[indexID].currentAddr)]
}

// indexWrite stores a byte at the address pointed to by the selected MIA memory index.
func (c *emulated_mia) indexWrite(indexID uint8, value uint8) {
	c.memory[c.memoryOffset(c.indexes[indexID].currentAddr)] = value
}

// indexStepAndRead steps an index for a read access and returns the new pointed byte.
func (c *emulated_mia) indexStepAndRead(indexID uint8, window miaIndexWindow) uint8 {
	entry := &c.indexes[indexID]
	c.stepIndex(entry, miaIndexFlagReadStep, window)

	return c.memory[c.memoryOffset(entry.currentAddr)]
}

// indexWriteAndStep writes through an index and then steps it for a write access.
func (c *emulated_mia) indexWriteAndStep(indexID uint8, value uint8, window miaIndexWindow) {
	entry := &c.indexes[indexID]
	c.memory[c.memoryOffset(entry.currentAddr)] = value
	c.stepIndex(entry, miaIndexFlagWriteStep, window)
}

// resetIndex moves the selected index current address back to its default address.
func (c *emulated_mia) resetIndex(indexID uint8) {
	c.indexes[indexID].currentAddr = c.indexes[indexID].defaultAddr
}

// stepIndex applies step, wrap, and IRQ side effects for an indexed access.
func (c *emulated_mia) stepIndex(entry *miaIndex, enableFlag uint8, window miaIndexWindow) {
	if bitSet(entry.flags, enableFlag) {
		if bitSet(entry.flags, miaIndexFlagStepDir) {
			entry.currentAddr += uint32(entry.step)
		} else {
			entry.currentAddr -= uint32(entry.step)
		}
	}

	entry.currentAddr &= miaAddressMask

	if entry.currentAddr >= entry.limitAddr && bitSet(entry.flags, miaIndexFlagWrap) {
		entry.currentAddr = entry.defaultAddr
		c.irqSetFlag(c.wrapIRQFlag(window))
	}
}

// wrapIRQFlag returns the IRQ status bit corresponding to an index window wrap.
func (c *emulated_mia) wrapIRQFlag(window miaIndexWindow) uint16 {
	if window == miaIndexWindowA {
		return miaIRQIdxAWrap
	}

	return miaIRQIdxBWrap
}

// memoryOffset converts a MIA 24-bit address into an offset in emulated MIA RAM.
func (c *emulated_mia) memoryOffset(address uint32) int {
	return int(address & uint32(len(c.memory)-1))
}

// bitSet reports whether a bit is set in an 8-bit value.
func bitSet(value uint8, bit uint8) bool {
	return ((value >> bit) & 0x01) != 0
}
