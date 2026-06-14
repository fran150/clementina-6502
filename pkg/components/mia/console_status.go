package mia

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

// This file implements the expanded terminal 'status' diagnostics and the 'exec'
// command. It mirrors the Pico firmware con_cmds.c additions: 'status' prints a
// compact dashboard and 'status <subsystem>' drills into a subsystem, while
// 'exec [status|pause|resume]' controls and reports PHI2 execution from the
// terminal. Subsystem detail that depends on real-hardware paths the emulator
// does not have (cross-core request/done latches, video repair queue, lwIP
// netif/IP) is adapted to the emulator's equivalent state rather than ported
// literally, matching the divergences already noted for the firmware port.

// miaFlagName pairs a flag bit with the symbolic name printed for it.
type miaFlagName struct {
	bit  uint16
	name string
}

// writeFlagNames appends " (NAME,NAME)" for every set flag, mirroring the
// firmware print_flag_name helper. It reports whether any flag was written.
func writeFlagNames(out *strings.Builder, value uint16, flags []miaFlagName) bool {
	wrote := false
	for _, flag := range flags {
		if value&flag.bit == 0 {
			continue
		}

		if wrote {
			out.WriteString(",")
		} else {
			out.WriteString(" (")
		}
		out.WriteString(flag.name)
		wrote = true
	}

	if wrote {
		out.WriteString(")")
	}

	return wrote
}

// consoleStatus dispatches 'status' and its subsystem subcommands.
func (c *emulated_mia) consoleStatus(args string) string {
	args = strings.TrimSpace(args)

	switch args {
	case "", "summary", "all":
		return c.consoleStatusSummary()
	case "video":
		return c.consoleVideoDetail()
	case "input":
		return c.consoleInputDetail()
	case "wifi":
		return c.consoleWifiDetail()
	case "irq":
		return c.consoleStatusIRQ()
	case "speed":
		return c.consoleStatusSpeed()
	case "exec":
		return c.consoleStatusExec()
	case "errors":
		return c.consoleErrorsList()
	case "mem", "memory":
		return c.consoleStatusMem()
	}

	if strings.HasPrefix(args, "index") && isCommandBoundary(args, 5) {
		return c.consoleStatusIndex(strings.TrimLeft(args[5:], " \t"))
	}

	return "Usage: status [video|input|wifi|irq|speed|exec|errors|mem|index [id]]\n"
}

// consoleStatusSummary renders the compact dashboard, mirroring cmd_status_summary.
func (c *emulated_mia) consoleStatusSummary() string {
	c.mu.Lock()
	status := c.status()
	execPaused := c.execIsPaused()
	applied := c.appliedPhi2Hz
	requested := c.requestedPhi2Hz
	speedChanging := c.speedChangeRequested || status&miaStatusSpeedChanging != 0
	irqStatus := c.irqStatus()
	irqMask := c.irqMask()
	errCount := (c.errors.last - c.errors.first) & 0x0F
	currentErr := c.readRegister(miaRegErrorLSB)
	idxa := c.readRegister(miaRegIdxASelector)
	idxb := c.readRegister(miaRegIdxBSelector)
	mode := "Bootloader"
	if status&miaStatusMasterMode != 0 {
		mode = "Normal"
	}
	c.mu.Unlock()

	var out strings.Builder
	fmt.Fprintf(&out, "MIA Status:\n")
	fmt.Fprintf(&out, "  Mode:   %s\n", mode)

	execState := "Running"
	if execPaused {
		execState = "Paused (PHI2 stopped low)"
	}
	fmt.Fprintf(&out, "  Exec:   %s\n", execState)

	fmt.Fprintf(&out, "  PHI2:   %d Hz", applied)
	if speedChanging {
		fmt.Fprintf(&out, "  requested:%d Hz", requested)
	}
	out.WriteString("\n")

	fmt.Fprintf(&out, "  RAM:    %dKB  ($00000-$%05X)\n", miaRAMSize/1024, miaRAMSize-1)
	fmt.Fprintf(&out, "  Status: 0x%04X", status)
	writeStatusFlags(&out, status)
	out.WriteString("\n")
	fmt.Fprintf(&out, "  IRQ:    status:0x%04X  mask:0x%04X\n", irqStatus, irqMask)
	fmt.Fprintf(&out, "  Errors: %d queued", errCount)
	if errCount != 0 {
		fmt.Fprintf(&out, "  current:0x%02X %s", currentErr, errorName(currentErr))
	}
	out.WriteString("\n")
	fmt.Fprintf(&out, "  IDXA:   index %d\n", idxa)
	fmt.Fprintf(&out, "  IDXB:   index %d\n", idxb)

	out.WriteString(c.consoleWifiStatus())
	out.WriteString(c.consoleVideoSummary())
	out.WriteString(c.consoleInputStatus())

	return out.String()
}

