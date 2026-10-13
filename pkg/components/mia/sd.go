package mia

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// This file ports the Pico MIA SD-card and FAT filesystem subsystem (firmware
// src/mia/sd/sd.{c,h} + sd_diskio.c) to the emulator. The 6502-facing contract is
// reproduced exactly: the SD/FS RAM block at $13000, the control block / sector /
// path / directory-entry / transfer buffers, the $E0-$E4 indexes, commands
// $70-$73 and $78-$81, the MIA_STAT_SD_PRESENT/SD_BUSY/FS_MOUNTED status bits, the
// IRQ_SD_DONE/SD_ERROR/FS_EVENT IRQ bits, and the $70-$88 error codes.
//
// SD_HANDLE_SELECT ($2E, protocol v5): FS_OPEN/FS_READ/FS_WRITE/FS_SEEK/FS_SYNC/
// FS_CLOSE all act on whichever of miaSDMaxHandles slots this names, mirroring the
// firmware's per-slot FIL array. Size/position/EOF are never cached per slot -
// like the firmware (f_size/f_tell/f_eof read straight off the FIL), this reads
// them live from the slot's *os.File via Stat/Seek. The directory cursor and the
// whole-file load/save jobs stay singular, unaffected by handle selection.
//
// The platform-specific divergence is the backend. The firmware drives a real SD
// card over SPI and parses FAT through the bundled FatFs (ff.c). The emulator
// instead maps the FAT file/directory commands onto a host folder passed with the
// CLI `--sd` flag (see SetSDFolder); each FAT command becomes an os/filepath
// operation rooted at that folder. Raw sector commands ($71/$72) operate on a
// sparse in-memory virtual block device (rawSectors) because a host folder has no
// sector image; this keeps the raw API functional and safe (it never touches the
// host folder), matching how audio replaced the PWM pins with host output and
// video replaced GPIO with UDP. With no folder attached the subsystem behaves like
// a Pico with no card inserted: SD_INIT/FS_MOUNT fail and the SD status bits stay
// clear.
//
// Threading: unlike audio there is no background goroutine. SD/FS commands run
// synchronously from executeCommand (and the console) under the chip mutex c.mu,
// like the DMA/video/exec commands. The firmware request/service split (busy bit
// set on request, work done later on core 0) collapses to a synchronous call here:
// busy is set, the work runs, busy clears, all within the same command, mirroring
// the emulator's "commands run synchronously in executeCommand" model. The
// SD_FATFS_RESULT byte is filled with an approximate FatFs FRESULT derived from the
// host error so 6502 diagnostics stay meaningful.

const (
	miaSDStateOffset    = 0x13000
	miaSDControlOffset  = miaSDStateOffset
	miaSDControlSize    = 0x40
	miaSDSectorOffset   = 0x13040
	miaSDSectorSize     = 512
	miaFSPathOffset     = 0x13240
	miaFSPathSize       = 256
	miaFSDirEntryOffset = 0x13340
	miaFSDirEntrySize   = 256
	miaFSTransferOffset = 0x13440
	miaFSTransferSize   = 0x07C0

	// PATH2 (the second path buffer used by FS_RENAME) deliberately overlaps the
	// first 256 bytes of the transfer buffer, mirroring the firmware.
	miaFSPath2Offset = miaFSTransferOffset
	miaFSPath2Size   = miaFSPathSize

	miaSDVersion = 7

	// miaSDMaxHandles is the number of concurrent file-handle slots. BASIC-visible
	// file numbers map directly onto these slots, one-to-one, with no allocator in
	// between - see the SD_HANDLE_SELECT note above.
	miaSDMaxHandles = 16

	// Virtual raw block device capacity reported by SD_INIT/SD_GET_INFO and used
	// to bound raw sector access. A host folder has no inherent size, so this is a
	// fixed nominal capacity (1 GiB = sectors/2048 MiB) for the simulated card.
	miaSDVirtualSectors uint32 = 2 * 1024 * 1024

	// Virtual FAT geometry for FS_GET_FREE. A host folder has no real cluster
	// structure, so free/total space is synthesized from this nominal geometry and
	// the folder's current byte usage (32 KiB clusters over the virtual capacity).
	miaSDClusterSectors uint16 = 64

	// sdJobChunkSize and miaSDServiceBudgetUS mirror the firmware's chunked SD job
	// engine (SD_JOB_CHUNK_SIZE / MIA_SD_SERVICE_BUDGET_US). On the Pico the
	// FS_LOAD_TO_MIA_RAM / FS_SAVE_FROM_MIA_RAM jobs process one 512-byte chunk per
	// core-0 service call and yield once the per-call time budget expires, so
	// video/Wi-Fi/console work keeps breathing during big transfers. The emulator
	// runs each SD command synchronously (see the file header), so these are
	// informational only: the console reports them for parity, but they do not gate
	// the transfer.
	sdJobChunkSize       = 512
	miaSDServiceBudgetUS = 1000
)

// Control block field offsets, relative to miaSDControlOffset.
const (
	miaSDControlVersion         = 0x00
	miaSDControlStatus          = 0x01
	miaSDControlLastError       = 0x02
	miaSDControlCardType        = 0x03
	miaSDControlLBA0            = 0x04
	miaSDControlRequestLenL     = 0x08
	miaSDControlResultLenL      = 0x0A
	miaSDControlDestAddrL       = 0x0C
	miaSDControlFileHandle      = 0x0F
	miaSDControlOpenMode        = 0x10
	miaSDControlEOF             = 0x11
	miaSDControlFatfsResult     = 0x12
	miaSDControlCardSectors0    = 0x14
	miaSDControlFileSize0       = 0x18
	miaSDControlFilePos0        = 0x1C
	miaSDControlFreeClusters0   = 0x20
	miaSDControlTotalClusters0  = 0x24
	miaSDControlClusterSectorsL = 0x28
	miaSDControlTransferLen0    = 0x2A
	miaSDControlHandleSelect    = 0x2E

	// FS_LOAD_PART (protocol 7). SD_TRANSFER_LEN holds the bytes per row;
	// with PART_ROWS 0 the command loads one contiguous run.
	miaSDControlPartOffset0     = 0x30
	miaSDControlPartRowsL       = 0x34
	miaSDControlPartFileStride0 = 0x36
	miaSDControlPartRAMStrideL  = 0x3A
)

// SD_STATUS byte flags (mirrored into the control block).
const (
	miaSDStatusPresent     uint8 = 1 << 0
	miaSDStatusInitialized uint8 = 1 << 1
	miaSDStatusMounted     uint8 = 1 << 2
	miaSDStatusBusy        uint8 = 1 << 3
	miaSDStatusFileOpen    uint8 = 1 << 4
	miaSDStatusDirOpen     uint8 = 1 << 5
	miaSDStatusEOF         uint8 = 1 << 6
	miaSDStatusError       uint8 = 1 << 7
)

// Card type codes.
const (
	miaSDCardNone uint8 = 0
	miaSDCardSDV1 uint8 = 1
	miaSDCardSDV2 uint8 = 2
	miaSDCardSDHC uint8 = 3
)

// File open modes.
const (
	miaFSOpenRead        uint8 = 0
	miaFSOpenWriteCreate uint8 = 1
	miaFSOpenWriteAppend uint8 = 2
	miaFSOpenReadWrite   uint8 = 3
)

// Directory entry field offsets, relative to miaFSDirEntryOffset.
const (
	miaFSDirAttr    = 0x00
	miaFSDirNameLen = 0x01
	miaFSDirSize0   = 0x04
	miaFSDirDateL   = 0x08
	miaFSDirTimeL   = 0x0A
	miaFSDirName    = 0x0C

	miaFSDirAttrDirectory uint8 = 0x10
	miaFSDirAttrArchive   uint8 = 0x20
)

// SD/FS indexes.
const (
	miaSDIndexControl  uint8 = 0xE0
	miaSDIndexSector   uint8 = 0xE1
	miaFSIndexPath     uint8 = 0xE2
	miaFSIndexDirEntry uint8 = 0xE3
	miaFSIndexTransfer uint8 = 0xE4
	miaFSIndexPath2    uint8 = 0xE5
)

