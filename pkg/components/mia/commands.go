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
	// miaErrorDMAQueueFull mirrors ERROR_DMA_QUEUE_FULL: firmware queues a
	// copy behind a running one and reports this when its queue is full.
	// Emulated copies finish before the command returns, so it never fires.
	miaErrorDMAQueueFull         uint8 = 0x13
	miaErrorCmdQueueFull         uint8 = 0x20
	miaErrorCmdUnknown           uint8 = 0x21
	miaErrorCtxOverflow          uint8 = 0x22
	miaErrorCtxUnderflow         uint8 = 0x23
	miaErrorWifiInitFailed       uint8 = 0x30
	miaErrorWifiConnectFailed    uint8 = 0x31
	miaErrorVideoUDPAllocFailed  uint8 = 0x40
	miaErrorVideoUDPBindFailed   uint8 = 0x41
	miaErrorInputModeUnavailable uint8 = 0x50
	miaErrorInputProbeInvalid    uint8 = 0x51
	miaErrorInputUDPAllocFailed  uint8 = 0x52
	miaErrorInputUDPBindFailed   uint8 = 0x53
	miaErrorAudioQueueOverflow   uint8 = 0x60
	miaErrorSDBusy               uint8 = 0x70
	miaErrorSDInitFailed         uint8 = 0x71
	miaErrorSDNotReady           uint8 = 0x72
	miaErrorSDReadFailed         uint8 = 0x73
	miaErrorSDWriteFailed        uint8 = 0x74
	miaErrorFSMountFailed        uint8 = 0x78
	miaErrorFSOpenFailed         uint8 = 0x79
	miaErrorFSReadFailed         uint8 = 0x7A
	miaErrorFSCloseFailed        uint8 = 0x7B
	miaErrorFSDirFailed          uint8 = 0x7C
	miaErrorFSInvalidRequest     uint8 = 0x7D
	miaErrorFSNoFileOpen         uint8 = 0x7E
	miaErrorFSWriteFailed        uint8 = 0x7F
	miaErrorFSSeekFailed         uint8 = 0x80
	miaErrorFSSyncFailed         uint8 = 0x81
	miaErrorFSStatFailed         uint8 = 0x82
	miaErrorFSMkdirFailed        uint8 = 0x83
	miaErrorFSDeleteFailed       uint8 = 0x84
	miaErrorFSRenameFailed       uint8 = 0x85
	miaErrorFSFreeFailed         uint8 = 0x86
	miaErrorFSHandleAlreadyOpen  uint8 = 0x87
	miaErrorFSInvalidHandle      uint8 = 0x88
)

// executeCommand runs the MIA command identified by the trigger register value.
func (c *emulated_mia) executeCommand(id uint8, params [3]uint8) {
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
		c.dmaTransferFromIndexes(params[0], params[1], params[2])
	case 0x11:
		c.dmaRectFromIndexes(params[0], params[1], params[2])
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
	case 0x52:
		c.inputRepeatDelay(uint16(params[0]) | uint16(params[1])<<8)
	case 0x53:
		c.inputRepeatInterval(uint16(params[0]) | uint16(params[1])<<8)
	case 0x54:
		c.inputRepeatKey(params[0], params[1])
	case miaCmdClockSnapshot:
		c.timingSnapshot()
	case miaCmdClockSetTI:
		c.timingSetTI(uint32(params[0]) | uint32(params[1])<<8 | uint32(params[2])<<16)
	case 0x60:
		c.audioEnable()
	case 0x61:
		c.audioStop()
	case 0x62:
		c.audioReset()
	case miaCmdAudioSeqLoad:
		for v := 0; v < miaAudioVoiceCount; v++ {
			if params[0]&(1<<uint(v)) != 0 {
				c.audioSeqLoadTrack(v)
			}
		}
	case miaCmdAudioSeqStart:
		c.audioSeqStart(params[0])
	case miaCmdAudioSeqStop:
		c.audioSeqStop(params[0])
	case miaCmdAudioVoiceTake:
		c.audioVoiceTake(params[0])
	case miaCmdAudioVoiceRelease:
		c.audioVoiceRelease(params[0])
	case miaCmdAudioSeqSetBase0:
		c.audioSeqSetBase(0, uint32(params[0])|uint32(params[1])<<8|uint32(params[2])<<16)
	case miaCmdAudioSeqSetBase1:
		c.audioSeqSetBase(1, uint32(params[0])|uint32(params[1])<<8|uint32(params[2])<<16)
	case miaCmdAudioSeqSetBase2:
		c.audioSeqSetBase(2, uint32(params[0])|uint32(params[1])<<8|uint32(params[2])<<16)
	case miaCmdAudioSeqSetBase3:
		c.audioSeqSetBase(3, uint32(params[0])|uint32(params[1])<<8|uint32(params[2])<<16)
	case miaCmdSDInit, miaCmdSDReadSector, miaCmdSDWriteSector, miaCmdSDGetInfo,
		miaCmdFSMount, miaCmdFSOpendir, miaCmdFSReaddir, miaCmdFSOpen,
		miaCmdFSRead, miaCmdFSClose, miaCmdFSLoadToMiaRAM, miaCmdFSWrite,
		miaCmdFSSync, miaCmdFSSeek, miaCmdFSStat, miaCmdFSMkdir,
		miaCmdFSDelete, miaCmdFSRename, miaCmdFSGetFree, miaCmdFSSaveFromMiaRAM,
		miaCmdFSChdir, miaCmdFSFileInfo, miaCmdFSLoadPart:
		c.sdRequest(id)
	default:
		// Unassigned command ids report ERROR_CMD_UNKNOWN, matching the firmware
		// command table where every unregistered id maps to command_empty.
		c.errors.Push(c, miaErrorCmdUnknown)
	}

	c.statusClear(miaStatusCmdRunning)
	c.irqSetFlag(miaIRQCommand)
}

