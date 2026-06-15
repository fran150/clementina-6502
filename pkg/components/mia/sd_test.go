package mia

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newSDTestCircuit builds a normal-mode MIA test circuit, optionally backed by a
// host folder acting as the emulated SD card.
func newSDTestCircuit(t *testing.T, folder string) *emulatedMiaTestCircuit {
	t.Helper()

	circuit := newEmulatedMiaTestCircuit()
	circuit.chip.state = miaStateNormal
	if folder != "" {
		circuit.chip.SetSDFolder(folder)
	}

	return circuit
}

// sdWritePath stores a null-terminated path into the path buffer.
func sdWritePath(chip *emulated_mia, p string) {
	clear(chip.memory[miaFSPathOffset : miaFSPathOffset+miaFSPathSize])
	copy(chip.memory[miaFSPathOffset:miaFSPathOffset+miaFSPathSize-1], p)
}

// sdWritePath2 stores a null-terminated path into the second path buffer (PATH2,
// used as the FS_RENAME destination).
func sdWritePath2(chip *emulated_mia, p string) {
	clear(chip.memory[miaFSPath2Offset : miaFSPath2Offset+miaFSPath2Size])
	copy(chip.memory[miaFSPath2Offset:miaFSPath2Offset+miaFSPath2Size-1], p)
}

// sdDirEntryName reads the directory-entry name field.
func sdDirEntryName(chip *emulated_mia) string {
	nameLen := chip.memory[miaFSDirEntryOffset+miaFSDirNameLen]
	return string(chip.memory[miaFSDirEntryOffset+miaFSDirName : miaFSDirEntryOffset+miaFSDirName+uint32(nameLen)])
}

func sdControlByte(chip *emulated_mia, field uint32) uint8 {
	return chip.memory[miaSDControlOffset+field]
}

// TestEmulatedMiaSDResetDefaults verifies the control block defaults and the fixed
// $E0-$E4 indexes match the firmware reset state, with no card attached.
func TestEmulatedMiaSDResetDefaults(t *testing.T) {
	chip := newEmulatedMiaTestCircuit().chip

	assert.Equal(t, uint8(miaSDVersion), sdControlByte(chip, miaSDControlVersion))
	assert.Equal(t, uint16(miaFSTransferSize), chip.sdReadU16(miaSDControlOffset+miaSDControlRequestLenL))
	assert.False(t, chip.sd.initialized)
	assert.False(t, chip.sd.mounted)

	expected := []struct {
		id    uint8
		start uint32
		size  uint32
	}{
		{miaSDIndexControl, miaSDControlOffset, miaSDControlSize},
		{miaSDIndexSector, miaSDSectorOffset, miaSDSectorSize},
		{miaFSIndexPath, miaFSPathOffset, miaFSPathSize},
		{miaFSIndexDirEntry, miaFSDirEntryOffset, miaFSDirEntrySize},
		{miaFSIndexTransfer, miaFSTransferOffset, miaFSTransferSize},
	}

	for _, e := range expected {
		idx := chip.indexes[e.id]
		assert.Equalf(t, e.start, idx.currentAddr, "index $%02X current", e.id)
		assert.Equalf(t, e.start+e.size, idx.limitAddr, "index $%02X limit", e.id)
		assert.Equalf(t, uint16(1), idx.step, "index $%02X step", e.id)
		assert.NotZerof(t, idx.flags&(1<<miaIndexFlagWrap), "index $%02X wrap", e.id)
	}
}

