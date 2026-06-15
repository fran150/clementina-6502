package mia

import (
	"encoding/binary"
	"net"
	"time"

	"github.com/fran150/clementina-6502/assets"
)

const (
	DefaultVideoUDPAddress = "127.0.0.1:6502"

	miaVideoStateSize       = 68944
	miaVideoPageSize        = 32
	miaVideoPageShift       = 5
	miaVideoPageCount       = 2155
	miaVideoFirstSyncPage   = 1
	miaVideoSyncPageCount   = miaVideoPageCount - miaVideoFirstSyncPage
	miaVideoDirtyMapSize    = 270
	miaVideoHeaderSize      = 32
	miaVideoPageRecordSize  = 34
	miaVideoUDPPayloadSize  = 512
	miaVideoRecordsPerChunk = (miaVideoUDPPayloadSize - miaVideoHeaderSize) / miaVideoPageRecordSize
	miaVideoLayoutVersion   = 1
	miaPETSCIIPlaneSize     = 2048

	miaVideoLocalVersionOffset    = 0x00000
	miaVideoLocalFrameIDOffset    = 0x00004
	miaVideoLocalDirtyPagesOffset = 0x00008
	miaVideoRenderControlOffset   = 0x00020
	miaVideoModeOffset            = 0x00020
	miaVideoPaletteOffset         = 0x00100
	miaVideoCHROffset             = 0x00200
	miaVideoBGNTOffset            = 0x0C200
	miaVideoBGAttrOffset          = 0x0E140
	miaVideoOverlayNTOffset       = 0x10080
	miaVideoOverlayAttrOffset     = 0x10468
	miaVideoOAMOffset             = 0x10850

	miaVideoMagic   = 0x4D56
	miaVideoVersion = 1

	miaVideoPacketHello        = 0x01
	miaVideoPacketWelcome      = 0x02
	miaVideoPacketRequestFrame = 0x05
	miaVideoPacketAckResponse  = 0x06
	miaVideoPacketNackChunks   = 0x07
	miaVideoPacketFrameData    = 0x20
	miaVideoPacketStatus       = 0x30

	miaVideoStatusNoDirtyPages = 0
	miaVideoStatusProtocolErr  = 1
)

type miaVideoState struct {
	conn        *net.UDPConn
	bindAddress string

	dirtyMaps [2][miaVideoDirtyMapSize]uint8
	activeMap uint8

	sessionActive bool
	remote        *net.UDPAddr
	sessionID     uint32
	nextSeq       uint32
	lastPeerSeq   uint32
	frameID       uint32
	clientFrameID uint32

	pendingValid        bool
	pendingMap          uint8
	pendingRequestID    uint16
	pendingLastComplete uint32
	pendingFrameID      uint32
	pendingPages        []uint16
	pendingChunkCount   uint16
	pendingInitialSent  bool
}

type miaVideoHeader struct {
	packetType uint8
	sessionID  uint32
	seq        uint32
	ack        uint32
	frameID    uint32
	requestID  uint16
	chunkIndex uint16
	chunkCount uint16
	payloadLen uint16
}

type miaVideoOutgoing struct {
	addr *net.UDPAddr
	data []byte
}

type miaVideoFrameJob struct {
	addr        *net.UDPAddr
	requestID   uint16
	frameID     uint32
	chunks      []uint16
	initialSend bool
}

type miaVideoDatagramPlan struct {
	packets []miaVideoOutgoing
	frame   *miaVideoFrameJob
}

// StartVideoUDP starts the emulated MIA video UDP service.
func (c *emulated_mia) StartVideoUDP(bindAddress string) error {
	if bindAddress == "" {
		bindAddress = DefaultVideoUDPAddress
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
	if c.video.conn != nil {
		c.mu.Unlock()
		conn.Close()
		return nil
	}
	c.video.conn = conn
	c.video.bindAddress = conn.LocalAddr().String()
	c.mu.Unlock()

	go c.videoReadLoop(conn)

	return nil
}

// VideoUDPAddress returns the actual UDP address used by the video service.
func (c *emulated_mia) VideoUDPAddress() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.video.bindAddress
}

