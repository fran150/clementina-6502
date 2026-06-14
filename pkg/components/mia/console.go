package mia

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/fran150/clementina-6502/pkg/components/cpu"
	"go.bug.st/serial"
)

const (
	miaConsoleLineMax       = 80
	miaConsoleSerialBaud    = 115200
	miaMonitorDefaultDump   = 128
	miaMonitorDefaultDisasm = 16
	miaConsoleCtrlQ         = 0x11
)

type miaConsoleMode uint8

const (
	miaConsoleModeNormal miaConsoleMode = iota
	miaConsoleModeMonitor
	miaConsoleModeInput
)

type miaConsoleWifiMode uint8

const (
	miaConsoleWifiOff miaConsoleWifiMode = iota
	miaConsoleWifiSTA
	miaConsoleWifiAP
)

type miaConsoleState struct {
	port serial.Port

	running atomic.Bool
	wg      sync.WaitGroup

	mode miaConsoleMode
	line []byte

	lastCR bool

	wifiMode miaConsoleWifiMode
	wifiSSID string

	lastDumpAddr   uint32
	lastDisasmAddr uint32
}

// ConnectToPort exposes the emulated MIA USB-style console over a host serial port.
func (c *emulated_mia) ConnectToPort(port serial.Port) error {
	if port == nil {
		return fmt.Errorf("MIA console serial port is nil")
	}

	if err := port.SetReadTimeout(100 * time.Millisecond); err != nil {
		return err
	}

	if err := port.SetMode(&serial.Mode{
		BaudRate: miaConsoleSerialBaud,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}); err != nil {
		return err
	}

	c.consoleClose()
	c.console.port = port
	c.console.mode = miaConsoleModeNormal
	c.console.line = c.console.line[:0]
	c.console.lastCR = false
	c.console.running.Store(true)
	c.console.wg.Add(1)
	go c.consoleReadLoop()

	return nil
}

func (c *emulated_mia) consoleClose() {
	c.console.running.Store(false)
	c.console.wg.Wait()
	c.console.port = nil
}

func (c *emulated_mia) consoleReadLoop() {
	defer c.console.wg.Done()

	c.consoleWriteString("MIA ready. Type 'help' for commands.\n")
	c.consoleWritePrompt()

	buf := make([]byte, 64)
	for c.console.running.Load() {
		clear(buf)
		n, err := c.console.port.Read(buf)
		if err != nil {
			if c.console.running.Load() {
				panic(err)
			}
			return
		}

		if n > len(buf) {
			n = len(buf)
		}

		for _, value := range buf[:n] {
			c.consoleHandleByte(value)
		}
	}
}

func (c *emulated_mia) consoleHandleByte(value byte) {
	if c.console.mode == miaConsoleModeInput {
		c.consoleHandleInputByte(value)
		return
	}

	switch {
	case value == '\r' || value == '\n':
		if value == '\n' && c.console.lastCR {
			c.console.lastCR = false
			return
		}
		c.console.lastCR = value == '\r'
		c.consoleWriteString("\n")
		line := string(c.console.line)
		c.console.line = c.console.line[:0]
		c.consoleWriteString(c.consoleExecLine(line))
		c.consoleWritePrompt()

	case (value == '\b' || value == 127) && len(c.console.line) > 0:
		c.console.lastCR = false
		c.console.line = c.console.line[:len(c.console.line)-1]
		c.consoleWriteString("\b \b")

	case value >= 0x20 && value < 0x7F && len(c.console.line) < miaConsoleLineMax-1:
		c.console.lastCR = false
		c.console.line = append(c.console.line, value)
		c.consoleWrite([]byte{value})

	default:
		c.console.lastCR = false
	}
}

// consoleHandleInputByte forwards a terminal byte to the input text FIFO while
// console input capture is active. Ctrl+Q returns the terminal to command mode.
func (c *emulated_mia) consoleHandleInputByte(value byte) {
	if value == miaConsoleCtrlQ {
		c.mu.Lock()
		c.inputConsoleEndCapture()
		c.mu.Unlock()

		c.console.mode = miaConsoleModeNormal
		c.console.line = c.console.line[:0]
		c.consoleWriteString("\nConsole input ended.\n")
		c.consoleWritePrompt()
		return
	}

	if value == '\n' {
		value = '\r'
	} else if value == 127 {
		value = '\b'
	}

	c.mu.Lock()
	c.inputConsoleByte(value)
	c.mu.Unlock()
}