// TestEmulatedMiaSDInitNoCard verifies SD_INIT fails when no folder is attached,
// reporting ERROR_SD_INIT_FAILED and IRQ_SD_ERROR with no present bit.
func TestEmulatedMiaSDInitNoCard(t *testing.T) {
	circuit := newSDTestCircuit(t, "")
	chip := circuit.chip

	circuit.write(miaRegCmdTrigger, miaCmdSDInit)

	assert.False(t, chip.sd.initialized)
	assert.Zero(t, chip.status()&miaStatusSDPresent)
	assert.Zero(t, chip.status()&miaStatusSDBusy)
	assert.Equal(t, miaErrorSDInitFailed, sdControlByte(chip, miaSDControlLastError))
	assert.NotZero(t, chip.irqStatus()&miaIRQSDError)
	assert.Zero(t, chip.irqStatus()&miaIRQSDDone)
	assert.Equal(t, miaErrorSDInitFailed, chip.readRegister(miaRegErrorLSB))
}

// TestEmulatedMiaSDInitWithFolder verifies SD_INIT succeeds against a host folder,
// reporting a present SDHC card and IRQ_SD_DONE.
func TestEmulatedMiaSDInitWithFolder(t *testing.T) {
	circuit := newSDTestCircuit(t, t.TempDir())
	chip := circuit.chip

	circuit.write(miaRegCmdTrigger, miaCmdSDInit)

	assert.True(t, chip.sd.initialized)
	assert.NotZero(t, chip.status()&miaStatusSDPresent)
	assert.Zero(t, chip.status()&miaStatusSDBusy, "busy clears synchronously")
	assert.Equal(t, uint8(0), sdControlByte(chip, miaSDControlLastError))
	assert.Equal(t, miaSDCardSDHC, sdControlByte(chip, miaSDControlCardType))
	assert.Equal(t, miaSDVirtualSectors, chip.sdReadU32(miaSDControlOffset+miaSDControlCardSectors0))
	assert.NotZero(t, chip.irqStatus()&miaIRQSDDone)
	assert.Zero(t, chip.irqStatus()&miaIRQSDError)

	status := sdControlByte(chip, miaSDControlStatus)
	assert.NotZero(t, status&miaSDStatusPresent)
	assert.NotZero(t, status&miaSDStatusInitialized)
}

// TestEmulatedMiaFSMount verifies FS_MOUNT mounts the folder, sets FS_MOUNTED and
// raises both the done and FS-event IRQs.
func TestEmulatedMiaFSMount(t *testing.T) {
	circuit := newSDTestCircuit(t, t.TempDir())
	chip := circuit.chip

	circuit.write(miaRegCmdTrigger, miaCmdFSMount)

	assert.True(t, chip.sd.mounted)
	assert.NotZero(t, chip.status()&miaStatusFSMounted)
	assert.NotZero(t, chip.status()&miaStatusSDPresent, "mount initializes the card")
	assert.Equal(t, uint8(0), sdControlByte(chip, miaSDControlLastError))
	assert.NotZero(t, chip.irqStatus()&miaIRQSDDone)
	assert.NotZero(t, chip.irqStatus()&miaIRQFSEvent)
}

// TestEmulatedMiaFSLoadToMiaRAM verifies a whole-file load into MIA RAM and the
// reported byte count / EOF.
func TestEmulatedMiaFSLoadToMiaRAM(t *testing.T) {
	dir := t.TempDir()
	payload := []byte("CLEMENTINA SD LOAD TEST")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "DEMO.PRG"), payload, 0o644))

	circuit := newSDTestCircuit(t, dir)
	chip := circuit.chip

	sdWritePath(chip, "/DEMO.PRG")
	chip.sdWriteU16(miaSDControlOffset+miaSDControlRequestLenL, 0) // load until EOF
	dest := uint32(0x08000)
	chip.memory[miaSDControlOffset+miaSDControlDestAddrL] = uint8(dest)
	chip.memory[miaSDControlOffset+miaSDControlDestAddrL+1] = uint8(dest >> 8)
	chip.memory[miaSDControlOffset+miaSDControlDestAddrL+2] = uint8(dest >> 16)

	circuit.write(miaRegCmdTrigger, miaCmdFSLoadToMiaRAM)

	assert.Equal(t, uint8(0), sdControlByte(chip, miaSDControlLastError))
	assert.Equal(t, uint16(len(payload)), chip.sdReadU16(miaSDControlOffset+miaSDControlResultLenL))
	assert.Equal(t, uint32(len(payload)), chip.sdReadU32(miaSDControlOffset+miaSDControlFilePos0))
	assert.NotZero(t, sdControlByte(chip, miaSDControlEOF))
	assert.Equal(t, payload, chip.memory[dest:dest+uint32(len(payload))])
}

