package mia

import (
	"fmt"
	"strings"
)

// This file ports MIA's background note sequencer (firmware
// src/mia/audio/audio.{c,h}, the TRACK/SEQ_LOAD/SEQ_START/SEQ_STOP/
// VOICE_TAKE/VOICE_RELEASE additions) to the emulator. See
// clementina-mia/docs/audio-sequencer.md for the full contract; this comment
// only covers where the Go port has to diverge from it.
//
// On real hardware the sequencer decodes straight out of MIA RAM inside the
// same Core 0 audio ISR that steps oscillators/envelopes. The emulator's
// audio engine runs on the oto callback goroutine and, per the design at the
// top of audio.go, must stay RAM-free - c.memory belongs to the CPU-Tick
// goroutine. So the sequencer keeps its own copy of each voice's track bytes
// (seqRAM, populated by audioSeqLoadTrack when TRACK/SEQ_LOAD run) and the
// render goroutine never touches c.memory:
//
//   - "Track finished" (no loop) can't call irqSetFlag directly - that
//     mutates shared registers unsynchronized with the CPU-Tick goroutine.
//     It sets seqDonePending instead; audioSeqService (called every Tick,
//     mirroring core 1's act_loop draining mia_irq_set_requests) drains it.
//   - A voice's register bytes in MIA RAM (FREQ_L, WAVEFORM, ...) would go
//     stale while the sequencer alone is driving them, since nothing else
//     writes c.memory for a sequencer-originated change. audioSeqService
//     also mirrors the live engine state for every running, non-taken voice
//     back into c.memory once per Tick - safe, since Tick already owns
//     c.memory - so a plain 6502 peek still reads what the sequencer is
//     actually doing, just refreshed once per bus cycle rather than
//     immediately.
//   - PLAYING(v)/CUE(v) (SEQ_NOTE_INDEX/SEQ_STATUS, voice offsets $09-$0B)
//     are the one place read fidelity is intentionally scoped down: they are
//     computed on demand from seq state only when read through the dedicated
//     $EC-$EF indexes (see indexes.go), not mirrored into c.memory. A generic
//     peek at those offsets through a different index will not reflect them
//     in the emulator, unlike real hardware.

const (
	miaSeqTrackSize         = 0x400 // 1024 bytes per voice
	miaSeqStateOffset       = 0x13000
	miaSeqTrackHeaderSize   = 4
	miaSeqTrackLoopL        = 0x00
	miaSeqTrackLoopH        = 0x01
	miaSeqTrackEventsOffset = miaSeqTrackHeaderSize
	miaSeqNoLoop     uint16 = 0xFFFF

	miaSeqOpEnd      uint8 = 0x00
	miaSeqOpNote     uint8 = 0x01
	miaSeqOpRest     uint8 = 0x02
	miaSeqOpSetWave  uint8 = 0x03
	miaSeqOpSetADSR  uint8 = 0x04
	miaSeqOpSetPan   uint8 = 0x05
	miaSeqOpSetVol   uint8 = 0x06
	miaSeqOpSetPulse uint8 = 0x07

	// Live sequencer status, read through the dedicated $EC-$EF indexes only
	// (see the file comment above and indexes.go).
	miaAudioVoiceSeqNoteIndexL          = 0x09
	miaAudioVoiceSeqNoteIndexH          = 0x0A
	miaAudioVoiceSeqStatus              = 0x0B
	miaAudioSeqStatusRunning      uint8 = 1 << 0
	miaAudioSeqStatusTaken        uint8 = 1 << 1

	miaAudioIndexSeqVoice0 uint8 = 0xEC
	miaAudioIndexSeqVoice1 uint8 = 0xED
	miaAudioIndexSeqVoice2 uint8 = 0xEE
	miaAudioIndexSeqVoice3 uint8 = 0xEF

	// Commands. Each takes one parameter byte: a voice bitmask (bit v =
	// voice v). There is no "0 means all" case - callers spell out the mask.
	miaCmdAudioSeqLoad      uint8 = 0x63
	miaCmdAudioSeqStart     uint8 = 0x64
	miaCmdAudioSeqStop      uint8 = 0x65
	miaCmdAudioVoiceTake    uint8 = 0x66
	miaCmdAudioVoiceRelease uint8 = 0x67

	// Max events silently skipped per rendered sample while reconciling a
	// VOICE_RELEASE, mirroring MIA_SEQ_CATCHUP_BUDGET.
	miaSeqCatchupBudget = 8
)

