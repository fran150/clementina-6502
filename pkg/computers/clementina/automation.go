package clementina

import (
	"fmt"
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/fran150/clementina-6502/pkg/core"
	"github.com/fran150/clementina-6502/pkg/core/managers"
	"runtime"
	"sync"
)

// Automation exclusively owns a headless computer. No UI or UDP loops share it.
// All operations, including Close, are serialized at complete cycle boundaries.
type Automation struct {
	mu              sync.Mutex
	computer        *ClementinaComputer
	step            common.StepContext
	paused          bool
	closed          bool
	running         bool
	reason          string
	breakpoints     core.BreakpointManager
	bankBreakpoints []AutomationBreakpoint
	trace           []automationCall
	pending         *automationInstruction
	skipBreakpoint  bool
	wake            chan struct{}
	done            chan struct{}
}

// AutomationRequest is one JSON request to the automation API. Method selects
// the operation; the other fields are its arguments.
type AutomationRequest struct {
	Version   int    `json:"version"`
	Method    string `json:"method"`
	Count     int    `json:"count,omitempty"`
	Address   int    `json:"address,omitempty"`
	Bank      *int   `json:"bank,omitempty"`
	Data      []int  `json:"data,omitempty"`
	UsagePage *int   `json:"usagePage,omitempty"`
	UsageID   *int   `json:"usageId,omitempty"`
	Down      *bool  `json:"down,omitempty"`
	Player    *int   `json:"player,omitempty"`
}

// AutomationState is the CPU and run state returned by automation calls.
type AutomationState struct {
	Cycles              string `json:"cycles"`
	PC                  uint16 `json:"pc"`
	A                   uint8  `json:"a"`
	X                   uint8  `json:"x"`
	Y                   uint8  `json:"y"`
	SP                  uint8  `json:"sp"`
	P                   uint8  `json:"p"`
	Paused              bool   `json:"paused"`
	Running             bool   `json:"running"`
	StopReason          string `json:"stopReason"`
	InstructionBoundary bool   `json:"instructionBoundary"`
	Bank                uint8  `json:"bank"`
}

// AutomationBreakpoint identifies one opcode location in the 16 KiB ExRAM window.
type AutomationBreakpoint struct {
	Address uint16 `json:"address"`
	Bank    uint8  `json:"bank"`
}

// AutomationCaller is one entry of a reconstructed call stack: the calling
// instruction, its ExRAM bank and the kind of call.
type AutomationCaller struct {
	PC   uint16 `json:"pc"`
	Bank uint8  `json:"bank"`
	Kind string `json:"kind"`
}

// AutomationStack is the reconstructed call stack. Callers lists the calls the
// trace can verify, innermost first; UnknownCaller says older callers may exist
// beyond them.
type AutomationStack struct {
	Callers       []AutomationCaller `json:"callers"`
	UnknownCaller bool               `json:"unknownCaller"`
}
type automationInstruction struct {
	pc     uint16
	sp     uint8
	bank   uint8
	opcode uint8
}
type automationCall struct {
	caller  AutomationCaller
	sp      uint8
	entrySP uint8
	stackPC uint16
}

// NewAutomation creates an exclusively owned headless machine with no SD card.
func NewAutomation() (*Automation, error) {
	return NewAutomationWithSDFolder("")
}

// NewAutomationWithSDFolder creates an exclusively owned headless machine and
// mounts folder as its emulated SD root. An empty folder leaves the slot empty.
func NewAutomationWithSDFolder(folder string) (*Automation, error) {
	c, err := NewClementinaComputer()
	if err != nil {
		return nil, err
	}
	c.SetMiaSDFolder(folder)
	a := &Automation{computer: c, step: common.NewStepContext(), reason: "initial", breakpoints: managers.NewBreakpointManager(), wake: make(chan struct{}, 1), done: make(chan struct{})}
	c.SetMiaExecPausedHandler(func(paused bool) { a.paused = paused })
	go a.loop()
	return a, nil
}

// Close stops the machine and waits for its run loop to exit.
func (a *Automation) Close() {
	a.mu.Lock()
	if !a.closed {
		a.running = false
		a.closed = true
		a.computer.Close()
		close(a.wake)
	}
	a.mu.Unlock()
	<-a.done
}