func (c *emulated_mia) consoleWritePrompt() {
	switch c.console.mode {
	case miaConsoleModeMonitor:
		c.consoleWriteString("MON> ")
	case miaConsoleModeInput:
		// No prompt is shown while capturing console input.
	default:
		c.consoleWriteString("> ")
	}
}

func (c *emulated_mia) consoleWriteString(value string) {
	c.consoleWrite([]byte(value))
}

func (c *emulated_mia) consoleWrite(value []byte) {
	if len(value) == 0 || c.console.port == nil {
		return
	}

	if _, err := c.console.port.Write(value); err != nil {
		panic(err)
	}
}

func (c *emulated_mia) consoleExecLine(line string) string {
	if c.console.mode == miaConsoleModeMonitor {
		out, keepMonitor := c.consoleMonitorExecLine(line)
		if !keepMonitor {
			c.console.mode = miaConsoleModeNormal
			out += "Exiting monitor.\n"
		}
		return out
	}

	return c.consoleDispatch(line)
}

func (c *emulated_mia) consoleDispatch(line string) string {
	cmd, args := splitConsoleCommand(line)
	if cmd == "" {
		return ""
	}

	switch cmd {
	case "?", "help":
		return c.consoleHelp()
	case "status":
		return c.consoleStatus(args)
	case "errors":
		return c.consoleErrors(args)
	case "speed":
		return c.consoleSpeed(args)
	case "wifi":
		return c.consoleWifi(args)
	case "input":
		return c.consoleInput(args)
	case "audio":
		return c.consoleAudio(args)
	case "exec":
		return c.consoleExec(args)
	case "monitor":
		c.console.mode = miaConsoleModeMonitor
		return c.consoleMonitorBanner()
	case "quit":
		return "Rebooting to BOOTSEL...\n"
	default:
		return fmt.Sprintf("Unknown command '%s'. Try 'help'.\n", cmd)
	}
}

func (c *emulated_mia) consoleHelp() string {
	var out strings.Builder
	out.WriteString("Commands:\n")
	out.WriteString("  status     status [video|input|audio|wifi|irq|speed|exec|mem|index]\n")
	out.WriteString("  errors     errors [list|clear]\n")
	out.WriteString("  speed      speed HZ  - set PHI2 clock frequency\n")
	out.WriteString("  wifi       wifi [status|off|connect|ap]\n")
	out.WriteString("  input      input [status|console|wifi]\n")
	out.WriteString("  audio      audio [status|enable|stop|reset]\n")
	out.WriteString("  exec       exec [status|pause|resume]\n")
	out.WriteString("  monitor    Enter 65C02 machine language monitor\n")
	out.WriteString("  quit       Reboot to BOOTSEL\n")
	out.WriteString("  help       Show this help\n")
	return out.String()
}

func writeStatusFlags(out *strings.Builder, status uint16) {
	writeFlagNames(out, status, []miaFlagName{
		{miaStatusMasterMode, "NORMAL"},
		{miaStatusErrors, "ERRORS"},
		{miaStatusCmdRunning, "CMD"},
		{miaStatusDMARunning, "DMA"},
		{miaStatusSpeedChanging, "SPEED"},
		{miaStatusVideoRequested, "VID_REQ"},
		{miaStatusVideoSent, "VID_SENT"},
		{miaStatusExecPaused, "PAUSED"},
		{miaStatusAudioActive, "AUDIO"},
	})
}

func (c *emulated_mia) consoleErrors(args string) string {
	switch strings.TrimSpace(args) {
	case "list":
		return c.consoleErrorsList()
	case "clear":
		c.mu.Lock()
		c.errors.reset(c)
		c.mu.Unlock()
		return "MIA Errors cleared.\n"
	default:
		return "Usage: errors [list|clear]\n"
	}
}

func (c *emulated_mia) consoleErrorsList() string {
	c.mu.Lock()
	first := c.errors.first
	last := c.errors.last
	buf := c.errors.buf
	current := c.readRegister(miaRegErrorLSB)
	c.mu.Unlock()

	count := (last - first) & 0x0F

	var out strings.Builder
	fmt.Fprintf(&out, "MIA Errors: %d queued", count)
	if count != 0 {
		fmt.Fprintf(&out, "  current: 0x%02X %s", current, errorName(current))
	}
	out.WriteString("\n")

	if count == 0 {
		out.WriteString("  none\n")
		return out.String()
	}

	for i := uint8(0); i < count && i < 15; i++ {
		pos := (first + i) & 0x0F
		code := buf[pos]
		fmt.Fprintf(&out, "  %2d: 0x%02X %s\n", i, code, errorName(code))
	}

	return out.String()
}