// consoleVideoSummary renders the one-line video summary, mirroring mia_video_print_summary.
func (c *emulated_mia) consoleVideoSummary() string {
	c.mu.Lock()
	udpReady := c.video.conn != nil
	clientActive := c.video.sessionActive
	frame := c.video.frameID
	dirty := c.videoCountDirtyPages(c.video.activeMap)
	pipeline := "idle"
	if c.video.pendingValid {
		if c.video.pendingInitialSent {
			pipeline = "sent"
		} else {
			pipeline = "sending"
		}
	} else if c.videoHasDirtyPages(c.video.activeMap) {
		pipeline = "dirty"
	}
	c.mu.Unlock()

	return fmt.Sprintf("Video: enabled  UDP:%s  client:%s  frame:%d  dirty:%d  pipeline:%s\n",
		readyOrNot(udpReady), activeOrNone(clientActive), frame, dirty, pipeline)
}

// consoleStatusIRQ renders the IRQ detail, mirroring cmd_status_irq. The emulator
// raises IRQ flags synchronously, so there is no separate set-request accumulator.
func (c *emulated_mia) consoleStatusIRQ() string {
	c.mu.Lock()
	status := c.irqStatus()
	mask := c.irqMask()
	c.mu.Unlock()

	enabled := status & mask

	var out strings.Builder
	out.WriteString("IRQ:\n")
	fmt.Fprintf(&out, "  status:   0x%04X", status)
	writeIRQSources(&out, status)
	out.WriteString("\n")
	fmt.Fprintf(&out, "  mask:     0x%04X", mask)
	writeIRQSources(&out, mask)
	out.WriteString("\n")
	fmt.Fprintf(&out, "  enabled:  0x%04X", enabled)
	writeIRQSources(&out, enabled)
	out.WriteString("\n")

	line := "released"
	if status&miaIRQTriggered != 0 {
		line = "asserted"
	}
	fmt.Fprintf(&out, "  line:     %s\n", line)

	return out.String()
}

func writeIRQSources(out *strings.Builder, value uint16) {
	wrote := writeFlagNames(out, value, []miaFlagName{
		{miaIRQError, "ERROR"},
		{miaIRQIdxAWrap, "IDXA_WRAP"},
		{miaIRQIdxBWrap, "IDXB_WRAP"},
		{miaIRQCommand, "COMMAND"},
		{miaIRQSpeedChanged, "SPEED"},
		{miaIRQVideoRequest, "VID_REQ"},
		{miaIRQVideoSent, "VID_SENT"},
		{miaIRQVideoAcked, "VID_ACK"},
		{miaIRQInputKeyboard, "INPUT_KEY"},
		{miaIRQInputMouse, "INPUT_MOUSE"},
		{miaIRQInputGamepad, "INPUT_PAD"},
		{miaIRQTriggered, "TRIGGERED"},
	})

	if !wrote {
		out.WriteString(" none")
	}
}

