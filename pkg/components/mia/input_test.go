package mia

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- 6502-facing register interface -----------------------------------------

func TestEmulatedMiaInputTextFifoConsoleSource(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip
	chip.state = miaStateNormal

	chip.mu.Lock()
	chip.inputConsoleByte('H')
	chip.inputConsoleByte('I')
	chip.mu.Unlock()

	assert.Equal(t, uint8(2), circuit.read(miaRegInputCharCount))
	status := circuit.read(miaRegInputStatus)
	assert.Equal(t, miaInputTextReady, status&miaInputTextReady)
	assert.Equal(t, miaInputSourceConsole, status&miaInputSourceConsole)

	assert.Equal(t, uint8('H'), circuit.read(miaRegInputChar))
	assert.Equal(t, uint8(1), circuit.read(miaRegInputCharCount))
	assert.Equal(t, uint8('I'), circuit.read(miaRegInputChar))
	assert.Equal(t, uint8(0), circuit.read(miaRegInputCharCount))
	assert.Equal(t, uint8(0), circuit.read(miaRegInputChar))
	assert.Zero(t, circuit.read(miaRegInputStatus)&miaInputTextReady)
}

func TestEmulatedMiaInputTextFifoOverflowDropsOldest(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip
	chip.state = miaStateNormal

	chip.mu.Lock()
	for i := 0; i < miaInputTextFifoSize+1; i++ {
		chip.inputConsoleByte(uint8('A' + i%26))
	}
	chip.mu.Unlock()

	assert.Equal(t, uint8(miaInputTextFifoSize), circuit.read(miaRegInputCharCount))
	// The first byte ('A') was dropped, so the front of the FIFO is now 'B'.
	assert.Equal(t, uint8('B'), circuit.read(miaRegInputChar))
}

func TestEmulatedMiaInputRegistersAreReadOnlyInNormalMode(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip
	chip.state = miaStateNormal

	chip.mu.Lock()
	chip.inputConsoleByte('Z')
	chip.mu.Unlock()

	circuit.write(miaRegInputCharCount, 0x00)
	circuit.write(miaRegInputStatus, 0x00)

	assert.Equal(t, uint8(1), circuit.read(miaRegInputCharCount))
	assert.Equal(t, miaInputTextReady, circuit.read(miaRegInputStatus)&miaInputTextReady)
}

func TestEmulatedMiaInputHidEventDecodesEditingKeys(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip
	chip.state = miaStateNormal

	keyDown := func(usage uint16, text uint8) {
		hid := make([]byte, 6)
		binary.LittleEndian.PutUint16(hid[0:2], miaHidPageKeyboard)
		binary.LittleEndian.PutUint16(hid[2:4], usage)
		hid[4] = 0x01 // down
		hid[5] = text
		chip.mu.Lock()
		chip.inputHandleHidEvent(hid)
		chip.mu.Unlock()
	}

	// Arrow keys carry no attached text; MIA decodes them to PETSCII cursor bytes.
	keyDown(0x4F, 0) // Right
	keyDown(0x50, 0) // Left
	keyDown(0x51, 0) // Down
	keyDown(0x52, 0) // Up

	assert.Equal(t, uint8(4), circuit.read(miaRegInputCharCount))
	assert.Equal(t, uint8(0x1D), circuit.read(miaRegInputChar)) // Right
	assert.Equal(t, uint8(0x9D), circuit.read(miaRegInputChar)) // Left
	assert.Equal(t, uint8(0x11), circuit.read(miaRegInputChar)) // Down
	assert.Equal(t, uint8(0x91), circuit.read(miaRegInputChar)) // Up

	// A printable key's text arrives via a TEXT packet, so its HID event carries
	// text 0 and has no decode entry: it must not enqueue anything.
	keyDown(0x04, 0) // 'a' down, no attached text
	assert.Equal(t, uint8(0), circuit.read(miaRegInputCharCount))

	// A key that already carries attached text uses it verbatim (table is skipped).
	keyDown(0x28, 0x0D) // Enter with attached CR
	assert.Equal(t, uint8(0x0D), circuit.read(miaRegInputChar))
}