// miaSeqTrackOffset returns the MIA RAM offset of a voice's track buffer.
func miaSeqTrackOffset(voice int) uint32 {
	return uint32(miaSeqStateOffset + voice*miaSeqTrackSize)
}

// miaSeqRecordSize returns the number of bytes the opcode occupies, opcode
// byte included, or 0 for END/unrecognized - mirroring audio_seq_record_size.
func miaSeqRecordSize(op uint8) uint8 {
	switch op {
	case miaSeqOpNote:
		return 6 // op + freq_l + freq_h + dur(3)
	case miaSeqOpRest:
		return 4 // op + dur(3)
	case miaSeqOpSetADSR:
		return 3
	case miaSeqOpSetWave, miaSeqOpSetPan, miaSeqOpSetVol, miaSeqOpSetPulse:
		return 2
	default:
		return 0
	}
}

func miaSeqReadDuration24(buf []byte, offset uint32) uint32 {
	return uint32(buf[offset]) | uint32(buf[offset+1])<<8 | uint32(buf[offset+2])<<16
}

// miaAudioSeqVoice holds one voice's sequencer runtime state. Guarded by
// miaAudioState.mu, same as the engine state it drives.
type miaAudioSeqVoice struct {
	running    bool
	taken      bool
	catchingUp bool
	hasLoop    bool

	cursor        uint32 // offset within this voice's seqRAM buffer
	countdown     uint32 // samples remaining until the current event ends
	eventDuration uint32 // full duration of the current event
	catchupRemain uint32 // samples still to silently skip during catch-up
	takenAt       uint32 // seqClock snapshot when VOICE_TAKE was issued

	loopCursor    uint32
	loopNoteIndex uint16
	noteIndex     uint16 // 1-based index of the current note/rest; 0 = not started
}

/**************************************************************************************************
 * Pure engine helpers (run under a.mu, alongside the rest of the audio engine)
 **************************************************************************************************/

func (a *miaAudioState) seqWriteReg(voice int, field uint8, value uint8) {
	a.applyRegister(uint8(miaAudioHeaderSize+voice*miaAudioVoiceSize)+field, value)
}

func (a *miaAudioState) seqGate(voice int, on bool) {
	var control uint8
	if on {
		control = miaAudioControlGate | miaAudioControlResetPhase
	}
	a.seqWriteReg(voice, miaAudioVoiceControl, control)
}

func (a *miaAudioState) seqApplyNote(voice int, cursor uint32) {
	buf := a.seqRAM[voice][:]
	a.seqWriteReg(voice, miaAudioVoiceFreqL, buf[cursor+1])
	a.seqWriteReg(voice, miaAudioVoiceFreqH, buf[cursor+2])
	a.seqGate(voice, true)
}

func (a *miaAudioState) seqApplyConfigOp(voice int, cursor uint32, op uint8) {
	buf := a.seqRAM[voice][:]
	switch op {
	case miaSeqOpSetWave:
		a.seqWriteReg(voice, miaAudioVoiceWaveform, buf[cursor+1])
	case miaSeqOpSetADSR:
		a.seqWriteReg(voice, miaAudioVoiceAttackDecay, buf[cursor+1])
		a.seqWriteReg(voice, miaAudioVoiceSustainRelease, buf[cursor+2])
	case miaSeqOpSetPan:
		a.seqWriteReg(voice, miaAudioVoicePan, buf[cursor+1])
	case miaSeqOpSetVol:
		a.seqWriteReg(voice, miaAudioVoiceVolume, buf[cursor+1])
	case miaSeqOpSetPulse:
		a.seqWriteReg(voice, miaAudioVoicePulseWidth, buf[cursor+1])
	}
}

// seqFinishTrack mirrors audio_seq_finish_track. It cannot call irqSetFlag
// directly (see the file comment); it defers to seqDonePending instead.
func (a *miaAudioState) seqFinishTrack(voice int) {
	seq := &a.seq[voice]
	seq.running = false
	seq.catchingUp = false
	a.seqGate(voice, false)
	a.seqDonePending = true
}

