package mia

import (
	"fmt"
	"strings"
)

// This file ports MIA's background note sequencer (firmware
// src/mia/audio/audio.{c,h}, the TRACK/SEQ_LOAD/SEQ_START/SEQ_STOP/
// VOICE_TAKE/VOICE_RELEASE/SEQ_SET_BASE additions) to the emulator. See
// clementina-mia/docs/audio-sequencer.md for the full contract.
//
// Firmware decodes straight out of MIA RAM inside the Core 0 audio ISR. The
// emulator's audio engine runs on the oto callback goroutine, while CPU-Tick
// writes MIA RAM. SEQ_LOAD installs a view from trackBase to the end of
// c.memory, so later CPU writes remain visible. The audio reader takes c.mu
// before a.mu while rendering; CPU writes take c.mu, avoiding a data race.
// A track has no declared length (just a base and an END/JUMP), so the view
// ends at miaRAMSize. Other necessary Go-port differences remain:
//
//   - "Track finished" (no loop) can't call irqSetFlag directly - that
//     mutates shared registers while the audio render loop runs.
//     It sets seqDonePending instead; audioSeqService (called every Tick,
//     mirroring core 1's act_loop draining mia_irq_set_requests) drains it.
//   - A voice's register bytes in MIA RAM (FREQ_L, WAVEFORM, ...) would go
//     stale while the sequencer alone is driving them, since nothing else
//     writes c.memory for a sequencer-originated change. audioSeqService
//     mirrors the live engine state for every running, non-taken voice
//     back into c.memory once per Tick, so a plain 6502 peek reads it after
//     the next bus cycle rather than immediately.
//   - PLAYING(v)/CUE(v) (SEQ_NOTE_INDEX/SEQ_STATUS, voice offsets $09-$0B)
//     are computed on demand from seq state for every index pointing at those
//     addresses (see indexes.go), rather than mirrored into c.memory.

const (
	// miaSeqNoBase marks a voice no program has given a track, mirroring
	// MIA_SEQ_NO_BASE: boot and AUDIO_RESET store it, the 24-bit SET_BASE
	// parameter can never produce it, and SEQ_START leaves such a voice
	// stopped. Nothing in MIA RAM is reserved for tracks.
	miaSeqNoBase uint32 = 0xFFFFFFFF

	miaSeqOpEnd      uint8 = 0x00
	miaSeqOpNote     uint8 = 0x01
	miaSeqOpRest     uint8 = 0x02
	miaSeqOpSetWave  uint8 = 0x03
	miaSeqOpSetADSR  uint8 = 0x04
	miaSeqOpSetPan   uint8 = 0x05
	miaSeqOpSetVol   uint8 = 0x06
	miaSeqOpSetPulse uint8 = 0x07
	// miaSeqOpJump: offset_l, offset_m, offset_h - a signed 24-bit
	// little-endian offset, relative to the address right after this 4-byte
	// record (self-relative, so a track keeps working if its base moves).
	// Zero-duration, does not advance noteIndex. "Loop the whole track" is
	// just a JUMP back to the top - there is no separate loop opcode.
	miaSeqOpJump uint8 = 0x08

	// Live sequencer status, read through any index pointing at these bytes.
	miaAudioVoiceSeqNoteIndexL       = 0x09
	miaAudioVoiceSeqNoteIndexH       = 0x0A
	miaAudioVoiceSeqStatus           = 0x0B
	miaAudioSeqStatusRunning   uint8 = 1 << 0
	miaAudioSeqStatusTaken     uint8 = 1 << 1

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
	// AUDIO_SEQ_SET_BASE<voice>: one command id per voice (a base address is
	// necessarily distinct per voice, unlike the other commands' shared
	// bitmask); params[0..2] = 24-bit little-endian MIA RAM address. Sets
	// trackBase only - follow with SEQ_LOAD to (re)start decoding from it.
	miaCmdAudioSeqSetBase0 uint8 = 0x68
	miaCmdAudioSeqSetBase1 uint8 = 0x69
	miaCmdAudioSeqSetBase2 uint8 = 0x6A
	miaCmdAudioSeqSetBase3 uint8 = 0x6B

	// Max events silently skipped per tick while reconciling a
	// VOICE_RELEASE, mirroring MIA_SEQ_CATCHUP_BUDGET.
	miaSeqCatchupBudget = 8

	// Max zero-duration opcodes (SET_*/JUMP) decoded in a row within a
	// single seqAdvance call before it gives up and stops the voice,
	// mirroring MIA_SEQ_DECODE_BUDGET - without this, a track whose JUMPs
	// cycle with no NOTE/REST/END in between would spin forever.
	miaSeqDecodeBudget = 32
)

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
	case miaSeqOpJump:
		return 4 // op + offset(3)
	default:
		return 0
	}
}