// consoleStatusSpeed renders the speed detail, mirroring cmd_status_speed.
func (c *emulated_mia) consoleStatusSpeed() string {
	c.mu.Lock()
	applied := c.appliedPhi2Hz
	requested := c.requestedPhi2Hz
	staged := c.stagedPhi2Hz
	pending := c.speedChangeRequested
	c.mu.Unlock()

	var out strings.Builder
	out.WriteString("Speed:\n")
	fmt.Fprintf(&out, "  applied:   %d Hz\n", applied)
	fmt.Fprintf(&out, "  requested: %d Hz\n", requested)
	fmt.Fprintf(&out, "  staged:    %d Hz\n", staged)
	fmt.Fprintf(&out, "  pending:   %s\n", yesNo(pending))
	fmt.Fprintf(&out, "  range:     %d-%d Hz\n", miaMinPhi2Hz, miaMaxPhi2Hz)

	return out.String()
}

// consoleStatusExec renders the exec status line, mirroring 'status exec'.
func (c *emulated_mia) consoleStatusExec() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.execStatusString()
}

// consoleStatusIndex renders one index or both selected indexes, mirroring cmd_status_index.
func (c *emulated_mia) consoleStatusIndex(args string) string {
	args = strings.TrimSpace(args)

	var out strings.Builder
	out.WriteString("Index:\n")

	if args != "" {
		id, ok := parseU8(args)
		if !ok {
			return "Usage: status index [id]\n"
		}
		c.mu.Lock()
		entry := c.indexes[id]
		c.mu.Unlock()
		writeIndexDetail(&out, id, entry)
		return out.String()
	}

	c.mu.Lock()
	idxa := c.readRegister(miaRegIdxASelector)
	idxb := c.readRegister(miaRegIdxBSelector)
	entryA := c.indexes[idxa]
	entryB := c.indexes[idxb]
	c.mu.Unlock()

	fmt.Fprintf(&out, "  IDXA selector: %d\n", idxa)
	writeIndexDetail(&out, idxa, entryA)
	fmt.Fprintf(&out, "  IDXB selector: %d\n", idxb)
	writeIndexDetail(&out, idxb, entryB)

	return out.String()
}

// consoleStatusMem renders the RAM/register summary, mirroring cmd_status_mem.
func (c *emulated_mia) consoleStatusMem() string {
	c.mu.Lock()
	idxa := c.readRegister(miaRegIdxASelector)
	idxb := c.readRegister(miaRegIdxBSelector)
	entryA := c.indexes[idxa]
	entryB := c.indexes[idxb]
	c.mu.Unlock()

	var out strings.Builder
	out.WriteString("Memory:\n")
	fmt.Fprintf(&out, "  RAM: %dKB  range:$00000-$%05X  mask:0x%05X\n",
		miaRAMSize/1024, miaRAMSize-1, miaRAMMask)
	fmt.Fprintf(&out, "  regs: %d bytes\n", miaRegisterCount)
	out.WriteString("  selected indexes:\n")
	writeIndexDetail(&out, idxa, entryA)
	writeIndexDetail(&out, idxb, entryB)

	return out.String()
}

func writeIndexDetail(out *strings.Builder, id uint8, entry miaIndex) {
	fmt.Fprintf(out, "  index %d: current:$%06X  default:$%06X  limit:$%06X  step:%d  flags:0x%02X",
		id,
		entry.currentAddr&miaAddressMask,
		entry.defaultAddr&miaAddressMask,
		entry.limitAddr&miaAddressMask,
		entry.step,
		entry.flags)

	writeFlagNames(out, uint16(entry.flags), []miaFlagName{
		{1 << miaIndexFlagReadStep, "R_STEP"},
		{1 << miaIndexFlagWriteStep, "W_STEP"},
		{1 << miaIndexFlagStepDir, "BACKWARD"},
		{1 << miaIndexFlagWrap, "WRAP"},
		{1 << miaIndexFlagWrapIRQ, "WRAP_IRQ"},
	})
	out.WriteString("\n")
}

