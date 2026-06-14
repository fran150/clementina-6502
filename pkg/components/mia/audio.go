package mia

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
)

// This file ports the Pico MIA PWM PSG audio subsystem (firmware
// src/mia/audio/audio.{c,h}) to the emulator. The 6502-facing behavior is
// reproduced exactly: the audio RAM block at $12000, the voice/header register
// layout, the $D0-$D5 audio indexes, commands $60-$62, the MIA_STAT_AUDIO_ACTIVE
// status bit, ERROR_AUDIO_QUEUE_OVERFLOW, and the 4-voice oscillator/ADSR/pan/mix
// engine. The only platform-specific divergence is the output stage: the firmware
// drives a stereo PWM pin pair, while the emulator renders the same mixed samples
// to the host sound device through ebitengine/oto.
//
// Threading: the firmware audio IRQ runs on core 0 and observes RAM writes
// queued by core 1. The emulator mirrors this with a single-producer queue. The
// producer is the 6502 bus path (audioCore1OnWrite, always called under the chip
// mutex c.mu); the consumer is the oto render goroutine (miaAudioReader.Read,
// which only takes the audio mutex a.mu and never c.mu). Lock order is always
// c.mu -> a.mu so the two can never deadlock. State that must be read from MIA
// RAM (the AUDIO_ENABLE/overflow resync) is captured by the producer while it
// holds c.mu and copied into the engine, so the render goroutine never needs RAM.

const (
	miaAudioSampleRate = 24000
	miaAudioVoiceCount = 4
	miaAudioVoiceSize  = 8

	miaAudioStateOffset  = 0x12000
	miaAudioStateSize    = 0x40
	miaAudioHeaderOffset = miaAudioStateOffset
	miaAudioHeaderSize   = 0x10
	miaAudioVoicesOffset = miaAudioStateOffset + miaAudioHeaderSize

	miaAudioVersion = 1

	miaAudioHeaderVersion  = 0x00
	miaAudioHeaderControl  = 0x01
	miaAudioHeaderStatus   = 0x02
	miaAudioHeaderChannels = 0x03
	miaAudioHeaderRateL    = 0x04
	miaAudioHeaderRateH    = 0x05
	miaAudioHeaderFlags    = 0x06

	miaAudioStatusActive        uint8 = 1 << 0
	miaAudioStatusQueueOverflow uint8 = 1 << 1
	miaAudioFlagStereo          uint8 = 1 << 0

	miaAudioVoiceFreqL          = 0x00
	miaAudioVoiceFreqH          = 0x01
	miaAudioVoicePulseWidth     = 0x02
	miaAudioVoiceAttackDecay    = 0x03
	miaAudioVoiceSustainRelease = 0x04
	miaAudioVoiceWaveform       = 0x05
	miaAudioVoicePan            = 0x06
	miaAudioVoiceControl        = 0x07

	miaAudioWaveSine     uint8 = 0x00
	miaAudioWavePulse    uint8 = 0x01
	miaAudioWaveSaw      uint8 = 0x02
	miaAudioWaveTriangle uint8 = 0x03
	miaAudioWaveNoise    uint8 = 0x04

	miaAudioControlGate       uint8 = 1 << 0
	miaAudioControlResetPhase uint8 = 1 << 1

	miaAudioIndexAll    uint8 = 0xD0
	miaAudioIndexVoice0 uint8 = 0xD1
	miaAudioIndexVoice1 uint8 = 0xD2
	miaAudioIndexVoice2 uint8 = 0xD3
	miaAudioIndexVoice3 uint8 = 0xD4
	miaAudioIndexHeader uint8 = 0xD5
)

// Internal engine constants, mirroring audio.c.
const (
	miaAudioPWMBits   = 10
	miaAudioPWMCenter = 1 << (miaAudioPWMBits - 1)
	miaAudioQueueSize = 64
	miaAudioEnvMax    = 256 << 16

	// Host output is rendered at full 16-bit resolution. The firmware clamps the
	// mix to a signed 10-bit PWM sample; we keep that exact clamp (so clipping
	// distortion matches real hardware) and then scale 10-bit -> 16-bit.
	miaAudioOutputShift = 16 - miaAudioPWMBits
)