// SD/FS command ids.
const (
	miaCmdSDInit        uint8 = 0x70
	miaCmdSDReadSector  uint8 = 0x71
	miaCmdSDWriteSector uint8 = 0x72
	miaCmdSDGetInfo     uint8 = 0x73

	miaCmdFSMount        uint8 = 0x78
	miaCmdFSOpendir      uint8 = 0x79
	miaCmdFSReaddir      uint8 = 0x7A
	miaCmdFSOpen         uint8 = 0x7B
	miaCmdFSRead         uint8 = 0x7C
	miaCmdFSClose        uint8 = 0x7D
	miaCmdFSLoadToMiaRAM uint8 = 0x7E
	miaCmdFSWrite        uint8 = 0x7F
	miaCmdFSSync         uint8 = 0x80
	miaCmdFSSeek         uint8 = 0x81
	miaCmdFSStat         uint8 = 0x82
	miaCmdFSMkdir        uint8 = 0x83
	miaCmdFSDelete       uint8 = 0x84
	miaCmdFSRename       uint8 = 0x85
	miaCmdFSGetFree      uint8 = 0x86

	miaCmdFSSaveFromMiaRAM uint8 = 0x87
	miaCmdFSChdir          uint8 = 0x88
	miaCmdFSFileInfo       uint8 = 0x89
	miaCmdFSLoadPart       uint8 = 0x8A
)

// Subset of FatFs FRESULT codes, written to SD_FATFS_RESULT for diagnostics.
const (
	frOK               uint8 = 0
	frDiskErr          uint8 = 1
	frNotReady         uint8 = 3
	frNoFile           uint8 = 4
	frInvalidName      uint8 = 6
	frDenied           uint8 = 7
	frExist            uint8 = 8
	frInvalidObject    uint8 = 9
	frNoFilesystem     uint8 = 13
	frInvalidParameter uint8 = 19
)

// miaSDState is the emulator's SD/FS subsystem state, embedded in emulated_mia and
// guarded by the chip mutex c.mu. rootDir, rawSectors and initialized are config /
// virtual-card state that survives a 6502 reset; the rest is runtime state cleared
// by sdResetRuntimeState.
type miaSDState struct {
	// rootDir is the host folder backing the emulated SD. Empty means no card.
	rootDir string

	initialized bool
	mounted     bool
	dirOpen     bool
	eof         bool

	// cwd is the current directory, FatFs-style ("" means the root, matching
	// a fresh f_chdir-less mount). Firmware gets this for free from FatFs's
	// own FF_FS_RPATH-tracked FATFS work area (persists until the next
	// f_mount); the emulator has no such library underneath it, so it is
	// tracked explicitly here instead - reset on every (re)mount, same as a
	// real f_mount resets FatFs's cdir to root. See sdServiceChdir and
	// sdResolveHostPathFrom.
	cwd string

	lastError       uint8
	lastFatfsResult uint8
	cardType        uint8
	sectors         uint32

	// Per-handle file state, indexed by slot (0..miaSDMaxHandles-1). See the
	// SD_HANDLE_SELECT note in this file's header comment.
	fileOpen        [miaSDMaxHandles]bool
	currentOpenMode [miaSDMaxHandles]uint8
	file            [miaSDMaxHandles]*os.File

	dirEntries []os.DirEntry
	dirIndex   int

	// rawSectors is the sparse virtual block device for SD_READ_SECTOR /
	// SD_WRITE_SECTOR. Unwritten sectors read back as zero.
	rawSectors map[uint32][]uint8
}

/**************************************************************************************************
 * Little-endian RAM helpers
 **************************************************************************************************/

func (c *emulated_mia) sdReadU16(offset uint32) uint16 {
	return uint16(c.memory[offset]) | uint16(c.memory[offset+1])<<8
}

func (c *emulated_mia) sdReadU24(offset uint32) uint32 {
	return uint32(c.memory[offset]) |
		uint32(c.memory[offset+1])<<8 |
		uint32(c.memory[offset+2])<<16
}

func (c *emulated_mia) sdReadU32(offset uint32) uint32 {
	return uint32(c.memory[offset]) |
		uint32(c.memory[offset+1])<<8 |
		uint32(c.memory[offset+2])<<16 |
		uint32(c.memory[offset+3])<<24
}

func (c *emulated_mia) sdWriteU16(offset uint32, value uint16) {
	c.memory[offset] = uint8(value)
	c.memory[offset+1] = uint8(value >> 8)
}

func (c *emulated_mia) sdWriteU32(offset uint32, value uint32) {
	c.memory[offset] = uint8(value)
	c.memory[offset+1] = uint8(value >> 8)
	c.memory[offset+2] = uint8(value >> 16)
	c.memory[offset+3] = uint8(value >> 24)
}

func (c *emulated_mia) sdControlLBA() uint32 {
	return c.sdReadU32(miaSDControlOffset + miaSDControlLBA0)
}

func (c *emulated_mia) sdControlRequestLen() uint16 {
	return c.sdReadU16(miaSDControlOffset + miaSDControlRequestLenL)
}

func (c *emulated_mia) sdControlDestAddr() uint32 {
	return c.sdReadU24(miaSDControlOffset+miaSDControlDestAddrL) & miaRAMMask
}

func (c *emulated_mia) sdControlFilePos() uint32 {
	return c.sdReadU32(miaSDControlOffset + miaSDControlFilePos0)
}

func (c *emulated_mia) sdControlTransferLen() uint32 {
	return c.sdReadU32(miaSDControlOffset + miaSDControlTransferLen0)
}

// sdRequireValidHandle reads SD_HANDLE_SELECT and validates it against
// miaSDMaxHandles, mirroring sd_require_valid_handle. The second return is false
// (with the first meaningless) when the selected slot is out of range, so
// callers can fail the command with ERROR_FS_INVALID_HANDLE.
func (c *emulated_mia) sdRequireValidHandle() (uint8, bool) {
	slot := c.memory[miaSDControlOffset+miaSDControlHandleSelect]
	if int(slot) >= miaSDMaxHandles {
		return 0, false
	}
	return slot, true
}

/**************************************************************************************************
 * Lifecycle (run under c.mu)
 **************************************************************************************************/

// SetSDFolder attaches a host folder as the emulated SD card. An empty path
// detaches the card. The folder is the FAT root. It stands for the card being in
// the slot at power-on, so it repeats MIA's boot: the firmware reads WIFI.CFG
// before the 6502 starts, which initializes the card and leaves it unmounted.
// The 6502 then sees MIA_STAT_SD_PRESENT and mounts with FS_MOUNT or any FS
// command, exactly as with a real card.
func (c *emulated_mia) SetSDFolder(folder string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.sd.rootDir = folder
	c.sdCardInit()
	c.wifiBoot()
}

// sdResetRuntimeState mirrors mia_sd_reset_runtime_state: it unmounts, closes any
// open file/dir, clears the SD/FS RAM buffers, reseeds the control block defaults,
// reconfigures the fixed indexes, and republishes state. rootDir, the virtual card
// content, and the initialized flag persist (sd_initialized is static in firmware).
func (c *emulated_mia) sdResetRuntimeState() {
	c.sdCloseAllFiles()
	c.sdCloseDir()
	c.sd.mounted = false
	c.sd.eof = false
	c.sd.lastError = 0
	c.sd.lastFatfsResult = frOK

	c.sdClearBuffers()
	c.memory[miaSDControlOffset+miaSDControlVersion] = miaSDVersion
	c.sdWriteU16(miaSDControlOffset+miaSDControlRequestLenL, miaFSTransferSize)

	c.statusClear(miaStatusSDBusy | miaStatusFSMounted)
	if c.sd.initialized {
		c.statusSet(miaStatusSDPresent)
	} else {
		c.statusClear(miaStatusSDPresent)
	}

	c.sdConfigureIndexes()
	c.sdPublishState()
}

func (c *emulated_mia) sdClearBuffers() {
	clear(c.memory[miaSDControlOffset : miaSDControlOffset+miaSDControlSize])
	clear(c.memory[miaSDSectorOffset : miaSDSectorOffset+miaSDSectorSize])
	clear(c.memory[miaFSPathOffset : miaFSPathOffset+miaFSPathSize])
	clear(c.memory[miaFSDirEntryOffset : miaFSDirEntryOffset+miaFSDirEntrySize])
	clear(c.memory[miaFSTransferOffset : miaFSTransferOffset+miaFSTransferSize])
}

// sdClose tears down host handles. Called from the chip Close path.
func (c *emulated_mia) sdClose() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.sdCloseAllFiles()
	c.sdCloseDir()
}

// sdCloseFile closes one handle slot, mirroring the firmware's per-slot
// FS_CLOSE. It no longer also closes the directory cursor - with multiple
// handles that coupling would let closing an unrelated file silently kill an
// in-progress FS_OPENDIR/FS_READDIR listing. See sdCloseAllFiles for teardown.
func (c *emulated_mia) sdCloseFile(slot uint8) {
	if c.sd.file[slot] != nil {
		_ = c.sd.file[slot].Close()
		c.sd.file[slot] = nil
	}
	c.sd.fileOpen[slot] = false
	c.sd.currentOpenMode[slot] = miaFSOpenRead
}