// TestEmulatedMiaFSOpenReadStreamClose exercises open/read/close and streams the
// transfer buffer back through index $E4 as a 6502 program would.
func TestEmulatedMiaFSOpenReadStreamClose(t *testing.T) {
	dir := t.TempDir()
	payload := []byte("HELLO SD\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "READ.TXT"), payload, 0o644))

	circuit := newSDTestCircuit(t, dir)
	chip := circuit.chip

	circuit.write(miaRegCmdTrigger, miaCmdFSMount)
	require.True(t, chip.sd.mounted)

	sdWritePath(chip, "/READ.TXT")
	chip.memory[miaSDControlOffset+miaSDControlOpenMode] = miaFSOpenRead
	circuit.write(miaRegCmdTrigger, miaCmdFSOpen)
	require.True(t, chip.sd.fileOpen)
	require.Equal(t, uint8(0), sdControlByte(chip, miaSDControlLastError))
	assert.Equal(t, uint32(len(payload)), chip.sdReadU32(miaSDControlOffset+miaSDControlFileSize0))

	chip.sdWriteU16(miaSDControlOffset+miaSDControlRequestLenL, 0) // full buffer
	circuit.write(miaRegCmdTrigger, miaCmdFSRead)

	resultLen := chip.sdReadU16(miaSDControlOffset + miaSDControlResultLenL)
	require.Equal(t, uint16(len(payload)), resultLen)
	assert.NotZero(t, sdControlByte(chip, miaSDControlEOF))

	// Stream the transfer buffer through index $E4 like the 6502 would.
	circuit.write(miaRegIdxASelector, miaFSIndexTransfer)
	got := make([]byte, resultLen)
	for i := range got {
		got[i] = circuit.read(miaRegIdxAPort)
	}
	assert.Equal(t, payload, got)

	circuit.write(miaRegCmdTrigger, miaCmdFSClose)
	assert.False(t, chip.sd.fileOpen)
	assert.Equal(t, uint8(0), sdControlByte(chip, miaSDControlLastError))
}

// TestEmulatedMiaFSWriteCreate verifies creating, writing, and syncing a save file
// lands the bytes on the host folder.
func TestEmulatedMiaFSWriteCreate(t *testing.T) {
	dir := t.TempDir()
	circuit := newSDTestCircuit(t, dir)
	chip := circuit.chip

	circuit.write(miaRegCmdTrigger, miaCmdFSMount)
	require.True(t, chip.sd.mounted)

	sdWritePath(chip, "/STATE.BIN")
	chip.memory[miaSDControlOffset+miaSDControlOpenMode] = miaFSOpenWriteCreate
	circuit.write(miaRegCmdTrigger, miaCmdFSOpen)
	require.True(t, chip.sd.fileOpen)

	payload := []byte{0x43, 0x4C, 0x45, 0x4D, 0x01, 0x00, 0x00, 0x00}
	copy(chip.memory[miaFSTransferOffset:], payload)
	chip.sdWriteU16(miaSDControlOffset+miaSDControlRequestLenL, uint16(len(payload)))
	circuit.write(miaRegCmdTrigger, miaCmdFSWrite)
	assert.Equal(t, uint16(len(payload)), chip.sdReadU16(miaSDControlOffset+miaSDControlResultLenL))

	circuit.write(miaRegCmdTrigger, miaCmdFSSync)
	assert.Equal(t, uint8(0), sdControlByte(chip, miaSDControlLastError))

	circuit.write(miaRegCmdTrigger, miaCmdFSClose)

	saved, err := os.ReadFile(filepath.Join(dir, "STATE.BIN"))
	require.NoError(t, err)
	assert.Equal(t, payload, saved)
}

// TestEmulatedMiaFSDirectoryListing verifies opendir/readdir enumerate the folder
// and signal EOF when exhausted.
func TestEmulatedMiaFSDirectoryListing(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "FILE.TXT"), []byte("abc"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "SUBDIR"), 0o755))

	circuit := newSDTestCircuit(t, dir)
	chip := circuit.chip

	sdWritePath(chip, "/")
	circuit.write(miaRegCmdTrigger, miaCmdFSOpendir)
	require.True(t, chip.sd.dirOpen)
	require.Equal(t, uint8(0), sdControlByte(chip, miaSDControlLastError))

	names := map[string]uint8{}
	for i := 0; i < 8; i++ {
		circuit.write(miaRegCmdTrigger, miaCmdFSReaddir)
		require.Equal(t, uint8(0), sdControlByte(chip, miaSDControlLastError))
		if sdControlByte(chip, miaSDControlEOF) != 0 {
			break
		}
		nameLen := chip.memory[miaFSDirEntryOffset+miaFSDirNameLen]
		name := string(chip.memory[miaFSDirEntryOffset+miaFSDirName : miaFSDirEntryOffset+miaFSDirName+uint32(nameLen)])
		names[name] = chip.memory[miaFSDirEntryOffset+miaFSDirAttr]
	}

	require.Contains(t, names, "FILE.TXT")
	require.Contains(t, names, "SUBDIR")
	assert.Zero(t, names["FILE.TXT"]&miaFSDirAttrDirectory)
	assert.NotZero(t, names["SUBDIR"]&miaFSDirAttrDirectory)
}