// miaAudioVoiceOffset returns the RAM offset of a voice's 8-byte register record.
func miaAudioVoiceOffset(voice int) uint32 {
	return uint32(miaAudioVoicesOffset + voice*miaAudioVoiceSize)
}

// miaAudioRate converts an envelope time in milliseconds into the per-sample
// fixed-point step used by the ADSR engine, mirroring the AUDIO_RATE macro.
func miaAudioRate(ms uint32) uint32 {
	return uint32((uint64(miaAudioEnvMax) * 1000) / (uint64(miaAudioSampleRate) * uint64(ms)))
}

type miaAudioADSR uint8

const (
	miaADSRRelease miaAudioADSR = iota
	miaADSRAttack
	miaADSRDecay
	miaADSRSustain
)

// miaAudioVoice holds the live engine state for one voice. The leading fields
// mirror the cached registers; the trailing fields are the oscillator/envelope
// runtime state.
type miaAudioVoice struct {
	freqQ4     uint16
	phaseInc   uint32
	pulseWidth uint8
	attack     uint8
	decay      uint8
	sustain    uint8
	release    uint8
	waveform   uint8
	pan        int8
	control    uint8
	panL       uint8
	panR       uint8

	sample int8
	adsr   miaAudioADSR
	vol    uint32
	phase  uint32
	noise1 uint32
	noise2 uint32
}

type miaAudioQueueEntry struct {
	loc   uint8
	value uint8
}

// miaAudioState is the emulator's audio subsystem state. It is embedded in
// emulated_mia. Its own mutex guards the engine and queue so the oto render
// goroutine can run without touching the chip mutex.
type miaAudioState struct {
	mu sync.Mutex

	voices [miaAudioVoiceCount]miaAudioVoice

	active        bool
	queueOverflow bool
	queueHead     uint8
	queueTail     uint8
	queue         [miaAudioQueueSize]miaAudioQueueEntry

	// Host output. The oto context is process-wide (oto allows only one), so it
	// is created lazily and shared; each chip owns one player. hostDisabled
	// suppresses host output entirely (used by tests so they never open a real
	// audio device); outputErr records a failed device open for diagnostics.
	player       *oto.Player
	hostDisabled bool
	outputErr    error
}

// The sine table and envelope-rate tables are deterministic and shared by all
// chips, mirroring the firmware's static tables.
var (
	miaAudioSineTable [256]int8

	miaAudioLevelTable = [16]uint32{
		0 << 16, 17 << 16, 34 << 16, 51 << 16,
		68 << 16, 85 << 16, 102 << 16, 119 << 16,
		137 << 16, 154 << 16, 171 << 16, 188 << 16,
		205 << 16, 222 << 16, 239 << 16, 256 << 16,
	}

	miaAudioAttackTable = [16]uint32{
		miaAudioRate(2), miaAudioRate(8), miaAudioRate(16), miaAudioRate(24),
		miaAudioRate(38), miaAudioRate(56), miaAudioRate(68), miaAudioRate(80),
		miaAudioRate(100), miaAudioRate(250), miaAudioRate(500), miaAudioRate(800),
		miaAudioRate(1000), miaAudioRate(3000), miaAudioRate(5000), miaAudioRate(8000),
	}

	miaAudioDecayReleaseTable = [16]uint32{
		miaAudioRate(6), miaAudioRate(24), miaAudioRate(48), miaAudioRate(72),
		miaAudioRate(114), miaAudioRate(168), miaAudioRate(204), miaAudioRate(240),
		miaAudioRate(300), miaAudioRate(750), miaAudioRate(1500), miaAudioRate(2400),
		miaAudioRate(3000), miaAudioRate(9000), miaAudioRate(15000), miaAudioRate(24000),
	}
)

func init() {
	for i := 0; i < 256; i++ {
		miaAudioSineTable[i] = int8(math.Round(math.Sin((math.Pi*2.0/256.0)*float64(i)) * 127.0))
	}
}