// sdCloseAllFiles closes every handle slot, used by sdResetRuntimeState and
// sdCardInit (both mirror firmware paths that used to reset the single implicit
// handle).
func (c *emulated_mia) sdCloseAllFiles() {
	for slot := uint8(0); slot < miaSDMaxHandles; slot++ {
		c.sdCloseFile(slot)
	}
}

func (c *emulated_mia) sdCloseDir() {
	c.sd.dirEntries = nil
	c.sd.dirIndex = 0
	c.sd.dirOpen = false
}

/**************************************************************************************************
 * Request entry, busy/finish, state publishing
 **************************************************************************************************/

// sdRequest validates and runs an SD/FS command. It mirrors mia_sd_request plus
// the synchronous mia_sd_service: the firmware queues the request and services it
// later on core 0, while the emulator runs it inline (busy is set, the work runs,
// busy clears). The busy precheck is kept for parity but is unreachable
// synchronously.
func (c *emulated_mia) sdRequest(command uint8) bool {
	switch command {
	case miaCmdSDInit, miaCmdSDReadSector, miaCmdSDWriteSector, miaCmdSDGetInfo,
		miaCmdFSMount, miaCmdFSOpendir, miaCmdFSReaddir, miaCmdFSOpen,
		miaCmdFSRead, miaCmdFSClose, miaCmdFSLoadToMiaRAM, miaCmdFSWrite,
		miaCmdFSSync, miaCmdFSSeek, miaCmdFSStat, miaCmdFSMkdir,
		miaCmdFSDelete, miaCmdFSRename, miaCmdFSGetFree, miaCmdFSSaveFromMiaRAM,
		miaCmdFSChdir, miaCmdFSFileInfo, miaCmdFSLoadPart:
	default:
		return false
	}

	if c.status()&miaStatusSDBusy != 0 {
		c.errors.Push(c, miaErrorSDBusy)
		c.irqSetFlag(miaIRQSDError)
		return false
	}

	c.sdSetBusy(true)
	c.sdService(command)
	return true
}

func (c *emulated_mia) sdSetBusy(busy bool) {
	if busy {
		c.statusSet(miaStatusSDBusy)
	} else {
		c.statusClear(miaStatusSDBusy)
	}
	c.sdPublishState()
}

func (c *emulated_mia) sdSetLastError(errorCode uint8, fr uint8) {
	c.sd.lastError = errorCode
	c.sd.lastFatfsResult = fr
	c.memory[miaSDControlOffset+miaSDControlLastError] = errorCode
	c.memory[miaSDControlOffset+miaSDControlFatfsResult] = fr
}

// sdFinish mirrors sd_finish: it clears busy, records the result, raises the
// completion/error IRQ, and (for filesystem commands) the FS event IRQ.
func (c *emulated_mia) sdFinish(ok bool, errorCode uint8, fr uint8, fsEvent bool) {
	c.sdSetBusy(false)

	if ok {
		c.sdSetLastError(0, fr)
		c.irqSetFlag(miaIRQSDDone)
	} else {
		c.sdSetLastError(errorCode, fr)
		c.errors.Push(c, errorCode)
		c.irqSetFlag(miaIRQSDError)
	}

	if fsEvent {
		c.irqSetFlag(miaIRQFSEvent)
	}

	c.sdPublishState()
}

// sdPublishState mirrors sd_publish_state: it rebuilds the SD_STATUS byte and the
// read-only control fields and tracks the global FS_MOUNTED status bit.
func (c *emulated_mia) sdPublishState() {
	var status uint8

	// FILE_HANDLE/EOF describe whichever slot SD_HANDLE_SELECT currently names.
	// EOF falls back to the transient c.sd.eof (set by FS_READDIR / the load-save
	// jobs, neither of which has "a slot") when the selected slot has no file
	// open - exactly today's single-handle behavior for anyone who never touches
	// SD_HANDLE_SELECT, since slot 0 is the reset-time default.
	sel, selValid := c.sdRequireValidHandle()
	selOpen := selValid && c.sd.fileOpen[sel]
	selEOF := c.sd.eof
	if selOpen {
		selEOF = c.sdFileAtEOF(sel)
	}

	if c.sd.initialized {
		status |= miaSDStatusPresent | miaSDStatusInitialized
	}
	if c.sd.mounted {
		status |= miaSDStatusMounted
	}
	if c.status()&miaStatusSDBusy != 0 {
		status |= miaSDStatusBusy
	}
	if selOpen {
		status |= miaSDStatusFileOpen
	}
	if c.sd.dirOpen {
		status |= miaSDStatusDirOpen
	}
	if selEOF {
		status |= miaSDStatusEOF
	}
	if c.sd.lastError != 0 {
		status |= miaSDStatusError
	}

	c.memory[miaSDControlOffset+miaSDControlStatus] = status
	c.memory[miaSDControlOffset+miaSDControlLastError] = c.sd.lastError
	c.memory[miaSDControlOffset+miaSDControlCardType] = c.sd.cardType
	if selEOF {
		c.memory[miaSDControlOffset+miaSDControlEOF] = 1
	} else {
		c.memory[miaSDControlOffset+miaSDControlEOF] = 0
	}
	if selOpen {
		c.memory[miaSDControlOffset+miaSDControlFileHandle] = 1
	} else {
		c.memory[miaSDControlOffset+miaSDControlFileHandle] = 0
	}
	c.memory[miaSDControlOffset+miaSDControlFatfsResult] = c.sd.lastFatfsResult
	c.sdWriteU32(miaSDControlOffset+miaSDControlCardSectors0, c.sd.sectors)

	if c.sd.mounted {
		c.statusSet(miaStatusFSMounted)
	} else {
		c.statusClear(miaStatusFSMounted)
	}
}

/**************************************************************************************************
 * Command service (mirrors mia_sd_service)
 **************************************************************************************************/

func (c *emulated_mia) sdService(command uint8) {
	var ok bool
	var errorCode uint8
	fr := frOK
	var loaded uint32
	fsEvent := command >= miaCmdFSMount

	c.sd.eof = false
	c.sdWriteU16(miaSDControlOffset+miaSDControlResultLenL, 0)

	switch command {
	case miaCmdSDInit:
		ok = c.sdCardInit()
		errorCode = miaErrorSDInitFailed

	case miaCmdSDReadSector:
		ok = c.sdBlockRead(c.sdControlLBA())
		errorCode = c.sdNotReadyOr(miaErrorSDReadFailed)
		if ok {
			c.sdWriteU16(miaSDControlOffset+miaSDControlResultLenL, miaSDSectorSize)
		}

	case miaCmdSDWriteSector:
		ok = c.sdBlockWrite(c.sdControlLBA())
		errorCode = c.sdNotReadyOr(miaErrorSDWriteFailed)
		if ok {
			c.sdWriteU16(miaSDControlOffset+miaSDControlResultLenL, miaSDSectorSize)
		}

	case miaCmdSDGetInfo:
		ok = c.sd.initialized
		errorCode = miaErrorSDNotReady

	case miaCmdFSMount:
		ok, fr = c.sdMountFilesystem()
		errorCode = miaErrorFSMountFailed

	case miaCmdFSOpendir:
		ok, fr, errorCode = c.sdServiceOpendir()

	case miaCmdFSReaddir:
		ok, fr, errorCode = c.sdServiceReaddir()

	case miaCmdFSOpen:
		ok, fr, errorCode = c.sdServiceOpen()

	case miaCmdFSRead:
		ok, fr, errorCode = c.sdServiceRead()

	case miaCmdFSWrite:
		ok, fr, errorCode = c.sdServiceWrite()

	case miaCmdFSFileInfo:
		ok, fr, errorCode = c.sdServiceFileInfo()

	case miaCmdFSSync:
		ok, fr, errorCode = c.sdServiceSync()

	case miaCmdFSSeek:
		ok, fr, errorCode = c.sdServiceSeek()

	case miaCmdFSStat:
		ok, fr, errorCode = c.sdServiceStat()

	case miaCmdFSMkdir:
		ok, fr, errorCode = c.sdServiceMkdir()

	case miaCmdFSChdir:
		ok, fr, errorCode = c.sdServiceChdir()

	case miaCmdFSDelete:
		ok, fr, errorCode = c.sdServiceDelete()

	case miaCmdFSRename:
		ok, fr, errorCode = c.sdServiceRename()

	case miaCmdFSGetFree:
		ok, fr, errorCode = c.sdServiceGetFree()

	case miaCmdFSClose:
		ok, fr, errorCode = c.sdServiceClose()

	case miaCmdFSLoadToMiaRAM:
		if mountOK, mountFR := c.sdRequireMounted(); !mountOK {
			ok = false
			fr = mountFR
			errorCode = miaErrorFSMountFailed
			break
		}
		maxLen := uint32(c.sdControlRequestLen())
		loaded, ok, fr = c.sdLoadToRAM(c.sdControlDestAddr(), maxLen)
		errorCode = miaErrorFSReadFailed
		c.sdWriteU16(miaSDControlOffset+miaSDControlResultLenL, uint16(loaded))
		c.sdWriteU32(miaSDControlOffset+miaSDControlFilePos0, loaded)

	case miaCmdFSSaveFromMiaRAM:
		ok, fr, errorCode = c.sdServiceSave()

	case miaCmdFSLoadPart:
		if mountOK, mountFR := c.sdRequireMounted(); !mountOK {
			ok = false
			fr = mountFR
			errorCode = miaErrorFSMountFailed
			break
		}
		control := uint32(miaSDControlOffset)
		var loaded uint32
		loaded, ok, fr, errorCode = c.sdLoadPart(c.sdControlDestAddr(), c.sdControlTransferLen(),
			c.sdReadU32(control+miaSDControlPartOffset0), uint32(c.sdReadU16(control+miaSDControlPartRowsL)),
			c.sdReadU32(control+miaSDControlPartFileStride0), c.sdReadU24(control+miaSDControlPartRAMStrideL))
		c.sdWriteU16(miaSDControlOffset+miaSDControlResultLenL, uint16(loaded))
		c.sdWriteU32(miaSDControlOffset+miaSDControlFilePos0, loaded)

	default:
		ok = false
		errorCode = miaErrorFSInvalidRequest
		fr = frInvalidParameter
	}

	c.sdFinish(ok, errorCode, fr, fsEvent)
}

