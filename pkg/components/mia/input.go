package mia

import "net"

// This file implements the MIA input subsystem. It mirrors the Pico firmware
// (src/mia/input) so the 6502-facing interface is identical regardless of the
// active input source. MIA exposes input in three forms: a text FIFO, fixed
// Keyboard/Keypad and Consumer HID bitmaps, and fixed mouse and gamepad state
// structures. Two sources can own the live input state: the terminal console
// (text only) and the Wi-Fi input client (full state). USB host is reserved.

const (
	// DefaultInputUDPAddress is the address used by the Wi-Fi input service when
	// no explicit bind address is provided.
	DefaultInputUDPAddress = "127.0.0.1:6503"

	// Input state memory block. It sits outside the video state region and is
	// not synchronized to the video client.
	miaInputStateOffset = 0x11000
	miaInputStateSize   = 0x80

	miaInputKeyboardBitmapOffset = 0x11000
	miaInputConsumerBitmapOffset = 0x11020
	miaInputMouseStateOffset     = 0x11040
	miaInputControlOffset        = 0x11045
	miaInputGamepadOffset        = 0x11050

	miaInputTextFifoSize = 64
	miaInputTextFifoMask = miaInputTextFifoSize - 1

	miaInputGamepadSlotSize = 10
	miaInputGamepadSlots    = 4

	miaHidPageKeyboard = 0x0007
	miaHidPageConsumer = 0x000C

	miaMouseButtonMask = 0x1F
)

// INPUT_STATUS bits ($FFF2).
const (
	miaInputTextReady     uint8 = 1 << 0
	miaInputKeyboardDown  uint8 = 1 << 1
	miaInputConsumerDown  uint8 = 1 << 2
	miaInputMouseDown     uint8 = 1 << 3
	miaInputGamepadDown   uint8 = 1 << 4
	miaInputSourceConsole uint8 = 1 << 5
	miaInputSourceWifi    uint8 = 1 << 6
	miaInputSourceUSBHost uint8 = 1 << 7
)

// INPUT_DEVICE_FLAGS bits ($11045).
const (
	miaInputDeviceKeyboard uint8 = 1 << 0
	miaInputDeviceConsumer uint8 = 1 << 1
	miaInputDeviceMouse    uint8 = 1 << 2
	miaInputDeviceGamepad0 uint8 = 1 << 3
	miaInputDeviceGamepad1 uint8 = 1 << 4
	miaInputDeviceGamepad2 uint8 = 1 << 5
	miaInputDeviceGamepad3 uint8 = 1 << 6
)

// KEYBOARD_EVENT_FLAGS/MASK/ACK bits.
const (
	miaKeyEventText         uint8 = 1 << 0
	miaKeyEventKeyDown      uint8 = 1 << 1
	miaKeyEventKeyUp        uint8 = 1 << 2
	miaKeyEventConsumerDown uint8 = 1 << 3
	miaKeyEventConsumerUp   uint8 = 1 << 4
	miaKeyEventDevice       uint8 = 1 << 5
)

// MOUSE_EVENT_FLAGS/MASK/ACK bits.
const (
	miaMouseEventButtonDown uint8 = 1 << 0
	miaMouseEventButtonUp   uint8 = 1 << 1
	miaMouseEventMove       uint8 = 1 << 2
	miaMouseEventScroll     uint8 = 1 << 3
	miaMouseEventDevice     uint8 = 1 << 4
)

// GAMEPAD_EVENT_FLAGS/MASK/ACK bits.
const (
	miaGamepadEventButtonDown uint8 = 1 << 0
	miaGamepadEventButtonUp   uint8 = 1 << 1
	miaGamepadEventDpad       uint8 = 1 << 2
	miaGamepadEventStick      uint8 = 1 << 3
	miaGamepadEventTrigger    uint8 = 1 << 4
	miaGamepadEventDevice     uint8 = 1 << 5
)

// Input control block byte offsets, laid out from miaInputControlOffset.
const (
	miaInputDeviceFlagsOffset = miaInputControlOffset + iota
	miaKeyboardEventFlagsOffset
	miaKeyboardEventMaskOffset
	miaKeyboardEventAckOffset
	miaMouseEventFlagsOffset
	miaMouseEventMaskOffset
	miaMouseEventAckOffset
	miaGamepadEventFlagsOffset
	miaGamepadEventMaskOffset
	miaGamepadEventAckOffset
)

