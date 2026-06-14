package mia

// MIA error codes. These mirror the Pico firmware err.h definitions one-for-one
// so the 6502-visible ERROR register and the console 'errors' command report the
// same codes. Some codes only originate on real hardware paths the emulator does
// not exercise (Wi-Fi/CYW43 setup, UDP bind failures, the core1->core0 command
// FIFO); they are defined for naming parity and so the console can decode them.
const (
	miaErrorMIACannotAllocateRAM uint8 = 0x01
	miaErrorQueueOverflow        uint8 = 0x02
	miaErrorDMASizeZero          uint8 = 0x10
	miaErrorDMASourceOverflow    uint8 = 0x11
	miaErrorDMATargetOverflow    uint8 = 0x12
	miaErrorCmdQueueFull         uint8 = 0x20
	miaErrorCmdUnknown           uint8 = 0x21
	miaErrorWifiInitFailed       uint8 = 0x30
	miaErrorWifiConnectFailed    uint8 = 0x31
	miaErrorVideoUDPAllocFailed  uint8 = 0x40
	miaErrorVideoUDPBindFailed   uint8 = 0x41
	miaErrorInputModeUnavailable uint8 = 0x50
	miaErrorInputProbeInvalid    uint8 = 0x51
	miaErrorInputUDPAllocFailed  uint8 = 0x52
	miaErrorInputUDPBindFailed   uint8 = 0x53
	miaErrorAudioQueueOverflow   uint8 = 0x60
)

// executeCommand runs the MIA command identified by the trigger register value.
func (c *emulated_mia) executeCommand(id uint8, params [3]uint8) {
	c.statusSet(miaStatusCmdRunning)

	switch id {
	case 0x00:
		c.resetIndex(c.readRegister(miaRegIdxASelector))
	case 0x01:
		c.resetIndex(c.readRegister(miaRegIdxBSelector))
	case 0x02:
		indexID := params[0]
		c.resetIndex(indexID)
	case 0x03:
		indexID := params[0]
		c.indexes[indexID].defaultAddr = c.indexes[indexID].currentAddr
	case 0x04:
		indexID := params[0]
		c.indexes[indexID].limitAddr = c.indexes[indexID].currentAddr
	case 0x05:
		for i := range c.indexes {
			c.indexes[i].currentAddr = c.indexes[i].defaultAddr
		}
	case 0x06:
		c.writeRegister(miaRegIdxAPort, c.indexRead(params[0]))
	case 0x07:
		c.writeRegister(miaRegIdxBPort, c.indexRead(params[0]))
	case 0x10:
		c.dmaTransfer(c.indexes[params[0]].currentAddr, c.indexes[params[1]].currentAddr, uint16(params[2]))
	case 0x30:
		// Pause is 6502-facing (a program can freeze itself at a diagnostic
		// point) but there is no 6502 resume command: once PHI2 is stopped the
		// CPU cannot fetch the resume trigger. Resume comes from the MIA console
		// 'exec resume' or a reset. This mirrors the firmware command table.
		c.execPause()
	case 0x42:
		c.videoForceFullRefresh()
	case 0x43:
		c.videoSetMode(params[0])
	case 0x50:
		if !c.inputSetMode(miaInputMode(params[0])) {
			c.errors.Push(c, miaErrorInputModeUnavailable)
		}
	case 0x51:
		if !c.inputSetProbe(params[0], params[1]) {
			c.errors.Push(c, miaErrorInputProbeInvalid)
		}
	case 0x60:
		c.audioEnable()
	case 0x61:
		c.audioStop()
	case 0x62:
		c.audioReset()
	default:
		// Unassigned command ids report ERROR_CMD_UNKNOWN, matching the firmware
		// command table where every unregistered id maps to command_empty.
		c.errors.Push(c, miaErrorCmdUnknown)
	}

	c.statusClear(miaStatusCmdRunning)
	c.irqSetFlag(miaIRQCommand)
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
	c.videoMarkDirtyRange(dstOffset, uint32(length))
	c.statusClear(miaStatusDMARunning)
	c.irqSetFlag(miaIRQCommand)

	return true
}