// sdNotReadyOr returns ERROR_SD_NOT_READY when the card is not initialized, else
// the supplied error, mirroring the firmware's raw-sector error selection.
func (c *emulated_mia) sdNotReadyOr(errorCode uint8) uint8 {
	if c.sd.initialized {
		return errorCode
	}
	return miaErrorSDNotReady
}

func (c *emulated_mia) sdServiceOpendir() (bool, uint8, uint8) {
	if ok, fr := c.sdRequireMounted(); !ok {
		return false, fr, miaErrorFSMountFailed
	}

	c.sdCloseDir()

	hostPath, ok := c.sdResolveHostPath()
	if !ok {
		c.sdClearDirEntry()
		return false, frInvalidName, miaErrorFSDirFailed
	}

	entries, err := os.ReadDir(hostPath)
	if err != nil {
		c.sdClearDirEntry()
		return false, sdErrToFresult(err), miaErrorFSDirFailed
	}

	c.sd.dirEntries = entries
	c.sd.dirIndex = 0
	c.sd.dirOpen = true
	c.sdClearDirEntry()
	return true, frOK, miaErrorFSDirFailed
}

func (c *emulated_mia) sdServiceReaddir() (bool, uint8, uint8) {
	if !c.sd.dirOpen {
		return false, frInvalidObject, miaErrorFSDirFailed
	}

	if c.sd.dirIndex >= len(c.sd.dirEntries) {
		c.sd.eof = true
		c.sdClearDirEntry()
		return true, frOK, miaErrorFSDirFailed
	}

	entry := c.sd.dirEntries[c.sd.dirIndex]
	c.sd.dirIndex++
	c.sdFillDirEntry(entry)
	return true, frOK, miaErrorFSDirFailed
}

func (c *emulated_mia) sdServiceOpen() (bool, uint8, uint8) {
	slot, slotOK := c.sdRequireValidHandle()
	if !slotOK {
		return false, frInvalidParameter, miaErrorFSInvalidHandle
	}

	if ok, fr := c.sdRequireMounted(); !ok {
		return false, fr, miaErrorFSMountFailed
	}

	openMode := c.memory[miaSDControlOffset+miaSDControlOpenMode]
	flag, ok := sdOpenModeToFlag(openMode)
	if !ok {
		return false, frDenied, miaErrorFSInvalidRequest
	}

	// Multi-handle FS_OPEN requires an explicit FS_CLOSE first, unlike the old
	// single-handle behavior that silently replaced whatever was open - an
	// unexpected re-open of a slot is much more likely a program bug than an
	// intentional replace.
	if c.sd.fileOpen[slot] {
		return false, frDenied, miaErrorFSHandleAlreadyOpen
	}

	hostPath, ok := c.sdResolveHostPath()
	if !ok {
		c.sdUpdateFilePosition(slot)
		return false, frInvalidName, miaErrorFSOpenFailed
	}

	file, err := os.OpenFile(hostPath, flag, 0o644)
	if err != nil {
		c.sdUpdateFilePosition(slot)
		return false, sdErrToFresult(err), miaErrorFSOpenFailed
	}

	// FatFs FA_OPEN_APPEND places the file pointer at EOF; mirror it so FILE_POS
	// reflects the append position immediately after open.
	if openMode == miaFSOpenWriteAppend {
		_, _ = file.Seek(0, io.SeekEnd)
	}

	c.sd.file[slot] = file
	c.sd.fileOpen[slot] = true
	c.sd.currentOpenMode[slot] = openMode
	c.sdUpdateFilePosition(slot)
	return true, frOK, miaErrorFSOpenFailed
}

func (c *emulated_mia) sdServiceRead() (bool, uint8, uint8) {
	slot, slotOK := c.sdRequireValidHandle()
	if !slotOK {
		return false, frInvalidParameter, miaErrorFSInvalidHandle
	}
	if !c.sd.fileOpen[slot] || c.sd.file[slot] == nil {
		return false, frInvalidObject, miaErrorFSNoFileOpen
	}

	requested := c.sdControlRequestLen()
	if requested == 0 || requested > miaFSTransferSize {
		requested = miaFSTransferSize
	}

	start := uint32(miaFSTransferOffset)
	n, err := c.sd.file[slot].Read(c.memory[start : start+uint32(requested)])
	if err != nil && err != io.EOF {
		c.sdUpdateFilePosition(slot)
		return false, sdErrToFresult(err), miaErrorFSReadFailed
	}

	c.sdWriteU16(miaSDControlOffset+miaSDControlResultLenL, uint16(n))
	c.sdUpdateFilePosition(slot)
	return true, frOK, miaErrorFSReadFailed
}

func (c *emulated_mia) sdServiceWrite() (bool, uint8, uint8) {
	slot, slotOK := c.sdRequireValidHandle()
	if !slotOK {
		return false, frInvalidParameter, miaErrorFSInvalidHandle
	}
	if !c.sd.fileOpen[slot] || c.sd.file[slot] == nil {
		return false, frInvalidObject, miaErrorFSNoFileOpen
	}

	requested := c.sdControlRequestLen()
	if requested == 0 || requested > miaFSTransferSize {
		requested = miaFSTransferSize
	}

	start := uint32(miaFSTransferOffset)
	n, err := c.sd.file[slot].Write(c.memory[start : start+uint32(requested)])
	ok := err == nil && n == int(requested)
	c.sdWriteU16(miaSDControlOffset+miaSDControlResultLenL, uint16(n))
	c.sdUpdateFilePosition(slot)
	if !ok {
		return false, sdErrToFresult(err), miaErrorFSWriteFailed
	}
	return true, frOK, miaErrorFSWriteFailed
}

// A metadata query must not flush, seek to a target, or change EOF state.
func (c *emulated_mia) sdServiceFileInfo() (bool, uint8, uint8) {
	slot, valid := c.sdRequireValidHandle()
	if !valid {
		return false, frInvalidParameter, miaErrorFSInvalidHandle
	}
	if !c.sd.fileOpen[slot] || c.sd.file[slot] == nil {
		return false, frInvalidObject, miaErrorFSNoFileOpen
	}
	info, err := c.sd.file[slot].Stat()
	if err != nil {
		return false, sdErrToFresult(err), miaErrorFSStatFailed
	}
	pos, err := c.sd.file[slot].Seek(0, io.SeekCurrent)
	if err != nil {
		return false, sdErrToFresult(err), miaErrorFSSeekFailed
	}
	c.sdWriteU32(miaSDControlOffset+miaSDControlFileSize0, uint32(info.Size()))
	c.sdWriteU32(miaSDControlOffset+miaSDControlFilePos0, uint32(pos))
	return true, frOK, 0
}

