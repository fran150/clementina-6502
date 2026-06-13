package mia

import (
	"encoding/binary"
	"net"
)

// This file implements the MIA Wi-Fi input source: a UDP server speaking the
// "MIIN" protocol. One accepted client owns the Wi-Fi input session at a time.
// The protocol carries text, keyboard/consumer HID state, mouse deltas, and
// gamepad slots. It mirrors the Pico firmware (src/mia/input/input_wifi.c).

const (
	miaInputMagic0 = 'M'
	miaInputMagic1 = 'I'
	miaInputMagic2 = 'I'
	miaInputMagic3 = 'N'

	miaInputVersion      = 1
	miaInputHeaderSize   = 12
	miaInputRxPacketSize = 384
	miaInputWelcomeSize  = miaInputHeaderSize + 7
)

const (
	miaInputPacketHello        = 0x01
	miaInputPacketWelcome      = 0x02
	miaInputPacketDisconnect   = 0x04
	miaInputPacketText         = 0x10
	miaInputPacketHidEvent     = 0x11
	miaInputPacketHidBitmap    = 0x12
	miaInputPacketMouseDelta   = 0x20
	miaInputPacketGamepadState = 0x30
	miaInputPacketGamepadClear = 0x31
	miaInputPacketClearState   = 0x40
)

const (
	miaInputWelcomeAccepted           = 0x00
	miaInputWelcomeBusy               = 0x01
	miaInputWelcomeUnsupportedVersion = 0x02
)

const (
	miaInputCapText     uint16 = 1 << 0
	miaInputCapKeyboard uint16 = 1 << 1
	miaInputCapConsumer uint16 = 1 << 2
	miaInputCapMouse    uint16 = 1 << 3
	miaInputCapGamepad  uint16 = 1 << 4
	miaInputCapAll             = miaInputCapText | miaInputCapKeyboard | miaInputCapConsumer | miaInputCapMouse | miaInputCapGamepad
)

type miaInputOutgoing struct {
	addr *net.UDPAddr
	data []byte
}

// StartInputUDP starts the emulated MIA Wi-Fi input UDP service. Once running,
// the Wi-Fi input mode becomes available.
func (c *emulated_mia) StartInputUDP(bindAddress string) error {
	if bindAddress == "" {
		bindAddress = DefaultInputUDPAddress
	}

	addr, err := net.ResolveUDPAddr("udp4", bindAddress)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return err
	}

	c.mu.Lock()
	if c.input.conn != nil {
		c.mu.Unlock()
		conn.Close()
		return nil
	}
	c.input.conn = conn
	c.input.bindAddress = conn.LocalAddr().String()
	c.input.udpReady = true
	c.mu.Unlock()

	go c.inputReadLoop(conn)

	return nil
}

// InputUDPAddress returns the actual UDP address used by the input service.
func (c *emulated_mia) InputUDPAddress() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.input.bindAddress
}

func (c *emulated_mia) inputReadLoop(conn *net.UDPConn) {
	buf := make([]byte, miaInputRxPacketSize+64)
	for {
		n, remote, err := conn.ReadFromUDP(buf)
		if err != nil {
			return
		}

		if n > miaInputRxPacketSize {
			continue
		}

		packet := make([]byte, n)
		copy(packet, buf[:n])

		reply := c.inputHandleDatagram(packet, remote)
		if reply != nil {
			_, _ = conn.WriteToUDP(reply.data, reply.addr)
		}
	}
}

func (c *emulated_mia) inputHandleDatagram(packet []byte, remote *net.UDPAddr) *miaInputOutgoing {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(packet) < miaInputHeaderSize ||
		packet[0] != miaInputMagic0 ||
		packet[1] != miaInputMagic1 ||
		packet[2] != miaInputMagic2 ||
		packet[3] != miaInputMagic3 {
		return nil
	}

	version := packet[4]
	packetType := packet[5]
	seq := binary.LittleEndian.Uint16(packet[6:8])
	session := binary.LittleEndian.Uint32(packet[8:12])
	payload := packet[miaInputHeaderSize:]

	if packetType == miaInputPacketHello {
		return c.inputHandleHello(payload, seq, version, remote)
	}

	if version != miaInputVersion ||
		c.input.mode != miaInputModeWifi ||
		!c.inputAcceptsSessionPacket(session, seq, remote) {
		return nil
	}

	switch packetType {
	case miaInputPacketDisconnect:
		if len(payload) == 0 {
			c.inputInvalidateWifiSession(true)
		}
	case miaInputPacketText:
		c.inputHandleText(payload)
	case miaInputPacketHidEvent:
		c.inputHandleHidEvent(payload)
	case miaInputPacketHidBitmap:
		c.inputHandleHidBitmap(payload)
	case miaInputPacketMouseDelta:
		c.inputHandleMouseDelta(payload)
	case miaInputPacketGamepadState:
		c.inputHandleGamepadState(payload)
	case miaInputPacketGamepadClear:
		c.inputHandleGamepadClear(payload)
	case miaInputPacketClearState:
		c.inputHandleClearState(payload)
	}

	return nil
}