func miaSeqReadDuration24(buf []byte, offset uint32) uint32 {
	return uint32(buf[offset]) | uint32(buf[offset+1])<<8 | uint32(buf[offset+2])<<16
}

// miaSeqEventTicks mirrors audio_seq_event_ticks: a NOTE/REST lasts exactly
// dur ticks, counting the tick that decodes it, so the next event decodes dur
// ticks later. dur = 0 still takes that one tick.
func miaSeqEventTicks(dur uint32) uint32 {
	if dur == 0 {
		return 1
	}
	return dur
}

// miaSeqJumpTarget resolves a JUMP's 3-byte signed offset (stored at
// cursor+1) into a new cursor, relative to the position right after this
// 4-byte record. cursor here is the snapshot-relative offset seqAdvance
// already uses, not an absolute MIA address, but the arithmetic is
// translation-invariant - the same offset lands correctly either way, since
// it was computed as a difference of two absolute addresses that both
// carried the same trackBase. A malformed offset landing outside the
// snapshot is caught by the ordinary cursor >= len(buf) check on the next
// iteration, same as any other corrupt cursor.
func miaSeqJumpTarget(buf []byte, cursor uint32) uint32 {
	raw := miaSeqReadDuration24(buf, cursor+1)
	offset := int32(raw<<8) >> 8 // sign-extend bit 23 into 32 bits
	return uint32(int32(cursor+4) + offset)
}

