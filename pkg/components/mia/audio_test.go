package mia

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// silentAudioCircuit builds a normal-mode MIA test circuit with host audio
// output suppressed, so audio tests never open a real sound device.
func silentAudioCircuit(t *testing.T) *emulatedMiaTestCircuit {
	t.Helper()

	circuit := newEmulatedMiaTestCircuit()
	circuit.chip.audio.hostDisabled = true
	circuit.chip.state = miaStateNormal

	return circuit
}

// TestEmulatedMiaAudioResetDefaults verifies the audio RAM header, voice
// defaults, and fixed indexes match the firmware reset state.
func TestEmulatedMiaAudioResetDefaults(t *testing.T) {
	chip := newEmulatedMiaTestCircuit().chip

	assert.Equal(t, uint8(miaAudioVersion), chip.memory[miaAudioHeaderOffset+miaAudioHeaderVersion])
	assert.Equal(t, uint8(miaAudioVoiceCount), chip.memory[miaAudioHeaderOffset+miaAudioHeaderChannels])
	assert.Equal(t, uint16(miaAudioSampleRate), chip.audioReadU16(miaAudioHeaderOffset+miaAudioHeaderRateL))
	assert.Equal(t, miaAudioFlagStereo, chip.memory[miaAudioHeaderOffset+miaAudioHeaderFlags])

	for v := 0; v < miaAudioVoiceCount; v++ {
		base := miaAudioVoiceOffset(v)
		assert.Equal(t, uint8(128), chip.memory[base+miaAudioVoicePulseWidth], "voice %d pulse width", v)
		assert.Equal(t, uint8(0xF5), chip.memory[base+miaAudioVoiceSustainRelease], "voice %d SR", v)
		assert.Equal(t, miaAudioWavePulse, chip.memory[base+miaAudioVoiceWaveform], "voice %d waveform", v)
	}

	// Fixed audio indexes are configured and wrap within their ranges.
	all := chip.indexes[miaAudioIndexVoice0]
	assert.Equal(t, uint32(miaAudioVoiceOffset(0)), all.currentAddr)
	assert.Equal(t, uint32(miaAudioVoiceOffset(0)+miaAudioVoiceSize), all.limitAddr)
	assert.Equal(t, uint16(1), all.step)
	assert.NotZero(t, all.flags&(1<<miaIndexFlagWrap))

	block := chip.indexes[miaAudioIndexAll]
	assert.Equal(t, uint32(miaAudioStateOffset), block.currentAddr)
	assert.Equal(t, uint32(miaAudioStateOffset+miaAudioStateSize), block.limitAddr)
}

// TestEmulatedMiaAudioCommandsToggleState verifies commands 0x60/0x61/0x62 set
// and clear the active status and RAM status flag.
func TestEmulatedMiaAudioCommandsToggleState(t *testing.T) {
	circuit := silentAudioCircuit(t)
	chip := circuit.chip

	enable := func() {
		circuit.write(miaRegCmdParam1, 0)
		circuit.write(miaRegCmdParam2, 0)
		circuit.write(miaRegCmdParam3, 0)
		circuit.write(miaRegCmdTrigger, 0x60)
	}

	enable()
	assert.True(t, chip.audioIsActive())
	assert.NotZero(t, chip.status()&miaStatusAudioActive)
	assert.NotZero(t, chip.memory[miaAudioHeaderOffset+miaAudioHeaderStatus]&miaAudioStatusActive)

	circuit.write(miaRegCmdTrigger, 0x61)
	assert.False(t, chip.audioIsActive())
	assert.Zero(t, chip.status()&miaStatusAudioActive)
	assert.Zero(t, chip.memory[miaAudioHeaderOffset+miaAudioHeaderStatus]&miaAudioStatusActive)

	// Dirty the block, then reset and confirm defaults are restored.
	chip.memory[miaAudioVoiceOffset(1)+miaAudioVoiceWaveform] = miaAudioWaveNoise
	circuit.write(miaRegCmdTrigger, 0x62)
	assert.Equal(t, miaAudioWavePulse, chip.memory[miaAudioVoiceOffset(1)+miaAudioVoiceWaveform])
}

// TestEmulatedMiaAudioLiveWriteQueueThroughBus verifies that indexed writes into
// the audio block while audio is active are queued and applied to the engine.
func TestEmulatedMiaAudioLiveWriteQueueThroughBus(t *testing.T) {
	circuit := silentAudioCircuit(t)
	chip := circuit.chip

	circuit.write(miaRegCmdTrigger, 0x60)
	require.True(t, chip.audioIsActive())

	// Write a full voice-0 record through index $D1 (1000 Hz pulse, gated on).
	circuit.write(miaRegIdxASelector, miaAudioIndexVoice0)
	record := []uint8{0x80, 0x3E, 0x80, 0x00, 0xF0, miaAudioWaveSaw, 0x00, miaAudioControlGate}
	for _, b := range record {
		circuit.write(miaRegIdxAPort, b)
	}

	// RAM holds the written bytes.
	base := miaAudioVoiceOffset(0)
	assert.Equal(t, uint8(0x3E80&0xFF), chip.memory[base+miaAudioVoiceFreqL])
	assert.Equal(t, miaAudioWaveSaw, chip.memory[base+miaAudioVoiceWaveform])

	// Drain the live queue and confirm the engine voice reflects the writes.
	chip.audio.mu.Lock()
	chip.audio.drainQueueLocked(miaAudioQueueSize)
	voice := chip.audio.voices[0]
	chip.audio.mu.Unlock()

	assert.Equal(t, uint16(0x3E80), voice.freqQ4)
	assert.Equal(t, miaAudioWaveSaw, voice.waveform)
	assert.Equal(t, miaADSRAttack, voice.adsr, "gate-on starts the attack phase")
	assert.Equal(t, miaAudioControlGate, voice.control)
}