// videoClose tears down the video UDP service. It is part of the chip Close path.
func (c *emulated_mia) videoClose() {
	c.mu.Lock()
	conn := c.video.conn
	c.video.conn = nil
	c.video.bindAddress = ""
	c.mu.Unlock()

	if conn != nil {
		conn.Close()
	}
}

func (c *emulated_mia) videoReadLoop(conn *net.UDPConn) {
	buf := make([]byte, miaVideoUDPPayloadSize)
	for {
		n, remote, err := conn.ReadFromUDP(buf)
		if err != nil {
			return
		}

		packet := make([]byte, n)
		copy(packet, buf[:n])

		plan := c.videoHandleDatagram(packet, remote)
		c.videoSendPlan(conn, plan)
	}
}

func (c *emulated_mia) videoSendPlan(conn *net.UDPConn, plan miaVideoDatagramPlan) {
	for _, packet := range plan.packets {
		_, _ = conn.WriteToUDP(packet.data, packet.addr)
	}

	if plan.frame == nil {
		return
	}

	for _, chunk := range plan.frame.chunks {
		packet := c.videoBuildFrameChunk(plan.frame.requestID, plan.frame.frameID, chunk)
		if packet == nil {
			continue
		}
		_, _ = conn.WriteToUDP(packet, plan.frame.addr)
	}

	if plan.frame.initialSend {
		c.videoMarkInitialSendComplete(plan.frame.requestID, plan.frame.frameID)
	}
}

func (c *emulated_mia) videoHandleDatagram(packet []byte, remote *net.UDPAddr) miaVideoDatagramPlan {
	c.mu.Lock()
	defer c.mu.Unlock()

	header, payload, ok := c.videoParseHeader(packet)
	if !ok {
		return c.videoProtocolErrorForMalformed(packet, remote)
	}

	if header.packetType == miaVideoPacketHello {
		if !c.videoValidateHello(header) {
			return c.videoProtocolErrorPlan(remote, header.requestID, header.frameID)
		}

		c.video.lastPeerSeq = header.seq
		c.videoResetSession(remote)
		return miaVideoDatagramPlan{
			packets: []miaVideoOutgoing{{
				addr: remote,
				data: c.videoBuildHeaderLocked(miaVideoPacketWelcome, c.video.sessionID, 0, 0, 0, 0, 0),
			}},
		}
	}

	if !c.videoAcceptsSessionPacket(header, remote) {
		return miaVideoDatagramPlan{}
	}

	c.video.lastPeerSeq = header.seq

	switch header.packetType {
	case miaVideoPacketRequestFrame:
		return c.videoHandleRequestFrame(header, payload, remote)
	case miaVideoPacketAckResponse:
		return c.videoHandleAckResponse(header, remote)
	case miaVideoPacketNackChunks:
		return c.videoHandleNackChunks(header, payload, remote)
	case miaVideoPacketStatus:
		return c.videoHandleClientStatus(header, payload, remote)
	default:
		return c.videoProtocolErrorPlan(remote, header.requestID, header.frameID)
	}
}

func (c *emulated_mia) videoParseHeader(packet []byte) (miaVideoHeader, []byte, bool) {
	if len(packet) < miaVideoHeaderSize {
		return miaVideoHeader{}, nil, false
	}

	if binary.LittleEndian.Uint16(packet[0:2]) != miaVideoMagic || packet[2] != miaVideoVersion {
		return miaVideoHeader{}, nil, false
	}

	payloadLen := binary.LittleEndian.Uint16(packet[26:28])
	if int(payloadLen) != len(packet)-miaVideoHeaderSize {
		return miaVideoHeader{}, nil, false
	}

	if binary.LittleEndian.Uint16(packet[28:30]) != 0 || binary.LittleEndian.Uint16(packet[30:32]) != 0 {
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
		payloadLen: payloadLen,
	}

	return header, packet[miaVideoHeaderSize:], true
}