// seqAdvance decodes and applies config opcodes at the current cursor, live,
// until a NOTE/REST becomes the new current (audible) event, or END
// stops/loops the track. Mirrors audio_seq_advance.
func (a *miaAudioState) seqAdvance(voice int) {
	seq := &a.seq[voice]
	buf := a.seqRAM[voice][:]

	for {
		if seq.cursor >= miaSeqTrackSize {
			a.seqFinishTrack(voice)
			return
		}

		op := buf[seq.cursor]
		size := miaSeqRecordSize(op)

		if size == 0 {
			if seq.hasLoop {
				seq.cursor = seq.loopCursor
				seq.noteIndex = seq.loopNoteIndex
				continue
			}
			a.seqFinishTrack(voice)
			return
		}

		if op == miaSeqOpNote || op == miaSeqOpRest {
			var dur uint32
			if op == miaSeqOpNote {
				dur = miaSeqReadDuration24(buf, seq.cursor+3)
				a.seqApplyNote(voice, seq.cursor)
			} else {
				dur = miaSeqReadDuration24(buf, seq.cursor+1)
				a.seqGate(voice, false)
			}
			seq.noteIndex++
			seq.eventDuration = dur
			seq.countdown = dur
			seq.cursor += uint32(size)
			return
		}

		a.seqApplyConfigOp(voice, seq.cursor, op)
		seq.cursor += uint32(size)
	}
}

// seqCatchupStep silently walks forward through events entirely in the past,
// bounded to miaSeqCatchupBudget per sample, then resumes live on the first
// event that isn't fully covered. Mirrors audio_seq_catchup_step; see
// docs/audio-sequencer.md, "Catch-up (VOICE_RELEASE reconciliation)".
func (a *miaAudioState) seqCatchupStep(voice int) {
	seq := &a.seq[voice]
	buf := a.seqRAM[voice][:]
	budget := miaSeqCatchupBudget

	for budget > 0 && seq.catchingUp {
		budget--

		if seq.cursor >= miaSeqTrackSize {
			a.seqFinishTrack(voice)
			return
		}

		op := buf[seq.cursor]
		size := miaSeqRecordSize(op)

		if size == 0 {
			if seq.hasLoop {
				seq.cursor = seq.loopCursor
				seq.noteIndex = seq.loopNoteIndex
				continue
			}
			a.seqFinishTrack(voice)
			return
		}

		if op == miaSeqOpNote || op == miaSeqOpRest {
			var dur uint32
			if op == miaSeqOpNote {
				dur = miaSeqReadDuration24(buf, seq.cursor+3)
			} else {
				dur = miaSeqReadDuration24(buf, seq.cursor+1)
			}

			if dur <= seq.catchupRemain {
				// Fully in the past: skip it silently, no register writes.
				seq.catchupRemain -= dur
				seq.noteIndex++
				seq.cursor += uint32(size)
				continue
			}

			// Not fully elapsed: this is the event to resume on, live, in full.
			seq.catchingUp = false
			if op == miaSeqOpNote {
				a.seqApplyNote(voice, seq.cursor)
			} else {
				a.seqGate(voice, false)
			}
			seq.noteIndex++
			seq.eventDuration = dur
			seq.countdown = dur
			seq.cursor += uint32(size)
			return
		}

		a.seqApplyConfigOp(voice, seq.cursor, op)
		seq.cursor += uint32(size)
	}
}

// seqStep advances one voice's sequencer by one sample. Mirrors
// audio_seq_step; called from renderSampleLocked, once per voice per sample,
// before the oscillator/envelope pass - same placement as the firmware ISR.
func (a *miaAudioState) seqStep(voice int) {
	seq := &a.seq[voice]

	if !seq.running || seq.taken {
		return
	}

	switch {
	case seq.catchingUp:
		a.seqCatchupStep(voice)
	case seq.countdown > 0:
		seq.countdown--
	default:
		a.seqAdvance(voice)
	}
}

// seqReleaseOne reconciles a VOICE_RELEASE, mirroring audio_seq_release_one:
// already_spent credits however far into the current event this voice was
// when taken, so a short TAKE resumes the same event without retriggering
// it, and a long one skips forward by exactly the real elapsed time - not
// just the time since VOICE_TAKE.
func (a *miaAudioState) seqReleaseOne(voice int) {
	seq := &a.seq[voice]

	if !seq.taken {
		return
	}
	seq.taken = false

	if !seq.running {
		return
	}

	elapsedSinceTake := a.seqClock - seq.takenAt
	alreadySpent := seq.eventDuration - seq.countdown
	totalElapsed := alreadySpent + elapsedSinceTake

	if totalElapsed < seq.eventDuration {
		// Real time hasn't reached this event's natural end yet: resume the
		// same event mid-flight, no retrigger.
		seq.countdown = seq.eventDuration - totalElapsed
		return
	}

	// This event is over: the next sample's catch-up step walks forward from
	// here using the leftover budget.
	seq.catchingUp = true
	seq.catchupRemain = totalElapsed - seq.eventDuration
}

