package mia

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEmulatedMiaTraceFreezesAtFirstCtxError checks that the trace counts
// MIA_CTX pushes and pops and freezes the events that led to an underflow,
// the failing pop last.
func TestEmulatedMiaTraceFreezesAtFirstCtxError(t *testing.T) {
	circuit := contextTestCircuit(t)
	chip := circuit.chip

	circuit.write(miaRegContext, miaCtxPush)
	circuit.write(miaRegContext, miaCtxPop)
	circuit.write(miaRegIdxASelector, 0x10)
	circuit.write(miaRegContext, miaCtxPop) // underflow: freezes
	circuit.write(miaRegIdxASelector, 0x11) // after the capture

	out := chip.consoleTrace("")
	assert.Contains(t, out, "ctx pushes:1 pops:2 overflows:0 underflows:1")
	assert.Contains(t, out, "Captured at the first ctx error (event #4)")
	assert.Contains(t, out, "#3         W $FFF5 MIA_CTX      <- $02\n")
	assert.NotContains(t, out, "<- $11", "events after the capture are not in it")

	words := chip.traceWords()
	require.Len(t, words, 4)
	assert.Equal(t, uint32(1<<13|uint32(miaRegContext)<<8|uint32(miaCtxPop)), words[3])

	assert.Equal(t, "Trace: cleared and re-armed\n", chip.consoleTrace("clear"))
	assert.Contains(t, chip.consoleTrace(""), "No ctx error captured")
}

// TestEmulatedMiaTraceSkipsUnwatchedReads mirrors the action PIO: writes are
// always reported, reads only for every fourth register and the watched one.
func TestEmulatedMiaTraceSkipsUnwatchedReads(t *testing.T) {
	circuit := contextTestCircuit(t)
	chip := circuit.chip

	circuit.read(miaRegIdxASelector) // $FFE1: not watched in normal mode
	circuit.read(miaRegIdxAPort)     // $FFE0: watched
	circuit.read(miaRegInputChar)    // $FFF3: watched in normal mode
	assert.Len(t, chip.traceWords(), 2)
}

func TestEmulatedMiaConsoleStatusCtxAndPIO(t *testing.T) {
	circuit := contextTestCircuit(t)
	chip := circuit.chip

	circuit.write(miaRegIdxASelector, 0x10)
	circuit.write(miaRegContext, miaCtxPush)

	out := chip.consoleStatus("ctx")
	assert.Contains(t, out, "depth: 1 of 4\n")
	assert.Contains(t, out, "saved A:   16 ($10): current:$020000")

	assert.Equal(t, "Bus PIOs: not emulated (the emulator has no PIOs).\n", chip.consoleStatus("pio"))
	assert.Equal(t, "PIO: fdebug flags cleared\n", chip.consoleStatus("pio clear"))
}

func TestEmulatedMiaConsoleStatusSys(t *testing.T) {
	circuit := contextTestCircuit(t)
	chip := circuit.chip

	out := chip.consoleStatus("sys")
	assert.Contains(t, out, "System:\n")
	assert.Contains(t, out, "  uptime:   0d 00:00:")
	assert.Contains(t, out, "Core 1 bus loop: not emulated")
	assert.Equal(t, "System: peaks cleared\n", chip.consoleStatus("sys clear"))
	assert.Equal(t, "Usage: status sys [clear]\n", chip.consoleStatus("sys bogus"))
}

func TestEmulatedMiaConsoleDumpToConsole(t *testing.T) {
	circuit := contextTestCircuit(t)
	chip := circuit.chip
	circuit.write(miaRegCmdParam1, 0x5A)

	assert.Contains(t, chip.consoleDump("regs"), "  $FFE6 CMD_PARAM1   $5A\n")

	out := chip.consoleDump("idx")
	assert.Contains(t, out, "   16 ($10): current:$020000 default:$020000 limit:$020010 step:1")
	assert.Contains(t, out, "of 256 in use\n")
	assert.Contains(t, chip.consoleDump("idx $11"), "current:$021000")

	assert.Contains(t, chip.consoleDump("ram 20000 4"), "$20000: A0 A1 A2 A3")
	assert.Contains(t, chip.consoleDump("ram 40000 4"), "address out of range")
	assert.Contains(t, chip.consoleDump("bogus"), "Usage: dump")
	assert.Contains(t, chip.consoleDump("all"), "Error queue: 0\n")
}

func TestEmulatedMiaConsoleDumpToFiles(t *testing.T) {
	folder := t.TempDir()
	circuit := newSDTestCircuit(t, folder)
	chip := circuit.chip
	chip.memory[0x14000] = 0xEA
	chip.memory[0x14001] = 0x60
	chip.indexes[7] = miaIndex{currentAddr: 0x14000, step: 1}

	assert.Equal(t, "Dump: wrote 2 bytes to PART.BIN\n", chip.consoleDump("ram 14000 2 PART.BIN"))
	part, err := os.ReadFile(filepath.Join(folder, "PART.BIN"))
	require.NoError(t, err)
	assert.Equal(t, []uint8{0xEA, 0x60}, part)

	out := chip.consoleDump("all SNAP.BIN")
	assert.Contains(t, out, "Dump: wrote ")
	snap, err := os.ReadFile(filepath.Join(folder, "SNAP.BIN"))
	require.NoError(t, err)
	require.Equal(t, "MIASNAP\x01", string(snap[:8]))

	sections := map[string][]uint8{}
	var order []string
	for p := 8; p < len(snap); {
		tag := string(snap[p : p+4])
		length := int(binary.LittleEndian.Uint32(snap[p+4:]))
		sections[tag] = snap[p+8 : p+8+length]
		order = append(order, tag)
		p += 8 + length
	}
	assert.Equal(t, []string{"INFO", "REGS", "IDX ", "CTX ", "ERRQ", "TRCE", "RAM ", "END "}, order)
	assert.Len(t, sections["IDX "], 4096)
	assert.Equal(t, uint32(0x14000), binary.LittleEndian.Uint32(sections["IDX "][7*16:]))
	assert.Len(t, sections["RAM "], miaRAMSize)
	assert.Equal(t, uint8(0xEA), sections["RAM "][0x14000])
	assert.Equal(t, []uint8{0}, sections["CTX "])
	assert.Equal(t, uint8(0x04), sections["INFO"][6]&0x04, "trace flag")

	noCard := newSDTestCircuit(t, "")
	assert.Equal(t, "Dump: no card, or the card does not mount\n", noCard.chip.consoleDump("ram 0 10 X.BIN"))
}