type miaInputMode uint8

const (
	miaInputModeConsole miaInputMode = iota
	miaInputModeWifi
	miaInputModeUSBHost
)

// miaInputDefaultMode is the input source MIA selects at reset when it is
// available, falling back to console otherwise. It mirrors the firmware
// MIA_INPUT_DEFAULT_MODE build option.
const miaInputDefaultMode = miaInputModeWifi

type miaInputState struct {
	conn        *net.UDPConn
	bindAddress string
	udpReady    bool

	mode miaInputMode

	textFifo [miaInputTextFifoSize]uint8
	textHead uint32
	textTail uint32

	wifiActive  bool
	wifiRemote  *net.UDPAddr
	wifiSession uint32
	wifiLastSeq uint16
	wifiCaps    uint16

	nextSessionValue uint32
	txSeq            uint32

	// Auto-repeat for held editing keys: while repeatUsage is held, repeatByte
	// is re-enqueued into the text FIFO once repeatDeadline (a nowNs timestamp)
	// passes. repeatUsage == 0 means nothing is repeating.
	repeatUsage    uint16
	repeatByte     uint8
	repeatDeadline int64
}

// inputResetRuntimeState clears the input state block, resets the text FIFO and
// Wi-Fi session, reconfigures input indexes, and selects the default mode. It
// preserves the UDP listener so the Wi-Fi source survives a 6502 reset.
func (c *emulated_mia) inputResetRuntimeState() {
	conn := c.input.conn
	bindAddress := c.input.bindAddress
	udpReady := c.input.udpReady
	txSeq := c.input.txSeq
	nextSessionValue := c.input.nextSessionValue

	c.input = miaInputState{
		conn:             conn,
		bindAddress:      bindAddress,
		udpReady:         udpReady,
		txSeq:            txSeq,
		nextSessionValue: nextSessionValue,
		mode:             miaInputModeConsole,
	}

	clear(c.memory[miaInputStateOffset : miaInputStateOffset+miaInputStateSize])
	c.inputConfigureIndexes()

	// Select the configured default source when it is available, mirroring the
	// firmware MIA_INPUT_DEFAULT_MODE selection with its fallback to console.
	if miaInputDefaultMode != miaInputModeConsole && c.inputModeAvailable(miaInputDefaultMode) {
		c.input.mode = miaInputDefaultMode
	}

	c.inputRecomputeStatus()
}

// inputClose tears down the input UDP service. It is part of the chip Close path.
func (c *emulated_mia) inputClose() {
	c.mu.Lock()
	conn := c.input.conn
	c.input.conn = nil
	c.input.bindAddress = ""
	c.input.udpReady = false
	c.mu.Unlock()

	if conn != nil {
		conn.Close()
	}
}

// inputService applies queued 6502 event acknowledgements and re-evaluates the
// input IRQ bits. It runs once per MIA service cycle.
func (c *emulated_mia) inputService() {
	c.inputApplyEventAcks()
	c.inputRepeatService()
	c.inputUpdateIRQs()
}

// inputModeAvailable reports whether a requested input mode can be activated.
func (c *emulated_mia) inputModeAvailable(mode miaInputMode) bool {
	switch mode {
	case miaInputModeConsole:
		return true
	case miaInputModeWifi:
		return c.input.udpReady
	case miaInputModeUSBHost:
		// USB host input is not emulated; the mode is recognized but unavailable.
		return false
	default:
		return false
	}
}

// inputSetMode changes the active input source, clearing the live state owned by
// the previous source. It returns false when the requested mode is unavailable.
func (c *emulated_mia) inputSetMode(mode miaInputMode) bool {
	if !c.inputModeAvailable(mode) {
		return false
	}

	if mode == c.input.mode {
		return true
	}

	if c.input.mode == miaInputModeWifi {
		c.inputInvalidateWifiSession(true)
	} else {
		c.inputClearLiveState(true)
	}

	c.input.mode = mode

	if mode == miaInputModeWifi {
		c.inputInvalidateWifiSession(false)
	} else {
		c.inputClearLiveState(false)
	}

	c.inputRecomputeStatus()
	return true
}