/**************************************************************************************************
 * RAM-backed command entry points (run under c.mu)
 **************************************************************************************************/

// audioSeqLoadTrack mirrors mia_audio_seq_load_track. It copies the voice's
// track buffer out of MIA RAM into the audio engine's own copy - the render
// goroutine never touches c.memory, see the file comment - and resets the
// voice's runtime position to the top of its event stream.
func (c *emulated_mia) audioSeqLoadTrack(voice int) {
	if voice < 0 || voice >= miaAudioVoiceCount {
		return
	}

	trackOffset := miaSeqTrackOffset(voice)
	loopRel := uint16(c.memory[trackOffset+miaSeqTrackLoopL]) |
		uint16(c.memory[trackOffset+miaSeqTrackLoopH])<<8

	a := &c.audio
	a.mu.Lock()
	defer a.mu.Unlock()

	copy(a.seqRAM[voice][:], c.memory[trackOffset:trackOffset+miaSeqTrackSize])

	seq := &a.seq[voice]
	*seq = miaAudioSeqVoice{}
	seq.cursor = miaSeqTrackEventsOffset

	loopCursor := uint32(miaSeqTrackEventsOffset) + uint32(loopRel)
	if loopRel == miaSeqNoLoop || loopCursor >= miaSeqTrackSize {
		return
	}

	seq.hasLoop = true
	seq.loopCursor = loopCursor

	// Count NOTE/REST events from the top of the stream up to the loop
	// point, once, so looping restores the right note index. Bounded by the
	// track buffer size; a malformed stream just stops the count at whatever
	// opcode it can't recognize rather than running away.
	buf := a.seqRAM[voice][:]
	var index uint16
	for scan := uint32(miaSeqTrackEventsOffset); scan < loopCursor; {
		op := buf[scan]
		size := miaSeqRecordSize(op)
		if size == 0 {
			break
		}
		if op == miaSeqOpNote || op == miaSeqOpRest {
			index++
		}
		scan += uint32(size)
	}
	seq.loopNoteIndex = index
}

func (c *emulated_mia) audioSeqStart(mask uint8) {
	a := &c.audio
	a.mu.Lock()
	defer a.mu.Unlock()

	for v := 0; v < miaAudioVoiceCount; v++ {
		if mask&(1<<uint(v)) == 0 {
			continue
		}
		a.seq[v].running = true
		a.seq[v].taken = false
		a.seq[v].catchingUp = false
	}
}

func (c *emulated_mia) audioSeqStop(mask uint8) {
	a := &c.audio
	a.mu.Lock()
	defer a.mu.Unlock()

	for v := 0; v < miaAudioVoiceCount; v++ {
		if mask&(1<<uint(v)) == 0 {
			continue
		}
		a.seq[v].running = false
		a.seq[v].taken = false
		a.seq[v].catchingUp = false
		a.seqGate(v, false)
	}
}

func (c *emulated_mia) audioVoiceTake(mask uint8) {
	a := &c.audio
	a.mu.Lock()
	defer a.mu.Unlock()

	for v := 0; v < miaAudioVoiceCount; v++ {
		if mask&(1<<uint(v)) == 0 {
			continue
		}
		a.seq[v].taken = true
		a.seq[v].takenAt = a.seqClock
	}
}

func (c *emulated_mia) audioVoiceRelease(mask uint8) {
	a := &c.audio
	a.mu.Lock()
	defer a.mu.Unlock()

	for v := 0; v < miaAudioVoiceCount; v++ {
		if mask&(1<<uint(v)) != 0 {
			a.seqReleaseOne(v)
		}
	}
}