func (c *emulated_mia) inputHandleHello(payload []byte, seq uint16, version uint8, remote *net.UDPAddr) *miaInputOutgoing {
	if version != miaInputVersion {
		return c.inputBuildWelcome(miaInputWelcomeUnsupportedVersion, 0, remote)
	}

	if len(payload) < 3 {
		return nil
	}

	capabilities := binary.LittleEndian.Uint16(payload[0:2])
	nameLen := int(payload[2])
	if len(payload) != 3+nameLen {
		return nil
	}

	if c.input.mode != miaInputModeWifi {
		return c.inputBuildWelcome(miaInputWelcomeBusy, 0, remote)
	}

	c.inputAcceptWifiSession(seq, capabilities, remote)
	return c.inputBuildWelcome(miaInputWelcomeAccepted, c.input.wifiSession, remote)
}

// inputAcceptsSessionPacket validates a session-owned packet by session token,
// source address/port, and sequence freshness, advancing the last seen seq.
func (c *emulated_mia) inputAcceptsSessionPacket(session uint32, seq uint16, remote *net.UDPAddr) bool {
	if !c.input.wifiActive || session != c.input.wifiSession {
		return false
	}
	if c.input.wifiRemote == nil ||
		c.input.wifiRemote.Port != remote.Port ||
		!c.input.wifiRemote.IP.Equal(remote.IP) {
		return false
	}
	if !inputSeqIsNewer(seq, c.input.wifiLastSeq) {
		return false
	}

	c.input.wifiLastSeq = seq
	return true
}

// inputInvalidateWifiSession drops the active Wi-Fi session and clears the live
// state it owned. Queued text bytes remain in the FIFO.
func (c *emulated_mia) inputInvalidateWifiSession(publishEvents bool) {
	c.input.wifiActive = false
	c.input.wifiRemote = nil
	c.input.wifiSession = 0
	c.input.wifiLastSeq = 0
	c.input.wifiCaps = 0
	c.inputClearLiveState(publishEvents)
}

// inputAcceptWifiSession installs a new Wi-Fi session, replacing any existing
// one and publishing the client's capabilities as device flags.
func (c *emulated_mia) inputAcceptWifiSession(seq uint16, capabilities uint16, remote *net.UDPAddr) {
	replacing := c.input.wifiActive

	c.inputClearLiveState(replacing)
	c.input.wifiActive = true
	c.input.wifiRemote = &net.UDPAddr{IP: append(net.IP(nil), remote.IP...), Port: remote.Port, Zone: remote.Zone}
	c.input.wifiSession = c.inputNextSessionID(remote)
	c.input.wifiLastSeq = seq
	c.input.wifiCaps = capabilities
	c.inputApplyDeviceFlagsFromWifiCapabilities(capabilities)
	c.inputRecomputeStatus()
}

func (c *emulated_mia) inputNextSessionID(remote *net.UDPAddr) uint32 {
	c.input.nextSessionValue++
	if c.input.nextSessionValue == 0 {
		c.input.nextSessionValue = 1
	}

	id := (c.input.nextSessionValue << 16) ^
		(uint32(remote.Port) << 1) ^
		c.input.txSeq ^
		0xA5C31E7D
	if id == 0 {
		id = 1
	}
	return id
}

func (c *emulated_mia) inputNextTxSeq() uint16 {
	c.input.txSeq++
	if c.input.txSeq == 0 {
		c.input.txSeq = 1
	}
	return uint16(c.input.txSeq)
}

func (c *emulated_mia) inputBuildWelcome(status uint8, session uint32, remote *net.UDPAddr) *miaInputOutgoing {
	if c.input.conn == nil {
		return nil
	}

	packet := make([]byte, miaInputWelcomeSize)
	packet[0] = miaInputMagic0
	packet[1] = miaInputMagic1
	packet[2] = miaInputMagic2
	packet[3] = miaInputMagic3
	packet[4] = miaInputVersion
	packet[5] = miaInputPacketWelcome
	binary.LittleEndian.PutUint16(packet[6:8], c.inputNextTxSeq())
	binary.LittleEndian.PutUint32(packet[8:12], 0)
	packet[12] = status
	binary.LittleEndian.PutUint32(packet[13:17], session)
	binary.LittleEndian.PutUint16(packet[17:19], miaInputCapAll)

	return &miaInputOutgoing{addr: remote, data: packet}
}