// inputSetProbe positions one of the parked single-byte keyboard or consumer
// probe indexes. Probe ids $00-$07 are keyboard probes, $08-$0F are consumer
// probes, and the byte offset is masked to the 32-byte bitmap range.
func (c *emulated_mia) inputSetProbe(probeID uint8, byteOffset uint8) bool {
	offset := uint32(byteOffset & 0x1F)

	if probeID < 8 {
		c.inputConfigureIndex(0x50+probeID, miaInputKeyboardBitmapOffset+offset, 1, false)
		return true
	}

	if probeID < 16 {
		c.inputConfigureIndex(0x58+probeID-8, miaInputConsumerBitmapOffset+offset, 1, false)
		return true
	}

	return false
}

// inputConfigureIndexes preconfigures the input probe and streaming indexes.
func (c *emulated_mia) inputConfigureIndexes() {
	for i := uint8(0); i < 8; i++ {
		c.inputConfigureIndex(0x50+i, miaInputKeyboardBitmapOffset, 1, false)
		c.inputConfigureIndex(0x58+i, miaInputConsumerBitmapOffset, 1, false)
	}

	c.inputConfigureIndex(0x60, miaInputStateOffset, miaInputStateSize, true)
	c.inputConfigureIndex(0x61, miaInputKeyboardBitmapOffset, 32, true)
	c.inputConfigureIndex(0x62, miaInputConsumerBitmapOffset, 32, true)
	c.inputConfigureIndex(0x63, miaInputMouseStateOffset, 5, true)
	c.inputConfigureIndex(0x64, miaInputGamepadOffset, miaInputGamepadSlotSize, true)
	c.inputConfigureIndex(0x65, miaInputGamepadOffset+miaInputGamepadSlotSize, miaInputGamepadSlotSize, true)
	c.inputConfigureIndex(0x66, miaInputGamepadOffset+miaInputGamepadSlotSize*2, miaInputGamepadSlotSize, true)
	c.inputConfigureIndex(0x67, miaInputGamepadOffset+miaInputGamepadSlotSize*3, miaInputGamepadSlotSize, true)
	c.inputConfigureIndex(0x68, miaInputControlOffset, 11, true)
}

// inputConfigureIndex sets up a single input index. Streaming indexes step and
// wrap over their range; probe indexes wrap but keep stepping disabled.
func (c *emulated_mia) inputConfigureIndex(indexID uint8, start uint32, length uint32, step bool) {
	flags := uint8(1 << miaIndexFlagWrap)
	if step {
		flags |= (1 << miaIndexFlagReadStep) | (1 << miaIndexFlagWriteStep)
	}

	c.indexes[indexID] = miaIndex{
		currentAddr: start,
		defaultAddr: start,
		limitAddr:   start + length,
		step:        1,
		flags:       flags,
	}
}

// inputRecomputeStatus rebuilds INPUT_STATUS from the active source and the held
// digital input, then refreshes the text FIFO register snapshot.
func (c *emulated_mia) inputRecomputeStatus() {
	var status uint8

	switch c.input.mode {
	case miaInputModeConsole:
		status |= miaInputSourceConsole
	case miaInputModeWifi:
		status |= miaInputSourceWifi
	case miaInputModeUSBHost:
		status |= miaInputSourceUSBHost
	}

	if c.inputBitmapAny(miaInputKeyboardBitmapOffset) {
		status |= miaInputKeyboardDown
	}
	if c.inputBitmapAny(miaInputConsumerBitmapOffset) {
		status |= miaInputConsumerDown
	}
	if c.memory[miaInputMouseStateOffset]&miaMouseButtonMask != 0 {
		status |= miaInputMouseDown
	}
	if c.inputGamepadAnyDigitalDown() {
		status |= miaInputGamepadDown
	}

	c.registers[miaRegInputStatus] = status
	c.inputPublishTextSnapshot()
}