// The runner releases ownership every 256 cycles, allowing inspection and pause.
// This is an unthrottled debugger runner, not a PHI2 frequency scheduler.
func (a *Automation) loop() {
	defer close(a.done)
	for range a.wake {
		for {
			a.mu.Lock()
			if !a.running || a.closed {
				a.mu.Unlock()
				break
			}
			for i := 0; i < 256; i++ {
				if a.stopCondition() {
					break
				}
				if a.boundary() && a.hasBreakpoint() && !a.skipBreakpoint {
					a.stop("breakpoint")
					break
				}
				a.skipBreakpoint = false
				a.tick()
				if a.stopCondition() {
					break
				}
			}
			a.mu.Unlock()
			runtime.Gosched()
		}
	}
}
func (a *Automation) boundary() bool {
	return a.computer.chips.cpu.(interface{ DebugInstructionBoundary() bool }).DebugInstructionBoundary()
}
func (a *Automation) bank() uint8 { return a.computer.circuit.portABus.Read() & 0x1f }
func (a *Automation) hasBreakpoint() bool {
	pc := a.computer.GetProgramCounter()
	if a.breakpoints.HasBreakpoint(pc) {
		return true
	}
	if pc < 0x8000 || pc >= 0xc000 {
		return false
	}
	for _, bp := range a.bankBreakpoints {
		if bp.Address == pc && bp.Bank == a.bank() {
			return true
		}
	}
	return false
}
func (a *Automation) stop(reason string) { a.running = false; a.reason = reason }
func (a *Automation) stopCondition() bool {
	if a.paused {
		a.stop("mia-paused")
		return true
	}
	if a.computer.chips.cpu.(interface{ DebugStopped() bool }).DebugStopped() {
		a.stop("cpu-stopped")
		return true
	}
	return false
}
func (a *Automation) tick() {
	if a.boundary() && a.pending == nil {
		pc := a.computer.GetProgramCounter()
		opcode, ok := a.computer.peekMappedMemory(pc)
		if ok {
			a.pending = &automationInstruction{pc, a.computer.chips.cpu.GetStackPointer(), a.bank(), opcode}
		}
	}
	a.computer.Tick(&a.step)
	a.computer.PostTick(&a.step)
	a.step.NextCycle()
	if a.pending != nil && a.boundary() {
		a.traceInstruction(*a.pending)
		a.pending = nil
	}
}
func (a *Automation) stackWord(sp uint8) uint16 {
	return uint16(a.computer.chips.baseram.Peek(0x100|uint32(sp-1))) |
		uint16(a.computer.chips.baseram.Peek(0x100|uint32(sp)))<<8
}
func (a *Automation) traceInstruction(start automationInstruction) {
	endSP := a.computer.chips.cpu.GetStackPointer()
	endPC := a.computer.GetProgramCounter()
	// A changed stack origin or a wrapped hardware stack makes older callers
	// ambiguous even if bytes at a previous return slot happen to still match.
	if start.opcode == 0x9a || ((start.opcode == 0x08 || start.opcode == 0x48 || start.opcode == 0x5a || start.opcode == 0xda) && endSP > start.sp) ||
		((start.opcode == 0x28 || start.opcode == 0x68 || start.opcode == 0x7a || start.opcode == 0xfa) && endSP < start.sp) {
		a.trace = nil
		return
	}
	call := start.opcode == 0x20 && endSP == start.sp-2 && a.stackWord(start.sp) == start.pc+2
	brk := start.opcode == 0x00 && endSP == start.sp-3 && a.stackWord(start.sp) == start.pc+2
	interrupt := endSP == start.sp-3 && a.stackWord(start.sp) == start.pc
	if call || brk || interrupt {
		kind, returnPC := "call", start.pc+3
		if brk || interrupt {
			kind, returnPC = "interrupt", start.pc
			if brk {
				returnPC = start.pc + 2
			}
		}
		a.trace = append(a.trace, automationCall{AutomationCaller{returnPC, start.bank, kind}, endSP, start.sp, a.stackWord(start.sp)})
		return
	}
	if start.opcode == 0x60 || start.opcode == 0x40 {
		if len(a.trace) == 0 {
			return
		}
		top := a.trace[len(a.trace)-1]
		matching := (start.opcode == 0x60 && top.caller.Kind == "call" && endSP == top.entrySP && endPC == top.caller.PC) ||
			(start.opcode == 0x40 && top.caller.Kind == "interrupt" && endSP == top.entrySP && endPC == top.caller.PC)
		if matching {
			a.trace = a.trace[:len(a.trace)-1]
		} else {
			a.trace = nil
		}
	}
}
func (a *Automation) stack() AutomationStack {
	result := AutomationStack{Callers: []AutomationCaller{}, UnknownCaller: true}
	if !a.boundary() {
		return result
	}
	sp := a.computer.chips.cpu.GetStackPointer()
	for i := len(a.trace) - 1; i >= 0; i-- {
		frame := a.trace[i]
		if sp > frame.sp || a.stackWord(frame.entrySP) != frame.stackPC {
			break
		}
		result.Callers = append(result.Callers, frame.caller)
		sp = frame.entrySP
	}
	return result
}
func (a *Automation) state() AutomationState {
	c := a.computer.chips.cpu
	var p uint8
	for i := 0; i < 8; i++ {
		if c.GetProcessorStatusRegister().Flag(components.StatusBit(i)) {
			p |= 1 << i
		}
	}
	return AutomationState{fmt.Sprint(a.step.Cycle), c.GetProgramCounter(), c.GetAccumulatorRegister(), c.GetXRegister(), c.GetYRegister(), c.GetStackPointer(), p, a.paused, a.running, a.reason, a.boundary(), a.bank()}
}