// errorName maps a MIA error code to its symbolic name, mirroring the firmware
// con_cmds error_name table.
func errorName(code uint8) string {
	switch code {
	case miaErrorMIACannotAllocateRAM:
		return "ERROR_MIA_CANNOT_ALLOCATE_RAM"
	case miaErrorQueueOverflow:
		return "ERROR_QUEUE_OVERFLOW"
	case miaErrorDMASizeZero:
		return "ERROR_DMA_SIZE_ZERO"
	case miaErrorDMASourceOverflow:
		return "ERROR_DMA_SRC_WILL_OVERFLOW"
	case miaErrorDMATargetOverflow:
		return "ERROR_DMA_TGT_WILL_OVERFLOW"
	case miaErrorCmdQueueFull:
		return "ERROR_CMD_QUEUE_FULL"
	case miaErrorCmdUnknown:
		return "ERROR_CMD_UNKNOWN"
	case miaErrorWifiInitFailed:
		return "ERROR_WIFI_INIT_FAILED"
	case miaErrorWifiConnectFailed:
		return "ERROR_WIFI_CONNECT_FAILED"
	case miaErrorVideoUDPAllocFailed:
		return "ERROR_VIDEO_UDP_ALLOC_FAILED"
	case miaErrorVideoUDPBindFailed:
		return "ERROR_VIDEO_UDP_BIND_FAILED"
	case miaErrorInputModeUnavailable:
		return "ERROR_INPUT_MODE_UNAVAILABLE"
	case miaErrorInputProbeInvalid:
		return "ERROR_INPUT_PROBE_INVALID"
	case miaErrorInputUDPAllocFailed:
		return "ERROR_INPUT_UDP_ALLOC_FAILED"
	case miaErrorInputUDPBindFailed:
		return "ERROR_INPUT_UDP_BIND_FAILED"
	case miaErrorAudioQueueOverflow:
		return "ERROR_AUDIO_QUEUE_OVERFLOW"
	default:
		return "UNKNOWN_ERROR"
	}
}

func (c *emulated_mia) consoleSpeed(args string) string {
	args = strings.TrimLeft(args, " \t")
	if args == "" {
		c.mu.Lock()
		phi2 := c.appliedPhi2Hz
		c.mu.Unlock()
		return fmt.Sprintf("PHI2: %d Hz\nUsage: speed HZ  (range: %d-%d, e.g. speed 1000000)\n",
			phi2, miaMinPhi2Hz, miaMaxPhi2Hz)
	}

	hz, ok := parseLeadingDecimal(args)
	if !ok {
		return "Invalid value. Usage: speed HZ  (e.g. speed 1000000)\n"
	}

	c.mu.Lock()
	c.stagedPhi2Hz = hz
	c.commitPhi2Hz()
	c.mu.Unlock()

	return fmt.Sprintf("PHI2 speed requested: %d Hz (use 'status' to confirm applied value)\n", hz)
}

func (c *emulated_mia) consoleWifi(args string) string {
	args = strings.TrimSpace(args)
	if args == "" || args == "status" {
		return c.consoleWifiStatus()
	}

	if args == "off" {
		c.console.wifiMode = miaConsoleWifiOff
		c.console.wifiSSID = ""
		return "Wi-Fi: off\n"
	}

	if strings.HasPrefix(args, "ap") && isCommandBoundary(args, 2) {
		return c.consoleWifiStartAP(strings.TrimLeft(args[2:], " \t"))
	}

	if strings.HasPrefix(args, "connect") && isCommandBoundary(args, 7) {
		return c.consoleWifiConnect(strings.TrimLeft(args[7:], " \t"))
	}

	return "Usage: wifi [status|off|connect <ssid> [password]|ap <ssid> [password]]\n" +
		"  Notes: SSID and password may not contain spaces.\n" +
		"         AP clients must use a static IP; no DHCP is provided.\n"
}

