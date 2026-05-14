package mia

// getCfg returns the value exposed by a MIA configuration register id.
func (c *emulated_mia) getCfg(id uint8) uint8 {
	indexID := (id >> 4) & 0x01
	field := id & 0x0F
	entry := c.indexes[indexID]

	switch field {
	case 0x00:
		return byteFrom24(entry.currentAddr, 0)
	case 0x01:
		return byteFrom24(entry.currentAddr, 1)
	case 0x02:
		return byteFrom24(entry.currentAddr, 2)
	case 0x03:
		return byteFrom24(entry.defaultAddr, 0)
	case 0x04:
		return byteFrom24(entry.defaultAddr, 1)
	case 0x05:
		return byteFrom24(entry.defaultAddr, 2)
	case 0x06:
		return byteFrom24(entry.limitAddr, 0)
	case 0x07:
		return byteFrom24(entry.limitAddr, 1)
	case 0x08:
		return byteFrom24(entry.limitAddr, 2)
	case 0x09:
		return uint8(entry.step)
	case 0x0A:
		return uint8(entry.step >> 8)
	case 0x0B:
		return entry.flags
	default:
		return 0
	}
}

// setCfg updates the MIA index field selected by a configuration register id.
func (c *emulated_mia) setCfg(id uint8, value uint8) {
	indexID := (id >> 4) & 0x01
	field := id & 0x0F
	entry := &c.indexes[indexID]

	switch field {
	case 0x00:
		entry.currentAddr = setByteIn24(entry.currentAddr, 0, value)
	case 0x01:
		entry.currentAddr = setByteIn24(entry.currentAddr, 1, value)
	case 0x02:
		entry.currentAddr = setByteIn24(entry.currentAddr, 2, value)
	case 0x03:
		entry.defaultAddr = setByteIn24(entry.defaultAddr, 0, value)
	case 0x04:
		entry.defaultAddr = setByteIn24(entry.defaultAddr, 1, value)
	case 0x05:
		entry.defaultAddr = setByteIn24(entry.defaultAddr, 2, value)
	case 0x06:
		entry.limitAddr = setByteIn24(entry.limitAddr, 0, value)
	case 0x07:
		entry.limitAddr = setByteIn24(entry.limitAddr, 1, value)
	case 0x08:
		entry.limitAddr = setByteIn24(entry.limitAddr, 2, value)
	case 0x09:
		entry.step = (entry.step & 0xFF00) | uint16(value)
	case 0x0A:
		entry.step = (entry.step & 0x00FF) | (uint16(value) << 8)
	case 0x0B:
		entry.flags = value
	}
}

// byteFrom24 returns one byte from a 24-bit little-endian value.
func byteFrom24(value uint32, index uint8) uint8 {
	return uint8(value >> (index * 8))
}

// setByteIn24 replaces one byte in a 24-bit little-endian value.
func setByteIn24(current uint32, index uint8, value uint8) uint32 {
	shift := index * 8
	mask := uint32(0xFF) << shift
	next := (current &^ mask) | (uint32(value) << shift)

	return next & miaAddressMask
}