// inputPublishTextSnapshot updates INPUT_CHAR, INPUT_CHAR_COUNT, and the
// INPUT_TEXT_READY status bit from the current text FIFO contents.
func (c *emulated_mia) inputPublishTextSnapshot() {
	head := c.input.textHead
	tail := c.inputFifoEffectiveTail(head, c.input.textTail)
	count := c.inputFifoCount(head, tail)

	c.registers[miaRegInputCharCount] = count
	if count == 0 {
		c.registers[miaRegInputChar] = 0
		c.registers[miaRegInputStatus] &^= miaInputTextReady
	} else {
		c.registers[miaRegInputChar] = c.input.textFifo[inputFifoIndex(tail)]
		c.registers[miaRegInputStatus] |= miaInputTextReady
	}
}

// inputOnCharRead pops one byte from the text FIFO. It is the read-to-pop side
// effect of a 6502 read from INPUT_CHAR ($FFF3).
func (c *emulated_mia) inputOnCharRead() {
	head := c.input.textHead
	tail := c.inputFifoEffectiveTail(head, c.input.textTail)

	if head != tail {
		c.input.textTail = tail + 1
	}

	c.inputPublishTextSnapshot()
}

// inputEnqueueText appends a byte to the text FIFO, dropping the oldest byte when
// the FIFO is full, and latches a text event.
func (c *emulated_mia) inputEnqueueText(value uint8) {
	head := c.input.textHead
	c.input.textFifo[inputFifoIndex(head)] = value
	c.input.textHead = head + 1

	c.inputPublishTextSnapshot()
	c.inputSetKeyboardEvents(miaKeyEventText)
}

// inputClearTextFifo discards all queued text bytes.
func (c *emulated_mia) inputClearTextFifo() {
	c.input.textTail = c.input.textHead
	c.inputPublishTextSnapshot()
}

// inputFifoEffectiveTail clamps the tail so the FIFO never reports more than its
// capacity, dropping the oldest bytes on overflow.
func (c *emulated_mia) inputFifoEffectiveTail(head, tail uint32) uint32 {
	if head-tail > miaInputTextFifoSize {
		return head - miaInputTextFifoSize
	}
	return tail
}

// inputFifoCount returns the queued byte count, capped at the FIFO capacity.
func (c *emulated_mia) inputFifoCount(head, tail uint32) uint8 {
	count := head - c.inputFifoEffectiveTail(head, tail)
	if count > miaInputTextFifoSize {
		return miaInputTextFifoSize
	}
	return uint8(count)
}

func inputFifoIndex(sequence uint32) uint8 {
	return uint8(sequence & miaInputTextFifoMask)
}

// inputBitmapAny reports whether any bit is set in a 32-byte HID bitmap.
func (c *emulated_mia) inputBitmapAny(offset int) bool {
	for i := 0; i < 32; i++ {
		if c.memory[offset+i] != 0 {
			return true
		}
	}
	return false
}

// inputGamepadAnyDigitalDown reports whether a connected gamepad has any active
// d-pad, digital stick summary, or digital button bit.
func (c *emulated_mia) inputGamepadAnyDigitalDown() bool {
	for player := 0; player < miaInputGamepadSlots; player++ {
		base := miaInputGamepadOffset + player*miaInputGamepadSlotSize
		if c.memory[base]&0x80 == 0 {
			continue
		}
		if c.memory[base]&0x0F != 0 || c.memory[base+1] != 0 || c.memory[base+2] != 0 || c.memory[base+3] != 0 {
			return true
		}
	}
	return false
}

// inputSetKeyboardEvents latches keyboard/consumer/text event causes.
func (c *emulated_mia) inputSetKeyboardEvents(flags uint8) {
	if flags == 0 {
		return
	}
	c.memory[miaKeyboardEventFlagsOffset] |= flags
	c.inputUpdateIRQs()
}

// inputSetMouseEvents latches mouse event causes.
func (c *emulated_mia) inputSetMouseEvents(flags uint8) {
	if flags == 0 {
		return
	}
	c.memory[miaMouseEventFlagsOffset] |= flags
	c.inputUpdateIRQs()
}

// inputSetGamepadEvents latches gamepad event causes.
func (c *emulated_mia) inputSetGamepadEvents(flags uint8) {
	if flags == 0 {
		return
	}
	c.memory[miaGamepadEventFlagsOffset] |= flags
	c.inputUpdateIRQs()
}