func (c *emulated_mia) dmaTransferFromIndexes(srcIndex uint8, dstIndex uint8, count uint8) bool {
	length := uint16(count)
	if count == 0 {
		current := c.indexes[srcIndex].currentAddr
		limit := c.indexes[srcIndex].limitAddr
		if limit <= current {
			length = 0
		} else if limit-current > uint32(^uint16(0)) {
			c.errors.Push(c, miaErrorDMASourceOverflow)
			return false
		} else {
			length = uint16(limit - current)
		}
	}

	return c.dmaTransfer(c.indexes[srcIndex].currentAddr, c.indexes[dstIndex].currentAddr, length)
}

// dmaRectFromIndexes mirrors command_copy_rect: rowsParam rows (0 means
// 256), each as long as the source index's limit minus its current address;
// after each row the source advances by its index's step and the destination
// by its own. A source step of 0 repeats the first row. Indexes don't move.
func (c *emulated_mia) dmaRectFromIndexes(srcIndex uint8, dstIndex uint8, rowsParam uint8) bool {
	rows := uint32(rowsParam)
	if rows == 0 {
		rows = 256
	}

	src := c.indexes[srcIndex]
	var length uint32
	if src.limitAddr > src.currentAddr {
		length = src.limitAddr - src.currentAddr
	}
	if length > uint32(^uint16(0)) {
		c.errors.Push(c, miaErrorDMASourceOverflow)
		return false
	}

	dst := c.indexes[dstIndex]
	return c.dmaRect(src.currentAddr, dst.currentAddr, length, rows, uint32(src.step), uint32(dst.step))
}

// dmaTransfer copies a bounded byte range inside emulated MIA RAM.
func (c *emulated_mia) dmaTransfer(srcOffset uint32, dstOffset uint32, length uint16) bool {
	return c.dmaRect(srcOffset, dstOffset, uint32(length), 1, 0, 0)
}

// dmaRect mirrors mia_dma_rect_init: rows of length bytes, the source
// advancing srcStride and the destination dstStride per row. The firmware
// queues copies behind a running one and raises IRQ_COMMAND after the last
// row; emulated copies finish at once, so the queue never fills here.
func (c *emulated_mia) dmaRect(srcOffset, dstOffset, length, rows, srcStride, dstStride uint32) bool {
	if length == 0 || rows == 0 {
		c.errors.Push(c, miaErrorDMASizeZero)
		return false
	}

	srcEnd := uint64(srcOffset) + uint64(rows-1)*uint64(srcStride) + uint64(length)
	if srcOffset >= miaRAMSize || srcEnd > miaRAMSize {
		c.errors.Push(c, miaErrorDMASourceOverflow)
		return false
	}

	dstEnd := uint64(dstOffset) + uint64(rows-1)*uint64(dstStride) + uint64(length)
	if dstOffset >= miaRAMSize || dstEnd > miaRAMSize {
		c.errors.Push(c, miaErrorDMATargetOverflow)
		return false
	}

	c.statusSet(miaStatusDMARunning)
	for r := uint32(0); r < rows; r++ {
		s, d := srcOffset+r*srcStride, dstOffset+r*dstStride
		copy(c.memory[d:d+length], c.memory[s:s+length])
		c.videoMarkDirtyRange(d, length)
	}
	c.statusClear(miaStatusDMARunning)
	c.irqSetFlag(miaIRQCommand)

	return true
}