func TestEmulatedMiaInputAutoRepeatsHeldArrow(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip
	chip.state = miaStateNormal

	const ms = int64(1_000_000)

	// Drive the chip directly with a deterministic clock. (circuit.read/write
	// tick the chip with the circuit's own StepContext time, which would race the
	// repeat clock this test controls.)
	setKey := func(usage uint16, down bool, nowNs int64) {
		hid := make([]byte, 6)
		binary.LittleEndian.PutUint16(hid[0:2], miaHidPageKeyboard)
		binary.LittleEndian.PutUint16(hid[2:4], usage)
		if down {
			hid[4] = 0x01
		}
		chip.mu.Lock()
		chip.nowNs = nowNs
		chip.inputHandleHidEvent(hid)
		chip.mu.Unlock()
	}
	serviceAt := func(nowNs int64) {
		chip.mu.Lock()
		chip.nowNs = nowNs
		chip.inputService()
		chip.mu.Unlock()
	}
	count := func() uint8 {
		chip.mu.Lock()
		defer chip.mu.Unlock()
		return chip.registers[miaRegInputCharCount]
	}
	frontChar := func() uint8 {
		chip.mu.Lock()
		defer chip.mu.Unlock()
		return chip.registers[miaRegInputChar]
	}
	pop := func() {
		chip.mu.Lock()
		chip.inputOnCharRead()
		chip.mu.Unlock()
	}

	// Down arrow pressed at t=0: one $11 is enqueued and repeat is armed.
	setKey(0x51, true, 0)
	assert.Equal(t, uint8(1), count())
	assert.Equal(t, uint8(0x11), frontChar())
	pop()
	assert.Equal(t, uint8(0), count())

	serviceAt(399 * ms) // before the initial delay: nothing repeats
	assert.Equal(t, uint8(0), count())

	serviceAt(400 * ms) // initial delay elapsed: first repeat
	assert.Equal(t, uint8(1), count())
	assert.Equal(t, uint8(0x11), frontChar())
	pop()

	serviceAt(459 * ms) // before the next interval: nothing
	assert.Equal(t, uint8(0), count())

	serviceAt(460 * ms) // 400 + 60: second repeat
	assert.Equal(t, uint8(1), count())
	pop()

	// Release stops the repeat; later services enqueue nothing.
	setKey(0x51, false, 1000*ms)
	serviceAt(2000 * ms)
	assert.Equal(t, uint8(0), count())
}

// --- Indexes and commands ---------------------------------------------------

func TestEmulatedMiaInputConfiguresIndexes(t *testing.T) {
	chip := newEmulatedMiaTestCircuit().chip

	streamFlags := uint8((1 << miaIndexFlagReadStep) | (1 << miaIndexFlagWriteStep) | (1 << miaIndexFlagWrap))
	probeFlags := uint8(1 << miaIndexFlagWrap)

	assert.Equal(t, miaIndex{
		currentAddr: miaInputKeyboardBitmapOffset,
		defaultAddr: miaInputKeyboardBitmapOffset,
		limitAddr:   miaInputKeyboardBitmapOffset + 1,
		step:        1,
		flags:       probeFlags,
	}, chip.indexes[0x50])

	assert.Equal(t, miaIndex{
		currentAddr: miaInputStateOffset,
		defaultAddr: miaInputStateOffset,
		limitAddr:   miaInputStateOffset + miaInputStateSize,
		step:        1,
		flags:       streamFlags,
	}, chip.indexes[0x60])

	assert.Equal(t, uint32(miaInputGamepadOffset+3*miaInputGamepadSlotSize), chip.indexes[0x67].currentAddr)
	assert.Equal(t, uint32(miaInputControlOffset), chip.indexes[0x68].currentAddr)
	assert.Equal(t, uint32(miaInputControlOffset+11), chip.indexes[0x68].limitAddr)
}

func TestEmulatedMiaInputSetProbeCommand(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip
	chip.state = miaStateNormal

	circuit.write(miaRegCmdParam1, 0x03) // keyboard probe 3 -> index 0x53
	circuit.write(miaRegCmdParam2, 0x08) // byte offset 8
	circuit.write(miaRegCmdParam3, 0x00)
	circuit.write(miaRegCmdTrigger, 0x51)

	assert.Equal(t, uint32(miaInputKeyboardBitmapOffset+8), chip.indexes[0x53].currentAddr)
	assert.Equal(t, uint32(miaInputKeyboardBitmapOffset+8), chip.indexes[0x53].defaultAddr)
}

