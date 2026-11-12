package mia

import (
	"encoding/binary"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// This file mirrors the firmware's con_diag.c: 'status ctx', 'status pio',
// 'dump' and 'trace' (see clementina-mia docs/diagnostics.md).
//
// Divergences from the firmware, all because the emulator is not timing bound:
//   - The bus event trace is always on. The firmware keeps it out of normal
//     builds (make diag) so core 1's action loop has no extra work.
//   - Copies are taken under the chip lock, so they are never torn and the
//     dumps do not print the firmware's "use exec pause" note.
//   - The emulator has no PIOs: 'status pio' says so and snapshots have no
//     PIO section.
//   - Files go to the SD host folder; there is no mount to borrow.

const (
	miaTraceRing        = 64
	miaSnapshotVersion  = 1
	miaSnapshotDiagFlag = 0x04
)

var miaRegisterNames = [miaRegisterCount]string{
	"IDXA_PORT", "IDXA_SELECT", "CFG_SELECT", "CFG_PORT",
	"IDXB_PORT", "IDXB_SELECT", "CMD_PARAM1", "CMD_PARAM2",
	"CMD_PARAM3", "CMD_TRIGGER", "STATUS_L", "STATUS_H",
	"ERROR_L", "ERROR_H", "IRQ_MASK_L", "IRQ_MASK_H",
	"IRQ_STATUS_L", "IRQ_STATUS_H", "INPUT_STATUS", "INPUT_CHAR",
	"INPUT_COUNT", "MIA_CTX", "(reserved)", "(reserved)",
	"(reserved)", "(reserved)", "NMI_VEC_L", "NMI_VEC_H",
	"RESET_VEC_L", "RESET_VEC_H", "IRQ_VEC_L", "IRQ_VEC_H",
}

// miaTraceState mirrors the diagnostic build's trace in mia.c: the last
// miaTraceRing action words, MIA_CTX counters, and a copy frozen at the first
// MIA_CTX overflow or underflow.
type miaTraceState struct {
	ring       [miaTraceRing]uint32
	pos        uint32
	pushes     uint32
	pops       uint32
	overflows  uint32
	underflows uint32
	frozen     bool
	frozenAt   uint32
	snap       [miaTraceRing]uint32
}

// traceBusEvent records one register access the way the firmware's action
// PIO reports it: every write, and only the watched reads (every fourth
// register, plus $FFE1 in loader mode or $FFF3 in normal mode). The word is
// R/W (bit 13) | register offset (bits 8-12) | data (bits 0-7).
func (c *emulated_mia) traceBusEvent(address uint8, data uint8, write bool) {
	address &= miaRegisterMask
	if !write {
		watch := uint8(miaRegIdxASelector)
		if c.state == miaStateNormal {
			watch = miaRegInputChar
		}
		if address%4 != 0 && address != watch {
			return
		}
	}

	word := uint32(address)<<8 | uint32(data)
	if write {
		word |= 1 << 13
	}
	t := &c.trace
	t.ring[t.pos%miaTraceRing] = word
	t.pos++
}

func (c *emulated_mia) traceCtxPush()  { c.trace.pushes++ }
func (c *emulated_mia) traceCtxPop()   { c.trace.pops++ }
func (c *emulated_mia) traceOverflow() { c.trace.overflows++; c.traceFreeze() }
func (c *emulated_mia) traceUnderflow() {
	c.trace.underflows++
	c.traceFreeze()
}

func (c *emulated_mia) traceFreeze() {
	t := &c.trace
	if t.frozen {
		return
	}
	for i := range t.snap {
		t.snap[i] = t.ring[(t.pos+uint32(i))%miaTraceRing]
	}
	t.frozenAt = t.pos
	t.frozen = true
}

// traceWords returns the frozen capture, or the latest events, oldest first.
func (c *emulated_mia) traceWords() []uint32 {
	t := &c.trace
	pos := t.pos
	var words [miaTraceRing]uint32
	if t.frozen {
		pos = t.frozenAt
		words = t.snap
	} else {
		for i := range words {
			words[i] = t.ring[(pos+uint32(i))%miaTraceRing]
		}
	}
	if pos < miaTraceRing {
		return append([]uint32(nil), words[miaTraceRing-pos:]...)
	}
	return append([]uint32(nil), words[:]...)
}

func (c *emulated_mia) traceClear() {
	c.trace = miaTraceState{}
}

/**************************************************************************************************
 * status ctx / status pio
 **************************************************************************************************/

func writeIndexLine(out *strings.Builder, id uint8, entry miaIndex) {
	fmt.Fprintf(out, "  %3d ($%02X): current:$%06X default:$%06X limit:$%06X step:%d flags:$%02X\n",
		id, id,
		entry.currentAddr&miaAddressMask,
		entry.defaultAddr&miaAddressMask,
		entry.limitAddr&miaAddressMask,
		entry.step, entry.flags)
}

func (c *emulated_mia) consoleStatusCtx() string {
	c.mu.Lock()
	depth := c.ctxDepth
	stack := c.ctxStack
	c.mu.Unlock()
	return formatCtx(depth, stack)
}

func formatCtx(depth int, stack [miaCtxDepth]miaContext) string {
	var out strings.Builder
	out.WriteString("MIA_CTX:\n")
	fmt.Fprintf(&out, "  depth: %d of %d\n", depth, miaCtxDepth)
	for i := 0; i < depth; i++ {
		e := stack[i]
		fmt.Fprintf(&out, "  [%d] IDXA sel:%d  IDXB sel:%d  CFG sel:$%02X  params:$%02X $%02X $%02X\n",
			i, e.idxASelector, e.idxBSelector, e.cfgSelector,
			e.cmdParams[0], e.cmdParams[1], e.cmdParams[2])
		out.WriteString("      saved A:")
		writeIndexLine(&out, e.idxASelector, e.idxA)
		out.WriteString("      saved B:")
		writeIndexLine(&out, e.idxBSelector, e.idxB)
	}
	return out.String()
}

func (c *emulated_mia) consoleStatusPIO(args string) string {
	switch strings.TrimSpace(args) {
	case "":
		return "Bus PIOs: not emulated (the emulator has no PIOs).\n"
	case "clear":
		return "PIO: fdebug flags cleared\n"
	default:
		return "Usage: status pio [clear]\n"
	}
}

/**************************************************************************************************
 * trace
 **************************************************************************************************/

func (c *emulated_mia) consoleTrace(args string) string {
	switch strings.TrimSpace(args) {
	case "":
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.formatTraceLocked()
	case "clear":
		c.mu.Lock()
		c.traceClear()
		c.mu.Unlock()
		return "Trace: cleared and re-armed\n"
	default:
		return "Usage: trace [clear]\n"
	}
}

func (c *emulated_mia) formatTraceLocked() string {
	t := &c.trace
	words := c.traceWords()

	var out strings.Builder
	out.WriteString("Bus trace (always on in the emulator):\n")
	fmt.Fprintf(&out, "  events:%d  ctx pushes:%d pops:%d overflows:%d underflows:%d\n",
		t.pos, t.pushes, t.pops, t.overflows, t.underflows)

	last := t.pos
	if t.frozen {
		last = t.frozenAt
		fmt.Fprintf(&out, "  Captured at the first ctx error (event #%d); 'trace clear' re-arms it.\n", t.frozenAt)
	} else {
		out.WriteString("  No ctx error captured; showing the latest events.\n")
	}

	first := last - uint32(len(words))
	for i, w := range words {
		data := w & 0xFF
		offset := (w >> 8) & 0x1F
		write := (w>>13)&1 != 0
		dir, arrow := 'R', "->"
		if write {
			dir, arrow = 'W', "<-"
		}
		fmt.Fprintf(&out, "  #%-9d %c $%04X %-12s %s $%02X\n",
			first+uint32(i), dir, 0xFFE0+offset, miaRegisterNames[offset], arrow, data)
	}
	return out.String()
}

/**************************************************************************************************
 * dump
 **************************************************************************************************/

func dumpUsage() string {
	return "Usage: dump regs | idx [id] | ram ADDR LEN [FILE] | all [FILE]\n" +
		"  ADDR and LEN are hex. With FILE, ram writes the raw bytes and all\n" +
		"  writes a snapshot (docs/diagnostics.md) to the SD card.\n"
}

func (c *emulated_mia) consoleDump(args string) string {
	fields := strings.Fields(args)
	if len(fields) == 0 {
		return dumpUsage()
	}

	switch fields[0] {
	case "regs":
		if len(fields) != 1 {
			return dumpUsage()
		}
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.formatRegsLocked()

	case "idx":
		if len(fields) == 1 {
			c.mu.Lock()
			defer c.mu.Unlock()
			return c.formatIndexesInUseLocked()
		}
		id, ok := parseU8(fields[1])
		if !ok || len(fields) != 2 {
			return "Usage: dump idx [id]  (id: 0-255 or $00-$FF)\n"
		}
		c.mu.Lock()
		entry := c.indexes[id]
		c.mu.Unlock()
		var out strings.Builder
		writeIndexLine(&out, id, entry)
		return out.String()

	case "ram":
		return c.consoleDumpRAM(fields[1:])

	case "all":
		switch len(fields) {
		case 1:
			c.mu.Lock()
			defer c.mu.Unlock()
			var out strings.Builder
			out.WriteString(c.formatRegsLocked())
			out.WriteString(formatCtx(c.ctxDepth, c.ctxStack))
			out.WriteString(c.consoleStatusPIO(""))
			out.WriteString(c.formatErrorQueueLocked())
			out.WriteString(c.formatIndexesInUseLocked())
			out.WriteString(c.formatTraceLocked())
			return out.String()
		case 2:
			c.mu.Lock()
			snapshot := c.snapshotLocked()
			c.mu.Unlock()
			if err := c.consoleWriteFile(fields[1], snapshot); err != "" {
				return fmt.Sprintf("Dump: %s\n", err)
			}
			return fmt.Sprintf("Dump: writing snapshot...\nDump: wrote %d bytes to %s\n", len(snapshot), fields[1])
		}
	}

	return dumpUsage()
}

func (c *emulated_mia) consoleDumpRAM(fields []string) string {
	usage := "Usage: dump ram ADDR LEN [FILE]  (hex, e.g. dump ram 14000 100)\n"
	if len(fields) < 2 || len(fields) > 3 {
		if len(fields) > 3 {
			return dumpUsage()
		}
		return usage
	}
	addr, okAddr := parseDumpHex(fields[0])
	length, okLen := parseDumpHex(fields[1])
	if !okAddr || !okLen || length == 0 {
		return usage
	}
	if addr >= miaRAMSize {
		return fmt.Sprintf("Dump: address out of range (max $%05X)\n", miaRAMSize-1)
	}
	if length > miaRAMSize-addr {
		length = miaRAMSize - addr
	}

	if len(fields) == 2 {
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.monitorDumpLocked(addr, length)
	}

	c.mu.Lock()
	data := append([]uint8(nil), c.memory[addr:addr+length]...)
	c.mu.Unlock()
	if err := c.consoleWriteFile(fields[2], data); err != "" {
		return fmt.Sprintf("Dump: %s\n", err)
	}
	return fmt.Sprintf("Dump: wrote %d bytes to %s\n", length, fields[2])
}

// parseDumpHex parses a hex number with an optional $ or 0x prefix.
func parseDumpHex(value string) (uint32, bool) {
	value = strings.TrimPrefix(value, "$")
	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		value = value[2:]
	}
	if value == "" || len(value) > 8 {
		return 0, false
	}
	var v uint32
	for i := 0; i < len(value); i++ {
		if !isHexDigit(value[i]) {
			return 0, false
		}
		var d uint32
		switch ch := value[i]; {
		case ch >= '0' && ch <= '9':
			d = uint32(ch - '0')
		case ch >= 'a' && ch <= 'f':
			d = uint32(ch-'a') + 10
		default:
			d = uint32(ch-'A') + 10
		}
		v = v<<4 | d
	}
	return v, true
}