func (c *emulated_mia) videoValidateHello(header miaVideoHeader) bool {
	return header.payloadLen == 0 &&
		header.sessionID == 0 &&
		header.frameID == 0 &&
		header.requestID == 0 &&
		header.chunkIndex == 0 &&
		header.chunkCount == 0
}

func (c *emulated_mia) videoAcceptsSessionPacket(header miaVideoHeader, remote *net.UDPAddr) bool {
	if !c.video.sessionActive || header.sessionID != c.video.sessionID {
		return false
	}

	if c.video.remote == nil {
		return true
	}

	return c.video.remote.Port == remote.Port && c.video.remote.IP.Equal(remote.IP)
}

func (c *emulated_mia) videoHandleRequestFrame(header miaVideoHeader, payload []byte, remote *net.UDPAddr) miaVideoDatagramPlan {
	if header.payloadLen != 4 || header.frameID != 0 || header.chunkIndex != 0 || header.chunkCount != 0 {
		return c.videoProtocolErrorPlan(remote, header.requestID, header.frameID)
	}

	lastComplete := binary.LittleEndian.Uint32(payload[0:4])

	if c.video.pendingValid {
		if header.requestID == c.video.pendingRequestID && lastComplete == c.video.pendingLastComplete {
			return c.videoResendPendingPlan(remote, true)
		}

		if lastComplete == c.video.pendingFrameID {
			c.videoReleasePendingResponse(true)
			return c.videoHandleRequestFrameNoPending(header.requestID, lastComplete, remote)
		}

		if lastComplete > c.video.pendingFrameID {
			return c.videoProtocolErrorPlan(remote, header.requestID, header.frameID)
		}

		return miaVideoDatagramPlan{}
	}

	if lastComplete < c.video.clientFrameID {
		return miaVideoDatagramPlan{}
	}

	if lastComplete > c.video.clientFrameID {
		return c.videoProtocolErrorPlan(remote, header.requestID, header.frameID)
	}

	return c.videoHandleRequestFrameNoPending(header.requestID, lastComplete, remote)
}

func (c *emulated_mia) videoHandleRequestFrameNoPending(requestID uint16, lastComplete uint32, remote *net.UDPAddr) miaVideoDatagramPlan {
	if !c.videoHasDirtyPages(c.video.activeMap) {
		c.videoSetLastResponseDirtyPages(0)
		return miaVideoDatagramPlan{
			packets: []miaVideoOutgoing{{
				addr: remote,
				data: c.videoBuildStatusLocked(miaVideoStatusNoDirtyPages, requestID, 0),
			}},
		}
	}

	pendingMap := c.video.activeMap
	c.video.activeMap ^= 1
	c.video.pendingMap = pendingMap
	c.video.pendingPages = c.videoScanDirtyPages(pendingMap)
	c.video.pendingChunkCount = uint16((len(c.video.pendingPages) + miaVideoRecordsPerChunk - 1) / miaVideoRecordsPerChunk)
	c.video.pendingFrameID = c.videoNextFrameID()
	c.video.pendingRequestID = requestID
	c.video.pendingLastComplete = lastComplete
	c.video.pendingInitialSent = false
	c.video.pendingValid = true
	c.videoSetFrameID(c.video.pendingFrameID)
	c.statusSet(miaStatusVideoRequested)
	c.irqSetFlag(miaIRQVideoRequest)

	return c.videoResendPendingPlan(remote, true)
}

func (c *emulated_mia) videoHandleAckResponse(header miaVideoHeader, remote *net.UDPAddr) miaVideoDatagramPlan {
	if header.payloadLen != 0 || header.chunkIndex != 0 || header.chunkCount != 0 {
		return c.videoProtocolErrorPlan(remote, header.requestID, header.frameID)
	}

	if c.video.pendingValid {
		if header.requestID == c.video.pendingRequestID && header.frameID == c.video.pendingFrameID {
			c.videoReleasePendingResponse(true)
			return miaVideoDatagramPlan{}
		}

		if header.frameID > c.video.pendingFrameID {
			return c.videoProtocolErrorPlan(remote, header.requestID, header.frameID)
		}

		return miaVideoDatagramPlan{}
	}

	if header.frameID > c.video.clientFrameID {
		return c.videoProtocolErrorPlan(remote, header.requestID, header.frameID)
	}

	return miaVideoDatagramPlan{}
}