// TestEmulatedMiaSDRawSectorRoundTrip verifies raw sector write then read back
// against the virtual block device.
func TestEmulatedMiaSDRawSectorRoundTrip(t *testing.T) {
	circuit := newSDTestCircuit(t, t.TempDir())
	chip := circuit.chip

	circuit.write(miaRegCmdTrigger, miaCmdSDInit)
	require.True(t, chip.sd.initialized)

	for i := 0; i < miaSDSectorSize; i++ {
		chip.memory[miaSDSectorOffset+i] = uint8(i*7 + 1)
	}
	chip.sdWriteU32(miaSDControlOffset+miaSDControlLBA0, 5)
	circuit.write(miaRegCmdTrigger, miaCmdSDWriteSector)
	require.Equal(t, uint8(0), sdControlByte(chip, miaSDControlLastError))
	assert.Equal(t, uint16(miaSDSectorSize), chip.sdReadU16(miaSDControlOffset+miaSDControlResultLenL))

	clear(chip.memory[miaSDSectorOffset : miaSDSectorOffset+miaSDSectorSize])
	chip.sdWriteU32(miaSDControlOffset+miaSDControlLBA0, 5)
	circuit.write(miaRegCmdTrigger, miaCmdSDReadSector)
	require.Equal(t, uint8(0), sdControlByte(chip, miaSDControlLastError))

	for i := 0; i < miaSDSectorSize; i++ {
		require.Equalf(t, uint8(i*7+1), chip.memory[miaSDSectorOffset+i], "sector byte %d", i)
	}

	// An unwritten sector reads back as zero.
	chip.sdWriteU32(miaSDControlOffset+miaSDControlLBA0, 6)
	circuit.write(miaRegCmdTrigger, miaCmdSDReadSector)
	for i := 0; i < miaSDSectorSize; i++ {
		require.Zerof(t, chip.memory[miaSDSectorOffset+i], "unwritten sector byte %d", i)
	}
}

// TestEmulatedMiaFSReadNoFileOpen verifies FS_READ without an open file reports
// ERROR_FS_NO_FILE_OPEN.
func TestEmulatedMiaFSReadNoFileOpen(t *testing.T) {
	circuit := newSDTestCircuit(t, t.TempDir())
	chip := circuit.chip

	circuit.write(miaRegCmdTrigger, miaCmdFSRead)

	assert.Equal(t, miaErrorFSNoFileOpen, sdControlByte(chip, miaSDControlLastError))
	assert.NotZero(t, chip.irqStatus()&miaIRQSDError)
}