// The oto context is process-wide; oto refuses to create more than one.
var (
	miaAudioCtxOnce sync.Once
	miaAudioCtx     *oto.Context
	miaAudioCtxErr  error
)

func miaAudioContext() (*oto.Context, error) {
	miaAudioCtxOnce.Do(func() {
		ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
			SampleRate:   miaAudioSampleRate,
			ChannelCount: 2,
			Format:       oto.FormatSignedInt16LE,
			BufferSize:   50 * time.Millisecond,
		})
		if err != nil {
			miaAudioCtxErr = err
			return
		}
		<-ready
		miaAudioCtx = ctx
	})

	return miaAudioCtx, miaAudioCtxErr
}

/**************************************************************************************************
 * Pure engine helpers
 **************************************************************************************************/

func miaAudioPhaseInc(freqQ4 uint16) uint32 {
	if freqQ4 == 0 {
		return 0
	}

	return uint32((uint64(freqQ4) << 32) / (uint64(miaAudioSampleRate) * 16))
}

func miaAudioUpdatePan(voice *miaAudioVoice, pan int8) {
	if pan < -64 {
		pan = -64
	}
	if pan > 63 {
		pan = 63
	}

	voice.pan = pan
	voice.panL = uint8(64 - int(pan))
	voice.panR = uint8(64 + int(pan))
}

// nextSample advances the oscillator one step and returns its raw -127..127 sample.
func (v *miaAudioVoice) nextSample() int8 {
	oldPhase := v.phase
	v.phase += v.phaseInc
	phase := uint8(v.phase >> 24)

	switch v.waveform {
	case miaAudioWaveSine:
		return miaAudioSineTable[phase]
	case miaAudioWavePulse:
		if phase < v.pulseWidth {
			return 127
		}
		return -127
	case miaAudioWaveSaw:
		return int8(127 - int16(phase))
	case miaAudioWaveTriangle:
		if phase < 128 {
			return int8(int16(phase)*2 - 128)
		}
		return int8(127 - (int16(phase-128) * 2))
	case miaAudioWaveNoise:
		if v.phase < oldPhase {
			v.noise1 ^= v.noise2
			v.noise2 += v.noise1
			v.sample = int8(v.noise2 & 0xFF)
		}
		return v.sample
	default:
		return 0
	}
}

// updateEnvelope advances the ADSR envelope one sample.
func (v *miaAudioVoice) updateEnvelope() {
	sustainTarget := miaAudioLevelTable[v.sustain]

	switch v.adsr {
	case miaADSRAttack:
		v.vol += miaAudioAttackTable[v.attack]
		if v.vol >= miaAudioEnvMax {
			v.vol = miaAudioEnvMax
			v.adsr = miaADSRDecay
		}
	case miaADSRDecay:
		if v.vol <= sustainTarget {
			v.vol = sustainTarget
			v.adsr = miaADSRSustain
		} else {
			rate := miaAudioDecayReleaseTable[v.decay]
			if v.vol <= sustainTarget+rate {
				v.vol = sustainTarget
				v.adsr = miaADSRSustain
			} else {
				v.vol -= rate
			}
		}
	case miaADSRSustain:
		v.vol = sustainTarget
	case miaADSRRelease:
		fallthrough
	default:
		rate := miaAudioDecayReleaseTable[v.release]
		if v.vol <= rate {
			v.vol = 0
		} else {
			v.vol -= rate
		}
	}
}

// resetVoiceState clears a voice's runtime engine state, mirroring
// audio_reset_voice_state.
func (a *miaAudioState) resetVoiceState(voice int) {
	state := &a.voices[voice]
	*state = miaAudioVoice{}
	state.pulseWidth = 128
	state.adsr = miaADSRRelease
	state.noise1 = 0x67452301 + uint32(voice)*0x11111111
	state.noise2 = 0xEFCDAB89 - uint32(voice)*0x01010101
	miaAudioUpdatePan(state, 0)
}