func (c *emulated_mia) videoHandleNackChunks(header miaVideoHeader, payload []byte, remote *net.UDPAddr) miaVideoDatagramPlan {
	if header.chunkIndex != 0 || header.chunkCount != 0 || header.payloadLen < 4 {
		return c.videoProtocolErrorPlan(remote, header.requestID, header.frameID)
	}

	missingCount := binary.LittleEndian.Uint16(payload[0:2])
	reserved := binary.LittleEndian.Uint16(payload[2:4])
	if reserved != 0 || missingCount == 0 || int(header.payloadLen) != 4+int(missingCount)*2 {
		return c.videoProtocolErrorPlan(remote, header.requestID, header.frameID)
	}

	if !c.video.pendingValid {
		if header.frameID > c.video.clientFrameID {
			return c.videoProtocolErrorPlan(remote, header.requestID, header.frameID)
		}

		return miaVideoDatagramPlan{}
	}

	if header.requestID != c.video.pendingRequestID || header.frameID != c.video.pendingFrameID {
		if header.frameID > c.video.pendingFrameID {
			return c.videoProtocolErrorPlan(remote, header.requestID, header.frameID)
		}

		return miaVideoDatagramPlan{}
	}

	seen := make(map[uint16]bool, missingCount)
	chunks := make([]uint16, 0, missingCount)
	for i := 0; i < int(missingCount); i++ {
		chunk := binary.LittleEndian.Uint16(payload[4+i*2 : 6+i*2])
		if chunk >= c.video.pendingChunkCount || seen[chunk] {
			return c.videoProtocolErrorPlan(remote, header.requestID, header.frameID)
		}

		seen[chunk] = true
		chunks = append(chunks, chunk)
	}

	return miaVideoDatagramPlan{
		frame: &miaVideoFrameJob{
			addr:      remote,
			requestID: c.video.pendingRequestID,
			frameID:   c.video.pendingFrameID,
			chunks:    chunks,
		},
	}
}

func (c *emulated_mia) videoHandleClientStatus(header miaVideoHeader, payload []byte, remote *net.UDPAddr) miaVideoDatagramPlan {
	if header.payloadLen != 2 || header.chunkIndex != 0 || header.chunkCount != 0 {
		return c.videoProtocolErrorPlan(remote, header.requestID, header.frameID)
	}

	status := binary.LittleEndian.Uint16(payload[0:2])
	if status == miaVideoStatusProtocolErr {
		c.videoInvalidateSession()
	}

	return miaVideoDatagramPlan{}
}

// videoProtocolErrorForMalformed sends a protocol error for a datagram that
// failed full header validation but is still recognizably ours: it has the
// magic and version, and its session/remote match the active session. This lets
// a peer that corrupts a packet learn the session is being torn down instead of
// silently stalling. Packets that are not even partially valid are ignored.
func (c *emulated_mia) videoProtocolErrorForMalformed(packet []byte, remote *net.UDPAddr) miaVideoDatagramPlan {
	if len(packet) < miaVideoHeaderSize ||
		binary.LittleEndian.Uint16(packet[0:2]) != miaVideoMagic ||
		packet[2] != miaVideoVersion {
		return miaVideoDatagramPlan{}
	}

	partial := miaVideoHeader{
		sessionID: binary.LittleEndian.Uint32(packet[4:8]),
		frameID:   binary.LittleEndian.Uint32(packet[16:20]),
		requestID: binary.LittleEndian.Uint16(packet[20:22]),
	}

	if !c.videoAcceptsSessionPacket(partial, remote) {
		return miaVideoDatagramPlan{}
	}

	return c.videoProtocolErrorPlan(remote, partial.requestID, partial.frameID)
}