func (c *emulated_mia) consoleWifiConnect(args string) string {
	ssid, _ := splitConsoleCommand(args)
	if ssid == "" {
		return "Usage: wifi connect <ssid> [password]\n"
	}

	c.console.wifiMode = miaConsoleWifiSTA
	c.console.wifiSSID = ssid

	return fmt.Sprintf("Wi-Fi: connecting to '%s'...\nWi-Fi: connected\n", ssid)
}

func (c *emulated_mia) consoleWifiStartAP(args string) string {
	ssid, _ := splitConsoleCommand(args)
	if ssid == "" {
		return "Usage: wifi ap <ssid> [password]\n"
	}

	c.console.wifiMode = miaConsoleWifiAP
	c.console.wifiSSID = ssid

	return fmt.Sprintf(
		"Wi-Fi: AP '%s' active at 192.168.4.1\nWi-Fi: clients must use a static IP in 192.168.4.x/24\n",
		ssid,
	)
}

func (c *emulated_mia) consoleWifiStatus() string {
	switch c.console.wifiMode {
	case miaConsoleWifiSTA:
		return fmt.Sprintf("Wi-Fi: STA  SSID: %s  (emulated)\n", c.console.wifiSSID)
	case miaConsoleWifiAP:
		return fmt.Sprintf("Wi-Fi: AP   SSID: %s  IP: 192.168.4.1\n", c.console.wifiSSID)
	default:
		return "Wi-Fi: off\n"
	}
}

func (c *emulated_mia) consoleInput(args string) string {
	args = strings.TrimSpace(args)
	if args == "" || args == "status" {
		return c.consoleInputStatus() + "Usage: input [status|console|wifi]\n"
	}

	switch args {
	case "console":
		c.mu.Lock()
		ok := c.inputSetMode(miaInputModeConsole)
		c.mu.Unlock()
		if !ok {
			return "Input: console mode is not available.\n"
		}
		c.console.mode = miaConsoleModeInput
		return "Console input active. Press Ctrl+Q to return to commands.\n"
	case "wifi":
		c.mu.Lock()
		ok := c.inputSetMode(miaInputModeWifi)
		address := c.input.bindAddress
		c.mu.Unlock()
		if !ok {
			return "Input: Wi-Fi mode is not available.\n"
		}
		return fmt.Sprintf("Input: Wi-Fi mode active on UDP %s.\n", address)
	default:
		return "Usage: input [status|console|wifi]\n"
	}
}

func (c *emulated_mia) consoleInputStatus() string {
	c.mu.Lock()
	mode := c.input.mode
	udpReady := c.input.udpReady
	wifiActive := c.input.wifiActive
	status := c.registers[miaRegInputStatus]
	count := c.registers[miaRegInputCharCount]
	c.mu.Unlock()

	var out strings.Builder
	fmt.Fprintf(&out, "Input: %s", inputModeName(mode))
	if mode == miaInputModeWifi {
		udp := "unavailable"
		if udpReady {
			udp = "ready"
		}
		client := "none"
		if wifiActive {
			client = "active"
		}
		fmt.Fprintf(&out, "  UDP:%s  client:%s", udp, client)
	}
	fmt.Fprintf(&out, "  status:0x%02X  chars:%d\n", status, count)

	return out.String()
}

func splitConsoleCommand(line string) (string, string) {
	line = strings.TrimLeft(line, " \t")
	if line == "" {
		return "", ""
	}

	end := 0
	for end < len(line) && line[end] != ' ' && line[end] != '\t' {
		end++
	}

	return line[:end], line[end:]
}

func isCommandBoundary(value string, index int) bool {
	return len(value) == index || value[index] == ' ' || value[index] == '\t'
}

func parseLeadingDecimal(value string) (uint32, bool) {
	end := 0
	for end < len(value) && value[end] >= '0' && value[end] <= '9' {
		end++
	}

	if end == 0 {
		return 0, false
	}

	parsed, err := strconv.ParseUint(value[:end], 10, 32)
	if err != nil {
		return miaMaxPhi2Hz + 1, true
	}

	return uint32(parsed), true
}

var miaMonitorInstructionSet = cpu.NewInstructionSet()

func (c *emulated_mia) consoleMonitorBanner() string {
	return fmt.Sprintf(
		"\n65C02 Monitor  [MIA RAM: %dKB, $00000-$%05X]\n%s\n",
		miaRAMSize/1024,
		miaRAMSize-1,
		consoleMonitorHelp(),
	)
}