// miaAudioSeqVoice holds one voice's sequencer runtime state. Guarded by
// miaAudioState.mu, same as the engine state it drives.
type miaAudioSeqVoice struct {
	running    bool
	taken      bool
	catchingUp bool

	trackBase     uint32 // this voice's track base, set by audioSeqSetBase
	cursor        uint32 // offset within this voice's seqRAM buffer (snapshot-relative)
	countdown     uint32 // ticks the current event still holds after the one being played
	eventDuration uint32 // full length of the current event in ticks (0 for none yet)
	catchupRemain uint32 // ticks still to silently skip during catch-up
	takenAt       uint32 // seqClock snapshot when VOICE_TAKE was issued
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

// seqAdvance decodes and applies config/JUMP opcodes at the current cursor,
// live, until a NOTE/REST becomes the new current (audible) event, or
// END/an unrecognized byte/running off the snapshot stops the track. Mirrors
// audio_seq_advance.
//
// Bounded by miaSeqDecodeBudget: JUMP and the SET_* opcodes are all
// zero-duration and loop straight back to the top without returning, so a
// track whose jumps cycle with no NOTE/REST/END in between would otherwise
// spin this call forever - fail-safe, silent, exactly like an unrecognized
// opcode byte.
func (a *miaAudioState) seqAdvance(voice int) {
	seq := &a.seq[voice]
	buf := a.seqRAM[voice]
	budget := miaSeqDecodeBudget

	for {
		if seq.cursor >= uint32(len(buf)) {
			a.seqFinishTrack(voice)
			return
		}

		op := buf[seq.cursor]
		size := miaSeqRecordSize(op)

		if size == 0 {
			a.seqFinishTrack(voice)
			return
		}

		if op == miaSeqOpNote || op == miaSeqOpRest {
			var ticks uint32
			if op == miaSeqOpNote {
				ticks = miaSeqEventTicks(miaSeqReadDuration24(buf, seq.cursor+3))
				a.seqApplyNote(voice, seq.cursor)
			} else {
				ticks = miaSeqEventTicks(miaSeqReadDuration24(buf, seq.cursor+1))
				a.seqGate(voice, false)
			}
			seq.noteIndex++
			seq.eventDuration = ticks
			seq.countdown = ticks - 1
			seq.cursor += uint32(size)
			return
		}

		if op == miaSeqOpJump {
			seq.cursor = miaSeqJumpTarget(buf, seq.cursor)
		} else {
			a.seqApplyConfigOp(voice, seq.cursor, op)
			seq.cursor += uint32(size)
		}

		budget--
		if budget < 0 {
			a.seqFinishTrack(voice)
			return
		}
	}
}

// seqCatchupStep silently walks forward through events entirely in the past,
// bounded to miaSeqCatchupBudget per tick, then resumes live on the first
// event that isn't fully covered. Mirrors audio_seq_catchup_step; see
// docs/audio-sequencer.md, "Catch-up (VOICE_RELEASE reconciliation)". This
// same per-iteration budget also bounds a cyclical run of JUMPs/config ops,
// the same way miaSeqDecodeBudget does in seqAdvance.
func (a *miaAudioState) seqCatchupStep(voice int) {
	seq := &a.seq[voice]
	buf := a.seqRAM[voice]
	budget := miaSeqCatchupBudget

	for budget > 0 && seq.catchingUp {
		budget--

		if seq.cursor >= uint32(len(buf)) {
			a.seqFinishTrack(voice)
			return
		}

		op := buf[seq.cursor]
		size := miaSeqRecordSize(op)

		if size == 0 {
			a.seqFinishTrack(voice)
			return
		}

		if op == miaSeqOpJump {
			seq.cursor = miaSeqJumpTarget(buf, seq.cursor)
			continue
		}

		if op == miaSeqOpNote || op == miaSeqOpRest {
			var ticks uint32
			if op == miaSeqOpNote {
				ticks = miaSeqEventTicks(miaSeqReadDuration24(buf, seq.cursor+3))
			} else {
				ticks = miaSeqEventTicks(miaSeqReadDuration24(buf, seq.cursor+1))
			}

			if ticks <= seq.catchupRemain {
				// Fully in the past: skip it silently, no register writes.
				seq.catchupRemain -= ticks
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
			seq.eventDuration = ticks
			seq.countdown = ticks - 1
			seq.cursor += uint32(size)
			return
		}

		a.seqApplyConfigOp(voice, seq.cursor, op)
		seq.cursor += uint32(size)
	}
}

// seqStep advances one voice's sequencer by one tick. Mirrors
// audio_seq_step; called from renderSampleLocked, once per voice per tick,
// before the envelope and oscillator pass - same placement as the firmware ISR.
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

	// This event is over: the next tick's catch-up step walks forward from
	// here using the leftover budget.
	seq.catchingUp = true
	seq.catchupRemain = totalElapsed - seq.eventDuration
}

/**************************************************************************************************
 * RAM-backed command entry points (run under c.mu)
 **************************************************************************************************/

// audioSeqSetBase mirrors mia_audio_seq_set_base: stores this voice's track
// base only. Does not itself move cursor/note_index - follow with
// audioSeqLoadTrack to (re)start decoding from the new base.
func (c *emulated_mia) audioSeqSetBase(voice int, base uint32) {
	if voice < 0 || voice >= miaAudioVoiceCount {
		return
	}

	a := &c.audio
	a.mu.Lock()
	a.seq[voice].trackBase = base
	a.mu.Unlock()
}

// audioSeqLoadTrack mirrors mia_audio_seq_load_track. A track has no
// declared length (just a base and an END/JUMP), so the audio engine keeps a
// view from this voice's trackBase to the top of MIA RAM. The view remains
// live as the CPU edits RAM. Loading resets cursor and note index to the top
// of the event stream; seqRAM's length is its effective bound.
func (c *emulated_mia) audioSeqLoadTrack(voice int) {
	if voice < 0 || voice >= miaAudioVoiceCount {
		return
	}

	a := &c.audio
	a.mu.Lock()
	defer a.mu.Unlock()

	base := a.seq[voice].trackBase
	if base >= miaRAMSize {
		a.seq[voice] = miaAudioSeqVoice{trackBase: base}
		a.seqRAM[voice] = nil
		return
	}

	a.seqRAM[voice] = c.memory[base:miaRAMSize]
	a.seq[voice] = miaAudioSeqVoice{trackBase: base}
}

func (c *emulated_mia) audioSeqStart(mask uint8) {
	a := &c.audio
	a.mu.Lock()
	defer a.mu.Unlock()

	for v := 0; v < miaAudioVoiceCount; v++ {
		if mask&(1<<uint(v)) == 0 {
			continue
		}
		// A voice no program has given a track stays stopped - no gate-off
		// and no sequence-done IRQ, exactly as if it weren't in the mask.
		if a.seq[v].trackBase == miaSeqNoBase {
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
// Reads of their addresses are supplied by live sequencer state through any
// index, rather than backed by c.memory - see the file comment.
func (c *emulated_mia) audioSeqConfigureIndexes() {
	c.audioConfigureIndex(miaAudioIndexSeqVoice0, miaAudioVoiceOffset(0)+miaAudioVoiceSeqNoteIndexL, 3)
	c.audioConfigureIndex(miaAudioIndexSeqVoice1, miaAudioVoiceOffset(1)+miaAudioVoiceSeqNoteIndexL, 3)
	c.audioConfigureIndex(miaAudioIndexSeqVoice2, miaAudioVoiceOffset(2)+miaAudioVoiceSeqNoteIndexL, 3)
	c.audioConfigureIndex(miaAudioIndexSeqVoice3, miaAudioVoiceOffset(3)+miaAudioVoiceSeqNoteIndexL, 3)
}

// audioSeqResetRuntimeState forgets every voice's track and resets
// per-voice runtime state, part of audioResetRuntimeState. The track bytes
// stay wherever the program put them - nothing in MIA RAM belongs to the
// sequencer (see docs/audio-sequencer.md).
func (c *emulated_mia) audioSeqResetRuntimeState() {
	a := &c.audio
	a.mu.Lock()
	for v := 0; v < miaAudioVoiceCount; v++ {
		a.seq[v] = miaAudioSeqVoice{trackBase: miaSeqNoBase}
		a.seqRAM[v] = nil
	}
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

// audioSeqStatusByte serves an indexed read of a sequencer field, computed
// live from sequencer state rather than c.memory. Caller holds c.mu.
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

// miaAudioSeqVoiceAtAddress finds the live fields within the audio block.
func miaAudioSeqVoiceAtAddress(address uint32) (int, bool) {
	if address < miaAudioVoicesOffset || address >= miaAudioVoicesOffset+miaAudioVoiceCount*miaAudioVoiceSize {
		return 0, false
	}
	rel := address - miaAudioVoicesOffset
	field := rel % miaAudioVoiceSize
	if field < miaAudioVoiceSeqNoteIndexL || field > miaAudioVoiceSeqStatus {
		return 0, false
	}
	return int(rel / miaAudioVoiceSize), true
}

/**************************************************************************************************
 * Terminal-only helper
 **************************************************************************************************/

// audioSeqWriteTestTrack writes a small built-in looping arpeggio at base and
// makes it the voice's track (does not load or start it), mirroring
// mia_audio_seq_write_test_track. For standalone bring-up before TRACK/BAND
// exist on the BASIC side.
func (c *emulated_mia) audioSeqWriteTestTrack(voice int, base uint32) {
	if voice < 0 || voice >= miaAudioVoiceCount {
		return
	}

	// attack=0/decay=3/sustain=F/release=5, then C4/E4/G4 (6000 ticks/
	// ~0.25s each), then JUMP -25 back to offset 0 (the SET_ADSR) to loop the
	// whole body forever. -25 as a signed 24-bit little-endian offset is
	// 0xFFFFE7 (0xE7, 0xFF, 0xFF).
	track := []byte{
		miaSeqOpSetADSR, 0x03, 0xF5,
		miaSeqOpNote, 0x5A, 0x10, 0x70, 0x17, 0x00, // C4
		miaSeqOpNote, 0x9A, 0x14, 0x70, 0x17, 0x00, // E4
		miaSeqOpNote, 0x80, 0x18, 0x70, 0x17, 0x00, // G4
		miaSeqOpJump, 0xE7, 0xFF, 0xFF,
	}

	if base > miaRAMSize-uint32(len(track)) {
		return
	}
	copy(c.memory[base:base+uint32(len(track))], track)
	c.audioSeqSetBase(voice, base)
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
		// The test arpeggio goes in the top 1 KiB of MIA RAM, 256 bytes per
		// voice, overwriting whatever a program stored there.
		base := miaRAMSize - 0x400 + uint32(voice)*0x100
		c.mu.Lock()
		c.audioSeqWriteTestTrack(int(voice), base)
		c.audioSeqLoadTrack(int(voice))
		c.audioSeqStart(mask)
		active := c.audioIsActive()
		if !active {
			c.audioEnable()
		}
		c.mu.Unlock()
		return fmt.Sprintf("Audio: voice %d playing built-in test arpeggio at $%05X\n", voice, base)
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
		if seq.trackBase == miaSeqNoBase {
			fmt.Fprintf(&out, "  ch%d: no track\n", v)
			continue
		}
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
		fmt.Fprintf(&out, "  ch%d: %s%s  note:%d  base:$%05X  cursor:$%03X  countdown:%d\n",
			v, state, extra, seq.noteIndex, seq.trackBase, seq.cursor, seq.countdown)
	}

	return out.String()
}