func (c *emulated_mia) videoProtocolErrorPlan(remote *net.UDPAddr, requestID uint16, frameID uint32) miaVideoDatagramPlan {
	if !c.video.sessionActive {
		return miaVideoDatagramPlan{}
	}

	packet := c.videoBuildStatusLocked(miaVideoStatusProtocolErr, requestID, frameID)
	c.videoInvalidateSession()

	return miaVideoDatagramPlan{
		packets: []miaVideoOutgoing{{
			addr: remote,
			data: packet,
		}},
	}
}

func (c *emulated_mia) videoResendPendingPlan(remote *net.UDPAddr, initialSend bool) miaVideoDatagramPlan {
	if !c.video.pendingValid {
		return miaVideoDatagramPlan{}
	}

	chunks := make([]uint16, c.video.pendingChunkCount)
	for i := range chunks {
		chunks[i] = uint16(i)
	}

	return miaVideoDatagramPlan{
		frame: &miaVideoFrameJob{
			addr:        remote,
			requestID:   c.video.pendingRequestID,
			frameID:     c.video.pendingFrameID,
			chunks:      chunks,
			initialSend: initialSend && !c.video.pendingInitialSent,
		},
	}
}

func (c *emulated_mia) videoBuildFrameChunk(requestID uint16, frameID uint32, chunkIndex uint16) []byte {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.video.pendingValid ||
		requestID != c.video.pendingRequestID ||
		frameID != c.video.pendingFrameID ||
		chunkIndex >= c.video.pendingChunkCount {
		return nil
	}

	firstRecord := int(chunkIndex) * miaVideoRecordsPerChunk
	lastRecord := firstRecord + miaVideoRecordsPerChunk
	if lastRecord > len(c.video.pendingPages) {
		lastRecord = len(c.video.pendingPages)
	}

	recordCount := lastRecord - firstRecord
	payloadLen := uint16(recordCount * miaVideoPageRecordSize)
	packet := c.videoBuildHeaderLocked(
		miaVideoPacketFrameData,
		c.video.sessionID,
		frameID,
		requestID,
		chunkIndex,
		c.video.pendingChunkCount,
		payloadLen,
	)

	for i := firstRecord; i < lastRecord; i++ {
		page := c.video.pendingPages[i]
		recordOffset := len(packet)
		packet = append(packet, 0, 0)
		binary.LittleEndian.PutUint16(packet[recordOffset:recordOffset+2], page)

		pageData := make([]byte, miaVideoPageSize)
		pageOffset := int(page) << miaVideoPageShift
		validLen := miaVideoPageSize
		if pageOffset+validLen > miaVideoStateSize {
			validLen = miaVideoStateSize - pageOffset
		}
		copy(pageData, c.memory[pageOffset:pageOffset+validLen])
		packet = append(packet, pageData...)
	}

	return packet
}

func (c *emulated_mia) videoMarkInitialSendComplete(requestID uint16, frameID uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.video.pendingValid ||
		c.video.pendingInitialSent ||
		requestID != c.video.pendingRequestID ||
		frameID != c.video.pendingFrameID {
		return
	}

	c.video.pendingInitialSent = true
	c.videoSetLastResponseDirtyPages(uint16(len(c.video.pendingPages)))
	c.statusSet(miaStatusVideoSent)
	c.irqSetFlag(miaIRQVideoSent)
}

func (c *emulated_mia) videoBuildStatusLocked(statusCode uint16, requestID uint16, frameID uint32) []byte {
	packet := c.videoBuildHeaderLocked(
		miaVideoPacketStatus,
		c.video.sessionID,
		frameID,
		requestID,
		0,
		0,
		2,
	)
	packet = append(packet, 0, 0)
	binary.LittleEndian.PutUint16(packet[miaVideoHeaderSize:miaVideoHeaderSize+2], statusCode)

	return packet
}