// applyRegister applies one queued register byte to the engine. It mirrors
// audio_apply_register but never reads MIA RAM: the 16-bit frequency is patched
// from the cached value so the render goroutine stays RAM-free.
func (a *miaAudioState) applyRegister(loc, value uint8) {
	if loc < miaAudioHeaderSize || loc >= miaAudioHeaderSize+miaAudioVoiceCount*miaAudioVoiceSize {
		return
	}

	rel := loc - miaAudioHeaderSize
	voice := rel / miaAudioVoiceSize
	field := rel % miaAudioVoiceSize
	if int(voice) >= miaAudioVoiceCount {
		return
	}

	state := &a.voices[voice]

	switch field {
	case miaAudioVoiceFreqL:
		state.freqQ4 = (state.freqQ4 & 0xFF00) | uint16(value)
		state.phaseInc = miaAudioPhaseInc(state.freqQ4)
	case miaAudioVoiceFreqH:
		state.freqQ4 = (state.freqQ4 & 0x00FF) | uint16(value)<<8
		state.phaseInc = miaAudioPhaseInc(state.freqQ4)
	case miaAudioVoicePulseWidth:
		state.pulseWidth = value
	case miaAudioVoiceAttackDecay:
		state.attack = value >> 4
		state.decay = value & 0x0F
	case miaAudioVoiceSustainRelease:
		state.sustain = value >> 4
		state.release = value & 0x0F
	case miaAudioVoiceWaveform:
		state.waveform = value & 0x0F
	case miaAudioVoicePan:
		miaAudioUpdatePan(state, int8(value))
	case miaAudioVoiceControl:
		oldGate := state.control&miaAudioControlGate != 0
		newGate := value&miaAudioControlGate != 0
		if value&miaAudioControlResetPhase != 0 {
			state.phase = 0
		}
		if !oldGate && newGate {
			state.adsr = miaADSRAttack
			state.vol = 0
		} else if oldGate && !newGate {
			state.adsr = miaADSRRelease
		}
		state.control = value
	}
}

// drainQueueLocked applies up to maxWork queued register writes. Caller holds a.mu.
func (a *miaAudioState) drainQueueLocked(maxWork int) {
	for maxWork > 0 && a.queueTail != a.queueHead {
		entry := a.queue[a.queueTail]
		a.queueTail = (a.queueTail + 1) & (miaAudioQueueSize - 1)
		a.applyRegister(entry.loc, entry.value)
		maxWork--
	}
}

// renderSampleLocked mixes one stereo sample. Caller holds a.mu. It mirrors the
// per-sample body of audio_irq_handler, keeping the firmware's 10-bit clamp and
// scaling the result up to the host's 16-bit range.
func (a *miaAudioState) renderSampleLocked() (int16, int16) {
	var sampleL, sampleR int32

	for i := 0; i < miaAudioVoiceCount; i++ {
		voice := &a.voices[i]

		sample := int32(voice.nextSample())
		voice.updateEnvelope()
		sample = (sample * int32(voice.vol>>16)) >> 8

		sampleL += (sample * int32(voice.panL)) >> 7
		sampleR += (sample * int32(voice.panR)) >> 7
	}

	const maxVal = int32(1<<(miaAudioPWMBits-1)) - 1
	const minVal = int32(-(1 << (miaAudioPWMBits - 1)))

	sampleL = miaAudioClamp(sampleL, minVal, maxVal)
	sampleR = miaAudioClamp(sampleR, minVal, maxVal)

	return int16(sampleL << miaAudioOutputShift), int16(sampleR << miaAudioOutputShift)
}

func miaAudioClamp(value, lo, hi int32) int32 {
	if value < lo {
		return lo
	}
	if value > hi {
		return hi
	}
	return value
}

/**************************************************************************************************
 * RAM-backed engine sync (run under c.mu)
 **************************************************************************************************/

func (c *emulated_mia) audioReadU16(offset uint32) uint16 {
	return uint16(c.memory[offset]) | uint16(c.memory[offset+1])<<8
}