func (c *emulated_mia) sdServiceSync() (bool, uint8, uint8) {
	slot, slotOK := c.sdRequireValidHandle()
	if !slotOK {
		return false, frInvalidParameter, miaErrorFSInvalidHandle
	}
	if !c.sd.fileOpen[slot] || c.sd.file[slot] == nil {
		return false, frInvalidObject, miaErrorFSNoFileOpen
	}

	err := c.sd.file[slot].Sync()
	c.sdUpdateFilePosition(slot)
	if err != nil {
		return false, sdErrToFresult(err), miaErrorFSSyncFailed
	}
	return true, frOK, miaErrorFSSyncFailed
}

func (c *emulated_mia) sdServiceSeek() (bool, uint8, uint8) {
	slot, slotOK := c.sdRequireValidHandle()
	if !slotOK {
		return false, frInvalidParameter, miaErrorFSInvalidHandle
	}
	if !c.sd.fileOpen[slot] || c.sd.file[slot] == nil {
		return false, frInvalidObject, miaErrorFSNoFileOpen
	}

	_, err := c.sd.file[slot].Seek(int64(c.sdControlFilePos()), io.SeekStart)
	c.sdUpdateFilePosition(slot)
	if err != nil {
		return false, sdErrToFresult(err), miaErrorFSSeekFailed
	}
	return true, frOK, miaErrorFSSeekFailed
}

// sdServiceClose closes only the selected handle slot. It used to also close
// the (then-singular, now-shared) directory cursor as a side effect - dropped,
// for the same reason noted on sdCloseFile.
func (c *emulated_mia) sdServiceClose() (bool, uint8, uint8) {
	slot, slotOK := c.sdRequireValidHandle()
	if !slotOK {
		return false, frInvalidParameter, miaErrorFSInvalidHandle
	}

	ok := true
	fr := frOK
	errorCode := miaErrorFSCloseFailed

	if c.sd.fileOpen[slot] && c.sd.file[slot] != nil {
		if err := c.sd.file[slot].Close(); err != nil {
			ok = false
			fr = sdErrToFresult(err)
			errorCode = miaErrorFSCloseFailed
		}
		c.sd.file[slot] = nil
	}
	c.sd.fileOpen[slot] = false
	c.sd.currentOpenMode[slot] = miaFSOpenRead
	c.sdUpdateFilePosition(slot)
	return ok, fr, errorCode
}

// sdServiceStat mirrors the firmware FS_STAT case: it stats a path and fills the
// directory-entry buffer with the result.
func (c *emulated_mia) sdServiceStat() (bool, uint8, uint8) {
	if ok, fr := c.sdRequireMounted(); !ok {
		return false, fr, miaErrorFSMountFailed
	}

	hostPath, ok := c.sdResolveHostPath()
	if !ok {
		return false, frInvalidName, miaErrorFSStatFailed
	}

	info, err := os.Stat(hostPath)
	if err != nil {
		return false, sdErrToFresult(err), miaErrorFSStatFailed
	}

	var size uint32
	if !info.IsDir() {
		size = uint32(info.Size())
	}
	c.sdWriteDirEntry(info.Name(), info.IsDir(), size, info.ModTime())
	c.sdWriteU16(miaSDControlOffset+miaSDControlResultLenL, miaFSDirEntrySize)
	return true, frOK, miaErrorFSStatFailed
}

// sdServiceMkdir mirrors FS_MKDIR (f_mkdir): it creates one directory.
func (c *emulated_mia) sdServiceMkdir() (bool, uint8, uint8) {
	if ok, fr := c.sdRequireMounted(); !ok {
		return false, fr, miaErrorFSMountFailed
	}

	hostPath, ok := c.sdResolveHostPath()
	if !ok {
		return false, frInvalidName, miaErrorFSMkdirFailed
	}

	if err := os.Mkdir(hostPath, 0o755); err != nil {
		return false, sdErrToFresult(err), miaErrorFSMkdirFailed
	}
	return true, frOK, miaErrorFSMkdirFailed
}

// sdServiceChdir mirrors FS_CHDIR (f_chdir): it changes the directory that
// relative paths resolve against (see sdResolveHostPathFrom and the cwd
// field's own comment). f_chdir requires an existing directory, so this
// stats the target and rejects a file or a missing path the same way.
func (c *emulated_mia) sdServiceChdir() (bool, uint8, uint8) {
	if ok, fr := c.sdRequireMounted(); !ok {
		return false, fr, miaErrorFSMountFailed
	}

	hostPath, ok := c.sdResolveHostPath()
	if !ok {
		return false, frInvalidName, miaErrorFSDirFailed
	}

	info, err := os.Stat(hostPath)
	if err != nil {
		return false, sdErrToFresult(err), miaErrorFSDirFailed
	}
	if !info.IsDir() {
		return false, frInvalidObject, miaErrorFSDirFailed
	}

	// Recompute cwd from the resolved host path (rather than combining the
	// raw argument with the old cwd by hand) so ".", "..", and mixed
	// relative/absolute arguments all land correctly - sdResolveHostPath
	// already did that resolution once above.
	rel, err := filepath.Rel(c.sd.rootDir, hostPath)
	if err != nil {
		return false, frInvalidName, miaErrorFSDirFailed
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		c.sd.cwd = ""
	} else {
		c.sd.cwd = "/" + rel
	}
	return true, frOK, miaErrorFSDirFailed
}

// sdServiceDelete mirrors FS_DELETE (f_unlink): it removes a file or empty
// directory.
func (c *emulated_mia) sdServiceDelete() (bool, uint8, uint8) {
	if ok, fr := c.sdRequireMounted(); !ok {
		return false, fr, miaErrorFSMountFailed
	}

	hostPath, ok := c.sdResolveHostPath()
	if !ok {
		return false, frInvalidName, miaErrorFSDeleteFailed
	}

	if err := os.Remove(hostPath); err != nil {
		return false, sdErrToFresult(err), miaErrorFSDeleteFailed
	}
	return true, frOK, miaErrorFSDeleteFailed
}

// sdServiceRename mirrors FS_RENAME (f_rename): the source path comes from the
// path buffer and the destination from the second path buffer (PATH2).
func (c *emulated_mia) sdServiceRename() (bool, uint8, uint8) {
	if ok, fr := c.sdRequireMounted(); !ok {
		return false, fr, miaErrorFSMountFailed
	}

	oldPath, ok := c.sdResolveHostPath()
	if !ok {
		return false, frInvalidName, miaErrorFSRenameFailed
	}
	newPath, ok := c.sdResolveHostPath2()
	if !ok {
		return false, frInvalidName, miaErrorFSRenameFailed
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return false, sdErrToFresult(err), miaErrorFSRenameFailed
	}
	return true, frOK, miaErrorFSRenameFailed
}

// sdServiceGetFree mirrors FS_GET_FREE (f_getfree): it reports the free/total
// cluster counts and sectors-per-cluster. The emulator synthesizes these from the
// virtual FAT geometry and the host folder's current byte usage.
func (c *emulated_mia) sdServiceGetFree() (bool, uint8, uint8) {
	if ok, fr := c.sdRequireMounted(); !ok {
		return false, fr, miaErrorFSMountFailed
	}

	free, total, clusterSectors := c.sdQueryFree()
	c.sdWriteU32(miaSDControlOffset+miaSDControlFreeClusters0, free)
	c.sdWriteU32(miaSDControlOffset+miaSDControlTotalClusters0, total)
	c.sdWriteU16(miaSDControlOffset+miaSDControlClusterSectorsL, clusterSectors)
	c.sdWriteU16(miaSDControlOffset+miaSDControlResultLenL, 10)
	return true, frOK, miaErrorFSFreeFailed
}

/**************************************************************************************************
 * Backend: virtual card + host folder
 **************************************************************************************************/

// sdCardInit mirrors mia_sd_card_init: it (re)initializes the virtual block
// device. With a host folder attached it reports a present SDHC card; with no
// folder it fails, like an empty card slot.
func (c *emulated_mia) sdCardInit() bool {
	c.sd.initialized = false
	c.sd.mounted = false
	c.sdCloseAllFiles()
	c.sdCloseDir()
	c.sd.eof = false
	c.sd.cardType = miaSDCardNone
	c.sd.sectors = 0
	c.statusClear(miaStatusSDPresent | miaStatusFSMounted)

	if !c.sdRootIsDir() {
		c.sdPublishState()
		return false
	}

	c.sd.initialized = true
	c.sd.cardType = miaSDCardSDHC
	c.sd.sectors = miaSDVirtualSectors
	if c.sd.rawSectors == nil {
		c.sd.rawSectors = make(map[uint32][]uint8)
	}
	c.statusSet(miaStatusSDPresent)
	c.sdPublishState()
	return true
}

