package clementina

import (
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestAutomation(t *testing.T) {
	a, err := NewAutomation()
	require.NoError(t, err)
	defer a.Close()
	call := func(method string, count, address int, data []int) any {
		v, e := a.Execute(AutomationRequest{1, method, count, address, data})
		require.NoError(t, e)
		return v
	}
	require.Equal(t, "0", call("state", 0, 0, nil).(AutomationState).Cycles)
	require.Equal(t, "3", call("reset", 0, 0, nil).(AutomationState).Cycles)
	call("step", 4000000, 0, nil)
	bytes := []int{}
	for _, b := range []byte("POKE 752,42\r") {
		bytes = append(bytes, int(b))
	}
	call("input", 0, 0, bytes)
	call("step", 600000, 0, nil)
	require.Equal(t, []any{42}, call("readMemory", 1, 752, nil))
	require.Equal(t, []any{nil}, call("readMemory", 1, 0xc000, nil))
	require.Len(t, call("video", 0, 0, nil), 68944)
	for _, r := range []AutomationRequest{{Version: 2, Method: "state"}, {Version: 1, Method: "step", Count: -1}, {Version: 1, Method: "readMemory", Address: 65535, Count: 2}, {Version: 1, Method: "input", Data: []int{65, 256}}, {Version: 1, Method: "missing"}} {
		_, err := a.Execute(r)
		require.Error(t, err)
	}
	a.Close()
	_, err = a.Execute(AutomationRequest{Version: 1, Method: "state"})
	require.Error(t, err)
}
func TestAutomationConcurrent(t *testing.T) {
	a, err := NewAutomation()
	require.NoError(t, err)
	defer a.Close()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_, e := a.Execute(AutomationRequest{Version: 1, Method: "step", Count: 10})
				if e != nil {
					t.Error(e)
				}
			}
		}()
	}
	wg.Wait()
	v, err := a.Execute(AutomationRequest{Version: 1, Method: "state"})
	require.NoError(t, err)
	require.Equal(t, "1600", v.(AutomationState).Cycles)
}
func TestAutomationHonorsMiaPause(t *testing.T) {
	a, err := NewAutomation()
	require.NoError(t, err)
	defer a.Close()
	a.paused = true
	v, err := a.Execute(AutomationRequest{Version: 1, Method: "step", Count: 10})
	require.NoError(t, err)
	require.Equal(t, "0", v.(AutomationState).Cycles)
}