// consoleVideoDetail renders the video subsystem detail, mirroring mia_video_print_status.
func (c *emulated_mia) consoleVideoDetail() string {
	c.mu.Lock()
	bindAddress := c.video.bindAddress
	udpReady := c.video.conn != nil
	sessionActive := c.video.sessionActive
	remote := ""
	if c.video.remote != nil {
		remote = c.video.remote.String()
	}
	sessionID := c.video.sessionID
	frameID := c.video.frameID
	clientFrameID := c.video.clientFrameID
	lastPeerSeq := c.video.lastPeerSeq
	nextSeq := c.video.nextSeq
	activeMap := c.video.activeMap
	dirty0 := c.videoCountDirtyPages(0)
	dirty1 := c.videoCountDirtyPages(1)
	pendingValid := c.video.pendingValid
	pendingRequestID := c.video.pendingRequestID
	pendingFrameID := c.video.pendingFrameID
	pendingLastComplete := c.video.pendingLastComplete
	pendingPages := len(c.video.pendingPages)
	pendingChunkCount := c.video.pendingChunkCount
	pendingInitialSent := c.video.pendingInitialSent
	layout := c.memory[miaVideoLocalVersionOffset]
	videoMode := c.memory[miaVideoModeOffset]
	frameField := binary.LittleEndian.Uint32(c.memory[miaVideoLocalFrameIDOffset : miaVideoLocalFrameIDOffset+4])
	dirtyField := binary.LittleEndian.Uint16(c.memory[miaVideoLocalDirtyPagesOffset : miaVideoLocalDirtyPagesOffset+2])
	status := c.status()
	c.mu.Unlock()

	var out strings.Builder
	out.WriteString("Video:\n")
	out.WriteString("  enabled: yes\n")
	fmt.Fprintf(&out, "  UDP:     %s  bind:%s\n", readyOrNot(udpReady), bindOrNone(bindAddress))

	if sessionActive {
		fmt.Fprintf(&out, "  client:  %s  session:0x%08X\n", remote, sessionID)
	} else {
		out.WriteString("  client:  none\n")
	}

	fmt.Fprintf(&out, "  frames:  local:%d  client:%d  last-peer-seq:%d  next-seq:%d\n",
		frameID, clientFrameID, lastPeerSeq, nextSeq)
	fmt.Fprintf(&out, "  dirty:   active-map:%d  map0:%d  map1:%d\n", activeMap, dirty0, dirty1)

	if pendingValid {
		fmt.Fprintf(&out, "  response: valid  request:%d  frame:%d  last-complete:%d\n",
			pendingRequestID, pendingFrameID, pendingLastComplete)
		fmt.Fprintf(&out, "            pages:%d  chunks:%d  initial-sent:%s\n",
			pendingPages, pendingChunkCount, yesNo(pendingInitialSent))
	} else {
		out.WriteString("  response: none\n")
	}

	fmt.Fprintf(&out, "  state:   layout:%d  mode:0x%02X  frame-field:%d  dirty-field:%d\n",
		layout, videoMode, frameField, dirtyField)
	fmt.Fprintf(&out, "  flags:   requested:%s  sent:%s\n",
		yesNo(status&miaStatusVideoRequested != 0),
		yesNo(status&miaStatusVideoSent != 0))

	return out.String()
}