func (c *emulated_mia) formatRegsLocked() string {
	var out strings.Builder
	out.WriteString("Registers ($FFE0-$FFFF):\n")
	for i := 0; i < miaRegisterCount; i++ {
		fmt.Fprintf(&out, "  $%04X %-12s $%02X\n", 0xFFE0+i, miaRegisterNames[i], c.registers[i])
	}
	return out.String()
}

func (c *emulated_mia) formatIndexesInUseLocked() string {
	var out strings.Builder
	out.WriteString("Indexes in use (any field non-zero):\n")
	used := 0
	for id := 0; id < miaIndexCount; id++ {
		if c.indexes[id] == (miaIndex{}) {
			continue
		}
		writeIndexLine(&out, uint8(id), c.indexes[id])
		used++
	}
	fmt.Fprintf(&out, "  %d of 256 in use\n", used)
	return out.String()
}

func (c *emulated_mia) formatErrorQueueLocked() string {
	first := c.errors.first
	count := (c.errors.last - first) & 0x0F
	var out strings.Builder
	fmt.Fprintf(&out, "Error queue: %d\n", count)
	for i := uint8(0); i < count; i++ {
		fmt.Fprintf(&out, "  %2d: $%02X\n", i, c.errors.buf[(first+i)&0x0F])
	}
	return out.String()
}

