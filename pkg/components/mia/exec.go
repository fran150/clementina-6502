package mia

// This file implements the MIA execution control. It mirrors the Pico firmware
// (src/mia/sys/exec.c). On real hardware "pause" stops PHI2 by disabling the
// write PIO state machine and holding PHI2 low through SIO, halting the 6502.
// In the emulator the 6502 clock is driven by the host emulation loop, so the
// pause is forwarded through a callback that stops and restarts that loop (see
// SetExecPausedHandler). The 6502-facing behavior is identical: the
// MIA_STAT_EXEC_PAUSED status bit tracks the state and command 0x30 pauses. There
// is no 6502 resume command — a frozen CPU cannot fetch it — so resume comes only
// from the terminal console ('exec resume') or a reset.

// execPause stops 6502 execution by stopping PHI2. It is idempotent and sets the
// MIA_STAT_EXEC_PAUSED status flag. Once paused, resume normally comes from the
// terminal with 'exec resume'.
func (c *emulated_mia) execPause() {
	if c.execPaused {
		return
	}

	c.execPaused = true
	c.statusSet(miaStatusExecPaused)
	c.notifyExecPaused(true)
}

// execResume restarts 6502 execution by restarting PHI2. It is idempotent and
// clears the MIA_STAT_EXEC_PAUSED status flag.
func (c *emulated_mia) execResume() {
	if !c.execPaused {
		return
	}

	c.execPaused = false
	c.statusClear(miaStatusExecPaused)
	c.notifyExecPaused(false)
}

// execIsPaused reports whether PHI2 is currently stopped by the exec control.
func (c *emulated_mia) execIsPaused() bool {
	return c.execPaused
}

// execResetRuntimeState resumes execution, mirroring mia_reset_runtime_state
// which calls mia_exec_resume() so a 6502 reset always releases the clock.
func (c *emulated_mia) execResetRuntimeState() {
	c.execResume()
}

// execStatusString renders the human-readable exec status line printed by the
// console, mirroring firmware mia_exec_print_status.
func (c *emulated_mia) execStatusString() string {
	if c.execPaused {
		return "Exec: paused  PHI2:stopped-low\n"
	}

	return "Exec: running  PHI2:running\n"
}

// notifyExecPaused forwards a pause/resume edge to the host emulation loop.
func (c *emulated_mia) notifyExecPaused(paused bool) {
	if c.execPausedChanged == nil {
		return
	}

	c.execPausedChanged(paused)
}

// SetExecPausedHandler installs a callback invoked when MIA pauses or resumes
// 6502 execution (stops or restarts PHI2). The callback receives true on pause
// and false on resume.
func (c *emulated_mia) SetExecPausedHandler(handler func(bool)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.execPausedChanged = handler
}