// inputSetDeviceFlags publishes the active-source capability and gamepad-slot
// flags, latching device-change events for any changed group.
func (c *emulated_mia) inputSetDeviceFlags(flags uint8) {
	old := c.memory[miaInputDeviceFlagsOffset]
	flags &= 0x7F
	if old == flags {
		return
	}

	changed := old ^ flags
	c.memory[miaInputDeviceFlagsOffset] = flags

	if changed&(miaInputDeviceKeyboard|miaInputDeviceConsumer) != 0 {
		c.inputSetKeyboardEvents(miaKeyEventDevice)
	}
	if changed&miaInputDeviceMouse != 0 {
		c.inputSetMouseEvents(miaMouseEventDevice)
	}
	if changed&(miaInputDeviceGamepad0|miaInputDeviceGamepad1|miaInputDeviceGamepad2|miaInputDeviceGamepad3) != 0 {
		c.inputSetGamepadEvents(miaGamepadEventDevice)
	}
}

// inputUpdateIRQs raises the input IRQ bits whose latched event flags intersect
// their enabled mask bits.
func (c *emulated_mia) inputUpdateIRQs() {
	if c.memory[miaKeyboardEventFlagsOffset]&c.memory[miaKeyboardEventMaskOffset] != 0 {
		c.irqSetFlag(miaIRQInputKeyboard)
	}
	if c.memory[miaMouseEventFlagsOffset]&c.memory[miaMouseEventMaskOffset] != 0 {
		c.irqSetFlag(miaIRQInputMouse)
	}
	if c.memory[miaGamepadEventFlagsOffset]&c.memory[miaGamepadEventMaskOffset] != 0 {
		c.irqSetFlag(miaIRQInputGamepad)
	}
}

// inputApplyEventAcks clears latched event bits that the 6502 acknowledged and
// resets the acknowledge bytes to zero.
func (c *emulated_mia) inputApplyEventAcks() {
	if ack := c.memory[miaKeyboardEventAckOffset]; ack != 0 {
		c.memory[miaKeyboardEventFlagsOffset] &^= ack
		c.memory[miaKeyboardEventAckOffset] = 0
	}
	if ack := c.memory[miaMouseEventAckOffset]; ack != 0 {
		c.memory[miaMouseEventFlagsOffset] &^= ack
		c.memory[miaMouseEventAckOffset] = 0
	}
	if ack := c.memory[miaGamepadEventAckOffset]; ack != 0 {
		c.memory[miaGamepadEventFlagsOffset] &^= ack
		c.memory[miaGamepadEventAckOffset] = 0
	}

	c.inputUpdateIRQs()
}

// inputClearKeyboardConsumer clears both HID bitmaps, optionally latching the
// matching release events.
func (c *emulated_mia) inputClearKeyboardConsumer(publishEvents bool) {
	var events uint8
	if publishEvents {
		if c.inputBitmapAny(miaInputKeyboardBitmapOffset) {
			events |= miaKeyEventKeyUp
		}
		if c.inputBitmapAny(miaInputConsumerBitmapOffset) {
			events |= miaKeyEventConsumerUp
		}
	}

	clear(c.memory[miaInputKeyboardBitmapOffset : miaInputKeyboardBitmapOffset+32])
	clear(c.memory[miaInputConsumerBitmapOffset : miaInputConsumerBitmapOffset+32])

	c.inputSetKeyboardEvents(events)
}

// inputClearMouse clears the mouse state, optionally latching the matching
// button, movement, and scroll events.
func (c *emulated_mia) inputClearMouse(publishEvents bool) {
	base := miaInputMouseStateOffset
	var events uint8
	if publishEvents {
		if c.memory[base]&miaMouseButtonMask != 0 {
			events |= miaMouseEventButtonUp
		}
		if c.memory[base+1] != 0 || c.memory[base+2] != 0 {
			events |= miaMouseEventMove
		}
		if c.memory[base+3] != 0 || c.memory[base+4] != 0 {
			events |= miaMouseEventScroll
		}
	}

	clear(c.memory[base : base+5])
	c.inputSetMouseEvents(events)
}