// Execute implements protocol v1. Limits are automation policy, not hardware limits.
func (a *Automation) Execute(r AutomationRequest) (any, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return nil, fmt.Errorf("session closed")
	}
	if r.Version != 1 {
		return nil, fmt.Errorf("unsupported protocol version")
	}
	switch r.Method {
	case "capabilities":
		return map[string]any{"version": 1, "methods": []string{"capabilities", "state", "reset", "step", "readMemory", "input", "setHidUsage", "setHidBitmap", "setGamepadState", "clearGamepad", "video", "audio", "run", "pause", "resume", "stepInstruction", "addBreakpoint", "removeBreakpoint", "breakpoints", "clearBreakpoints", "addBankBreakpoint", "removeBankBreakpoint", "bankBreakpoints", "clearBankBreakpoints", "stackTrace"}, "maxCycles": 10000000, "maxRead": 65536}, nil
	case "state":
		return a.state(), nil
	case "stackTrace":
		if a.running {
			return nil, fmt.Errorf("pause before stack inspection")
		}
		return a.stack(), nil
	case "reset":
		a.stop("reset")
		a.skipBreakpoint = false
		a.trace = nil
		a.pending = nil
		a.computer.Reset(true)
		for i := 0; i < 3; i++ {
			a.tick()
		}
		a.computer.Reset(false)
		a.trace = nil
		a.pending = nil
		return a.state(), nil
	case "step", "stepInstruction":
		if a.running {
			return nil, fmt.Errorf("pause before stepping")
		}
		if r.Count < 1 || r.Count > 10000000 {
			return nil, fmt.Errorf("count must be 1..10000000")
		}
		a.reason = "cycle-limit"
		a.skipBreakpoint = false
		for i := 0; i < r.Count; i++ {
			if a.stopCondition() {
				break
			}
			a.tick()
			if a.stopCondition() {
				break
			}
			if r.Method == "stepInstruction" && a.boundary() {
				a.reason = "instruction"
				break
			}
		}
		return a.state(), nil
	case "run", "resume":
		if a.running {
			return a.state(), nil
		}
		if r.Method == "resume" {
			a.computer.chips.mia.(interface{ DebugResumeExecution() }).DebugResumeExecution()
		}
		if a.stopCondition() {
			return a.state(), nil
		}
		a.skipBreakpoint = r.Method == "resume" && a.reason == "breakpoint" && a.boundary() && a.hasBreakpoint()
		a.running = true
		a.reason = "running"
		select {
		case a.wake <- struct{}{}:
		default:
		}
		return a.state(), nil
	case "pause":
		if a.running {
			a.stop("pause")
		}
		return a.state(), nil
	case "addBreakpoint", "removeBreakpoint":
		if r.Address < 0 || r.Address > 65535 {
			return nil, fmt.Errorf("invalid breakpoint address")
		}
		if r.Method == "addBreakpoint" {
			a.breakpoints.AddBreakpoint(uint16(r.Address))
		} else {
			a.breakpoints.RemoveBreakpoint(uint16(r.Address))
		}
		return a.breakpoints.GetBreakpoints(), nil
	case "clearBreakpoints":
		a.breakpoints.ClearAllBreakpoints()
		return a.breakpoints.GetBreakpoints(), nil
	case "breakpoints":
		return a.breakpoints.GetBreakpoints(), nil
	case "addBankBreakpoint", "removeBankBreakpoint":
		if r.Address < 0x8000 || r.Address >= 0xc000 || r.Bank == nil || *r.Bank < 0 || *r.Bank > 31 {
			return nil, fmt.Errorf("bank breakpoint requires address $8000-$BFFF and bank 0..31")
		}
		bp := AutomationBreakpoint{uint16(r.Address), uint8(*r.Bank)}
		index := -1
		for i, item := range a.bankBreakpoints {
			if item == bp {
				index = i
				break
			}
		}
		if r.Method == "addBankBreakpoint" && index < 0 {
			a.bankBreakpoints = append(a.bankBreakpoints, bp)
		}
		if r.Method == "removeBankBreakpoint" && index >= 0 {
			a.bankBreakpoints = append(a.bankBreakpoints[:index], a.bankBreakpoints[index+1:]...)
		}
		return append([]AutomationBreakpoint{}, a.bankBreakpoints...), nil
	case "bankBreakpoints":
		return append([]AutomationBreakpoint{}, a.bankBreakpoints...), nil
	case "clearBankBreakpoints":
		a.bankBreakpoints = nil
		return []AutomationBreakpoint{}, nil
	case "readMemory":
		if r.Address < 0 || r.Address > 65535 || r.Count < 1 || r.Count > 65536-r.Address {
			return nil, fmt.Errorf("invalid CPU memory range")
		}
		if r.Bank != nil && (*r.Bank < 0 || *r.Bank > 31 || r.Address < 0x8000 || r.Address+r.Count > 0xc000) {
			return nil, fmt.Errorf("bank read requires address range within $8000-$BFFF and bank 0..31")
		}
		data := make([]any, r.Count)
		for i := range data {
			if r.Bank != nil {
				data[i] = int(a.computer.chips.exram.Peek(uint32(*r.Bank)*0x4000 + uint32(r.Address+i-0x8000)))
				continue
			}
			v, ok := a.computer.peekMappedMemory(uint16(r.Address + i))
			if ok {
				data[i] = int(v)
			}
		}
		return data, nil
	case "input":
		if len(r.Data) < 1 || len(r.Data) > 64 {
			return nil, fmt.Errorf("input must contain 1..64 bytes")
		}
		for _, b := range r.Data {
			if b < 0 || b > 255 {
				return nil, fmt.Errorf("invalid input byte")
			}
		}
		chip := a.computer.chips.mia.(interface{ DebugQueueInput(uint8) })
		for _, b := range r.Data {
			chip.DebugQueueInput(uint8(b))
		}
		return a.state(), nil
	case "setHidUsage":
		if r.UsagePage == nil || (*r.UsagePage != 7 && *r.UsagePage != 12) || r.UsageID == nil || *r.UsageID < 0 || *r.UsageID > 255 || r.Down == nil {
			return nil, fmt.Errorf("expected HID page 7 or 12, usage ID 0..255, and down boolean")
		}
		a.computer.chips.mia.(interface{ DebugSetHidUsage(uint16, uint16, bool) }).DebugSetHidUsage(uint16(*r.UsagePage), uint16(*r.UsageID), *r.Down)
		return a.state(), nil
	case "setHidBitmap":
		if r.UsagePage == nil || (*r.UsagePage != 7 && *r.UsagePage != 12) {
			return nil, fmt.Errorf("expected HID page 7 or 12")
		}
		data, err := automationInputBytes(r.Data, 32)
		if err != nil {
			return nil, fmt.Errorf("HID bitmap: %w", err)
		}
		a.computer.chips.mia.(interface{ DebugSetHidBitmap(uint16, []byte) }).DebugSetHidBitmap(uint16(*r.UsagePage), data)
		return a.state(), nil
	case "setGamepadState":
		if r.Player == nil || *r.Player < 0 || *r.Player > 3 {
			return nil, fmt.Errorf("player must be 0..3")
		}
		data, err := automationInputBytes(r.Data, 10)
		if err != nil {
			return nil, fmt.Errorf("gamepad state: %w", err)
		}
		a.computer.chips.mia.(interface{ DebugSetGamepadState(int, []byte) }).DebugSetGamepadState(*r.Player, data)
		return a.state(), nil
	case "clearGamepad":
		if r.Player == nil || *r.Player < 0 || *r.Player > 3 {
			return nil, fmt.Errorf("player must be 0..3")
		}
		a.computer.chips.mia.(interface{ DebugClearGamepad(int) }).DebugClearGamepad(*r.Player)
		return a.state(), nil
	case "video":
		chip := a.computer.chips.mia.(interface{ DebugReadVideo(uint32) uint8 })
		data := make([]int, 68944)
		for i := range data {
			data[i] = int(chip.DebugReadVideo(uint32(i)))
		}
		return data, nil
	case "audio":
		bytes := a.computer.chips.mia.(interface{ DebugAudioSnapshot() []byte }).DebugAudioSnapshot()
		data := make([]int, len(bytes))
		for i, value := range bytes {
			data[i] = int(value)
		}
		return data, nil
	default:
		return nil, fmt.Errorf("unknown method")
	}
}

func automationInputBytes(data []int, size int) ([]byte, error) {
	if len(data) != size {
		return nil, fmt.Errorf("expected %d bytes", size)
	}
	bytes := make([]byte, size)
	for i, value := range data {
		if value < 0 || value > 255 {
			return nil, fmt.Errorf("invalid byte at index %d", i)
		}
		bytes[i] = byte(value)
	}
	return bytes, nil
}