func (c *emulated_mia) videoBuildHeaderLocked(packetType uint8, sessionID uint32, frameID uint32, requestID uint16, chunkIndex uint16, chunkCount uint16, payloadLen uint16) []byte {
	packet := make([]byte, miaVideoHeaderSize, miaVideoHeaderSize+int(payloadLen))
	binary.LittleEndian.PutUint16(packet[0:2], miaVideoMagic)
	packet[2] = miaVideoVersion
	packet[3] = packetType
	binary.LittleEndian.PutUint32(packet[4:8], sessionID)
	binary.LittleEndian.PutUint32(packet[8:12], c.videoNextSeq())
	binary.LittleEndian.PutUint32(packet[12:16], c.video.lastPeerSeq)
	binary.LittleEndian.PutUint32(packet[16:20], frameID)
	binary.LittleEndian.PutUint16(packet[20:22], requestID)
	binary.LittleEndian.PutUint16(packet[22:24], chunkIndex)
	binary.LittleEndian.PutUint16(packet[24:26], chunkCount)
	binary.LittleEndian.PutUint16(packet[26:28], payloadLen)

	return packet
}

func (c *emulated_mia) videoNextSeq() uint32 {
	c.video.nextSeq++
	if c.video.nextSeq == 0 {
		c.video.nextSeq = 1
	}

	return c.video.nextSeq
}

func (c *emulated_mia) videoNextFrameID() uint32 {
	c.video.frameID++
	if c.video.frameID == 0 {
		c.video.frameID = 1
	}

	return c.video.frameID
}

func (c *emulated_mia) videoNewSessionID(remote *net.UDPAddr) uint32 {
	seed := uint32(time.Now().UnixNano())
	seed ^= uint32(remote.Port) << 16
	seed ^= c.video.nextSeq + 0x9E3779B9
	if seed == 0 {
		seed = 1
	}

	return seed
}

func (c *emulated_mia) videoResetRuntimeState() {
	conn := c.video.conn
	bindAddress := c.video.bindAddress
	nextSeq := c.video.nextSeq
	c.video = miaVideoState{
		conn:        conn,
		bindAddress: bindAddress,
		nextSeq:     nextSeq,
		activeMap:   0,
	}
	c.memory[miaVideoLocalVersionOffset] = miaVideoLayoutVersion
	c.videoSetFrameID(0)
	c.videoSetLastResponseDirtyPages(0)
}

func (c *emulated_mia) videoResetSession(remote *net.UDPAddr) {
	c.videoClearDirtyMaps()
	c.video.activeMap = 0
	c.video.sessionActive = true
	c.video.remote = &net.UDPAddr{IP: append(net.IP(nil), remote.IP...), Port: remote.Port, Zone: remote.Zone}
	c.video.sessionID = c.videoNewSessionID(remote)
	c.video.clientFrameID = 0
	c.video.frameID = 0
	c.video.pendingValid = false
	c.video.pendingPages = nil
	c.video.pendingChunkCount = 0
	c.video.pendingInitialSent = false
	c.videoSetFrameID(0)
	c.videoSetLastResponseDirtyPages(0)
	c.statusClear(miaStatusVideoRequested | miaStatusVideoSent)
	c.videoMarkAllSyncPagesDirty()
}

func (c *emulated_mia) videoInvalidateSession() {
	c.videoClearDirtyMaps()
	c.video.activeMap = 0
	c.video.sessionActive = false
	c.video.remote = nil
	c.video.sessionID = 0
	c.video.clientFrameID = 0
	c.video.pendingValid = false
	c.video.pendingPages = nil
	c.video.pendingChunkCount = 0
	c.video.pendingInitialSent = false
	c.statusClear(miaStatusVideoRequested | miaStatusVideoSent)
}

func (c *emulated_mia) videoReleasePendingResponse(acknowledged bool) {
	if !c.video.pendingValid {
		return
	}

	pendingFrameID := c.video.pendingFrameID
	c.videoClearDirtyMap(c.video.pendingMap)
	c.video.pendingValid = false
	c.video.pendingPages = nil
	c.video.pendingChunkCount = 0
	c.video.pendingInitialSent = false

	if acknowledged {
		c.video.clientFrameID = pendingFrameID
		c.statusClear(miaStatusVideoRequested | miaStatusVideoSent)
		c.irqSetFlag(miaIRQVideoAcked)
	}
}