// inputClearGamepadSlot clears one gamepad slot and marks it disconnected,
// optionally latching the matching events.
func (c *emulated_mia) inputClearGamepadSlot(player int, publishEvents bool) {
	base := miaInputGamepadOffset + player*miaInputGamepadSlotSize
	var events uint8
	if publishEvents {
		if c.memory[base+2]|c.memory[base+3] != 0 {
			events |= miaGamepadEventButtonUp
		}
		if c.memory[base]&0x0F != 0 {
			events |= miaGamepadEventDpad
		}
		if c.memory[base+1] != 0 || c.memory[base+4] != 0 || c.memory[base+5] != 0 || c.memory[base+6] != 0 || c.memory[base+7] != 0 {
			events |= miaGamepadEventStick
		}
		if c.memory[base+8] != 0 || c.memory[base+9] != 0 {
			events |= miaGamepadEventTrigger
		}
	}

	clear(c.memory[base : base+miaInputGamepadSlotSize])
	c.inputUpdateGamepadDeviceFlag(player, false)
	c.inputSetGamepadEvents(events)
}

// inputClearGamepads clears every gamepad slot.
func (c *emulated_mia) inputClearGamepads(publishEvents bool) {
	for player := 0; player < miaInputGamepadSlots; player++ {
		c.inputClearGamepadSlot(player, publishEvents)
	}
}

// inputClearLiveState clears all held keyboard, consumer, mouse, and gamepad
// state and resets the device flags.
func (c *emulated_mia) inputClearLiveState(publishEvents bool) {
	c.inputClearKeyboardConsumer(publishEvents)
	c.inputClearMouse(publishEvents)
	c.inputClearGamepads(publishEvents)
	c.inputSetDeviceFlags(0)
	c.inputRecomputeStatus()
}

// inputUpdateGamepadDeviceFlag updates the connected bit for one gamepad slot.
func (c *emulated_mia) inputUpdateGamepadDeviceFlag(player int, connected bool) {
	mask := uint8(miaInputDeviceGamepad0 << player)
	flags := c.memory[miaInputDeviceFlagsOffset]
	if connected {
		flags |= mask
	} else {
		flags &^= mask
	}
	c.inputSetDeviceFlags(flags)
}

// inputSetHidUsage updates a single keyboard or consumer usage bit and latches
// the matching down/up event when the bit changes.
func (c *emulated_mia) inputSetHidUsage(usagePage uint16, usageID uint16, down bool) {
	if usageID > 0x00FF {
		return
	}

	bitmapOffset, downEvent, upEvent, ok := c.inputHidPage(usagePage)
	if !ok {
		return
	}

	byteIndex := bitmapOffset + int(usageID>>3)
	mask := uint8(1 << (usageID & 7))
	wasDown := c.memory[byteIndex]&mask != 0

	if down {
		c.memory[byteIndex] |= mask
	} else {
		c.memory[byteIndex] &^= mask
	}

	if wasDown != down {
		if down {
			c.inputSetKeyboardEvents(downEvent)
		} else {
			c.inputSetKeyboardEvents(upEvent)
		}
		c.inputRecomputeStatus()
	}
}

// inputSetHidBitmap replaces a full HID bitmap, latching down/up events for the
// bits that changed.
func (c *emulated_mia) inputSetHidBitmap(usagePage uint16, newBitmap []byte) {
	bitmapOffset, downEvent, upEvent, ok := c.inputHidPage(usagePage)
	if !ok {
		return
	}

	var events uint8
	for i := 0; i < 32; i++ {
		old := c.memory[bitmapOffset+i]
		next := newBitmap[i]
		changed := old ^ next
		if changed&next != 0 {
			events |= downEvent
		}
		if changed&old != 0 {
			events |= upEvent
		}
		c.memory[bitmapOffset+i] = next
	}

	c.inputSetKeyboardEvents(events)
	c.inputRecomputeStatus()
}