// sdMountFilesystem mirrors sd_mount_filesystem: it initializes the card if needed
// and "mounts" the folder.
func (c *emulated_mia) sdMountFilesystem() (bool, uint8) {
	if !c.sd.initialized {
		if !c.sdCardInit() {
			c.sd.mounted = false
			return false, frNotReady
		}
	}

	if !c.sdRootIsDir() {
		c.sd.mounted = false
		return false, frNoFilesystem
	}

	c.sd.mounted = true
	c.sd.cwd = ""
	return true, frOK
}

// sdRequireMounted mirrors sd_require_mounted.
func (c *emulated_mia) sdRequireMounted() (bool, uint8) {
	if c.sd.mounted {
		return true, frOK
	}
	return c.sdMountFilesystem()
}

func (c *emulated_mia) sdRootIsDir() bool {
	if c.sd.rootDir == "" {
		return false
	}
	info, err := os.Stat(c.sd.rootDir)
	return err == nil && info.IsDir()
}

func (c *emulated_mia) sdBlockRead(lba uint32) bool {
	if !c.sd.initialized || lba >= c.sd.sectors {
		return false
	}

	dst := uint32(miaSDSectorOffset)
	if sector, found := c.sd.rawSectors[lba]; found {
		copy(c.memory[dst:dst+miaSDSectorSize], sector)
	} else {
		clear(c.memory[dst : dst+miaSDSectorSize])
	}
	return true
}

func (c *emulated_mia) sdBlockWrite(lba uint32) bool {
	if !c.sd.initialized || lba >= c.sd.sectors {
		return false
	}

	if c.sd.rawSectors == nil {
		c.sd.rawSectors = make(map[uint32][]uint8)
	}
	src := uint32(miaSDSectorOffset)
	sector := make([]uint8, miaSDSectorSize)
	copy(sector, c.memory[src:src+miaSDSectorSize])
	c.sd.rawSectors[lba] = sector
	return true
}

// sdLoadToRAM mirrors sd_load_to_ram: it streams a file into MIA RAM, marking the
// touched range dirty so any overlap with the syncable video region is sent.
func (c *emulated_mia) sdLoadToRAM(dest, maxLen uint32) (uint32, bool, uint8) {
	hostPath, ok := c.sdResolveHostPath()
	if !ok {
		return 0, false, frInvalidName
	}

	file, err := os.Open(hostPath)
	if err != nil {
		return 0, false, sdErrToFresult(err)
	}
	defer file.Close()

	// Mirror sd_start_load_job: publish the file size before streaming so a 6502
	// program can compare SD_FILE_POS against SD_FILE_SIZE once the load completes.
	if info, statErr := file.Stat(); statErr == nil {
		c.sdWriteU32(miaSDControlOffset+miaSDControlFileSize0, uint32(info.Size()))
	}

	var total uint32
	temp := make([]uint8, sdJobChunkSize)

	for dest < miaRAMSize {
		remainingRAM := uint32(miaRAMSize) - dest
		var remainingLimit uint32
		if maxLen == 0 {
			remainingLimit = remainingRAM
		} else {
			remainingLimit = maxLen - total
		}
		if remainingLimit == 0 || remainingRAM == 0 {
			break
		}

		chunk := uint32(len(temp))
		if chunk > remainingLimit {
			chunk = remainingLimit
		}
		if chunk > remainingRAM {
			chunk = remainingRAM
		}

		n, rerr := file.Read(temp[:chunk])
		if n > 0 {
			copy(c.memory[dest:dest+uint32(n)], temp[:n])
			c.videoMarkDirtyRange(dest, uint32(n))
			dest += uint32(n)
			total += uint32(n)
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return total, false, sdErrToFresult(rerr)
		}
		if n == 0 {
			break
		}
	}

	if pos, err := file.Seek(0, io.SeekCurrent); err == nil {
		if info, err := file.Stat(); err == nil {
			c.sd.eof = pos >= info.Size()
		}
	}

	return total, true, frOK
}

// sdLoadPart mirrors the firmware FS_LOAD_PART job (sd_start_part_job +
// sd_job_step_load_part): rows of rowLen bytes, row r read from file offset
// offset + r*fileStride into MIA RAM at dest + r*ramStride; rows 0 loads one
// contiguous run. Reaching EOF ends the load early and still succeeds, and the
// loaded byte count lands in SD_FILE_POS, as with FS_LOAD_TO_MIA_RAM. The
// firmware spreads the job over core-0 service calls; the emulator runs it in
// one go.
func (c *emulated_mia) sdLoadPart(dest, rowLen, offset, rows, fileStride, ramStride uint32) (uint32, bool, uint8, uint8) {
	if rows == 0 {
		rows = 1
	}
	end := uint64(dest) + uint64(rows-1)*uint64(ramStride) + uint64(rowLen)
	if rowLen == 0 || end > miaRAMSize {
		return 0, false, frInvalidParameter, miaErrorFSInvalidRequest
	}

	hostPath, ok := c.sdResolveHostPath()
	if !ok {
		return 0, false, frInvalidName, miaErrorFSOpenFailed
	}
	file, err := os.Open(hostPath)
	if err != nil {
		return 0, false, sdErrToFresult(err), miaErrorFSOpenFailed
	}
	defer file.Close()

	var size int64
	if info, statErr := file.Stat(); statErr == nil {
		size = info.Size()
		c.sdWriteU32(miaSDControlOffset+miaSDControlFileSize0, uint32(size))
	}

	var total uint32
	for r := uint32(0); r < rows; r++ {
		pos := int64(offset) + int64(r)*int64(fileStride)
		if _, err := file.Seek(pos, io.SeekStart); err != nil {
			return total, false, sdErrToFresult(err), miaErrorFSSeekFailed
		}
		at := dest + r*ramStride
		n, err := io.ReadFull(file, c.memory[at:at+rowLen])
		if n > 0 {
			c.videoMarkDirtyRange(at, uint32(n))
			total += uint32(n)
		}
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			c.sd.eof = true
			return total, true, frOK, 0
		}
		if err != nil {
			return total, false, sdErrToFresult(err), miaErrorFSReadFailed
		}
		c.sd.eof = pos+int64(rowLen) >= size
	}

	return total, true, frOK, 0
}

// sdServiceSave mirrors the firmware FS_SAVE_FROM_MIA_RAM job (sd_start_save_job +
// sd_job_step_save + sd_finish_job): it opens the path with the requested write
// mode, streams SD_TRANSFER_LEN bytes from MIA RAM at SD_DEST_ADDR into the file,
// records the saved byte count, then closes the file. On the Pico this is a
// chunked job spread across core-0 service calls (see sdJobChunkSize); the emulator
// runs it in one synchronous transfer. The save uses its own file handle and never
// touches the implicit FS_OPEN file, exactly like the firmware job.
func (c *emulated_mia) sdServiceSave() (bool, uint8, uint8) {
	if ok, fr := c.sdRequireMounted(); !ok {
		return false, fr, miaErrorFSMountFailed
	}

	source := c.sdControlDestAddr()
	length := c.sdControlTransferLen()
	if source > miaRAMSize || length > miaRAMSize-source {
		return false, frInvalidParameter, miaErrorFSInvalidRequest
	}

	// Reject read-only opens: a save must be able to write. Mirrors the firmware
	// check that the resolved FatFs mode carries FA_WRITE.
	openMode := c.memory[miaSDControlOffset+miaSDControlOpenMode]
	flag, ok := sdOpenModeToFlag(openMode)
	if !ok || openMode == miaFSOpenRead {
		return false, frDenied, miaErrorFSInvalidRequest
	}

	hostPath, ok := c.sdResolveHostPath()
	if !ok {
		return false, frInvalidName, miaErrorFSOpenFailed
	}

	file, err := os.OpenFile(hostPath, flag, 0o644)
	if err != nil {
		return false, sdErrToFresult(err), miaErrorFSOpenFailed
	}

	// Mirror sd_start_save_job: reset progress and publish the open-time file size.
	c.sdWriteU16(miaSDControlOffset+miaSDControlResultLenL, 0)
	c.sdWriteU32(miaSDControlOffset+miaSDControlFilePos0, 0)
	if info, statErr := file.Stat(); statErr == nil {
		c.sdWriteU32(miaSDControlOffset+miaSDControlFileSize0, uint32(info.Size()))
	}

	saveOK := true
	fr := frOK
	errorCode := miaErrorFSWriteFailed

	n, werr := file.Write(c.memory[source : source+length])
	saved := uint32(n)
	c.sdWriteU16(miaSDControlOffset+miaSDControlResultLenL, uint16(saved))
	c.sdWriteU32(miaSDControlOffset+miaSDControlFilePos0, saved)
	if werr != nil || saved != length {
		saveOK = false
		fr = sdErrToFresult(werr)
	}

	// On success record the final file size (sd_finish_job writes f_size).
	if saveOK {
		if info, statErr := file.Stat(); statErr == nil {
			c.sdWriteU32(miaSDControlOffset+miaSDControlFileSize0, uint32(info.Size()))
		}
	}

	if closeErr := file.Close(); closeErr != nil && saveOK {
		saveOK = false
		fr = sdErrToFresult(closeErr)
		errorCode = miaErrorFSCloseFailed
	}

	return saveOK, fr, errorCode
}