func (c *emulated_mia) videoEnable() {
	clear(c.memory[:miaVideoStateSize])
	c.memory[miaVideoLocalVersionOffset] = miaVideoLayoutVersion
	c.videoLoadDefaultFont()
	c.video.frameID = 0
	c.video.clientFrameID = 0
	c.videoSetFrameID(0)
	c.videoSetLastResponseDirtyPages(0)
	c.videoClearDirtyMaps()
	c.video.activeMap = 0
	c.video.pendingValid = false
	c.video.pendingPages = nil
	c.video.pendingChunkCount = 0
	c.video.pendingInitialSent = false
	c.statusClear(miaStatusVideoRequested | miaStatusVideoSent)
	c.videoConfigureIndexes()
	c.videoMarkAllSyncPagesDirty()
}

func (c *emulated_mia) videoLoadDefaultFont() {
	if len(assets.MiaPETSCIICharset) < miaPETSCIIPlaneSize*2 {
		return
	}

	for i := 0; i < miaPETSCIIPlaneSize; i++ {
		c.memory[miaVideoCHROffset+i] = reverseByte(assets.MiaPETSCIICharset[i])
		c.memory[miaVideoCHROffset+0x800+i] = reverseByte(assets.MiaPETSCIICharset[miaPETSCIIPlaneSize+i])
	}
}

func (c *emulated_mia) videoForceFullRefresh() {
	c.videoMarkAllSyncPagesDirty()
}

func (c *emulated_mia) videoSetMode(mode uint8) {
	c.memory[miaVideoModeOffset] = mode
	c.videoMarkDirty(miaVideoModeOffset)
}

func (c *emulated_mia) videoConfigureIndexes() {
	for i := 0x70; i <= 0xFF; i++ {
		c.indexes[i] = miaIndex{}
	}

	c.videoConfigureIndex(0x70, 0x00000, 32)
	c.videoConfigureIndex(0x71, 0x00004, 4)
	c.videoConfigureIndex(0x72, 0x00008, 2)
	c.videoConfigureIndex(0x80, miaVideoRenderControlOffset, 32)
	c.videoConfigureIndex(0x81, 0x00021, 1)
	c.videoConfigureIndex(0x82, 0x00022, 2)
	c.videoConfigureIndex(0x83, 0x00024, 2)
	c.videoConfigureIndex(0x84, 0x00026, 2)
	c.videoConfigureIndex(0x85, 0x00028, 5)
	c.videoConfigureIndex(0x86, 0x0002D, 2)
	c.videoConfigureIndex(0x87, 0x0002F, 1)
	c.videoConfigureIndex(0x88, 0x00030, 1)

	for i := 0; i < 16; i++ {
		c.videoConfigureIndex(uint8(0x90+i), uint32(miaVideoPaletteOffset+i*16), 16)
	}

	for i := 0; i < 8; i++ {
		c.videoConfigureIndex(uint8(0xA0+i), uint32(miaVideoCHROffset+i*6144), 6144)
		c.videoConfigureIndex(uint8(0xA8+i), uint32(miaVideoBGNTOffset+i*1000), 1000)
		c.videoConfigureIndex(uint8(0xB0+i), uint32(miaVideoBGAttrOffset+i*1000), 1000)
	}

	c.videoConfigureIndex(0xB8, miaVideoOverlayNTOffset, 1000)
	c.videoConfigureIndex(0xB9, miaVideoOverlayAttrOffset, 1000)

	for i := 0; i < 32; i++ {
		c.videoConfigureIndex(uint8(0xC0+i), uint32(miaVideoOAMOffset+i*5), 5)
	}
}