// audioSeqConfigureIndexes installs the $EC-$EF sequencer status indexes.
// Their addresses match real hardware for diagnostic fidelity, but reads
// through them are special-cased in indexes.go rather than backed by
// c.memory - see the file comment.
func (c *emulated_mia) audioSeqConfigureIndexes() {
	c.audioConfigureIndex(miaAudioIndexSeqVoice0, miaAudioVoiceOffset(0)+miaAudioVoiceSeqNoteIndexL, 3)
	c.audioConfigureIndex(miaAudioIndexSeqVoice1, miaAudioVoiceOffset(1)+miaAudioVoiceSeqNoteIndexL, 3)
	c.audioConfigureIndex(miaAudioIndexSeqVoice2, miaAudioVoiceOffset(2)+miaAudioVoiceSeqNoteIndexL, 3)
	c.audioConfigureIndex(miaAudioIndexSeqVoice3, miaAudioVoiceOffset(3)+miaAudioVoiceSeqNoteIndexL, 3)
}

// audioSeqResetRuntimeState clears sequencer runtime state and track buffers,
// part of audioResetRuntimeState.
func (c *emulated_mia) audioSeqResetRuntimeState() {
	for v := 0; v < miaAudioVoiceCount; v++ {
		trackOffset := miaSeqTrackOffset(v)
		for i := uint32(0); i < miaSeqTrackSize; i++ {
			c.memory[trackOffset+i] = 0
		}
		c.memory[trackOffset+miaSeqTrackLoopL] = uint8(miaSeqNoLoop & 0xFF)
		c.memory[trackOffset+miaSeqTrackLoopH] = uint8(miaSeqNoLoop >> 8)
	}

	a := &c.audio
	a.mu.Lock()
	a.seq = [miaAudioVoiceCount]miaAudioSeqVoice{}
	a.seqRAM = [miaAudioVoiceCount][miaSeqTrackSize]byte{}
	a.seqClock = 0
	a.seqDonePending = false
	a.mu.Unlock()

	c.audioSeqConfigureIndexes()
}

// audioSeqService drains "track finished" signals raised by the audio render
// goroutine and refreshes MIA RAM for every running, non-taken voice from the
// live engine state. Called every Tick (mirroring core 1's act_loop draining
// mia_irq_set_requests every pass) - see the file comment for why this can't
// happen on the render goroutine itself.
func (c *emulated_mia) audioSeqService() {
	a := &c.audio
	a.mu.Lock()
	pending := a.seqDonePending
	a.seqDonePending = false

	for v := 0; v < miaAudioVoiceCount; v++ {
		if a.seq[v].running && !a.seq[v].taken {
			c.audioSeqMirrorVoiceLocked(v)
		}
	}
	a.mu.Unlock()

	if pending {
		c.irqSetFlag(miaIRQAudioSeqDone)
	}
}

// audioSeqMirrorVoiceLocked copies one voice's live engine register state
// back into MIA RAM. Caller holds c.mu and c.audio.mu.
func (c *emulated_mia) audioSeqMirrorVoiceLocked(voice int) {
	v := &c.audio.voices[voice]
	base := miaAudioVoiceOffset(voice)

	c.memory[base+miaAudioVoiceFreqL] = uint8(v.freqQ4)
	c.memory[base+miaAudioVoiceFreqH] = uint8(v.freqQ4 >> 8)
	c.memory[base+miaAudioVoicePulseWidth] = v.pulseWidth
	c.memory[base+miaAudioVoiceAttackDecay] = v.attack<<4 | v.decay
	c.memory[base+miaAudioVoiceSustainRelease] = v.sustain<<4 | v.release
	c.memory[base+miaAudioVoiceWaveform] = v.waveform
	c.memory[base+miaAudioVoicePan] = uint8(v.pan)
	c.memory[base+miaAudioVoiceControl] = v.control
	c.memory[base+miaAudioVoiceVolume] = v.volume
}

// audioSeqStatusByte serves a read through one of the $EC-$EF indexes,
// computed live from sequencer state rather than c.memory. Caller holds c.mu.
func (c *emulated_mia) audioSeqStatusByte(voice int, addr uint32) uint8 {
	field := addr - miaAudioVoiceOffset(voice)

	c.audio.mu.Lock()
	seq := &c.audio.seq[voice]
	noteIndex, running, taken := seq.noteIndex, seq.running, seq.taken
	c.audio.mu.Unlock()

	switch field {
	case miaAudioVoiceSeqNoteIndexL:
		return uint8(noteIndex)
	case miaAudioVoiceSeqNoteIndexH:
		return uint8(noteIndex >> 8)
	case miaAudioVoiceSeqStatus:
		var status uint8
		if running {
			status |= miaAudioSeqStatusRunning
		}
		if taken {
			status |= miaAudioSeqStatusTaken
		}
		return status
	default:
		return 0
	}
}

