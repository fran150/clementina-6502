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
	mu             sync.Mutex
	computer       *ClementinaComputer
	step           common.StepContext
	paused         bool
	closed         bool
	running        bool
	reason         string
	breakpoints    core.BreakpointManager
	skipBreakpoint bool
	wake           chan struct{}
	done           chan struct{}
}

type AutomationRequest struct {
	Version   int    `json:"version"`
	Method    string `json:"method"`
	Count     int    `json:"count,omitempty"`
	Address   int    `json:"address,omitempty"`
	Data      []int  `json:"data,omitempty"`
	UsagePage *int   `json:"usagePage,omitempty"`
	UsageID   *int   `json:"usageId,omitempty"`
	Down      *bool  `json:"down,omitempty"`
	Player    *int   `json:"player,omitempty"`
}
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
}

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
				if a.boundary() && a.breakpoints.HasBreakpoint(a.computer.GetProgramCounter()) && !a.skipBreakpoint {
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
	a.computer.Tick(&a.step)
	a.computer.PostTick(&a.step)
	a.step.NextCycle()
}
func (a *Automation) state() AutomationState {
	c := a.computer.chips.cpu
	var p uint8
	for i := 0; i < 8; i++ {
		if c.GetProcessorStatusRegister().Flag(components.StatusBit(i)) {
			p |= 1 << i
		}
	}
	return AutomationState{fmt.Sprint(a.step.Cycle), c.GetProgramCounter(), c.GetAccumulatorRegister(), c.GetXRegister(), c.GetYRegister(), c.GetStackPointer(), p, a.paused, a.running, a.reason, a.boundary()}
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
		return map[string]any{"version": 1, "methods": []string{"capabilities", "state", "reset", "step", "readMemory", "input", "setHidUsage", "setHidBitmap", "setGamepadState", "clearGamepad", "video", "audio", "run", "pause", "resume", "stepInstruction", "addBreakpoint", "removeBreakpoint", "breakpoints", "clearBreakpoints"}, "maxCycles": 10000000, "maxRead": 65536}, nil
	case "state":
		return a.state(), nil
	case "reset":
		a.stop("reset")
		a.skipBreakpoint = false
		a.computer.Reset(true)
		for i := 0; i < 3; i++ {
			a.tick()
		}
		a.computer.Reset(false)
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
		a.skipBreakpoint = r.Method == "resume" && a.reason == "breakpoint" && a.boundary() && a.breakpoints.HasBreakpoint(a.computer.GetProgramCounter())
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
	case "readMemory":
		if r.Address < 0 || r.Address > 65535 || r.Count < 1 || r.Count > 65536-r.Address {
			return nil, fmt.Errorf("invalid CPU memory range")
		}
		data := make([]any, r.Count)
		for i := range data {
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