// consoleWriteFile mirrors mia_sd_console_open/write/close: it refuses while
// an SD request runs and writes path relative to the current directory.
func (c *emulated_mia) consoleWriteFile(name string, data []uint8) string {
	c.mu.Lock()
	root := c.sd.rootDir
	cwd := c.sd.cwd
	busy := c.status()&miaStatusSDBusy != 0
	c.mu.Unlock()

	if busy {
		return "SD busy with a 6502 request; try again"
	}
	if root == "" {
		return "no card, or the card does not mount"
	}

	name = strings.ReplaceAll(name, "\\", "/")
	base := "/"
	if !strings.HasPrefix(name, "/") {
		base = "/" + cwd
	}
	hostPath := filepath.Join(root, filepath.FromSlash(path.Clean(base+"/"+name)))
	if err := os.WriteFile(hostPath, data, 0o644); err != nil {
		return fmt.Sprintf("%s: cannot create the file", name)
	}
	return ""
}

// snapshotLocked builds the snapshot file (docs/diagnostics.md) without the
// PIO section, which the emulator has nothing to put in.
func (c *emulated_mia) snapshotLocked() []uint8 {
	out := make([]uint8, 0, miaRAMSize+8192)
	out = append(out, "MIASNAP"...)
	out = append(out, miaSnapshotVersion)

	section := func(tag string, data []uint8) {
		out = append(out, tag...)
		out = binary.LittleEndian.AppendUint32(out, uint32(len(data)))
		out = append(out, data...)
	}

	status := c.status()
	info := make([]uint8, 16)
	binary.LittleEndian.PutUint32(info[0:], c.appliedPhi2Hz)
	binary.LittleEndian.PutUint16(info[4:], status)
	flags := uint8(miaSnapshotDiagFlag)
	if status&miaStatusMasterMode != 0 {
		flags |= 0x01
	}
	if c.execIsPaused() {
		flags |= 0x02
	}
	info[6] = flags
	binary.LittleEndian.PutUint32(info[8:], uint32(c.nowNs/1_000_000))
	binary.LittleEndian.PutUint32(info[12:], miaRAMSize)
	section("INFO", info)

	section("REGS", c.registers[:])

	idx := make([]uint8, 0, miaIndexCount*16)
	for _, e := range c.indexes {
		idx = appendIndex(idx, e)
	}
	section("IDX ", idx)

	ctx := []uint8{uint8(c.ctxDepth)}
	for i := 0; i < c.ctxDepth; i++ {
		e := c.ctxStack[i]
		ctx = append(ctx, e.idxASelector, e.idxBSelector, e.cfgSelector)
		ctx = append(ctx, e.cmdParams[:]...)
		ctx = appendIndex(ctx, e.idxA)
		ctx = appendIndex(ctx, e.idxB)
	}
	section("CTX ", ctx)

	first := c.errors.first
	count := (c.errors.last - first) & 0x0F
	errq := []uint8{count}
	for i := uint8(0); i < count; i++ {
		errq = append(errq, c.errors.buf[(first+i)&0x0F])
	}
	section("ERRQ", errq)

	t := &c.trace
	words := c.traceWords()
	frozen := uint32(0)
	if t.frozen {
		frozen = 1
	}
	trce := make([]uint8, 0, 40+4*len(words))
	for _, v := range []uint32{t.pos, t.pushes, t.pops, t.overflows, t.underflows,
		frozen, t.frozenAt, 0, 0, uint32(len(words))} {
		trce = binary.LittleEndian.AppendUint32(trce, v)
	}
	for _, w := range words {
		trce = binary.LittleEndian.AppendUint32(trce, w)
	}
	section("TRCE", trce)

	section("RAM ", c.memory[:miaRAMSize])
	section("END ", nil)
	return out
}

// appendIndex writes a descriptor in the firmware's 16-byte index_t layout.
func appendIndex(out []uint8, e miaIndex) []uint8 {
	out = binary.LittleEndian.AppendUint32(out, e.currentAddr)
	out = binary.LittleEndian.AppendUint32(out, e.defaultAddr)
	out = binary.LittleEndian.AppendUint32(out, e.limitAddr)
	out = binary.LittleEndian.AppendUint16(out, e.step)
	return append(out, e.flags, 0)
}