// TestEmulatedMiaSDPathStaysWithinRoot verifies a "../" path cannot escape the SD
// root: the resolved host path remains under rootDir.
func TestEmulatedMiaSDPathStaysWithinRoot(t *testing.T) {
	dir := t.TempDir()
	chip := newSDTestCircuit(t, dir).chip

	sdWritePath(chip, "/../../etc/passwd")
	hostPath, ok := chip.sdResolveHostPath()
	require.True(t, ok)

	rel, err := filepath.Rel(dir, hostPath)
	require.NoError(t, err)
	assert.False(t, filepath.IsAbs(rel))
	assert.NotContains(t, rel, "..", "resolved path must stay within the SD root")
}

// TestEmulatedMiaSDConsoleCommands exercises the terminal 'sd' command and the
// 'status sd' diagnostic.
func TestEmulatedMiaSDConsoleCommands(t *testing.T) {
	dir := t.TempDir()
	chip := newSDTestCircuit(t, dir).chip

	assert.Equal(t, "SD: init requested\n", chip.consoleSD("init"))
	assert.True(t, chip.sd.initialized)

	assert.Equal(t, "SD: mount requested\n", chip.consoleSD("mount"))
	assert.True(t, chip.sd.mounted)

	detail := chip.consoleSD("status")
	assert.Contains(t, detail, "SD/FS:\n")
	assert.Contains(t, detail, "initialized:yes mounted:yes")
	assert.Contains(t, detail, "folder:"+dir)
	assert.Contains(t, detail, "Usage: sd [status|init|mount]\n")

	assert.Contains(t, chip.consoleStatusSummary(), "SD: ready  card:SDHC/SDXC  fs:mounted\n")
	assert.Equal(t, "Usage: sd [status|init|mount]\n", chip.consoleSD("bogus"))
}

// TestEmulatedMiaFSMkdirStatDelete exercises FS_MKDIR, FS_STAT, and FS_DELETE
// against a host folder.
func TestEmulatedMiaFSMkdirStatDelete(t *testing.T) {
	dir := t.TempDir()
	circuit := newSDTestCircuit(t, dir)
	chip := circuit.chip

	circuit.write(miaRegCmdTrigger, miaCmdFSMount)
	require.True(t, chip.sd.mounted)

	sdWritePath(chip, "/NEWDIR")
	circuit.write(miaRegCmdTrigger, miaCmdFSMkdir)
	require.Equal(t, uint8(0), sdControlByte(chip, miaSDControlLastError))
	info, err := os.Stat(filepath.Join(dir, "NEWDIR"))
	require.NoError(t, err)
	require.True(t, info.IsDir())

	// FS_STAT fills the dir-entry buffer and reports its full size.
	sdWritePath(chip, "/NEWDIR")
	circuit.write(miaRegCmdTrigger, miaCmdFSStat)
	require.Equal(t, uint8(0), sdControlByte(chip, miaSDControlLastError))
	assert.Equal(t, uint16(miaFSDirEntrySize), chip.sdReadU16(miaSDControlOffset+miaSDControlResultLenL))
	assert.NotZero(t, chip.memory[miaFSDirEntryOffset+miaFSDirAttr]&miaFSDirAttrDirectory)
	assert.Equal(t, "NEWDIR", sdDirEntryName(chip))

	// FS_DELETE removes it; a subsequent FS_STAT fails.
	sdWritePath(chip, "/NEWDIR")
	circuit.write(miaRegCmdTrigger, miaCmdFSDelete)
	require.Equal(t, uint8(0), sdControlByte(chip, miaSDControlLastError))
	_, err = os.Stat(filepath.Join(dir, "NEWDIR"))
	assert.True(t, os.IsNotExist(err))

	sdWritePath(chip, "/NEWDIR")
	circuit.write(miaRegCmdTrigger, miaCmdFSStat)
	assert.Equal(t, miaErrorFSStatFailed, sdControlByte(chip, miaSDControlLastError))
	assert.NotZero(t, chip.irqStatus()&miaIRQSDError)
}