func consoleMonitorHelp() string {
	return fmt.Sprintf(
		"  m [ADDR [LEN]]    Dump memory, hex+ASCII (default %d bytes)\n"+
			"  u [ADDR [COUNT]]  Disassemble 65C02 (default %d instructions)\n"+
			"  e ADDR BYTE...    Edit memory (space-separated hex bytes)\n"+
			"  ? / help          Show this help\n"+
			"  quit              Return to console\n",
		miaMonitorDefaultDump,
		miaMonitorDefaultDisasm,
	)
}

func (c *emulated_mia) consoleMonitorExecLine(line string) (string, bool) {
	p := strings.TrimLeft(line, " \t")
	if p == "" {
		return "", true
	}

	cmd, rest := splitConsoleCommand(p)
	cmd = strings.ToLower(cmd)

	switch cmd {
	case "quit":
		return "", false
	case "?", "help":
		return consoleMonitorHelp(), true
	case "m":
		return c.consoleMonitorDumpCommand(rest), true
	case "u":
		return c.consoleMonitorDisassembleCommand(rest), true
	case "e":
		return c.consoleMonitorEditCommand(rest), true
	default:
		return fmt.Sprintf("Unknown command '%s'. Type ? for help.\n", cmd), true
	}
}

func (c *emulated_mia) consoleMonitorDumpCommand(args string) string {
	addr := c.console.lastDumpAddr
	length := uint32(miaMonitorDefaultDump)
	if value, rest, ok := nextHex(args); ok {
		addr = value
		args = rest
	}
	if value, _, ok := nextHex(args); ok {
		length = value
	}

	if addr >= miaRAMSize {
		return fmt.Sprintf("Address out of range (max $%05X)\n", miaRAMSize-1)
	}

	c.mu.Lock()
	out := c.monitorDumpLocked(addr, length)
	c.mu.Unlock()
	c.console.lastDumpAddr = addr + length

	return out
}

func (c *emulated_mia) consoleMonitorDisassembleCommand(args string) string {
	addr := c.console.lastDisasmAddr
	count := uint32(miaMonitorDefaultDisasm)
	if value, rest, ok := nextHex(args); ok {
		addr = value
		args = rest
	}
	if value, _, ok := nextHex(args); ok {
		count = value
	}

	if addr >= miaRAMSize {
		return fmt.Sprintf("Address out of range (max $%05X)\n", miaRAMSize-1)
	}

	c.mu.Lock()
	out, next := c.monitorDisassembleLocked(addr, count)
	c.mu.Unlock()
	c.console.lastDisasmAddr = next

	return out
}

func (c *emulated_mia) consoleMonitorEditCommand(args string) string {
	addr, rest, ok := nextHex(args)
	if !ok {
		return "Usage: e ADDR BYTE [BYTE ...]\n"
	}

	cur := addr
	wrote := false
	var out strings.Builder

	c.mu.Lock()
	for {
		value, next, ok := nextHex(rest)
		if !ok {
			break
		}
		rest = next

		if cur >= miaRAMSize {
			fmt.Fprintf(&out, "Address overflow at $%05X\n", cur)
			break
		}

		c.memory[cur] = uint8(value)
		c.videoMarkDirty(cur)
		cur++
		wrote = true
	}
	c.mu.Unlock()

	if !wrote {
		return "Usage: e ADDR BYTE [BYTE ...]\n"
	}

	return out.String()
}

func (c *emulated_mia) monitorDumpLocked(addr uint32, length uint32) string {
	if addr >= miaRAMSize || length == 0 {
		return ""
	}

	end := addr + length
	if end < addr || end > miaRAMSize {
		end = miaRAMSize
	}

	rowStart := addr &^ 0x0F
	var out strings.Builder
	for row := rowStart; row < end; row += 16 {
		fmt.Fprintf(&out, "$%05X: ", row)

		for i := uint32(0); i < 16; i++ {
			if i == 8 {
				out.WriteByte(' ')
			}
			a := row + i
			if a < addr || a >= end {
				out.WriteString("   ")
			} else {
				fmt.Fprintf(&out, "%02X ", c.memory[a])
			}
		}

		out.WriteByte(' ')
		for i := uint32(0); i < 16; i++ {
			a := row + i
			if a < addr || a >= end {
				out.WriteByte(' ')
				continue
			}

			value := c.memory[a]
			if value >= 0x20 && value < 0x7F {
				out.WriteByte(value)
			} else {
				out.WriteByte('.')
			}
		}
		out.WriteByte('\n')
	}

	return out.String()
}

