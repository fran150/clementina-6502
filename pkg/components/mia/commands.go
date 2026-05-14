package mia

const (
	miaErrorDMASizeZero       uint8 = 0x10
	miaErrorDMASourceOverflow uint8 = 0x11
	miaErrorDMATargetOverflow uint8 = 0x12
)

// executeCommand runs the MIA command identified by the trigger register value.
func (c *emulated_mia) executeCommand(id uint8, params [3]uint8) {
	c.statusSet(miaStatusCmdRunning)

	switch id {
	case 0x00:
		c.resetIndex(c.readRegister(miaRegIdxBSelector))
	case 0x01:
		c.resetIndex(c.readRegister(miaRegIdxASelector))
	case 0x02:
		indexID := params[0]
		c.indexes[indexID].currentAddr = c.indexes[indexID].limitAddr
	case 0x03:
		indexID := params[0]
		c.indexes[indexID].defaultAddr = c.indexes[indexID].currentAddr
	case 0x04:
		indexID := params[0]
		c.indexes[indexID].limitAddr = c.indexes[indexID].currentAddr
	case 0x05:
		for i := uint8(0); i < 255; i++ {
			c.indexes[i].currentAddr = c.indexes[i].limitAddr
		}
	case 0x06:
		c.writeRegister(miaRegIdxAPort, c.indexRead(params[0]))
	case 0x07:
		c.writeRegister(miaRegIdxBPort, c.indexRead(params[0]))
	case 0x10:
		c.dmaTransfer(c.indexes[params[0]].currentAddr, c.indexes[params[1]].currentAddr, uint16(params[2]))
	}

	c.statusClear(miaStatusCmdRunning)
}

// dmaTransfer copies a bounded byte range inside emulated MIA RAM.
func (c *emulated_mia) dmaTransfer(srcOffset uint32, dstOffset uint32, length uint16) bool {
	if length == 0 {
		c.errors.Push(c, miaErrorDMASizeZero)
		return false
	}

	if srcOffset >= miaRAMSize || uint32(length) > miaRAMSize-srcOffset {
		c.errors.Push(c, miaErrorDMASourceOverflow)
		return false
	}

	if dstOffset >= miaRAMSize || uint32(length) > miaRAMSize-dstOffset {
		c.errors.Push(c, miaErrorDMATargetOverflow)
		return false
	}

	c.statusSet(miaStatusDMARunning)
	copy(c.memory[dstOffset:dstOffset+uint32(length)], c.memory[srcOffset:srcOffset+uint32(length)])
	c.statusClear(miaStatusDMARunning)

	return true
}