// TestEmulatedMiaFSStatFile verifies FS_STAT reports a file's size and name.
func TestEmulatedMiaFSStatFile(t *testing.T) {
	dir := t.TempDir()
	payload := []byte("0123456789")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "F.TXT"), payload, 0o644))

	circuit := newSDTestCircuit(t, dir)
	chip := circuit.chip
	circuit.write(miaRegCmdTrigger, miaCmdFSMount)

	sdWritePath(chip, "/F.TXT")
	circuit.write(miaRegCmdTrigger, miaCmdFSStat)
	require.Equal(t, uint8(0), sdControlByte(chip, miaSDControlLastError))
	assert.Zero(t, chip.memory[miaFSDirEntryOffset+miaFSDirAttr]&miaFSDirAttrDirectory)
	assert.Equal(t, uint32(len(payload)), chip.sdReadU32(miaFSDirEntryOffset+miaFSDirSize0))
	assert.Equal(t, "F.TXT", sdDirEntryName(chip))
}

// TestEmulatedMiaFSRename verifies FS_RENAME moves a file using PATH and PATH2.
func TestEmulatedMiaFSRename(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "OLD.TXT"), []byte("data"), 0o644))

	circuit := newSDTestCircuit(t, dir)
	chip := circuit.chip
	circuit.write(miaRegCmdTrigger, miaCmdFSMount)

	sdWritePath(chip, "/OLD.TXT")
	sdWritePath2(chip, "/NEW.TXT")
	circuit.write(miaRegCmdTrigger, miaCmdFSRename)
	require.Equal(t, uint8(0), sdControlByte(chip, miaSDControlLastError))

	_, err := os.Stat(filepath.Join(dir, "OLD.TXT"))
	assert.True(t, os.IsNotExist(err))
	data, err := os.ReadFile(filepath.Join(dir, "NEW.TXT"))
	require.NoError(t, err)
	assert.Equal(t, []byte("data"), data)
}

// TestEmulatedMiaFSGetFree verifies FS_GET_FREE reports the virtual FAT geometry
// and that used space reduces the free cluster count.
func TestEmulatedMiaFSGetFree(t *testing.T) {
	dir := t.TempDir()
	circuit := newSDTestCircuit(t, dir)
	chip := circuit.chip
	circuit.write(miaRegCmdTrigger, miaCmdFSMount)

	circuit.write(miaRegCmdTrigger, miaCmdFSGetFree)
	require.Equal(t, uint8(0), sdControlByte(chip, miaSDControlLastError))

	total := chip.sdReadU32(miaSDControlOffset + miaSDControlTotalClusters0)
	freeEmpty := chip.sdReadU32(miaSDControlOffset + miaSDControlFreeClusters0)
	clusterSectors := chip.sdReadU16(miaSDControlOffset + miaSDControlClusterSectorsL)

	assert.Equal(t, miaSDVirtualSectors/uint32(miaSDClusterSectors), total)
	assert.Equal(t, miaSDClusterSectors, clusterSectors)
	assert.Equal(t, total, freeEmpty, "an empty folder leaves all clusters free")
	assert.Equal(t, uint16(10), chip.sdReadU16(miaSDControlOffset+miaSDControlResultLenL))

	// A 100 KiB file consumes 4 of the 32 KiB clusters.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "BIG.BIN"), make([]byte, 100*1024), 0o644))
	circuit.write(miaRegCmdTrigger, miaCmdFSGetFree)
	freeAfter := chip.sdReadU32(miaSDControlOffset + miaSDControlFreeClusters0)
	assert.Less(t, freeAfter, freeEmpty, "used space reduces the free cluster count")
}