// inputDecodeKeyUsage maps a keyboard HID usage to the control byte MIA pushes
// into the text FIFO for non-text editing keys: cursor moves, Home, and the
// editing control keys. It returns 0 for everything else - printable characters
// reach the FIFO as text instead, so they must not decode here or they would be
// enqueued twice. MIA owns this table (rather than the input client) so the
// keyboard decode lives in one place, mirroring how the C64 KERNAL, not the
// keyboard, owns the decode table. Cursor and Home codes use PETSCII values; the
// control keys reuse their ASCII codes.
func inputDecodeKeyUsage(usageID uint16) uint8 {
	switch usageID {
	case 0x28, 0x58: // Enter, Keypad Enter
		return 0x0D
	case 0x2B: // Tab
		return 0x09
	case 0x2A: // Backspace
		return 0x08
	case 0x29: // Escape
		return 0x1B
	case 0x49: // Insert
		return 0x94
	case 0x4A: // Home
		return 0x13
	case 0x4C: // Delete (forward)
		return 0x7F
	case 0x4F: // Right Arrow
		return 0x1D
	case 0x50: // Left Arrow
		return 0x9D
	case 0x51: // Down Arrow
		return 0x11
	case 0x52: // Up Arrow
		return 0x91
	default:
		return 0
	}
}

// Key auto-repeat timing. Held editing keys re-enqueue their byte after an
// initial delay, then at a steady interval, matching a typewriter-style repeat.
const (
	miaKeyRepeatDelayNs    int64 = 400 * 1_000_000 // wait before the first repeat
	miaKeyRepeatIntervalNs int64 = 60 * 1_000_000  // ~16 repeats/sec while held
)

// inputKeyRepeats reports whether a held key should auto-repeat. Only the keys
// where holding is useful repeat (cursor moves and Backspace); Enter and the
// other one-shot keys fire once per press.
func inputKeyRepeats(usageID uint16) bool {
	switch usageID {
	case 0x2A, 0x4C, 0x4F, 0x50, 0x51, 0x52: // Backspace, Delete, Right, Left, Down, Up
		return true
	default:
		return false
	}
}

// inputUsageDown reports whether a keyboard usage bit is currently held.
func (c *emulated_mia) inputUsageDown(usageID uint16) bool {
	if usageID > 0x00FF {
		return false
	}
	byteIndex := miaInputKeyboardBitmapOffset + int(usageID>>3)
	return c.memory[byteIndex]&uint8(1<<(usageID&7)) != 0
}

// inputArmRepeat starts auto-repeating usageID, re-enqueuing decoded byte b.
func (c *emulated_mia) inputArmRepeat(usageID uint16, b uint8) {
	c.input.repeatUsage = usageID
	c.input.repeatByte = b
	c.input.repeatDeadline = c.nowNs + miaKeyRepeatDelayNs
}

// inputReleaseRepeat stops auto-repeat if usageID is the repeating key.
func (c *emulated_mia) inputReleaseRepeat(usageID uint16) {
	if usageID == c.input.repeatUsage {
		c.input.repeatUsage = 0
	}
}

// inputRepeatService re-enqueues the held key's byte once its deadline passes,
// then schedules the next repeat. It also stops if the key was released without
// going through the HID-event path (e.g. a full bitmap replace).
func (c *emulated_mia) inputRepeatService() {
	if c.input.repeatUsage == 0 {
		return
	}
	if !c.inputUsageDown(c.input.repeatUsage) {
		c.input.repeatUsage = 0
		return
	}
	if c.nowNs < c.input.repeatDeadline {
		return
	}

	c.inputEnqueueText(c.input.repeatByte)
	c.inputRecomputeStatus()
	c.input.repeatDeadline = c.nowNs + miaKeyRepeatIntervalNs
}

// inputHidPage maps a HID usage page to its bitmap offset and event bits.
func (c *emulated_mia) inputHidPage(usagePage uint16) (offset int, downEvent uint8, upEvent uint8, ok bool) {
	switch usagePage {
	case miaHidPageKeyboard:
		return miaInputKeyboardBitmapOffset, miaKeyEventKeyDown, miaKeyEventKeyUp, true
	case miaHidPageConsumer:
		return miaInputConsumerBitmapOffset, miaKeyEventConsumerDown, miaKeyEventConsumerUp, true
	default:
		return 0, 0, 0, false
	}
}

