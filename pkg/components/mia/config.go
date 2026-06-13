package mia

// getCfg returns the value exposed by a MIA configuration register id.
func (c *emulated_mia) getCfg(id uint8) uint8 {
	if id >= 0x20 {
		switch id {
		case miaCfgSpeedL:
			return byteFrom24(c.appliedPhi2Hz, 0)
		case miaCfgSpeedM:
			return byteFrom24(c.appliedPhi2Hz, 1)
		case miaCfgSpeedH:
			return byteFrom24(c.appliedPhi2Hz, 2)
		default:
			return 0
		}
	}

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
	if id >= 0x20 {
		switch id {
		case miaCfgSpeedL:
			c.stagePhi2HzByte(0, value)
		case miaCfgSpeedM:
			c.stagePhi2HzByte(1, value)
		case miaCfgSpeedH:
			c.stagePhi2HzByte(2, value)
			c.commitPhi2Hz()
		}

		return
	}

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

// RequestPhi2Hz asks the emulated MIA to change PHI2 as if firmware wrote the
// speed configuration registers.
func (c *emulated_mia) RequestPhi2Hz(hz uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stagedPhi2Hz = hz
	c.commitPhi2Hz()
}

// AppliedPhi2Hz returns the last PHI2 frequency acknowledged by the emulated MIA.
func (c *emulated_mia) AppliedPhi2Hz() uint32 {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.appliedPhi2Hz
}

// SetPhi2HzChangedHandler installs a callback invoked when MIA receives a PHI2
// change request. The callback receives the clamped target frequency in Hz.
func (c *emulated_mia) SetPhi2HzChangedHandler(handler func(uint32)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.phi2HzChanged = handler
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

// speedResetRuntimeState resets staged PHI2 changes while keeping the applied speed.
func (c *emulated_mia) speedResetRuntimeState() {
	c.stagedPhi2Hz = c.appliedPhi2Hz
	c.requestedPhi2Hz = c.appliedPhi2Hz
	c.speedChangeRequested = false
	c.statusClear(miaStatusSpeedChanging)
}

// stagePhi2HzByte updates one byte of the staged PHI2 frequency.
func (c *emulated_mia) stagePhi2HzByte(index uint8, value uint8) {
	c.stagedPhi2Hz = setByteIn24(c.stagedPhi2Hz, index, value)
}

// commitPhi2Hz requests applying the staged PHI2 frequency.
func (c *emulated_mia) commitPhi2Hz() {
	c.requestedPhi2Hz = c.stagedPhi2Hz
	c.statusSet(miaStatusSpeedChanging)
	c.speedChangeRequested = true
	c.notifyPhi2HzChanged(clampPhi2Hz(c.requestedPhi2Hz))
}

// speedService applies a pending PHI2 speed change and raises the completion IRQ flag.
func (c *emulated_mia) speedService() {
	if !c.speedChangeRequested {
		return
	}

	c.speedChangeRequested = false
	c.appliedPhi2Hz = clampPhi2Hz(c.requestedPhi2Hz)
	c.stagedPhi2Hz = c.appliedPhi2Hz
	c.requestedPhi2Hz = c.appliedPhi2Hz
	c.statusClear(miaStatusSpeedChanging)
	c.irqSetFlag(miaIRQSpeedChanged)
}

func (c *emulated_mia) notifyPhi2HzChanged(hz uint32) {
	if c.phi2HzChanged == nil {
		return
	}

	c.phi2HzChanged(hz)
}

func clampPhi2Hz(value uint32) uint32 {
	if value < miaMinPhi2Hz {
		return miaMinPhi2Hz
	}

	if value > miaMaxPhi2Hz {
		return miaMaxPhi2Hz
	}

	return value
}