// consoleInputDetail renders the input subsystem detail, mirroring mia_input_print_detail.
func (c *emulated_mia) consoleInputDetail() string {
	c.mu.Lock()
	mode := c.input.mode
	statusReg := c.registers[miaRegInputStatus]
	charReg := c.registers[miaRegInputChar]
	countReg := c.registers[miaRegInputCharCount]
	udpReady := c.input.udpReady
	bindAddress := c.input.bindAddress
	wifiActive := c.input.wifiActive
	wifiRemote := ""
	if c.input.wifiRemote != nil {
		wifiRemote = c.input.wifiRemote.String()
	}
	wifiSession := c.input.wifiSession
	wifiLastSeq := c.input.wifiLastSeq
	wifiCaps := c.input.wifiCaps
	deviceFlags := c.memory[miaInputDeviceFlagsOffset]
	kbFlags := c.memory[miaKeyboardEventFlagsOffset]
	kbMask := c.memory[miaKeyboardEventMaskOffset]
	kbAck := c.memory[miaKeyboardEventAckOffset]
	msFlags := c.memory[miaMouseEventFlagsOffset]
	msMask := c.memory[miaMouseEventMaskOffset]
	msAck := c.memory[miaMouseEventAckOffset]
	gpFlags := c.memory[miaGamepadEventFlagsOffset]
	gpMask := c.memory[miaGamepadEventMaskOffset]
	gpAck := c.memory[miaGamepadEventAckOffset]
	var mouse [5]uint8
	copy(mouse[:], c.memory[miaInputMouseStateOffset:miaInputMouseStateOffset+5])
	var pads [miaInputGamepadSlots][miaInputGamepadSlotSize]uint8
	for i := 0; i < miaInputGamepadSlots; i++ {
		base := miaInputGamepadOffset + i*miaInputGamepadSlotSize
		copy(pads[i][:], c.memory[base:base+miaInputGamepadSlotSize])
	}
	c.mu.Unlock()

	var out strings.Builder
	out.WriteString("Input:\n")
	fmt.Fprintf(&out, "  mode:    %s\n", inputModeName(mode))
	fmt.Fprintf(&out, "  status:  0x%02X", statusReg)
	writeInputStatusBits(&out, statusReg)
	fmt.Fprintf(&out, "  chars:%d  current:0x%02X\n", countReg, charReg)

	fmt.Fprintf(&out, "  UDP:     %s  bind:%s\n", readyOrNot(udpReady), bindOrNone(bindAddress))

	if wifiActive {
		fmt.Fprintf(&out, "  client:  %s  session:0x%08X  last-seq:%d\n", wifiRemote, wifiSession, wifiLastSeq)
		fmt.Fprintf(&out, "  caps:    0x%04X", wifiCaps)
		writeInputCaps(&out, wifiCaps)
		out.WriteString("\n")
	} else {
		out.WriteString("  client:  none\n")
	}

	fmt.Fprintf(&out, "  devices: 0x%02X", deviceFlags)
	writeInputDeviceFlags(&out, deviceFlags)
	out.WriteString("\n")

	fmt.Fprintf(&out, "  keyboard events: flags:0x%02X  mask:0x%02X  ack:0x%02X\n", kbFlags, kbMask, kbAck)
	fmt.Fprintf(&out, "  mouse events:    flags:0x%02X  mask:0x%02X  ack:0x%02X\n", msFlags, msMask, msAck)
	fmt.Fprintf(&out, "  gamepad events:  flags:0x%02X  mask:0x%02X  ack:0x%02X\n", gpFlags, gpMask, gpAck)

	fmt.Fprintf(&out, "  mouse:   buttons:0x%02X  dx:0x%02X  dy:0x%02X  wheel-x:0x%02X  wheel-y:0x%02X\n",
		mouse[0]&miaMouseButtonMask, mouse[1], mouse[2], mouse[3], mouse[4])

	for i := 0; i < miaInputGamepadSlots; i++ {
		slot := pads[i]
		buttons := uint16(slot[2]) | uint16(slot[3])<<8
		connected := "none"
		if slot[0]&0x80 != 0 {
			connected = "connected"
		}
		fmt.Fprintf(&out, "  pad%d:    %s  dpad:0x%X  buttons:0x%04X  lx:%d  ly:%d  rx:%d  ry:%d  lt:%d  rt:%d\n",
			i, connected, slot[0]&0x0F, buttons, slot[4], slot[5], slot[6], slot[7], slot[8], slot[9])
	}

	return out.String()
}

func writeInputStatusBits(out *strings.Builder, status uint8) {
	writeFlagNames(out, uint16(status), []miaFlagName{
		{uint16(miaInputTextReady), "TEXT"},
		{uint16(miaInputKeyboardDown), "KEYBOARD"},
		{uint16(miaInputConsumerDown), "CONSUMER"},
		{uint16(miaInputMouseDown), "MOUSE"},
		{uint16(miaInputGamepadDown), "GAMEPAD"},
		{uint16(miaInputSourceConsole), "CONSOLE"},
		{uint16(miaInputSourceWifi), "WIFI"},
		{uint16(miaInputSourceUSBHost), "USB_HOST"},
	})
}