func TestEmulatedMiaInputSetModeCommandIgnoresUnavailableMode(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip
	chip.state = miaStateNormal

	// USB host and Wi-Fi are unavailable without a UDP listener; the mode stays
	// console and the source flag is unchanged.
	circuit.write(miaRegCmdParam1, uint8(miaInputModeUSBHost))
	circuit.write(miaRegCmdTrigger, 0x50)

	chip.mu.Lock()
	mode := chip.input.mode
	chip.mu.Unlock()

	assert.Equal(t, miaInputModeConsole, mode)
	assert.Equal(t, miaInputSourceConsole, circuit.read(miaRegInputStatus)&0xE0)
}

// --- Event and IRQ model ----------------------------------------------------

func TestEmulatedMiaInputKeyboardEventRaisesAndClearsIRQ(t *testing.T) {
	chip := newEmulatedMiaTestCircuit().chip
	chip.state = miaStateNormal

	chip.mu.Lock()
	defer chip.mu.Unlock()

	chip.setIRQStatus(0)
	chip.irqEval()
	chip.writeRegisterWord(miaRegIRQMaskLSB, miaIRQInputKeyboard)
	chip.memory[miaKeyboardEventMaskOffset] = miaKeyEventText

	chip.inputConsoleByte('X')

	assert.Equal(t, miaKeyEventText, chip.memory[miaKeyboardEventFlagsOffset]&miaKeyEventText)
	assert.Equal(t, miaIRQInputKeyboard, chip.irqStatus()&miaIRQInputKeyboard)
	assert.NotZero(t, chip.irqStatus()&miaIRQTriggered)
	assert.True(t, chip.irqAsserted)

	// Acknowledge the event flag, then clear the top-level status; the service
	// pass must not re-raise the IRQ.
	chip.memory[miaKeyboardEventAckOffset] = miaKeyEventText
	chip.inputService()
	chip.irqClearStatus()
	chip.inputService()

	assert.Zero(t, chip.memory[miaKeyboardEventFlagsOffset]&miaKeyEventText)
	assert.Zero(t, chip.irqStatus()&miaIRQInputKeyboard)
	assert.Zero(t, chip.memory[miaKeyboardEventAckOffset])
}

func TestEmulatedMiaInputModeSwitchClearsLiveState(t *testing.T) {
	chip := newEmulatedMiaTestCircuit().chip
	require.NoError(t, chip.StartInputUDP("127.0.0.1:0"))
	defer chip.Close()
	chip.state = miaStateNormal

	chip.mu.Lock()
	defer chip.mu.Unlock()

	require.True(t, chip.inputSetMode(miaInputModeWifi))
	chip.memory[miaInputKeyboardBitmapOffset] = 0x10
	chip.memory[miaInputMouseStateOffset] = 0x01
	chip.inputRecomputeStatus()
	require.Equal(t, miaInputKeyboardDown, chip.registers[miaRegInputStatus]&miaInputKeyboardDown)

	// Switching back to console clears the held keyboard and mouse state.
	require.True(t, chip.inputSetMode(miaInputModeConsole))
	assert.Zero(t, chip.memory[miaInputKeyboardBitmapOffset])
	assert.Zero(t, chip.memory[miaInputMouseStateOffset])
	assert.Equal(t, miaInputSourceConsole, chip.registers[miaRegInputStatus]&0xE0)
}

// --- Wi-Fi UDP protocol -----------------------------------------------------

