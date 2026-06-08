package mia

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmulatedMiaVideoEnableConfiguresIndexesAndDirtyTracking(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip
	chip.state = miaStateNormal

	circuit.write(miaRegCmdTrigger, 0x40)

	assert.Equal(t, uint8(miaVideoLayoutVersion), chip.memory[miaVideoLocalVersionOffset])
	assert.Len(t, chip.videoScanDirtyPages(chip.video.activeMap), miaVideoSyncPageCount)
	assert.Equal(t, miaIndex{
		currentAddr: 0x00024,
		defaultAddr: 0x00024,
		limitAddr:   0x00026,
		step:        1,
		flags:       (1 << miaIndexFlagReadStep) | (1 << miaIndexFlagWriteStep) | (1 << miaIndexFlagWrap),
	}, chip.indexes[0x83])
	assert.Equal(t, uint32(miaVideoOAMOffset+31*5), chip.indexes[0xDF].defaultAddr)
	assert.Equal(t, uint32(miaVideoOAMOffset+32*5), chip.indexes[0xDF].limitAddr)

	chip.videoClearDirtyMaps()
	circuit.write(miaRegIdxASelector, 0x83)
	circuit.write(miaRegIdxAPort, 0x34)
	circuit.write(miaRegIdxAPort, 0x12)

	assert.Equal(t, uint8(0x34), chip.memory[0x00024])
	assert.Equal(t, uint8(0x12), chip.memory[0x00025])
	assert.True(t, chip.videoPageDirty(chip.video.activeMap, 1))
	assert.False(t, chip.videoPageDirty(chip.video.activeMap, 0))
}

func TestEmulatedMiaVideoUDPFullRefreshAckAndEmptyRequest(t *testing.T) {
	chip := NewEmulatedMia().(*emulated_mia)
	require.NoError(t, chip.StartVideoUDP("127.0.0.1:0"))
	defer chip.Close()

	client, serverAddr := newMiaVideoUDPClient(t, chip.VideoUDPAddress())
	defer client.Close()

	chip.mu.Lock()
	chip.memory[miaVideoRenderControlOffset] = 0xAA
	chip.memory[miaVideoRenderControlOffset+1] = 0xBB
	chip.mu.Unlock()

	sendMiaVideoPacket(t, client, serverAddr, buildMiaVideoClientPacket(miaVideoPacketHello, 0, 1, 0, 0, nil))
	welcomeHeader, _ := readMiaVideoPacket(t, client)
	require.Equal(t, uint8(miaVideoPacketWelcome), welcomeHeader.packetType)
	require.NotZero(t, welcomeHeader.sessionID)
	assert.Equal(t, uint32(1), welcomeHeader.ack)

	requestPayload := make([]byte, 4)
	sendMiaVideoPacket(t, client, serverAddr, buildMiaVideoClientPacket(miaVideoPacketRequestFrame, welcomeHeader.sessionID, 2, 0, 0x1234, requestPayload))

	firstFrameHeader, firstFramePayload := readMiaVideoPacket(t, client)
	require.Equal(t, uint8(miaVideoPacketFrameData), firstFrameHeader.packetType)
	require.Equal(t, uint16(0x1234), firstFrameHeader.requestID)
	require.Equal(t, uint32(1), firstFrameHeader.frameID)
	require.Equal(t, uint16(154), firstFrameHeader.chunkCount)
	require.Equal(t, uint16(0), firstFrameHeader.chunkIndex)
	require.Equal(t, uint16(miaVideoRecordsPerChunk*miaVideoPageRecordSize), firstFrameHeader.payloadLen)
	assert.Equal(t, uint16(1), binary.LittleEndian.Uint16(firstFramePayload[0:2]))
	assert.Equal(t, []byte{0xAA, 0xBB}, firstFramePayload[2:4])

	for i := 1; i < int(firstFrameHeader.chunkCount); i++ {
		frameHeader, _ := readMiaVideoPacket(t, client)
		require.Equal(t, uint8(miaVideoPacketFrameData), frameHeader.packetType)
		assert.Equal(t, uint16(i), frameHeader.chunkIndex)
		assert.Equal(t, firstFrameHeader.frameID, frameHeader.frameID)
	}

	sendMiaVideoPacket(t, client, serverAddr, buildMiaVideoClientPacket(miaVideoPacketAckResponse, welcomeHeader.sessionID, 3, firstFrameHeader.frameID, 0x1234, nil))
	require.Eventually(t, func() bool {
		chip.mu.Lock()
		defer chip.mu.Unlock()
		return !chip.video.pendingValid &&
			chip.video.clientFrameID == firstFrameHeader.frameID &&
			chip.status()&(miaStatusVideoRequested|miaStatusVideoSent) == 0 &&
			chip.irqStatus()&miaIRQVideoAcked == miaIRQVideoAcked
	}, time.Second, time.Millisecond)

	chip.mu.Lock()
	chip.writeRegister(miaRegIRQStatusLSB, chip.readRegister(miaRegIRQStatusLSB)&^uint8(miaIRQVideoAcked))
	chip.irqEval()
	assert.Zero(t, chip.irqStatus()&miaIRQVideoAcked)
	chip.mu.Unlock()

	binary.LittleEndian.PutUint32(requestPayload, firstFrameHeader.frameID)
	sendMiaVideoPacket(t, client, serverAddr, buildMiaVideoClientPacket(miaVideoPacketRequestFrame, welcomeHeader.sessionID, 4, 0, 0x1235, requestPayload))
	statusHeader, statusPayload := readMiaVideoPacket(t, client)
	require.Equal(t, uint8(miaVideoPacketStatus), statusHeader.packetType)
	assert.Equal(t, uint16(0x1235), statusHeader.requestID)
	assert.Equal(t, uint16(miaVideoStatusNoDirtyPages), binary.LittleEndian.Uint16(statusPayload[0:2]))
}