// Synthetic code is installed only by this internal test harness.
func automationProgram(t *testing.T, program []byte) *Automation {
	t.Helper()
	a, err := NewAutomation()
	require.NoError(t, err)
	t.Cleanup(a.Close)
	for i, b := range program {
		a.computer.BaseRamPoke(uint16(0x600+i), b)
	}
	a.computer.chips.cpu.ForceProgramCounter(0x600)
	return a
}
func command(t *testing.T, a *Automation, method string, count, address int) any {
	t.Helper()
	v, err := a.Execute(AutomationRequest{Version: 1, Method: method, Count: count, Address: address})
	require.NoError(t, err)
	return v
}
func waitStopped(t *testing.T, a *Automation) AutomationState {
	t.Helper()
	var state AutomationState
	require.Eventually(t, func() bool { state = command(t, a, "state", 0, 0).(AutomationState); return !state.Running }, time.Second*3, time.Millisecond)
	return state
}
func TestAutomationInstructionAndBreakpoint(t *testing.T) {
	// LDA #42; STA $0700; INX; JMP $0600
	a := automationProgram(t, []byte{0xa9, 42, 0x8d, 0, 7, 0xe8, 0x4c, 0, 6})
	require.Equal(t, []uint16{0x602}, command(t, a, "addBreakpoint", 0, 0x602))
	require.Equal(t, []uint16{0x602}, command(t, a, "addBreakpoint", 0, 0x602))
	command(t, a, "run", 0, 0)
	s := waitStopped(t, a)
	require.Equal(t, "breakpoint", s.StopReason)
	require.Equal(t, uint16(0x602), s.PC)
	require.Equal(t, uint8(42), s.A)
	require.True(t, s.InstructionBoundary)
	require.Equal(t, []any{0}, command(t, a, "readMemory", 1, 0x700))
	command(t, a, "run", 0, 0)
	require.Equal(t, s.Cycles, waitStopped(t, a).Cycles)
	command(t, a, "resume", 0, 0)
	s = waitStopped(t, a)
	require.Equal(t, "breakpoint", s.StopReason)
	require.Equal(t, uint16(0x602), s.PC)
	require.Equal(t, uint8(1), s.X)
	require.Equal(t, []any{42}, command(t, a, "readMemory", 1, 0x700))
	s = command(t, a, "stepInstruction", 100, 0).(AutomationState)
	require.Equal(t, uint16(0x605), s.PC)
	require.Equal(t, "instruction", s.StopReason)
	s = command(t, a, "stepInstruction", 100, 0).(AutomationState)
	require.Equal(t, uint8(2), s.X)
	require.Equal(t, uint16(0x606), s.PC)
	require.Empty(t, command(t, a, "removeBreakpoint", 0, 0x602))
	command(t, a, "addBreakpoint", 0, 0)
	require.Empty(t, command(t, a, "clearBreakpoints", 0, 0))
}
func TestAutomationPauseAndClose(t *testing.T) {
	a := automationProgram(t, []byte{0x4c, 0, 6})
	command(t, a, "run", 0, 0)
	require.Eventually(t, func() bool { return command(t, a, "state", 0, 0).(AutomationState).Cycles != "0" }, time.Second, time.Millisecond)
	_, err := a.Execute(AutomationRequest{Version: 1, Method: "step", Count: 1})
	require.Error(t, err)
	s := command(t, a, "pause", 0, 0).(AutomationState)
	require.False(t, s.Running)
	require.Equal(t, "pause", s.StopReason)
	time.Sleep(time.Millisecond * 5)
	require.Equal(t, s, command(t, a, "state", 0, 0))
	command(t, a, "resume", 0, 0)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); a.Close() }()
	}
	wg.Wait()
	_, err = a.Execute(AutomationRequest{Version: 1, Method: "run"})
	require.Error(t, err)
}
func TestAutomationMidInstructionAndHalt(t *testing.T) {
	a := automationProgram(t, []byte{0xa9, 42, 0xdb})
	s := command(t, a, "step", 1, 0).(AutomationState)
	require.False(t, s.InstructionBoundary)
	s = command(t, a, "stepInstruction", 100, 0).(AutomationState)
	require.Equal(t, uint16(0x602), s.PC)
	require.Equal(t, uint8(42), s.A)
	s = command(t, a, "stepInstruction", 100, 0).(AutomationState)
	require.Equal(t, "cpu-stopped", s.StopReason)
	require.False(t, command(t, a, "resume", 0, 0).(AutomationState).Running)
}
func TestAutomationWaitingIsBounded(t *testing.T) {
	a := automationProgram(t, []byte{0xcb})
	// Isolate control lines as in the CPU WAI tests: board wiring shares VCC
	// with RDY, and the CPU currently drives RDY low during WAI.
	a.computer.chips.cpu.Ready().Connect(buses.NewStandaloneLine(true))
	a.computer.chips.cpu.InterruptRequest().Connect(buses.NewStandaloneLine(true))
	a.computer.chips.cpu.NonMaskableInterrupt().Connect(buses.NewStandaloneLine(true))
	s := command(t, a, "stepInstruction", 20, 0).(AutomationState)
	require.Equal(t, "cycle-limit", s.StopReason)
	require.Equal(t, "20", s.Cycles)
}
func TestAutomationMiaResume(t *testing.T) {
	a := automationProgram(t, nil)
	command(t, a, "reset", 0, 0)
	command(t, a, "step", 4000000, 0)
	command(t, a, "stepInstruction", 100, 0)
	a.computer.chips.cpu.ForceProgramCounter(0x600)
	for i, b := range []byte{0xa9, 0x30, 0x8d, 0xe9, 0xff, 0x4c, 5, 6} {
		a.computer.BaseRamPoke(uint16(0x600+i), b)
	}
	command(t, a, "run", 0, 0)
	s := waitStopped(t, a)
	require.Equal(t, "mia-paused", s.StopReason)
	require.True(t, s.Paused)
	command(t, a, "resume", 0, 0)
	s = command(t, a, "pause", 0, 0).(AutomationState)
	require.False(t, s.Paused)
}

func TestAutomationBranchBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name    string
		program []byte
		target  uint16
	}{
		{"not taken", []byte{0xa9, 0, 0xd0, 2, 0xe8, 0xdb}, 0x604},
		{"taken", []byte{0xa9, 1, 0xd0, 2, 0xea, 0xea, 0xe8, 0xdb}, 0x606},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stepped := automationProgram(t, tc.program)
			command(t, stepped, "stepInstruction", 100, 0)
			branch := command(t, stepped, "stepInstruction", 100, 0).(AutomationState)
			require.Equal(t, tc.target, branch.PC)
			require.Zero(t, branch.X)
			a := automationProgram(t, tc.program)
			// Operand address must not trigger a breakpoint.
			command(t, a, "addBreakpoint", 0, 0x601)
			command(t, a, "addBreakpoint", 0, int(tc.target))
			command(t, a, "run", 0, 0)
			s := waitStopped(t, a)
			require.Equal(t, "breakpoint", s.StopReason)
			require.Equal(t, tc.target, s.PC)
			require.Zero(t, s.X)
			s = command(t, a, "stepInstruction", 100, 0).(AutomationState)
			require.Equal(t, uint8(1), s.X)
		})
	}
}