func (c *emulated_mia) audioWriteU16(offset uint32, value uint16) {
	c.memory[offset] = uint8(value)
	c.memory[offset+1] = uint8(value >> 8)
}

// audioSyncVoiceRegisters loads one voice's engine state from its RAM record,
// mirroring audio_sync_voice_registers. Caller holds c.mu and c.audio.mu.
func (c *emulated_mia) audioSyncVoiceRegisters(voice int) {
	state := &c.audio.voices[voice]
	base := miaAudioVoiceOffset(voice)

	freqQ4 := c.audioReadU16(base + miaAudioVoiceFreqL)
	attackDecay := c.memory[base+miaAudioVoiceAttackDecay]
	sustainRelease := c.memory[base+miaAudioVoiceSustainRelease]
	control := c.memory[base+miaAudioVoiceControl]
	oldGate := state.control&miaAudioControlGate != 0
	newGate := control&miaAudioControlGate != 0

	state.freqQ4 = freqQ4
	state.phaseInc = miaAudioPhaseInc(freqQ4)
	state.pulseWidth = c.memory[base+miaAudioVoicePulseWidth]
	state.attack = attackDecay >> 4
	state.decay = attackDecay & 0x0F
	state.sustain = sustainRelease >> 4
	state.release = sustainRelease & 0x0F
	state.waveform = c.memory[base+miaAudioVoiceWaveform] & 0x0F
	miaAudioUpdatePan(state, int8(c.memory[base+miaAudioVoicePan]))

	if control&miaAudioControlResetPhase != 0 {
		state.phase = 0
	}
	if !oldGate && newGate {
		state.adsr = miaADSRAttack
		state.vol = 0
	} else if oldGate && !newGate {
		state.adsr = miaADSRRelease
	}
	state.control = control
}

// audioSyncFromMemory reloads every voice from RAM. Caller holds c.mu and c.audio.mu.
func (c *emulated_mia) audioSyncFromMemory() {
	for voice := 0; voice < miaAudioVoiceCount; voice++ {
		c.audioSyncVoiceRegisters(voice)
	}
}

/**************************************************************************************************
 * Lifecycle and command entry points (run under c.mu)
 **************************************************************************************************/

// audioInit builds the audio RAM defaults and indexes. It is part of the chip
// reset path. mia_audio_init's one-time work (sine/rate tables, PWM hardware
// setup) has no per-instance analog here: the tables are package globals and the
// host device is opened lazily on first enable.
func (c *emulated_mia) audioResetRuntimeState() {
	c.audioStop()

	a := &c.audio
	a.mu.Lock()
	for i := uint32(0); i < miaAudioStateSize; i++ {
		c.memory[miaAudioStateOffset+i] = 0
	}
	c.memory[miaAudioHeaderOffset+miaAudioHeaderVersion] = miaAudioVersion
	c.memory[miaAudioHeaderOffset+miaAudioHeaderChannels] = miaAudioVoiceCount
	c.audioWriteU16(miaAudioHeaderOffset+miaAudioHeaderRateL, miaAudioSampleRate)
	c.memory[miaAudioHeaderOffset+miaAudioHeaderFlags] = miaAudioFlagStereo

	for voice := 0; voice < miaAudioVoiceCount; voice++ {
		base := miaAudioVoiceOffset(voice)
		c.memory[base+miaAudioVoicePulseWidth] = 128
		c.memory[base+miaAudioVoiceSustainRelease] = 0xF5
		c.memory[base+miaAudioVoiceWaveform] = miaAudioWavePulse
		c.memory[base+miaAudioVoicePan] = 0
		a.resetVoiceState(voice)
	}

	a.queueHead = 0
	a.queueTail = 0
	a.queueOverflow = false
	a.mu.Unlock()

	c.audioConfigureIndexes()
}