// miaAudioSeqIndexVoice reports whether indexID is one of the dedicated
// sequencer-status indexes, and if so, which voice it serves.
func miaAudioSeqIndexVoice(indexID uint8) (int, bool) {
	switch indexID {
	case miaAudioIndexSeqVoice0:
		return 0, true
	case miaAudioIndexSeqVoice1:
		return 1, true
	case miaAudioIndexSeqVoice2:
		return 2, true
	case miaAudioIndexSeqVoice3:
		return 3, true
	default:
		return 0, false
	}
}

/**************************************************************************************************
 * Terminal-only helper
 **************************************************************************************************/

// audioSeqWriteTestTrack writes a small built-in looping arpeggio into a
// voice's track buffer (does not load or start it), mirroring
// mia_audio_seq_write_test_track. For standalone bring-up before TRACK/BAND
// exist on the BASIC side.
func (c *emulated_mia) audioSeqWriteTestTrack(voice int) {
	if voice < 0 || voice >= miaAudioVoiceCount {
		return
	}

	// LOOP=0 (loop the whole body), then attack=0/decay=3/sustain=F/release=5,
	// then C4/E4/G4, 6000 samples (~0.25s) each.
	track := []byte{
		0x00, 0x00, // LOOP
		0x00, 0x00, // reserved
		miaSeqOpSetADSR, 0x03, 0xF5,
		miaSeqOpNote, 0x5A, 0x10, 0x70, 0x17, 0x00, // C4
		miaSeqOpNote, 0x9A, 0x14, 0x70, 0x17, 0x00, // E4
		miaSeqOpNote, 0x80, 0x18, 0x70, 0x17, 0x00, // G4
		miaSeqOpEnd,
	}

	base := miaSeqTrackOffset(voice)
	copy(c.memory[base:base+uint32(len(track))], track)
}

/**************************************************************************************************
 * Console diagnostics, mirroring cmd_audio_seq
 **************************************************************************************************/

func (c *emulated_mia) consoleAudioSeq(args string) string {
	args = strings.TrimSpace(args)
	if args == "" || args == "status" {
		return c.consoleAudioSeqDetail()
	}

	verb, rest := splitConsoleCommand(args)
	voice, ok := parseU8(rest)
	if !ok || voice >= miaAudioVoiceCount {
		return "Usage: audio seq [status|test|stop|take|give] <voice 0-3>\n"
	}
	mask := uint8(1) << voice

	switch verb {
	case "test":
		c.mu.Lock()
		c.audioSeqWriteTestTrack(int(voice))
		c.audioSeqLoadTrack(int(voice))
		c.audioSeqStart(mask)
		active := c.audioIsActive()
		if !active {
			c.audioEnable()
		}
		c.mu.Unlock()
		return fmt.Sprintf("Audio: voice %d playing built-in test arpeggio\n", voice)
	case "stop":
		c.mu.Lock()
		c.audioSeqStop(mask)
		c.mu.Unlock()
		return fmt.Sprintf("Audio: voice %d sequencer stopped\n", voice)
	case "take":
		c.mu.Lock()
		c.audioVoiceTake(mask)
		c.mu.Unlock()
		return fmt.Sprintf("Audio: voice %d taken from the sequencer\n", voice)
	case "give":
		c.mu.Lock()
		c.audioVoiceRelease(mask)
		c.mu.Unlock()
		return fmt.Sprintf("Audio: voice %d released back to the sequencer\n", voice)
	default:
		return "Usage: audio seq [status|test|stop|take|give] <voice 0-3>\n"
	}
}

func (c *emulated_mia) consoleAudioSeqDetail() string {
	c.audio.mu.Lock()
	defer c.audio.mu.Unlock()

	var out strings.Builder
	out.WriteString("Sequencer:\n")
	for v := 0; v < miaAudioVoiceCount; v++ {
		seq := &c.audio.seq[v]
		state := "stopped"
		if seq.running {
			state = "running"
		}
		var extra string
		if seq.taken {
			extra += " taken"
		}
		if seq.catchingUp {
			extra += " catching-up"
		}
		loop := "no"
		if seq.hasLoop {
			loop = "yes"
		}
		fmt.Fprintf(&out, "  ch%d: %s%s  note:%d  cursor:$%03X  countdown:%d  loop:%s\n",
			v, state, extra, seq.noteIndex, seq.cursor, seq.countdown, loop)
	}

	return out.String()
}