// inputApplyMouseDelta writes the button snapshot and adds the signed movement
// deltas to the wrapping mouse accumulators. The payload is buttons, dx, dy,
// wheel, pan.
func (c *emulated_mia) inputApplyMouseDelta(payload []byte) {
	base := miaInputMouseStateOffset
	oldButtons := c.memory[base] & miaMouseButtonMask
	newButtons := payload[0] & miaMouseButtonMask
	changed := oldButtons ^ newButtons

	var events uint8
	if changed&newButtons != 0 {
		events |= miaMouseEventButtonDown
	}
	if changed&oldButtons != 0 {
		events |= miaMouseEventButtonUp
	}

	c.memory[base] = newButtons
	c.memory[base+1] += payload[1]
	c.memory[base+2] += payload[2]
	c.memory[base+3] += payload[3]
	c.memory[base+4] += payload[4]

	if payload[1] != 0 || payload[2] != 0 {
		events |= miaMouseEventMove
	}
	if payload[3] != 0 || payload[4] != 0 {
		events |= miaMouseEventScroll
	}

	c.inputSetMouseEvents(events)
	c.inputRecomputeStatus()
}

// inputApplyGamepadState copies a full 10-byte gamepad slot and latches the
// button, d-pad, stick, trigger, and connected-state events that changed.
func (c *emulated_mia) inputApplyGamepadState(player int, next []byte) {
	base := miaInputGamepadOffset + player*miaInputGamepadSlotSize
	slot := c.memory[base : base+miaInputGamepadSlotSize]

	var events uint8
	oldButtons := uint16(slot[2]) | uint16(slot[3])<<8
	newButtons := uint16(next[2]) | uint16(next[3])<<8
	changedButtons := oldButtons ^ newButtons

	if changedButtons&newButtons != 0 {
		events |= miaGamepadEventButtonDown
	}
	if changedButtons&oldButtons != 0 {
		events |= miaGamepadEventButtonUp
	}
	if (slot[0]^next[0])&0x0F != 0 {
		events |= miaGamepadEventDpad
	}
	if slot[1] != next[1] || slot[4] != next[4] || slot[5] != next[5] || slot[6] != next[6] || slot[7] != next[7] {
		events |= miaGamepadEventStick
	}
	if slot[8] != next[8] || slot[9] != next[9] {
		events |= miaGamepadEventTrigger
	}

	wasConnected := slot[0]&0x80 != 0
	connected := next[0]&0x80 != 0

	copy(slot, next[:miaInputGamepadSlotSize])
	c.inputUpdateGamepadDeviceFlag(player, connected)
	if wasConnected != connected {
		events |= miaGamepadEventDevice
	}

	c.inputSetGamepadEvents(events)
	c.inputRecomputeStatus()
}

// inputConsoleByte queues a console text byte. Console input is text-only and is
// ignored unless the console source owns input.
func (c *emulated_mia) inputConsoleByte(value uint8) {
	if c.input.mode != miaInputModeConsole {
		return
	}

	c.inputEnqueueText(value)
	c.inputRecomputeStatus()
}

// inputConsoleEndCapture refreshes the published registers when console capture
// ends.
func (c *emulated_mia) inputConsoleEndCapture() {
	c.inputRecomputeStatus()
}

// DebugQueueInput injects one byte into the text input FIFO as if it had been
// typed on the console (the default input source). It exists for headless tests
// and tools that need to drive the 6502's input without a live input client.
func (c *emulated_mia) DebugQueueInput(value uint8) {
	c.mu.Lock()
	c.inputConsoleByte(value)
	c.mu.Unlock()
}

// DebugReadVideo returns one byte of MIA video RAM at the given internal video
// offset (e.g. miaVideoOverlayNTOffset + row*40 + col). Debug/testing only.
func (c *emulated_mia) DebugReadVideo(offset uint32) uint8 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if int(offset) >= len(c.memory) {
		return 0
	}
	return c.memory[offset]
}

// inputModeName returns the human-readable name of an input mode.
func inputModeName(mode miaInputMode) string {
	switch mode {
	case miaInputModeConsole:
		return "console"
	case miaInputModeWifi:
		return "wifi"
	case miaInputModeUSBHost:
		return "usb_host"
	default:
		return "unknown"
	}
}