func (c *emulated_mia) videoConfigureIndex(indexID uint8, start uint32, length uint32) {
	c.indexes[indexID] = miaIndex{
		currentAddr: start,
		defaultAddr: start,
		limitAddr:   start + length,
		step:        1,
		flags:       (1 << miaIndexFlagReadStep) | (1 << miaIndexFlagWriteStep) | (1 << miaIndexFlagWrap),
	}
}

func (c *emulated_mia) videoMarkDirty(addr uint32) {
	if addr < miaVideoPageSize || addr >= miaVideoStateSize {
		return
	}

	page := addr >> miaVideoPageShift
	c.video.dirtyMaps[c.video.activeMap][page>>3] |= uint8(1 << (page & 0x07))
}

func (c *emulated_mia) videoMarkDirtyRange(start uint32, length uint32) {
	if length == 0 || start >= miaVideoStateSize {
		return
	}

	end := start + length
	if end < start {
		end = miaVideoStateSize
	}

	if end <= miaVideoPageSize {
		return
	}

	if start < miaVideoPageSize {
		start = miaVideoPageSize
	}
	if end > miaVideoStateSize {
		end = miaVideoStateSize
	}

	for page := start >> miaVideoPageShift; page <= (end-1)>>miaVideoPageShift; page++ {
		c.video.dirtyMaps[c.video.activeMap][page>>3] |= uint8(1 << (page & 0x07))
	}
}

func (c *emulated_mia) videoMarkAllSyncPagesDirty() {
	for page := uint32(miaVideoFirstSyncPage); page < miaVideoPageCount; page++ {
		c.video.dirtyMaps[c.video.activeMap][page>>3] |= uint8(1 << (page & 0x07))
	}
}

// videoCountDirtyPages counts the dirty syncable pages in a dirty map. It mirrors
// the firmware video_count_dirty_pages helper used by the console diagnostics.
func (c *emulated_mia) videoCountDirtyPages(mapIndex uint8) uint16 {
	var count uint16
	for page := uint16(miaVideoFirstSyncPage); page < miaVideoPageCount; page++ {
		if c.videoPageDirty(mapIndex, page) {
			count++
		}
	}

	return count
}

func (c *emulated_mia) videoHasDirtyPages(mapIndex uint8) bool {
	for page := uint16(miaVideoFirstSyncPage); page < miaVideoPageCount; page++ {
		if c.videoPageDirty(mapIndex, page) {
			return true
		}
	}

	return false
}

func (c *emulated_mia) videoScanDirtyPages(mapIndex uint8) []uint16 {
	pages := make([]uint16, 0, 64)
	for page := uint16(miaVideoFirstSyncPage); page < miaVideoPageCount; page++ {
		if c.videoPageDirty(mapIndex, page) {
			pages = append(pages, page)
		}
	}

	return pages
}

func (c *emulated_mia) videoPageDirty(mapIndex uint8, page uint16) bool {
	return c.video.dirtyMaps[mapIndex][page>>3]&(uint8(1)<<(page&0x07)) != 0
}

func (c *emulated_mia) videoClearDirtyMaps() {
	c.videoClearDirtyMap(0)
	c.videoClearDirtyMap(1)
}

func (c *emulated_mia) videoClearDirtyMap(mapIndex uint8) {
	clear(c.video.dirtyMaps[mapIndex][:])
}

func (c *emulated_mia) videoSetFrameID(frameID uint32) {
	binary.LittleEndian.PutUint32(c.memory[miaVideoLocalFrameIDOffset:miaVideoLocalFrameIDOffset+4], frameID)
}

func (c *emulated_mia) videoSetLastResponseDirtyPages(count uint16) {
	binary.LittleEndian.PutUint16(c.memory[miaVideoLocalDirtyPagesOffset:miaVideoLocalDirtyPagesOffset+2], count)
}

func reverseByte(value byte) byte {
	value = (value&0xF0)>>4 | (value&0x0F)<<4
	value = (value&0xCC)>>2 | (value&0x33)<<2
	return (value&0xAA)>>1 | (value&0x55)<<1
}