/**************************************************************************************************
 * Host path resolution and directory entries
 **************************************************************************************************/

// sdReadPathStringFrom reads a null-terminated path from a buffer and normalizes
// backslashes to forward slashes.
func (c *emulated_mia) sdReadPathStringFrom(offset uint32, size int) string {
	var b strings.Builder
	for i := 0; i < size; i++ {
		value := c.memory[offset+uint32(i)]
		if value == 0 {
			break
		}
		if value == '\\' {
			value = '/'
		}
		b.WriteByte(value)
	}
	return b.String()
}

// sdResolveHostPathFrom maps a 6502 path buffer onto a host path under rootDir. The
// path is cleaned against "/" so ".." segments can never escape the SD root.
func (c *emulated_mia) sdResolveHostPathFrom(offset uint32, size int) (string, bool) {
	if c.sd.rootDir == "" {
		return "", false
	}

	raw := c.sdReadPathStringFrom(offset, size)
	// An absolute path (leading "/" or "\", mirroring firmware's own "\" ->
	// "/" translation in sd_prepare_fatfs_path_from) is rooted at the SD
	// card root regardless of cwd - "0:/x" always means the volume root in
	// FatFs terms. A relative (or empty) path resolves against cwd instead,
	// same as FatFs resolves bare "0:x" (or "0:" alone) against the mounted
	// work area's current directory. See the cwd field's own comment.
	base := "/"
	if len(raw) == 0 || (raw[0] != '/' && raw[0] != '\\') {
		base = "/" + c.sd.cwd
	}
	cleaned := path.Clean(base + "/" + raw)
	hostPath := filepath.Join(c.sd.rootDir, filepath.FromSlash(cleaned))
	return hostPath, true
}

// sdResolveHostPath resolves the primary path buffer ($E2).
func (c *emulated_mia) sdResolveHostPath() (string, bool) {
	return c.sdResolveHostPathFrom(miaFSPathOffset, miaFSPathSize)
}

// sdResolveHostPath2 resolves the second path buffer ($E5, used by FS_RENAME).
func (c *emulated_mia) sdResolveHostPath2() (string, bool) {
	return c.sdResolveHostPathFrom(miaFSPath2Offset, miaFSPath2Size)
}

func (c *emulated_mia) sdClearDirEntry() {
	clear(c.memory[miaFSDirEntryOffset : miaFSDirEntryOffset+miaFSDirEntrySize])
}

// sdFillDirEntry writes a directory entry record from an os.DirEntry (FS_READDIR).
func (c *emulated_mia) sdFillDirEntry(entry os.DirEntry) {
	var size uint32
	var mod time.Time
	if info, err := entry.Info(); err == nil {
		mod = info.ModTime()
		if !entry.IsDir() {
			size = uint32(info.Size())
		}
	}
	c.sdWriteDirEntry(entry.Name(), entry.IsDir(), size, mod)
}

// sdWriteDirEntry mirrors sd_fill_dir_entry, writing one directory entry record
// from scalar fields. It backs both FS_READDIR and FS_STAT.
func (c *emulated_mia) sdWriteDirEntry(name string, isDir bool, size uint32, mod time.Time) {
	c.sdClearDirEntry()

	maxName := miaFSDirEntrySize - miaFSDirName - 1
	if len(name) > maxName {
		name = name[:maxName]
	}
	for i := 0; i < len(name); i++ {
		c.memory[miaFSDirEntryOffset+miaFSDirName+i] = name[i]
	}

	attr := miaFSDirAttrArchive
	if isDir {
		attr = miaFSDirAttrDirectory
		size = 0
	}
	fdate, ftime := sdFatDateTime(mod)

	c.memory[miaFSDirEntryOffset+miaFSDirAttr] = attr
	c.memory[miaFSDirEntryOffset+miaFSDirNameLen] = uint8(len(name))
	c.sdWriteU32(miaFSDirEntryOffset+miaFSDirSize0, size)
	c.sdWriteU16(miaFSDirEntryOffset+miaFSDirDateL, fdate)
	c.sdWriteU16(miaFSDirEntryOffset+miaFSDirTimeL, ftime)
}

// sdQueryFree synthesizes FAT free-space figures for the host folder. A folder has
// no real cluster map, so total clusters come from the virtual geometry and used
// clusters from the folder's current byte usage. Mirrors the firmware FS_GET_FREE
// outputs (free/total clusters, sectors per cluster).
func (c *emulated_mia) sdQueryFree() (free, total uint32, clusterSectors uint16) {
	clusterSectors = miaSDClusterSectors
	total = miaSDVirtualSectors / uint32(clusterSectors)
	bytesPerCluster := uint64(clusterSectors) * miaSDSectorSize

	var used uint64
	_ = filepath.WalkDir(c.sd.rootDir, func(_ string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, infoErr := d.Info(); infoErr == nil {
			used += uint64(info.Size())
		}
		return nil
	})

	usedClusters := uint32((used + bytesPerCluster - 1) / bytesPerCluster)
	if usedClusters > total {
		usedClusters = total
	}
	free = total - usedClusters
	return free, total, clusterSectors
}

// sdUpdateFilePosition publishes SD_FILE_SIZE/SD_FILE_POS for the given slot
// (pass anything >= miaSDMaxHandles, or a closed slot, to publish the "no file"
// zeros).
func (c *emulated_mia) sdUpdateFilePosition(slot uint8) {
	if int(slot) >= miaSDMaxHandles || !c.sd.fileOpen[slot] || c.sd.file[slot] == nil {
		c.sdWriteU32(miaSDControlOffset+miaSDControlFileSize0, 0)
		c.sdWriteU32(miaSDControlOffset+miaSDControlFilePos0, 0)
		return
	}

	var size int64
	if info, err := c.sd.file[slot].Stat(); err == nil {
		size = info.Size()
	}
	pos, _ := c.sd.file[slot].Seek(0, io.SeekCurrent)

	c.sdWriteU32(miaSDControlOffset+miaSDControlFileSize0, uint32(size))
	c.sdWriteU32(miaSDControlOffset+miaSDControlFilePos0, uint32(pos))
}

func (c *emulated_mia) sdFileAtEOF(slot uint8) bool {
	if int(slot) >= miaSDMaxHandles || c.sd.file[slot] == nil {
		return false
	}
	pos, err := c.sd.file[slot].Seek(0, io.SeekCurrent)
	if err != nil {
		return false
	}
	info, err := c.sd.file[slot].Stat()
	if err != nil {
		return false
	}
	return pos >= info.Size()
}

// sdOpenModeToFlag maps a MIA open mode to host os.OpenFile flags, mirroring
// sd_open_mode_to_fatfs.
func sdOpenModeToFlag(openMode uint8) (int, bool) {
	switch openMode {
	case miaFSOpenRead:
		return os.O_RDONLY, true
	case miaFSOpenWriteCreate:
		return os.O_WRONLY | os.O_CREATE | os.O_TRUNC, true
	case miaFSOpenWriteAppend:
		return os.O_WRONLY | os.O_CREATE | os.O_APPEND, true
	case miaFSOpenReadWrite:
		return os.O_RDWR | os.O_CREATE, true
	default:
		return 0, false
	}
}

// sdErrToFresult approximates a FatFs FRESULT from a host error.
func sdErrToFresult(err error) uint8 {
	switch {
	case err == nil:
		return frOK
	case errors.Is(err, os.ErrNotExist):
		return frNoFile
	case errors.Is(err, os.ErrPermission):
		return frDenied
	case errors.Is(err, os.ErrExist):
		return frExist
	default:
		return frDiskErr
	}
}