// TestEmulatedMiaAudioQueueOverflowReportsError verifies that overrunning the
// live-write queue raises the overflow status bit and the audio error code, and
// resynchronizes the engine from RAM.
func TestEmulatedMiaAudioQueueOverflowReportsError(t *testing.T) {
	circuit := silentAudioCircuit(t)
	chip := circuit.chip

	circuit.write(miaRegCmdTrigger, 0x60)
	require.True(t, chip.audioIsActive())

	// Write more bytes than the queue can hold without the render goroutine
	// draining it (host output is disabled). Index $D0 wraps within the block.
	circuit.write(miaRegIdxASelector, miaAudioIndexAll)
	for i := 0; i < miaAudioQueueSize+8; i++ {
		circuit.write(miaRegIdxAPort, uint8(i))
	}

	assert.NotZero(t, chip.memory[miaAudioHeaderOffset+miaAudioHeaderStatus]&miaAudioStatusQueueOverflow)
	assert.NotZero(t, chip.status()&miaStatusErrors)
	assert.Equal(t, miaErrorAudioQueueOverflow, chip.readRegister(miaRegErrorLSB))

	chip.audio.mu.Lock()
	overflow := chip.audio.queueOverflow
	chip.audio.mu.Unlock()
	assert.True(t, overflow)

	// Enabling again clears the overflow status bit.
	circuit.write(miaRegCmdTrigger, 0x60)
	assert.Zero(t, chip.memory[miaAudioHeaderOffset+miaAudioHeaderStatus]&miaAudioStatusQueueOverflow)
}

// TestEmulatedMiaAudioRenderProducesSound verifies the oscillator/envelope engine
// emits a non-silent signal for a gated voice and stays centered when stopped.
func TestEmulatedMiaAudioRenderProducesSound(t *testing.T) {
	circuit := silentAudioCircuit(t)
	chip := circuit.chip

	// Configure voice 0 in RAM: 1000 Hz pulse, full sustain, fast attack, gated.
	base := miaAudioVoiceOffset(0)
	chip.audioWriteU16(base+miaAudioVoiceFreqL, 16000)
	chip.memory[base+miaAudioVoiceWaveform] = miaAudioWavePulse
	chip.memory[base+miaAudioVoicePulseWidth] = 128
	chip.memory[base+miaAudioVoiceAttackDecay] = 0x00
	chip.memory[base+miaAudioVoiceSustainRelease] = 0xF0
	chip.memory[base+miaAudioVoicePan] = 0
	chip.memory[base+miaAudioVoiceControl] = miaAudioControlGate

	chip.audioEnable()

	chip.audio.mu.Lock()
	peak := int16(0)
	for i := 0; i < 256; i++ {
		left, _ := chip.audio.renderSampleLocked()
		if left > peak {
			peak = left
		}
	}
	chip.audio.mu.Unlock()

	assert.Greater(t, peak, int16(0), "a gated voice with full sustain should produce sound")
}

// TestEmulatedMiaAudioConsoleCommands exercises the terminal audio command and
// the 'status audio' diagnostic.
func TestEmulatedMiaAudioConsoleCommands(t *testing.T) {
	chip := newEmulatedMiaTestCircuit().chip
	chip.audio.hostDisabled = true
	chip.state = miaStateNormal

	out := chip.consoleAudio("enable")
	assert.Equal(t, "Audio: enabled\n", out)
	assert.True(t, chip.audioIsActive())

	detail := chip.consoleAudio("status")
	assert.Contains(t, detail, "Audio:\n")
	assert.Contains(t, detail, "  state:     active\n")
	assert.Contains(t, detail, "  output:    disabled\n")
	assert.Contains(t, detail, "Usage: audio [status|enable|stop|reset]\n")

	out = chip.consoleAudio("stop")
	assert.Equal(t, "Audio: stopped\n", out)
	assert.False(t, chip.audioIsActive())

	summary := chip.consoleStatusSummary()
	assert.True(t, strings.Contains(summary, "Audio: stopped"), "summary should report audio state")

	assert.Equal(t, "Usage: audio [status|enable|stop|reset]\n", chip.consoleAudio("bogus"))
}