// audioEnable synchronizes the engine from RAM and starts host output, mirroring
// mia_audio_enable. Host output is started asynchronously so a slow device open
// never stalls the 6502 command path.
func (c *emulated_mia) audioEnable() {
	a := &c.audio
	a.mu.Lock()
	a.queueHead = 0
	a.queueTail = 0
	a.queueOverflow = false
	c.audioSyncFromMemory()
	a.active = true
	a.mu.Unlock()

	status := c.memory[miaAudioHeaderOffset+miaAudioHeaderStatus]
	status = (status &^ miaAudioStatusQueueOverflow) | miaAudioStatusActive
	c.memory[miaAudioHeaderOffset+miaAudioHeaderStatus] = status
	c.statusSet(miaStatusAudioActive)

	go c.audioStartOutput()
}

// audioStop halts the engine and pauses host output, mirroring mia_audio_stop.
func (c *emulated_mia) audioStop() {
	a := &c.audio
	a.mu.Lock()
	wasActive := a.active
	a.active = false
	player := a.player
	a.mu.Unlock()

	if player != nil && wasActive {
		player.Pause()
	}

	c.memory[miaAudioHeaderOffset+miaAudioHeaderStatus] &^= miaAudioStatusActive
	c.statusClear(miaStatusAudioActive)
}

// audioReset stops audio and clears the audio block, mirroring mia_audio_reset.
func (c *emulated_mia) audioReset() {
	c.audioStop()
	c.audioResetRuntimeState()
}

// audioReclock mirrors mia_audio_reclock. On real hardware a PHI2 speed change
// alters clk_sys, so the PWM sample-timer wrap is recomputed to hold 24 kHz. The
// emulator renders to the host device at a fixed 24 kHz independent of PHI2, so
// there is nothing to re-derive.
func (c *emulated_mia) audioReclock() {}

// audioIsActive reports whether the audio engine is running.
func (c *emulated_mia) audioIsActive() bool {
	c.audio.mu.Lock()
	defer c.audio.mu.Unlock()

	return c.audio.active
}

// audioCore1OnWrite queues a live register write into the audio block, mirroring
// mia_audio_core1_on_write. Caller holds c.mu. On queue overflow the firmware
// defers a resync to the next audio IRQ; the emulator resyncs here instead,
// because the render goroutine has no access to MIA RAM.
func (c *emulated_mia) audioCore1OnWrite(offset uint32, value uint8) {
	if offset < miaAudioStateOffset || offset >= miaAudioStateOffset+miaAudioStateSize {
		return
	}

	a := &c.audio
	a.mu.Lock()
	if !a.active {
		a.mu.Unlock()
		return
	}

	next := (a.queueHead + 1) & (miaAudioQueueSize - 1)
	if next == a.queueTail {
		risingEdge := !a.queueOverflow
		a.queueOverflow = true
		c.audioSyncFromMemory()
		a.queueHead = 0
		a.queueTail = 0
		a.mu.Unlock()

		c.memory[miaAudioHeaderOffset+miaAudioHeaderStatus] |= miaAudioStatusQueueOverflow
		if risingEdge {
			c.errors.Push(c, miaErrorAudioQueueOverflow)
		}
		return
	}

	head := a.queueHead
	a.queue[head].loc = uint8(offset - miaAudioStateOffset)
	a.queue[head].value = value
	a.queueHead = next
	a.mu.Unlock()
}

// audioConfigureIndexes installs the fixed $D0-$D5 audio indexes, mirroring
// audio_configure_indexes.
func (c *emulated_mia) audioConfigureIndexes() {
	c.audioConfigureIndex(miaAudioIndexAll, miaAudioStateOffset, miaAudioStateSize)
	c.audioConfigureIndex(miaAudioIndexVoice0, miaAudioVoiceOffset(0), miaAudioVoiceSize)
	c.audioConfigureIndex(miaAudioIndexVoice1, miaAudioVoiceOffset(1), miaAudioVoiceSize)
	c.audioConfigureIndex(miaAudioIndexVoice2, miaAudioVoiceOffset(2), miaAudioVoiceSize)
	c.audioConfigureIndex(miaAudioIndexVoice3, miaAudioVoiceOffset(3), miaAudioVoiceSize)
	c.audioConfigureIndex(miaAudioIndexHeader, miaAudioHeaderOffset, miaAudioHeaderSize)
}