// sdFatDateTime packs a timestamp into FAT date/time words.
func sdFatDateTime(t time.Time) (uint16, uint16) {
	year := t.Year()
	if year < 1980 {
		return 0, 0
	}
	date := uint16((year-1980)<<9) | uint16(int(t.Month())<<5) | uint16(t.Day())
	clock := uint16(t.Hour()<<11) | uint16(t.Minute()<<5) | uint16(t.Second()/2)
	return date, clock
}

/**************************************************************************************************
 * Index configuration (mirrors sd_configure_indexes)
 **************************************************************************************************/

func (c *emulated_mia) sdConfigureIndexes() {
	c.sdConfigureIndex(miaSDIndexControl, miaSDControlOffset, miaSDControlSize)
	c.sdConfigureIndex(miaSDIndexSector, miaSDSectorOffset, miaSDSectorSize)
	c.sdConfigureIndex(miaFSIndexPath, miaFSPathOffset, miaFSPathSize)
	c.sdConfigureIndex(miaFSIndexDirEntry, miaFSDirEntryOffset, miaFSDirEntrySize)
	c.sdConfigureIndex(miaFSIndexTransfer, miaFSTransferOffset, miaFSTransferSize)
	c.sdConfigureIndex(miaFSIndexPath2, miaFSPath2Offset, miaFSPath2Size)
}

func (c *emulated_mia) sdConfigureIndex(indexID uint8, start, length uint32) {
	c.indexes[indexID] = miaIndex{
		currentAddr: start,
		defaultAddr: start,
		limitAddr:   start + length,
		step:        1,
		flags: (1 << miaIndexFlagReadStep) |
			(1 << miaIndexFlagWriteStep) |
			(1 << miaIndexFlagWrap),
	}
}

/**************************************************************************************************
 * Console diagnostics
 **************************************************************************************************/

// consoleSD controls and reports the SD/FS subsystem, mirroring cmd_sd.
func (c *emulated_mia) consoleSD(args string) string {
	args = strings.TrimSpace(args)
	if args == "" || args == "status" {
		return c.consoleSDDetail() + "Usage: sd [status|init|mount]\n"
	}

	var command uint8
	var name string
	switch args {
	case "init":
		command = miaCmdSDInit
		name = "init"
	case "mount":
		command = miaCmdFSMount
		name = "mount"
	default:
		return "Usage: sd [status|init|mount]\n"
	}

	c.mu.Lock()
	ok := c.sdRequest(command)
	c.mu.Unlock()

	if ok {
		return fmt.Sprintf("SD: %s requested\n", name)
	}
	return "SD: busy or invalid request\n"
}

// consoleSDSummary renders the one-line SD summary for the status dashboard,
// mirroring mia_sd_print_summary.
func (c *emulated_mia) consoleSDSummary() string {
	c.mu.Lock()
	initialized := c.sd.initialized
	cardType := c.sd.cardType
	mounted := c.sd.mounted
	c.mu.Unlock()

	ready := "not ready"
	if initialized {
		ready = "ready"
	}
	fsState := "unmounted"
	if mounted {
		fsState = "mounted"
	}

	return fmt.Sprintf("SD: %s  card:%s  fs:%s\n", ready, sdCardTypeName(cardType), fsState)
}

// consoleSDDetail renders the SD/FS subsystem detail, mirroring mia_sd_print_status.
// The firmware SPI-pin line is replaced by a host-folder backend line.
func (c *emulated_mia) consoleSDDetail() string {
	c.mu.Lock()
	initialized := c.sd.initialized
	mounted := c.sd.mounted
	busy := c.status()&miaStatusSDBusy != 0
	sel, selValid := c.sdRequireValidHandle()
	selOpen := selValid && c.sd.fileOpen[sel]
	selEOF := c.sd.eof
	if selOpen {
		selEOF = c.sdFileAtEOF(sel)
	}
	openCount := 0
	for i := 0; i < miaSDMaxHandles; i++ {
		if c.sd.fileOpen[i] {
			openCount++
		}
	}
	dirOpen := c.sd.dirOpen
	cardType := c.sd.cardType
	sectors := c.sd.sectors
	rootDir := c.sd.rootDir
	statusByte := c.memory[miaSDControlOffset+miaSDControlStatus]
	lastError := c.sd.lastError
	fatfs := c.sd.lastFatfsResult
	lba := c.sdControlLBA()
	requestLen := c.sdControlRequestLen()
	resultLen := c.sdReadU16(miaSDControlOffset + miaSDControlResultLenL)
	destAddr := c.sdControlDestAddr()
	fileSize := c.sdReadU32(miaSDControlOffset + miaSDControlFileSize0)
	filePos := c.sdReadU32(miaSDControlOffset + miaSDControlFilePos0)
	openMode := uint8(miaFSOpenRead)
	if selValid {
		openMode = c.sd.currentOpenMode[sel]
	}
	requestedOpenMode := c.memory[miaSDControlOffset+miaSDControlOpenMode]
	freeClusters := c.sdReadU32(miaSDControlOffset + miaSDControlFreeClusters0)
	totalClusters := c.sdReadU32(miaSDControlOffset + miaSDControlTotalClusters0)
	clusterSectors := c.sdReadU16(miaSDControlOffset + miaSDControlClusterSectorsL)
	transferLen := c.sdControlTransferLen()
	c.mu.Unlock()

	var out strings.Builder
	out.WriteString("SD/FS:\n")
	// The firmware reports the active SD job type here; the emulator runs every SD
	// command synchronously, so no job is ever in flight when the console reads
	// status, hence job:none.
	fmt.Fprintf(&out, "  state:     initialized:%s mounted:%s busy:%s dir:%s job:none\n",
		yesNo(initialized), yesNo(mounted), yesNo(busy), openClosed(dirOpen))
	fmt.Fprintf(&out, "  handles:   max:%d open:%d selected:%d (%s) selected-file:%s eof:%s\n",
		miaSDMaxHandles, openCount, sel, validInvalid(selValid), openClosed(selOpen), yesNo(selEOF))
	fmt.Fprintf(&out, "  card:      type:%d sectors:%d capacity:%d MiB\n",
		cardType, sectors, sectors/2048)
	fmt.Fprintf(&out, "  backend:   folder:%s\n", bindOrNone(rootDir))
	fmt.Fprintf(&out, "  block:     control:$%05X-$%05X sector:$%05X-$%05X path:$%05X-$%05X\n",
		miaSDControlOffset, miaSDControlOffset+miaSDControlSize-1,
		miaSDSectorOffset, miaSDSectorOffset+miaSDSectorSize-1,
		miaFSPathOffset, miaFSPathOffset+miaFSPathSize-1)
	fmt.Fprintf(&out, "             dir:$%05X-$%05X transfer:$%05X-$%05X\n",
		miaFSDirEntryOffset, miaFSDirEntryOffset+miaFSDirEntrySize-1,
		miaFSTransferOffset, miaFSTransferOffset+miaFSTransferSize-1)
	fmt.Fprintf(&out, "  indexes:   control:$%02X sector:$%02X path:$%02X dir:$%02X transfer:$%02X path2:$%02X\n",
		miaSDIndexControl, miaSDIndexSector, miaFSIndexPath, miaFSIndexDirEntry, miaFSIndexTransfer, miaFSIndexPath2)
	fmt.Fprintf(&out, "  control:   status:0x%02X last-error:0x%02X fatfs:%d lba:%d req:%d result:%d dest:$%05X\n",
		statusByte, lastError, fatfs, lba, requestLen, resultLen, destAddr)
	fmt.Fprintf(&out, "  file:      size:%d pos:%d\n", fileSize, filePos)
	fmt.Fprintf(&out, "             mode:%d requested-open-mode:%d\n", openMode, requestedOpenMode)
	fmt.Fprintf(&out, "  free:      clusters:%d/%d cluster-sectors:%d\n",
		freeClusters, totalClusters, clusterSectors)
	fmt.Fprintf(&out, "  service:   chunk:%d budget:%d us transfer-len:%d\n",
		sdJobChunkSize, miaSDServiceBudgetUS, transferLen)

	return out.String()
}

func sdCardTypeName(cardType uint8) string {
	switch cardType {
	case miaSDCardSDV1:
		return "SD v1"
	case miaSDCardSDV2:
		return "SD v2"
	case miaSDCardSDHC:
		return "SDHC/SDXC"
	default:
		return "none"
	}
}

func validInvalid(valid bool) string {
	if valid {
		return "valid"
	}
	return "invalid"
}

func openClosed(open bool) string {
	if open {
		return "open"
	}
	return "closed"
}
