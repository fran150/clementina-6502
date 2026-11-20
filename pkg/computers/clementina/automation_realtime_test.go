package clementina

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAutomationBreakpointFilter(t *testing.T) {
	// loop: INC $10; BNE loop; INC $11; JMP loop - a 16-bit counter at $10.
	a := automationProgram(t, []byte{0xe6, 0x10, 0xd0, 0xfc, 0xe6, 0x11, 0x4c, 0x00, 0x06})
	command(t, a, "addBreakpoint", 0, 0x600)
	watch := 0x10
	_, err := a.Execute(AutomationRequest{Version: 1, Method: "setBreakpointFilter", Address: 0x600, Watch: &watch, Values: []int{300}})
	require.NoError(t, err)
	command(t, a, "run", 0, 0)
	state := waitStopped(t, a)
	require.Equal(t, "breakpoint", state.StopReason)
	require.Equal(t, uint16(0x600), state.PC)
	require.Equal(t, []any{300 & 0xff, 300 >> 8}, command(t, a, "readMemory", 2, 0x10))

	// Without the filter the same breakpoint stops on the next pass.
	command(t, a, "clearBreakpointFilter", 0, 0x600)
	command(t, a, "resume", 0, 0)
	state = waitStopped(t, a)
	require.Equal(t, "breakpoint", state.StopReason)
	require.Equal(t, []any{301 & 0xff, 301 >> 8}, command(t, a, "readMemory", 2, 0x10))

	_, err = a.Execute(AutomationRequest{Version: 1, Method: "setBreakpointFilter", Address: 0x600, Watch: &watch})
	require.Error(t, err)
}

func TestAutomationUnthrottledHasNoSpeedControl(t *testing.T) {
	a := automationProgram(t, nil)
	speed := command(t, a, "speed", 0, 0).(AutomationSpeed)
	require.False(t, speed.Realtime)
	require.True(t, speed.Unthrottled)
	mhz := 2.0
	_, err := a.Execute(AutomationRequest{Version: 1, Method: "setSpeed", MHz: &mhz})
	require.Error(t, err)
}

func TestRealtimeAutomationKeepsPHI2Time(t *testing.T) {
	c, err := NewClementinaComputer()
	require.NoError(t, err)
	a := NewRealtimeAutomation(c, 0.5)
	t.Cleanup(a.Close)
	command(t, a, "reset", 0, 0)
	command(t, a, "run", 0, 0)
	time.Sleep(200 * time.Millisecond)
	start := command(t, a, "state", 0, 0).(AutomationState)
	time.Sleep(500 * time.Millisecond)
	end := command(t, a, "state", 0, 0).(AutomationState)
	cycles := parseCycles(t, end.Cycles) - parseCycles(t, start.Cycles)
	// 0.5 MHz for half a second, with room for a loaded test host.
	require.InDelta(t, 250_000, cycles, 50_000)

	speed := command(t, a, "speed", 0, 0).(AutomationSpeed)
	require.True(t, speed.Realtime)
	require.InDelta(t, 0.5, speed.TargetMHz, 1e-9)
	require.InDelta(t, 0.5, speed.MeasuredMHz, 0.1)

	zero := 0.0
	v, err := a.Execute(AutomationRequest{Version: 1, Method: "setSpeed", MHz: &zero})
	require.NoError(t, err)
	require.True(t, v.(AutomationSpeed).Unthrottled)
	tooFast := 101.0
	_, err = a.Execute(AutomationRequest{Version: 1, Method: "setSpeed", MHz: &tooFast})
	require.Error(t, err)
}

func parseCycles(t *testing.T, text string) int {
	t.Helper()
	n, err := strconv.Atoi(text)
	require.NoError(t, err)
	return n
}