func (c *emulated_mia) audioConfigureIndex(indexID uint8, start, length uint32) {
	c.indexes[indexID] = miaIndex{
		currentAddr: start,
		defaultAddr: start,
		limitAddr:   start + length,
		step:        1,
		flags: (1 << miaIndexFlagReadStep) |
			(1 << miaIndexFlagWriteStep) |
			(1 << miaIndexFlagWrap),
	}
}

/**************************************************************************************************
 * Host output (oto)
 **************************************************************************************************/

// miaAudioReader feeds rendered PCM to an oto player. Its Read runs on oto's
// goroutine and only takes c.audio.mu.
type miaAudioReader struct {
	chip *emulated_mia
}

func (r *miaAudioReader) Read(buf []byte) (int, error) {
	a := &r.chip.audio

	frames := len(buf) / 4
	if frames == 0 {
		return 0, nil
	}

	a.mu.Lock()
	active := a.active
	for f := 0; f < frames; f++ {
		var left, right int16
		if active {
			a.drainQueueLocked(16)
			left, right = a.renderSampleLocked()
		}

		off := f * 4
		buf[off+0] = byte(uint16(left))
		buf[off+1] = byte(uint16(left) >> 8)
		buf[off+2] = byte(uint16(right))
		buf[off+3] = byte(uint16(right) >> 8)
	}
	a.mu.Unlock()

	return frames * 4, nil
}

// audioStartOutput lazily opens the shared oto context, creates this chip's
// player, and plays it. It runs off the c.mu path. If no host device is
// available it records the error and degrades silently, leaving the emulated
// engine running so diagnostics still work.
func (c *emulated_mia) audioStartOutput() {
	a := &c.audio

	a.mu.Lock()
	if a.hostDisabled {
		a.mu.Unlock()
		return
	}
	player := a.player
	a.mu.Unlock()

	if player == nil {
		ctx, err := miaAudioContext()
		if err != nil {
			a.mu.Lock()
			a.outputErr = err
			a.mu.Unlock()
			return
		}

		newPlayer := ctx.NewPlayer(&miaAudioReader{chip: c})

		a.mu.Lock()
		if a.player == nil {
			a.player = newPlayer
		}
		player = a.player
		a.mu.Unlock()
	}

	a.mu.Lock()
	active := a.active
	a.mu.Unlock()

	if active {
		player.Play()
	} else {
		player.Pause()
	}
}

// audioClose tears down host output. It is called from the chip Close path.
func (c *emulated_mia) audioClose() {
	a := &c.audio
	a.mu.Lock()
	player := a.player
	a.player = nil
	a.active = false
	a.mu.Unlock()

	if player != nil {
		player.Close()
	}
}

/**************************************************************************************************
 * Console diagnostics
 **************************************************************************************************/

// consoleAudio controls and reports the audio subsystem, mirroring cmd_audio.
func (c *emulated_mia) consoleAudio(args string) string {
	switch strings.TrimSpace(args) {
	case "", "status":
		return c.consoleAudioDetail() + "Usage: audio [status|enable|stop|reset]\n"
	case "enable":
		c.mu.Lock()
		c.audioEnable()
		c.mu.Unlock()
		return "Audio: enabled\n"
	case "stop":
		c.mu.Lock()
		c.audioStop()
		c.mu.Unlock()
		return "Audio: stopped\n"
	case "reset":
		c.mu.Lock()
		c.audioReset()
		c.mu.Unlock()
		return "Audio: reset\n"
	default:
		return "Usage: audio [status|enable|stop|reset]\n"
	}
}

// consoleAudioSummary renders the one-line audio summary for the status dashboard,
// mirroring mia_audio_print_summary (adapted: host output instead of GPIO pins).
func (c *emulated_mia) consoleAudioSummary() string {
	c.audio.mu.Lock()
	active := c.audio.active
	c.audio.mu.Unlock()

	state := "stopped"
	if active {
		state = "active"
	}

	return fmt.Sprintf("Audio: %s  %d Hz  %d voices  stereo (host output)\n",
		state, miaAudioSampleRate, miaAudioVoiceCount)
}

