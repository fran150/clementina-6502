package clementina

import (
	"testing"
	"time"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/stretchr/testify/require"
)

func TestAutomationPhysicalBankBreakpoint(t *testing.T) {
	a := automationProgram(t, nil)
	step := common.NewStepContext()
	write := func(address uint16, value uint8) {
		a.computer.circuit.addressBus.Write(address)
		a.computer.circuit.dataBus.Write(value)
		a.computer.circuit.cpuRW.Set(false)
		a.computer.chips.csLogic.Tick(&step)
		a.computer.chips.oeRWSync.Tick(&step)
		a.computer.chips.via.Tick(&step)
		a.computer.chips.exram.Tick(&step)
		a.computer.circuit.cpuRW.Set(true)
		step.NextCycle()
	}
	write(0xc003, 0x1f)
	for i, b := range []uint8{0x4c, 0x00, 0x80} {
		a.computer.ExRamPoke(uint16(0x8000+i), 1, b)
	}
	for i, b := range []uint8{0xe8, 0x4c, 0x00, 0x80} {
		a.computer.ExRamPoke(uint16(0x8000+i), 2, b)
	}
	bank := 2
	_, err := a.Execute(AutomationRequest{Version: 1, Method: "addBankBreakpoint", Address: 0x8000, Bank: &bank})
	require.NoError(t, err)
	write(0xc001, 1)
	a.computer.chips.cpu.ForceProgramCounter(0x8000)
	command(t, a, "run", 0, 0)
	time.Sleep(10 * time.Millisecond)
	state := command(t, a, "pause", 0, 0).(AutomationState)
	require.Equal(t, "pause", state.StopReason)
	require.Equal(t, uint8(1), state.Bank)
	require.Equal(t, []any{0x4c}, command(t, a, "readMemory", 1, 0x8000))
	read, err := a.Execute(AutomationRequest{Version: 1, Method: "readMemory", Address: 0x8000, Count: 1, Bank: &bank})
	require.NoError(t, err)
	require.Equal(t, []any{0xe8}, read)
	write(0xc001, 2)
	command(t, a, "resume", 0, 0)
	state = waitStopped(t, a)
	require.Equal(t, "breakpoint", state.StopReason)
	require.Equal(t, uint8(2), state.Bank)
	require.Equal(t, uint16(0x8000), state.PC)
}

func TestAutomationVerifiedCallStack(t *testing.T) {
	program := []byte{0x20, 0x08, 0x06, 0xdb, 0xea, 0xea, 0xea, 0xea, 0x20, 0x10, 0x06, 0x60, 0xea, 0xea, 0xea, 0xea, 0xea, 0x60}
	a := automationProgram(t, program)
	stack := func() AutomationStack { return command(t, a, "stackTrace", 0, 0).(AutomationStack) }
	require.Empty(t, stack().Callers)
	command(t, a, "stepInstruction", 100, 0)
	require.Equal(t, []AutomationCaller{{PC: 0x603, Bank: 0, Kind: "call"}}, stack().Callers)
	command(t, a, "stepInstruction", 100, 0)
	require.Equal(t, []AutomationCaller{{PC: 0x60b, Bank: 0, Kind: "call"}, {PC: 0x603, Bank: 0, Kind: "call"}}, stack().Callers)
	command(t, a, "stepInstruction", 100, 0)
	command(t, a, "stepInstruction", 100, 0)
	require.Equal(t, []AutomationCaller{{PC: 0x603, Bank: 0, Kind: "call"}}, stack().Callers)
	command(t, a, "stepInstruction", 100, 0)
	require.Empty(t, stack().Callers)
	require.True(t, stack().UnknownCaller)

	ambiguous := automationProgram(t, program)
	command(t, ambiguous, "stepInstruction", 100, 0)
	frame := ambiguous.trace[0]
	ambiguous.computer.BaseRamPoke(0x100|uint16(frame.entrySP), 0xff)
	result := command(t, ambiguous, "stackTrace", 0, 0).(AutomationStack)
	require.Empty(t, result.Callers)
	require.True(t, result.UnknownCaller)
}

func TestAutomationInterruptFrameAndReturn(t *testing.T) {
	a := automationProgram(t, []byte{0x00, 0xea, 0xdb})
	state := command(t, a, "stepInstruction", 100, 0).(AutomationState)
	require.Equal(t, "instruction", state.StopReason)
	stack := command(t, a, "stackTrace", 0, 0).(AutomationStack)
	require.Equal(t, []AutomationCaller{{PC: 0x602, Bank: 0, Kind: "interrupt"}}, stack.Callers)
	a.computer.BaseRamPoke(0x610, 0x40) // RTI handler for this isolated test.
	a.computer.chips.cpu.ForceProgramCounter(0x610)
	state = command(t, a, "stepInstruction", 100, 0).(AutomationState)
	require.Equal(t, uint16(0x602), state.PC)
	require.Empty(t, command(t, a, "stackTrace", 0, 0).(AutomationStack).Callers)
}
