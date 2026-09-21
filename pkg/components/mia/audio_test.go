package mia

import (
	"math"
	"strings"
	"testing"
	"time"

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

// TestEmulatedMiaAudioLiveWriteThroughBus verifies indexed writes into the audio
// block while audio is active are applied to the engine immediately (lockstep -
// no queue).
func TestEmulatedMiaAudioLiveWriteThroughBus(t *testing.T) {
	circuit := silentAudioCircuit(t)
	chip := circuit.chip

	circuit.write(miaRegCmdTrigger, 0x60)
	require.True(t, chip.audioIsActive())

	// Write a full voice-0 record through index $E7 (1000 Hz pulse, gated on).
	circuit.write(miaRegIdxASelector, miaAudioIndexVoice0)
	record := []uint8{0x80, 0x3E, 0x80, 0x00, 0xF0, miaAudioWaveSaw, 0x00, miaAudioControlGate}
	for _, b := range record {
		circuit.write(miaRegIdxAPort, b)
	}

	base := miaAudioVoiceOffset(0)
	assert.Equal(t, uint8(0x3E80&0xFF), chip.memory[base+miaAudioVoiceFreqL])
	assert.Equal(t, miaAudioWaveSaw, chip.memory[base+miaAudioVoiceWaveform])

	chip.DebugAudioFlush() // apply the queued (time-stamped) writes now
	chip.audio.mu.Lock()
	voice := chip.audio.voices[0]
	chip.audio.mu.Unlock()

	assert.Equal(t, uint16(0x3E80), voice.freqQ4)
	assert.Equal(t, miaAudioWaveSaw, voice.waveform)
	assert.Equal(t, miaADSRAttack, voice.adsr, "gate-on starts the attack phase")
	assert.Equal(t, miaAudioControlGate, voice.control)
}

// TestEmulatedMiaAudioReAttackOnGateReassert verifies the write burst BASIC's
// PLAY uses per note (CONTROL=0, then FREQ, then CONTROL=GATE|RESET_PHASE)
// re-triggers the envelope from attack/vol=0 even when the voice was already
// gated and sitting at sustain - so repeated PLAY notes are audibly distinct.
func TestEmulatedMiaAudioReAttackOnGateReassert(t *testing.T) {
	circuit := silentAudioCircuit(t)
	chip := circuit.chip

	base := miaAudioVoiceOffset(0)
	chip.audioWriteU16(base+miaAudioVoiceFreqL, 8000)
	chip.memory[base+miaAudioVoiceSustainRelease] = 0xF0 // sustain full, fast release
	chip.memory[base+miaAudioVoiceControl] = miaAudioControlGate
	chip.audioEnable()

	// Let the first note settle into sustain.
	chip.audio.mu.Lock()
	for i := 0; i < 400; i++ {
		chip.audio.renderSampleLocked()
	}
	settled := chip.audio.voices[0]
	chip.audio.mu.Unlock()
	require.Equal(t, miaADSRSustain, settled.adsr, "first note should reach sustain")
	require.Greater(t, settled.vol, uint32(0))

	// PLAY's per-note write burst: CONTROL=0 (release), FREQ_L/H, then
	// CONTROL=GATE|RESET_PHASE (re-attack). Applied to the engine immediately.
	writeAt := func(off uint32, v uint8) {
		chip.mu.Lock()
		chip.memory[base+off] = v
		chip.audioCore1OnWrite(base+off, v)
		chip.mu.Unlock()
	}
	writeAt(miaAudioVoiceControl, 0x00)
	writeAt(miaAudioVoiceFreqL, 0x40)
	writeAt(miaAudioVoiceFreqH, 0x1F)
	writeAt(miaAudioVoiceControl, miaAudioControlGate|miaAudioControlResetPhase)

	chip.DebugAudioFlush()
	chip.audio.mu.Lock()
	v := chip.audio.voices[0]
	chip.audio.mu.Unlock()

	assert.Equal(t, miaADSRAttack, v.adsr, "re-asserting the gate re-attacks the envelope")
	assert.Equal(t, uint32(0), v.vol, "re-attack restarts volume from 0")
	assert.Equal(t, uint32(0), v.phase, "RESET_PHASE restarts the oscillator")
}

// TestEmulatedMiaAudioManyLiveWritesNoLoss verifies a long burst of live writes
// all reach the engine. The firmware queues these for its 24 kHz IRQ and can
// overrun a 64-entry queue; the emulator applies them in lockstep with the CPU,
// so there is no queue to overrun and nothing is lost or applied in a burst.
func TestEmulatedMiaAudioManyLiveWritesNoLoss(t *testing.T) {
	circuit := silentAudioCircuit(t)
	chip := circuit.chip

	circuit.write(miaRegCmdTrigger, 0x60)
	require.True(t, chip.audioIsActive())

	base := miaAudioVoiceOffset(0)
	last := uint8(0)
	for i := 0; i < 500; i++ {
		last = uint8(i % 5)
		chip.mu.Lock()
		chip.memory[base+miaAudioVoiceWaveform] = last
		chip.audioCore1OnWrite(base+miaAudioVoiceWaveform, last)
		chip.mu.Unlock()

		chip.DebugAudioFlush()
		chip.audio.mu.Lock()
		got := chip.audio.voices[0].waveform
		chip.audio.mu.Unlock()
		require.Equal(t, last, got, "write %d applied to the engine at once", i)
	}

	assert.Zero(t, chip.memory[miaAudioHeaderOffset+miaAudioHeaderStatus]&miaAudioStatusQueueOverflow,
		"no queue -> no overflow status")
	assert.Zero(t, chip.status()&miaStatusErrors, "no queue -> no overflow error")
}

// TestEmulatedMiaAudioWallClockPacedReader is the regression for "PLAY drops
// short notes / plays one note for ~0.5 s". oto's mux asks miaAudioReader.Read
// for up to half a second of frames at once; rendering that in one call applied
// every queued note write into the first handful of frames, so short notes were
// inaudible. The oto-driven reader (capToWallClock) must hand out audio no
// faster than wall-clock, so writes spread out in real time render as separate,
// audible spans.
func TestEmulatedMiaAudioWallClockPacedReader(t *testing.T) {
	circuit := silentAudioCircuit(t)
	chip := circuit.chip

	configTestVoice0(chip)
	base := miaAudioVoiceOffset(0)
	chip.memory[base+miaAudioVoiceControl] = 0 // start ungated
	chip.audioEnable()

	reader := &miaAudioReader{chip: chip, capToWallClock: true}
	drain := func() []int16 {
		buf := make([]byte, miaAudioSampleRate*4) // ask for a full second every time
		n, _ := reader.Read(buf)
		out := make([]int16, n/2)
		for i := range out {
			out[i] = int16(uint16(buf[i*2]) | uint16(buf[i*2+1])<<8)
		}
		return out
	}

	// A one-second request right after enable must not return a whole second -
	// only about the lead - because wall-clock has barely advanced.
	first := len(drain()) / 2
	require.Less(t, first, miaAudioLeadSamples+miaAudioSampleRate/10,
		"capped reader rendered %d frames with no wall-clock elapsed (lead is %d)", first, miaAudioLeadSamples)

	gate := func(v uint8) {
		chip.mu.Lock()
		chip.memory[base+miaAudioVoiceControl] = v
		chip.audioCore1OnWrite(base+miaAudioVoiceControl, v)
		chip.mu.Unlock()
	}

	// Hold a note for ~90 ms of real time, draining as we go, then release.
	gate(miaAudioControlGate)
	var note []int16
	for i := 0; i < 6; i++ {
		time.Sleep(15 * time.Millisecond)
		note = append(note, drain()...)
	}
	gate(0)

	noteFrames := len(note) / 2
	require.Greater(t, noteFrames, miaAudioSampleRate*50/1000,
		"a note held ~90 ms rendered only %d frames - it collapsed instead of tracking wall-clock", noteFrames)

	// The held note must carry energy the whole way through - a collapsed write
	// stream would show up as silent windows.
	const windows = 4
	w := noteFrames / windows
	for k := 0; k < windows; k++ {
		var sum int64
		for f := k * w; f < (k+1)*w; f++ {
			s := int64(note[f*2])
			sum += s * s
		}
		rms := math.Sqrt(float64(sum) / float64(w))
		require.Greater(t, rms, 100.0, "window %d/%d of the held note is silent (rms %.0f)", k+1, windows, rms)
	}
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

// configTestVoice0 writes a gated, full-sustain pulse voice 0 into audio RAM.
func configTestVoice0(chip *emulated_mia) {
	base := miaAudioVoiceOffset(0)
	chip.audioWriteU16(base+miaAudioVoiceFreqL, 16000)
	chip.memory[base+miaAudioVoiceWaveform] = miaAudioWavePulse
	chip.memory[base+miaAudioVoicePulseWidth] = 128
	chip.memory[base+miaAudioVoiceAttackDecay] = 0x00
	chip.memory[base+miaAudioVoiceSustainRelease] = 0xF0
	chip.memory[base+miaAudioVoicePan] = 0
	chip.memory[base+miaAudioVoiceVolume] = miaAudioVoiceVolumeDefault
	chip.memory[base+miaAudioVoiceControl] = miaAudioControlGate
}

// renderLeftPeak renders 256 samples off the engine and returns the loudest
// left-channel value. Caller must not hold a.mu.
func renderLeftPeak(chip *emulated_mia) int16 {
	chip.audio.mu.Lock()
	defer chip.audio.mu.Unlock()

	peak := int16(0)
	for i := 0; i < 256; i++ {
		left, _ := chip.audio.renderSampleLocked()
		if left > peak {
			peak = left
		}
	}

	return peak
}

// TestEmulatedMiaAudioVolumeResetDefaults verifies the layout-v2 master and
// per-voice volume bytes reset to full in both RAM and the engine.
func TestEmulatedMiaAudioVolumeResetDefaults(t *testing.T) {
	chip := newEmulatedMiaTestCircuit().chip

	assert.Equal(t, uint8(2), chip.memory[miaAudioHeaderOffset+miaAudioHeaderVersion], "layout version")
	assert.Equal(t, miaAudioMasterVolumeDefault, chip.memory[miaAudioHeaderOffset+miaAudioHeaderVolume])
	for v := 0; v < miaAudioVoiceCount; v++ {
		base := miaAudioVoiceOffset(v)
		assert.Equal(t, miaAudioVoiceVolumeDefault, chip.memory[base+miaAudioVoiceVolume], "voice %d volume byte", v)
	}

	// Voice records are 16 bytes apart in layout v2.
	assert.Equal(t, uint32(0x12010), miaAudioVoiceOffset(0))
	assert.Equal(t, uint32(0x12040), miaAudioVoiceOffset(3))

	chip.audio.mu.Lock()
	defer chip.audio.mu.Unlock()
	assert.Equal(t, miaAudioMasterVolumeDefault, chip.audio.masterVolume)
	for v := 0; v < miaAudioVoiceCount; v++ {
		assert.Equal(t, miaAudioVoiceVolumeDefault, chip.audio.voices[v].volume, "voice %d engine volume", v)
	}
}

// TestEmulatedMiaAudioPerVoiceVolumeScalesOutput verifies the per-voice VOLUME
// byte attenuates that voice after the envelope, and 0 silences it.
func TestEmulatedMiaAudioPerVoiceVolumeScalesOutput(t *testing.T) {
	circuit := silentAudioCircuit(t)
	chip := circuit.chip
	volByte := miaAudioVoiceOffset(0) + miaAudioVoiceVolume

	chip.audioReset()
	configTestVoice0(chip)
	chip.memory[volByte] = 0xFF
	chip.audioEnable()
	full := renderLeftPeak(chip)
	require.Greater(t, full, int16(0), "unity voice volume should produce sound")

	chip.audioReset()
	configTestVoice0(chip)
	chip.memory[volByte] = 0x40
	chip.audioEnable()
	quarter := renderLeftPeak(chip)
	assert.Less(t, quarter, full/2, "volume 0x40 should be well below unity")
	assert.Greater(t, quarter, int16(0), "volume 0x40 is still audible")

	chip.audioReset()
	configTestVoice0(chip)
	chip.memory[volByte] = 0x00
	chip.audioEnable()
	assert.Equal(t, int16(0), renderLeftPeak(chip), "voice volume 0 is silent")
}

// TestEmulatedMiaAudioMasterVolumeScalesOutput verifies the header AUDIO_VOLUME
// nibble scales the summed mix, matching the 0..15 -> 0..255 gain, and mutes at 0.
func TestEmulatedMiaAudioMasterVolumeScalesOutput(t *testing.T) {
	circuit := silentAudioCircuit(t)
	chip := circuit.chip
	masterByte := miaAudioHeaderOffset + miaAudioHeaderVolume

	chip.audioReset()
	configTestVoice0(chip)
	chip.memory[masterByte] = 15
	chip.audioEnable()
	full := renderLeftPeak(chip)
	require.Greater(t, full, int16(0))

	chip.audioReset()
	configTestVoice0(chip)
	chip.memory[masterByte] = 7
	chip.audioEnable()
	half := renderLeftPeak(chip)
	// gain(7)/gain(15) = 119/255 ~= 0.467.
	ratio := float64(half) / float64(full)
	assert.InDelta(t, 0.467, ratio, 0.06, "master 7 ~= 0.467 of master 15")

	chip.audioReset()
	configTestVoice0(chip)
	chip.memory[masterByte] = 0
	chip.audioEnable()
	assert.Equal(t, int16(0), renderLeftPeak(chip), "master volume 0 mutes all output")
}

// TestEmulatedMiaAudioVolumeLiveWriteThroughBus verifies live indexed writes to
// the per-voice VOLUME byte and the header master volume reach the engine.
func TestEmulatedMiaAudioVolumeLiveWriteThroughBus(t *testing.T) {
	circuit := silentAudioCircuit(t)
	chip := circuit.chip

	circuit.write(miaRegCmdTrigger, 0x60)
	require.True(t, chip.audioIsActive())

	// Per-voice VOLUME is voice-record offset 8: write nine bytes through $E7.
	circuit.write(miaRegIdxASelector, miaAudioIndexVoice0)
	for _, b := range []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0x55} {
		circuit.write(miaRegIdxAPort, b)
	}

	// Master AUDIO_VOLUME is header offset 1: write two bytes through $EB.
	circuit.write(miaRegIdxASelector, miaAudioIndexHeader)
	circuit.write(miaRegIdxAPort, miaAudioVersion) // offset 0: keep version
	circuit.write(miaRegIdxAPort, 0x09)            // offset 1: master volume

	chip.DebugAudioFlush()
	chip.audio.mu.Lock()
	voiceVol := chip.audio.voices[0].volume
	master := chip.audio.masterVolume
	chip.audio.mu.Unlock()

	assert.Equal(t, uint8(0x55), voiceVol, "engine per-voice volume follows the live write")
	assert.Equal(t, uint8(0x09), master, "engine master volume follows the live write")
	assert.Equal(t, uint8(0x55), chip.memory[miaAudioVoiceOffset(0)+miaAudioVoiceVolume])
	assert.Equal(t, uint8(0x09), chip.memory[miaAudioHeaderOffset+miaAudioHeaderVolume])
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
	assert.Contains(t, detail, "Usage: audio [status|enable|stop|reset|seq]\n")

	out = chip.consoleAudio("stop")
	assert.Equal(t, "Audio: stopped\n", out)
	assert.False(t, chip.audioIsActive())

	summary := chip.consoleStatusSummary()
	assert.True(t, strings.Contains(summary, "Audio: stopped"), "summary should report audio state")

	assert.Equal(t, "Usage: audio [status|enable|stop|reset|seq]\n", chip.consoleAudio("bogus"))
}