func TestEmulatedMiaInputWifiSession(t *testing.T) {
	chip := NewEmulatedMia().(*emulated_mia)
	require.NoError(t, chip.StartInputUDP("127.0.0.1:0"))
	defer chip.Close()
	chip.state = miaStateNormal

	chip.mu.Lock()
	require.True(t, chip.inputSetMode(miaInputModeWifi))
	chip.mu.Unlock()

	client, serverAddr := newMiaInputUDPClient(t, chip.InputUDPAddress())
	defer client.Close()

	sendMiaInputPacket(t, client, serverAddr, buildMiaInputHelloPacket(1, miaInputCapAll, "tester"))
	welcome := readMiaInputPacket(t, client)
	require.Equal(t, uint8(miaInputPacketWelcome), welcome.packetType)
	require.Equal(t, uint8(miaInputWelcomeAccepted), welcome.status)
	require.NotZero(t, welcome.payloadSession)

	session := welcome.payloadSession

	// TEXT "AB"
	sendMiaInputPacket(t, client, serverAddr, buildMiaInputPacket(miaInputPacketText, session, 2, []byte{2, 'A', 'B'}))
	requireInputState(t, chip, func() bool {
		return chip.registers[miaRegInputCharCount] == 2
	})

	// HID_EVENT: keyboard usage 0x04 (A) pressed, queues text 'a'.
	hid := make([]byte, 6)
	binary.LittleEndian.PutUint16(hid[0:2], miaHidPageKeyboard)
	binary.LittleEndian.PutUint16(hid[2:4], 0x04)
	hid[4] = 0x01
	hid[5] = 'a'
	sendMiaInputPacket(t, client, serverAddr, buildMiaInputPacket(miaInputPacketHidEvent, session, 3, hid))
	requireInputState(t, chip, func() bool {
		return chip.memory[miaInputKeyboardBitmapOffset]&(1<<4) != 0 &&
			chip.registers[miaRegInputCharCount] == 3
	})

	// MOUSE_DELTA: left button held, dx=5, dy=-3 (0xFD two's complement).
	mouse := []byte{0x01, 5, 0xFD, 0, 0}
	sendMiaInputPacket(t, client, serverAddr, buildMiaInputPacket(miaInputPacketMouseDelta, session, 4, mouse))
	requireInputState(t, chip, func() bool {
		return chip.memory[miaInputMouseStateOffset] == 0x01 &&
			chip.memory[miaInputMouseStateOffset+1] == 5 &&
			chip.memory[miaInputMouseStateOffset+2] == 253
	})

	// GAMEPAD_STATE: player 0 connected with button0 bit 0 set.
	gp := make([]byte, 11)
	gp[1] = 0x80 // dpad: connected
	gp[3] = 0x01 // button0
	sendMiaInputPacket(t, client, serverAddr, buildMiaInputPacket(miaInputPacketGamepadState, session, 5, gp))
	requireInputState(t, chip, func() bool {
		return chip.memory[miaInputDeviceFlagsOffset]&miaInputDeviceGamepad0 != 0 &&
			chip.memory[miaInputGamepadOffset+2] == 0x01 &&
			chip.registers[miaRegInputStatus]&miaInputGamepadDown != 0
	})

	// DISCONNECT releases the session but keeps queued text.
	sendMiaInputPacket(t, client, serverAddr, buildMiaInputPacket(miaInputPacketDisconnect, session, 6, nil))
	requireInputState(t, chip, func() bool {
		return !chip.input.wifiActive &&
			chip.memory[miaInputGamepadOffset+2] == 0x00 &&
			chip.registers[miaRegInputCharCount] == 3
	})
}

func TestEmulatedMiaInputWifiHelloBusyWhenNotWifiMode(t *testing.T) {
	chip := NewEmulatedMia().(*emulated_mia)
	require.NoError(t, chip.StartInputUDP("127.0.0.1:0"))
	defer chip.Close()

	client, serverAddr := newMiaInputUDPClient(t, chip.InputUDPAddress())
	defer client.Close()

	sendMiaInputPacket(t, client, serverAddr, buildMiaInputHelloPacket(1, miaInputCapAll, ""))
	welcome := readMiaInputPacket(t, client)
	assert.Equal(t, uint8(miaInputPacketWelcome), welcome.packetType)
	assert.Equal(t, uint8(miaInputWelcomeBusy), welcome.status)
	assert.Zero(t, welcome.payloadSession)
}

func TestEmulatedMiaInputWifiHelloRejectsBadVersion(t *testing.T) {
	chip := NewEmulatedMia().(*emulated_mia)
	require.NoError(t, chip.StartInputUDP("127.0.0.1:0"))
	defer chip.Close()

	client, serverAddr := newMiaInputUDPClient(t, chip.InputUDPAddress())
	defer client.Close()

	packet := buildMiaInputHelloPacket(1, miaInputCapAll, "")
	packet[4] = 0xFF // unsupported version

	sendMiaInputPacket(t, client, serverAddr, packet)
	welcome := readMiaInputPacket(t, client)
	assert.Equal(t, uint8(miaInputPacketWelcome), welcome.packetType)
	assert.Equal(t, uint8(miaInputWelcomeUnsupportedVersion), welcome.status)
}