func writeInputDeviceFlags(out *strings.Builder, flags uint8) {
	writeFlagNames(out, uint16(flags), []miaFlagName{
		{uint16(miaInputDeviceKeyboard), "KEYBOARD"},
		{uint16(miaInputDeviceConsumer), "CONSUMER"},
		{uint16(miaInputDeviceMouse), "MOUSE"},
		{uint16(miaInputDeviceGamepad0), "PAD0"},
		{uint16(miaInputDeviceGamepad1), "PAD1"},
		{uint16(miaInputDeviceGamepad2), "PAD2"},
		{uint16(miaInputDeviceGamepad3), "PAD3"},
	})
}

func writeInputCaps(out *strings.Builder, caps uint16) {
	writeFlagNames(out, caps, []miaFlagName{
		{miaInputCapText, "TEXT"},
		{miaInputCapKeyboard, "KEYBOARD"},
		{miaInputCapConsumer, "CONSUMER"},
		{miaInputCapMouse, "MOUSE"},
		{miaInputCapGamepad, "GAMEPAD"},
	})
}

// consoleWifiDetail renders the Wi-Fi detail. The emulator Wi-Fi is fully
// simulated (no lwIP netif), so this reports the console-tracked mode and SSID
// rather than the firmware's live netif/IP fields.
func (c *emulated_mia) consoleWifiDetail() string {
	mode := c.console.wifiMode
	ssid := c.console.wifiSSID

	modeName := "off"
	switch mode {
	case miaConsoleWifiSTA:
		modeName = "sta"
	case miaConsoleWifiAP:
		modeName = "ap"
	}

	var out strings.Builder
	out.WriteString("Wi-Fi:\n")
	fmt.Fprintf(&out, "  mode:       %s\n", modeName)
	if ssid == "" {
		out.WriteString("  ssid:       (none)\n")
	} else {
		fmt.Fprintf(&out, "  ssid:       %s\n", ssid)
	}

	switch mode {
	case miaConsoleWifiSTA:
		out.WriteString("  ip:         (emulated)\n")
	case miaConsoleWifiAP:
		out.WriteString("  ip:         192.168.4.1\n")
		out.WriteString("  AP clients: static IP in 192.168.4.x/24\n")
	}

	return out.String()
}

// consoleExec controls and reports PHI2 execution, mirroring cmd_exec.
func (c *emulated_mia) consoleExec(args string) string {
	switch strings.TrimSpace(args) {
	case "", "status":
		c.mu.Lock()
		status := c.execStatusString()
		c.mu.Unlock()
		return status + "Usage: exec [status|pause|resume]\n"
	case "pause":
		c.mu.Lock()
		c.execPause()
		c.mu.Unlock()
		return "Exec: paused\n"
	case "resume":
		c.mu.Lock()
		c.execResume()
		c.mu.Unlock()
		return "Exec: running\n"
	default:
		return "Usage: exec [status|pause|resume]\n"
	}
}

func parseU8(value string) (uint8, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	base := 10
	switch {
	case strings.HasPrefix(value, "$"):
		base = 16
		value = value[1:]
	case strings.HasPrefix(value, "0x"), strings.HasPrefix(value, "0X"):
		base = 16
		value = value[2:]
	}

	parsed, err := strconv.ParseUint(value, base, 8)
	if err != nil {
		return 0, false
	}

	return uint8(parsed), true
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}

	return "no"
}

func readyOrNot(ready bool) string {
	if ready {
		return "ready"
	}

	return "unavailable"
}

func activeOrNone(active bool) string {
	if active {
		return "active"
	}

	return "none"
}

func bindOrNone(addr string) string {
	if addr == "" {
		return "(none)"
	}

	return addr
}