func (c *emulated_mia) monitorDisassembleLocked(addr uint32, count uint32) (string, uint32) {
	var out strings.Builder
	for i := uint32(0); i < count && addr < miaRAMSize; i++ {
		addr = c.monitorDisassembleOneLocked(&out, addr)
	}

	return out.String(), addr
}

func (c *emulated_mia) monitorDisassembleOneLocked(out *strings.Builder, addr uint32) uint32 {
	if addr >= miaRAMSize {
		fmt.Fprintf(out, "$%05X: [out of range]\n", addr)
		return addr + 1
	}

	opcode := c.memory[addr]
	instruction, known := monitorDecodeInstruction(opcode)
	size := monitorInstructionSize(instruction, known)
	op1 := c.monitorByte(addr + 1)
	op2 := c.monitorByte(addr + 2)

	fmt.Fprintf(out, "$%05X: ", addr)
	for i := uint8(0); i < 3; i++ {
		if i < size {
			fmt.Fprintf(out, "%02X ", c.monitorByte(addr+uint32(i)))
		} else {
			out.WriteString("   ")
		}
	}

	mnemonic := string(instruction.Mnemonic())
	if !known {
		mnemonic = "???"
	}
	fmt.Fprintf(out, "%-5s", mnemonic)
	out.WriteString(monitorInstructionOperand(addr, instruction, known, op1, op2))
	out.WriteByte('\n')

	return addr + uint32(size)
}

func (c *emulated_mia) monitorByte(addr uint32) uint8 {
	if addr >= miaRAMSize {
		return 0
	}

	return c.memory[addr]
}

func monitorDecodeInstruction(opcode uint8) (components.CpuInstructionData, bool) {
	instruction := miaMonitorInstructionSet.GetByOpCode(components.OpCode(opcode))
	return instruction, instruction.OpCode() == components.OpCode(opcode)
}

func monitorInstructionSize(instruction components.CpuInstructionData, known bool) uint8 {
	if !known {
		return 1
	}

	if instruction.AddressMode() == cpu.AddressModeBreak {
		return 1
	}

	return cpu.GetAddressMode(instruction.AddressMode()).MemSize()
}

func monitorInstructionOperand(addr uint32, instruction components.CpuInstructionData, known bool, op1, op2 uint8) string {
	if !known {
		return ""
	}

	mode := instruction.AddressMode()
	switch mode {
	case cpu.AddressModeBreak:
		return ""
	case cpu.AddressModeAccumulator:
		return monitorFormattedOperand(cpu.GetAddressMode(mode).Format())
	case cpu.AddressModeRelative:
		target := uint16(int32(uint16(addr+2)) + int32(int8(op1)))
		return fmt.Sprintf("$%04X", target)
	case cpu.AddressModeRelativeExtended:
		target := uint16(int32(uint16(addr+3)) + int32(int8(op2)))
		return fmt.Sprintf("$%02X,$%04X", op1, target)
	}

	details := cpu.GetAddressMode(mode)
	switch details.MemSize() {
	case 1:
		return ""
	case 2:
		return monitorFormattedOperand(details.Format(), op1)
	default:
		word := uint16(op1) | uint16(op2)<<8
		return monitorFormattedOperand(details.Format(), word)
	}
}

func monitorFormattedOperand(format string, args ...any) string {
	operand := fmt.Sprintf(format, args...)
	if operand == "a" {
		return "A"
	}

	return strings.ReplaceAll(operand, ", ", ",")
}

func nextHex(value string) (uint32, string, bool) {
	value = strings.TrimLeft(value, " \t")
	if value == "" {
		return 0, value, false
	}

	if value[0] == '$' {
		value = value[1:]
	}

	end := 0
	for end < len(value) && isHexDigit(value[end]) {
		end++
	}

	if end == 0 {
		return 0, value, false
	}

	parsed, err := strconv.ParseUint(value[:end], 16, 32)
	if err != nil {
		return 0, value[end:], false
	}

	return uint32(parsed), value[end:], true
}

func isHexDigit(value byte) bool {
	return (value >= '0' && value <= '9') ||
		(value >= 'a' && value <= 'f') ||
		(value >= 'A' && value <= 'F')
}