// --- Console integration ----------------------------------------------------

func TestEmulatedMiaConsoleInputCaptureAndStatus(t *testing.T) {
	chip, mock := newMiaConsoleTest(t)

	waitForMiaConsoleOutput(t, mock, "> ")
	sendMiaConsoleInput(mock, "input wifi\n")
	waitForMiaConsoleOutput(t, mock, "Input: Wi-Fi mode is not available.\n")

	sendMiaConsoleInput(mock, "input console\n")
	waitForMiaConsoleOutput(t, mock, "Console input active. Press Ctrl+Q to return to commands.\n")

	sendMiaConsoleInput(mock, "Hi")
	require.Eventually(t, func() bool {
		chip.mu.Lock()
		defer chip.mu.Unlock()
		return chip.registers[miaRegInputCharCount] == 2
	}, time.Second, 10*time.Millisecond)

	sendMiaConsoleInput(mock, string([]byte{miaConsoleCtrlQ}))
	waitForMiaConsoleOutput(t, mock, "Console input ended.\n")

	sendMiaConsoleInput(mock, "input status\n")
	waitForMiaConsoleOutput(t, mock, "Input: console")
}

// --- helpers ----------------------------------------------------------------

type miaInputTestPacket struct {
	packetType     uint8
	seq            uint16
	session        uint32
	status         uint8
	payloadSession uint32
}

func newMiaInputUDPClient(t *testing.T, serverAddress string) (*net.UDPConn, *net.UDPAddr) {
	t.Helper()

	client, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	require.NoError(t, err)

	serverAddr, err := net.ResolveUDPAddr("udp4", serverAddress)
	require.NoError(t, err)

	return client, serverAddr
}

func sendMiaInputPacket(t *testing.T, client *net.UDPConn, serverAddr *net.UDPAddr, packet []byte) {
	t.Helper()

	_, err := client.WriteToUDP(packet, serverAddr)
	require.NoError(t, err)
}

func readMiaInputPacket(t *testing.T, client *net.UDPConn) miaInputTestPacket {
	t.Helper()

	require.NoError(t, client.SetReadDeadline(time.Now().Add(time.Second)))
	buf := make([]byte, 64)
	n, _, err := client.ReadFromUDP(buf)
	require.NoError(t, err)
	require.GreaterOrEqual(t, n, miaInputHeaderSize)

	packet := miaInputTestPacket{
		packetType: buf[5],
		seq:        binary.LittleEndian.Uint16(buf[6:8]),
		session:    binary.LittleEndian.Uint32(buf[8:12]),
	}

	if packet.packetType == miaInputPacketWelcome && n >= miaInputHeaderSize+7 {
		packet.status = buf[12]
		packet.payloadSession = binary.LittleEndian.Uint32(buf[13:17])
	}

	return packet
}

func buildMiaInputPacket(packetType uint8, session uint32, seq uint16, payload []byte) []byte {
	packet := make([]byte, miaInputHeaderSize+len(payload))
	packet[0] = miaInputMagic0
	packet[1] = miaInputMagic1
	packet[2] = miaInputMagic2
	packet[3] = miaInputMagic3
	packet[4] = miaInputVersion
	packet[5] = packetType
	binary.LittleEndian.PutUint16(packet[6:8], seq)
	binary.LittleEndian.PutUint32(packet[8:12], session)
	copy(packet[miaInputHeaderSize:], payload)

	return packet
}

func buildMiaInputHelloPacket(seq uint16, capabilities uint16, name string) []byte {
	payload := make([]byte, 3+len(name))
	binary.LittleEndian.PutUint16(payload[0:2], capabilities)
	payload[2] = uint8(len(name))
	copy(payload[3:], name)

	return buildMiaInputPacket(miaInputPacketHello, 0, seq, payload)
}

func requireInputState(t *testing.T, chip *emulated_mia, condition func() bool) {
	t.Helper()

	require.Eventually(t, func() bool {
		chip.mu.Lock()
		defer chip.mu.Unlock()
		return condition()
	}, time.Second, time.Millisecond)
}