func newMiaVideoUDPClient(t *testing.T, serverAddress string) (*net.UDPConn, *net.UDPAddr) {
	t.Helper()

	client, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	require.NoError(t, err)

	serverAddr, err := net.ResolveUDPAddr("udp4", serverAddress)
	require.NoError(t, err)

	return client, serverAddr
}

func sendMiaVideoPacket(t *testing.T, client *net.UDPConn, serverAddr *net.UDPAddr, packet []byte) {
	t.Helper()

	_, err := client.WriteToUDP(packet, serverAddr)
	require.NoError(t, err)
}

func readMiaVideoPacket(t *testing.T, client *net.UDPConn) (miaVideoHeader, []byte) {
	t.Helper()

	require.NoError(t, client.SetReadDeadline(time.Now().Add(time.Second)))
	buf := make([]byte, miaVideoUDPPayloadSize)
	n, _, err := client.ReadFromUDP(buf)
	require.NoError(t, err)

	header, payload, ok := parseMiaVideoTestPacket(buf[:n])
	require.True(t, ok)

	return header, payload
}

func buildMiaVideoClientPacket(packetType uint8, sessionID uint32, seq uint32, frameID uint32, requestID uint16, payload []byte) []byte {
	packet := make([]byte, miaVideoHeaderSize, miaVideoHeaderSize+len(payload))
	binary.LittleEndian.PutUint16(packet[0:2], miaVideoMagic)
	packet[2] = miaVideoVersion
	packet[3] = packetType
	binary.LittleEndian.PutUint32(packet[4:8], sessionID)
	binary.LittleEndian.PutUint32(packet[8:12], seq)
	binary.LittleEndian.PutUint32(packet[16:20], frameID)
	binary.LittleEndian.PutUint16(packet[20:22], requestID)
	binary.LittleEndian.PutUint16(packet[26:28], uint16(len(payload)))
	packet = append(packet, payload...)

	return packet
}

func parseMiaVideoTestPacket(packet []byte) (miaVideoHeader, []byte, bool) {
	if len(packet) < miaVideoHeaderSize {
		return miaVideoHeader{}, nil, false
	}

	header := miaVideoHeader{
		packetType: packet[3],
		sessionID:  binary.LittleEndian.Uint32(packet[4:8]),
		seq:        binary.LittleEndian.Uint32(packet[8:12]),
		ack:        binary.LittleEndian.Uint32(packet[12:16]),
		frameID:    binary.LittleEndian.Uint32(packet[16:20]),
		requestID:  binary.LittleEndian.Uint16(packet[20:22]),
		chunkIndex: binary.LittleEndian.Uint16(packet[22:24]),
		chunkCount: binary.LittleEndian.Uint16(packet[24:26]),
		payloadLen: binary.LittleEndian.Uint16(packet[26:28]),
	}

	return header, packet[miaVideoHeaderSize:], int(header.payloadLen) == len(packet)-miaVideoHeaderSize
}