type miaAudioVoiceDump struct {
	freqQ4         uint16
	waveform       uint8
	pulseWidth     uint8
	pan            int8
	attackDecay    uint8
	sustainRelease uint8
	control        uint8
}

// consoleAudioDetail renders the audio subsystem detail, mirroring
// mia_audio_print_status. The firmware pin line is replaced by a host-output line.
func (c *emulated_mia) consoleAudioDetail() string {
	c.mu.Lock()
	c.audio.mu.Lock()
	active := c.audio.active
	queueHead := c.audio.queueHead
	queueTail := c.audio.queueTail
	overflow := c.audio.queueOverflow
	hostDisabled := c.audio.hostDisabled
	hasPlayer := c.audio.player != nil
	outputErr := c.audio.outputErr
	c.audio.mu.Unlock()

	var dump [miaAudioVoiceCount]miaAudioVoiceDump
	for v := 0; v < miaAudioVoiceCount; v++ {
		base := miaAudioVoiceOffset(v)
		dump[v] = miaAudioVoiceDump{
			freqQ4:         c.audioReadU16(base + miaAudioVoiceFreqL),
			waveform:       c.memory[base+miaAudioVoiceWaveform],
			pulseWidth:     c.memory[base+miaAudioVoicePulseWidth],
			pan:            int8(c.memory[base+miaAudioVoicePan]),
			attackDecay:    c.memory[base+miaAudioVoiceAttackDecay],
			sustainRelease: c.memory[base+miaAudioVoiceSustainRelease],
			control:        c.memory[base+miaAudioVoiceControl],
		}
	}
	c.mu.Unlock()

	var out strings.Builder
	out.WriteString("Audio:\n")
	fmt.Fprintf(&out, "  state:     %s\n", activeOrStopped(active))
	fmt.Fprintf(&out, "  rate:      %d Hz\n", miaAudioSampleRate)
	fmt.Fprintf(&out, "  voices:    %d\n", miaAudioVoiceCount)
	fmt.Fprintf(&out, "  output:    %s\n", audioOutputState(hostDisabled, hasPlayer, outputErr))
	fmt.Fprintf(&out, "  block:     $%05X-$%05X\n",
		miaAudioStateOffset, miaAudioStateOffset+miaAudioStateSize-1)
	fmt.Fprintf(&out, "  indexes:   all:$%02X  ch0:$%02X ch1:$%02X ch2:$%02X ch3:$%02X  header:$%02X\n",
		miaAudioIndexAll, miaAudioIndexVoice0, miaAudioIndexVoice1,
		miaAudioIndexVoice2, miaAudioIndexVoice3, miaAudioIndexHeader)
	fmt.Fprintf(&out, "  queue:     head:%d tail:%d overflow:%s\n",
		queueHead, queueTail, yesNo(overflow))

	for v := 0; v < miaAudioVoiceCount; v++ {
		d := dump[v]
		gate := "off"
		if d.control&miaAudioControlGate != 0 {
			gate = "on"
		}
		fmt.Fprintf(&out,
			"  ch%d:      freq:%d.%d Hz  wave:%d  pulse:%d  pan:%d  AD:%X/%X  SR:%X/%X  gate:%s\n",
			v,
			d.freqQ4>>4,
			(d.freqQ4&0x0F)*10/16,
			d.waveform,
			d.pulseWidth,
			d.pan,
			d.attackDecay>>4, d.attackDecay&0x0F,
			d.sustainRelease>>4, d.sustainRelease&0x0F,
			gate)
	}

	return out.String()
}

func activeOrStopped(active bool) string {
	if active {
		return "active"
	}

	return "stopped"
}

func audioOutputState(hostDisabled, hasPlayer bool, outputErr error) string {
	switch {
	case hostDisabled:
		return "disabled"
	case outputErr != nil:
		return fmt.Sprintf("unavailable (%v)", outputErr)
	case hasPlayer:
		return "host device ready"
	default:
		return "host device idle"
	}
}