// inputApplyDeviceFlagsFromWifiCapabilities publishes keyboard, consumer, and
// mouse availability from the client capabilities, keeping the gamepad slot
// flags intact.
func (c *emulated_mia) inputApplyDeviceFlagsFromWifiCapabilities(capabilities uint16) {
	var flags uint8
	if capabilities&miaInputCapKeyboard != 0 {
		flags |= miaInputDeviceKeyboard
	}
	if capabilities&miaInputCapConsumer != 0 {
		flags |= miaInputDeviceConsumer
	}
	if capabilities&miaInputCapMouse != 0 {
		flags |= miaInputDeviceMouse
	}

	flags |= c.memory[miaInputDeviceFlagsOffset] &
		(miaInputDeviceGamepad0 | miaInputDeviceGamepad1 | miaInputDeviceGamepad2 | miaInputDeviceGamepad3)
	c.inputSetDeviceFlags(flags)
}

func (c *emulated_mia) inputHandleText(payload []byte) {
	if len(payload) < 1 {
		return
	}

	count := int(payload[0])
	if len(payload) != 1+count {
		return
	}

	for i := 0; i < count; i++ {
		c.inputEnqueueText(payload[1+i])
	}
	c.inputRecomputeStatus()
}

func (c *emulated_mia) inputHandleHidEvent(payload []byte) {
	if len(payload) != 6 {
		return
	}

	usagePage := binary.LittleEndian.Uint16(payload[0:2])
	usageID := binary.LittleEndian.Uint16(payload[2:4])
	flags := payload[4]
	text := payload[5]
	down := flags&0x01 != 0

	c.inputSetHidUsage(usagePage, usageID, down)

	if usagePage == miaHidPageKeyboard && down && text != 0 {
		c.inputEnqueueText(text)
		c.inputRecomputeStatus()
	}
}

func (c *emulated_mia) inputHandleHidBitmap(payload []byte) {
	if len(payload) != 34 {
		return
	}

	c.inputSetHidBitmap(binary.LittleEndian.Uint16(payload[0:2]), payload[2:34])
}

func (c *emulated_mia) inputHandleMouseDelta(payload []byte) {
	if len(payload) != 5 {
		return
	}

	c.inputApplyMouseDelta(payload)
}

func (c *emulated_mia) inputHandleGamepadState(payload []byte) {
	if len(payload) != 11 || payload[0] >= miaInputGamepadSlots {
		return
	}

	c.inputApplyGamepadState(int(payload[0]), payload[1:11])
}

func (c *emulated_mia) inputHandleGamepadClear(payload []byte) {
	if len(payload) != 1 || payload[0] >= miaInputGamepadSlots {
		return
	}

	c.inputClearGamepadSlot(int(payload[0]), true)
	c.inputRecomputeStatus()
}

func (c *emulated_mia) inputHandleClearState(payload []byte) {
	if len(payload) != 1 {
		return
	}

	mask := payload[0]
	if mask&0x80 != 0 {
		mask |= 0x1F
	}

	if mask&0x01 != 0 {
		c.inputClearTextFifo()
	}
	if mask&0x02 != 0 {
		var events uint8
		if c.inputBitmapAny(miaInputKeyboardBitmapOffset) {
			events = miaKeyEventKeyUp
		}
		clear(c.memory[miaInputKeyboardBitmapOffset : miaInputKeyboardBitmapOffset+32])
		c.inputSetKeyboardEvents(events)
	}
	if mask&0x04 != 0 {
		var events uint8
		if c.inputBitmapAny(miaInputConsumerBitmapOffset) {
			events = miaKeyEventConsumerUp
		}
		clear(c.memory[miaInputConsumerBitmapOffset : miaInputConsumerBitmapOffset+32])
		c.inputSetKeyboardEvents(events)
	}
	if mask&0x08 != 0 {
		c.inputClearMouse(true)
	}
	if mask&0x10 != 0 {
		c.inputClearGamepads(true)
	}

	c.inputApplyDeviceFlagsFromWifiCapabilities(c.input.wifiCaps)
	c.inputRecomputeStatus()
}

// inputSeqIsNewer reports whether seq is newer than lastSeq using unsigned
// 16-bit wraparound comparison.
func inputSeqIsNewer(seq, lastSeq uint16) bool {
	delta := seq - lastSeq
	return delta != 0 && delta < 0x8000
}
