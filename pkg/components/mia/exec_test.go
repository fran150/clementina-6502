package mia

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEmulatedMiaExecPauseCommand verifies the 6502 pause command (0x30) sets
// MIA_STAT_EXEC_PAUSED and forwards a single pause edge to the host. There is no
// 6502 resume command: once PHI2 is stopped the CPU cannot fetch the trigger, so
// resume is console/reset only. Issuing the old 0x31 id reports ERROR_CMD_UNKNOWN,
// mirroring the firmware command table.
func TestEmulatedMiaExecPauseCommand(t *testing.T) {
	circuit := newEmulatedMiaTestCircuit()
	chip := circuit.chip
	chip.state = miaStateNormal

	var paused atomic.Bool
	var edges atomic.Int32
	chip.SetExecPausedHandler(func(p bool) {
		paused.Store(p)
		edges.Add(1)
	})

	circuit.write(miaRegCmdTrigger, 0x30)
	assert.Equal(t, miaStatusExecPaused, chip.status()&miaStatusExecPaused)
	assert.True(t, chip.execIsPaused())
	assert.True(t, paused.Load())
	assert.Equal(t, int32(1), edges.Load())

	// Re-issuing pause while already paused is idempotent: no second edge.
	circuit.write(miaRegCmdTrigger, 0x30)
	assert.Equal(t, int32(1), edges.Load())

	// 0x31 is not a command: the 6502 cannot resume itself.
	circuit.write(miaRegCmdTrigger, 0x31)
	assert.True(t, chip.execIsPaused())
	assert.Equal(t, miaErrorCmdUnknown, circuit.read(miaRegErrorLSB))

	// Resume is the console/internal path; it forwards the resume edge.
	chip.execResume()
	assert.Zero(t, chip.status()&miaStatusExecPaused)
	assert.False(t, chip.execIsPaused())
	assert.False(t, paused.Load())
	assert.Equal(t, int32(2), edges.Load())
}

// TestEmulatedMiaExecResumesOnReset verifies a runtime-state reset releases the
// pause, mirroring mia_reset_runtime_state calling mia_exec_resume.
func TestEmulatedMiaExecResumesOnReset(t *testing.T) {
	chip := NewEmulatedMia().(*emulated_mia)
	chip.state = miaStateNormal

	var lastEdge atomic.Bool
	chip.SetExecPausedHandler(func(p bool) {
		lastEdge.Store(p)
	})

	chip.execPause()
	require.True(t, chip.execIsPaused())
	require.True(t, lastEdge.Load())

	chip.init()

	assert.False(t, chip.execIsPaused())
	assert.Zero(t, chip.status()&miaStatusExecPaused)
	assert.False(t, lastEdge.Load())
}

// TestEmulatedMiaConsoleExecCommand verifies the terminal 'exec' command pauses
// and resumes PHI2 and that 'status exec' reports the state.
func TestEmulatedMiaConsoleExecCommand(t *testing.T) {
	chip, mock := newMiaConsoleTest(t)

	var paused atomic.Bool
	chip.SetExecPausedHandler(func(p bool) {
		paused.Store(p)
	})

	waitForMiaConsoleOutput(t, mock, "MIA ready. Type 'help' for commands.\n> ")

	sendMiaConsoleInput(mock, "exec\n")
	waitForMiaConsoleOutput(t, mock, "Exec: running  PHI2:running\n")
	waitForMiaConsoleOutput(t, mock, "Usage: exec [status|pause|resume]\n")

	sendMiaConsoleInput(mock, "exec pause\n")
	waitForMiaConsoleOutput(t, mock, "Exec: paused\n")
	require.Eventually(t, func() bool {
		chip.mu.Lock()
		defer chip.mu.Unlock()
		return chip.execIsPaused() && chip.status()&miaStatusExecPaused != 0
	}, time.Second, 10*time.Millisecond)
	assert.True(t, paused.Load())

	sendMiaConsoleInput(mock, "status exec\n")
	waitForMiaConsoleOutput(t, mock, "Exec: paused  PHI2:stopped-low\n")

	sendMiaConsoleInput(mock, "exec resume\n")
	waitForMiaConsoleOutput(t, mock, "Exec: running\n")
	require.Eventually(t, func() bool {
		return !paused.Load()
	}, time.Second, 10*time.Millisecond)
}

// TestEmulatedMiaConsoleStatusSubcommands exercises the expanded 'status'
// dashboard and its subsystem subcommands.
func TestEmulatedMiaConsoleStatusSubcommands(t *testing.T) {
	_, mock := newMiaConsoleTest(t)

	waitForMiaConsoleOutput(t, mock, "MIA ready. Type 'help' for commands.\n> ")

	sendMiaConsoleInput(mock, "status\n")
	waitForMiaConsoleOutput(t, mock, "  Exec:   Running\n")
	waitForMiaConsoleOutput(t, mock, "Video: enabled")

	sendMiaConsoleInput(mock, "status irq\n")
	waitForMiaConsoleOutput(t, mock, "IRQ:\n")
	waitForMiaConsoleOutput(t, mock, "  line:     released\n")

	sendMiaConsoleInput(mock, "status speed\n")
	waitForMiaConsoleOutput(t, mock, "Speed:\n")
	waitForMiaConsoleOutput(t, mock, "  pending:   no\n")

	sendMiaConsoleInput(mock, "status mem\n")
	waitForMiaConsoleOutput(t, mock, "Memory:\n")

	sendMiaConsoleInput(mock, "status index 0\n")
	waitForMiaConsoleOutput(t, mock, "  index 0: current:$000000")

	sendMiaConsoleInput(mock, "status bogus\n")
	waitForMiaConsoleOutput(t, mock, "Usage: status [video|input|audio|wifi|irq|speed|exec|errors|mem|index [id]]\n")
}
