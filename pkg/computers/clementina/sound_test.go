package clementina

import (
	"testing"
)

// The BASIC sound statements (SNDON/SNDOFF/SNDCLR/VOL/WAVE/NOTE/FREQ/GATE/ADSR/
// PULSE/PAN, handlers in clementina-rom src/basic/clementina_extra.s) drive the
// MIA PSG by writing its audio RAM block at $12000 through index window B. These
// tests boot the real ROM, type a program, run it, and read the block back.
//
// SNDON is deliberately not exercised here: enabling the engine opens a host
// audio device. The $60/$61/$62 command path is covered in the mia package
// (TestEmulatedMiaAudioCommandsToggleState); SNDCLR ($62) is safe to drive from
// here because it only clears RAM.

// DebugReadVideo is a raw read of MIA internal RAM, so it reaches the audio
// block just as well as the video region.
type audioMemReader interface{ DebugReadVideo(uint32) uint8 }

const audioBlockBase = 0x12000

func peekAudio(computer *ClementinaComputer, blockOffset uint32) uint8 {
	return computer.chips.mia.(audioMemReader).DebugReadVideo(audioBlockBase + blockOffset)
}

// voiceField returns a block offset: 16-byte header, then 16-byte voice records.
func voiceField(voice, field uint32) uint32 { return 0x10 + voice*0x10 + field }

func TestBasicSoundRegisterWrites(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, "10 WAVE 0,2 : PULSE 0,64 : ADSR 0,1,3,10,6")
	typeLine(computer, step, "20 PAN 0,-64 : VOL 0,200")
	typeLine(computer, step, "30 NOTE 0,48")
	typeLine(computer, step, "40 VOL 9")
	typeLine(computer, step, "50 FREQ 1,1000 : GATE 1,1")
	typeLine(computer, step, "60 GOTO 60")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 2_000_000)

	// Master volume (header offset $01).
	require_eq(t, 9, peekAudio(computer, 0x01), "master VOL")

	// Voice 0. NOTE 0,48 is C4: octave-0 C is round(261.63*16)=262, <<4 = 4192 = $1060.
	require_eq(t, 0x60, peekAudio(computer, voiceField(0, 0)), "v0 FREQ_L")
	require_eq(t, 0x10, peekAudio(computer, voiceField(0, 1)), "v0 FREQ_H")
	require_eq(t, 64, peekAudio(computer, voiceField(0, 2)), "v0 PULSE_WIDTH")
	require_eq(t, 0x13, peekAudio(computer, voiceField(0, 3)), "v0 ATTACK_DECAY = (1<<4)|3")
	require_eq(t, 0xA6, peekAudio(computer, voiceField(0, 4)), "v0 SUSTAIN_RELEASE = (10<<4)|6")
	require_eq(t, 2, peekAudio(computer, voiceField(0, 5)), "v0 WAVEFORM = saw")
	require_eq(t, 0xC0, peekAudio(computer, voiceField(0, 6)), "v0 PAN = -64 (int8 $C0)")
	require_eq(t, 0x03, peekAudio(computer, voiceField(0, 7)), "v0 CONTROL = GATE|RESET_PHASE (NOTE)")
	require_eq(t, 200, peekAudio(computer, voiceField(0, 8)), "v0 VOLUME")

	// Voice 1. FREQ 1,1000 -> 1000*16 = 16000 = $3E80; GATE 1,1 -> CONTROL bit 0.
	require_eq(t, 0x80, peekAudio(computer, voiceField(1, 0)), "v1 FREQ_L")
	require_eq(t, 0x3E, peekAudio(computer, voiceField(1, 1)), "v1 FREQ_H")
	require_eq(t, 0x01, peekAudio(computer, voiceField(1, 7)), "v1 CONTROL = GATE (no retrigger)")
}

func TestBasicSoundNoteOctaves(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	// C0 = 262, C1 = 524, C7 = 262<<7 = 33536 = $8300.
	typeLine(computer, step, "10 NOTE 0,0 : NOTE 1,12 : NOTE 2,84")
	typeLine(computer, step, "20 GOTO 20")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_500_000)

	require_eq(t, 0x06, peekAudio(computer, voiceField(0, 0)), "C0 FREQ_L (262)")
	require_eq(t, 0x01, peekAudio(computer, voiceField(0, 1)), "C0 FREQ_H")
	require_eq(t, 0x0C, peekAudio(computer, voiceField(1, 0)), "C1 FREQ_L (524)")
	require_eq(t, 0x02, peekAudio(computer, voiceField(1, 1)), "C1 FREQ_H")
	require_eq(t, 0x00, peekAudio(computer, voiceField(2, 0)), "C7 FREQ_L (33536)")
	require_eq(t, 0x83, peekAudio(computer, voiceField(2, 1)), "C7 FREQ_H")
}

func TestBasicSoundClearResetsBlock(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)

	typeLine(computer, step, "10 WAVE 0,4 : PULSE 0,10 : VOL 0,20 : VOL 3")
	typeLine(computer, step, "20 SNDCLR")
	typeLine(computer, step, "30 GOTO 30")
	typeLine(computer, step, "RUN")
	editorTickN(computer, step, 1_500_000)

	// SNDCLR ($62) restores the layout-v2 defaults.
	require_eq(t, 0x0F, peekAudio(computer, 0x01), "master volume back to full")
	require_eq(t, 0x01, peekAudio(computer, voiceField(0, 5)), "WAVEFORM back to pulse")
	require_eq(t, 128, peekAudio(computer, voiceField(0, 2)), "PULSE_WIDTH back to 128")
	require_eq(t, 0xFF, peekAudio(computer, voiceField(0, 8)), "VOLUME back to unity")
}

// TestBasicSoundNoteRestartsSoundingNote verifies NOTE over a note that is
// still sounding starts a new one: MIA restarts the envelope only on the
// gate's rising edge, so NOTE drops the gate before raising it. The engine is
// forced active without a host device (see DebugSetAudioActive) so the
// writes reach it.
func TestBasicSoundNoteRestartsSoundingNote(t *testing.T) {
	computer, step := bootClementinaToPrompt(t)
	audio := computer.chips.mia.(interface {
		audioAdvancer
		DebugAudioFlush()
	})
	audio.DebugSetAudioActive(true)

	typeLine(computer, step, "ADSR 0,12,0,15,0 : NOTE 0,48") // 1 s attack
	audio.DebugAudioFlush()
	audio.DebugAudioReadPCM(2400) // 50 ms into the attack
	_, _, adsr, vol := audio.DebugAudioVoiceState(0)
	require_eq(t, 1, adsr, "first note is in its attack")
	if vol == 0 {
		t.Fatal("first note's envelope never rose")
	}

	typeLine(computer, step, "NOTE 0,50")
	audio.DebugAudioFlush()
	freq, control, adsr, vol := audio.DebugAudioVoiceState(0)
	require_eq_u16(t, 0x1260, freq, "D4 (note_freq(50) = 294<<4)")
	require_eq(t, 0x03, control, "gate+reset-phase")
	require_eq(t, 1, adsr, "second note restarts its attack")
	if vol != 0 {
		t.Fatalf("second note's envelope must restart from silence, got %d", vol)
	}
}

func TestBasicSoundIllegalQuantityRecovers(t *testing.T) {
	cases := []string{
		"WAVE 0,9",        // waveform 9 > 4
		"WAVE 4,0",        // voice 4 > 3
		"VOL 16",          // master > 15
		"NOTE 0,96",       // semitone 96 > 95
		"FREQ 0,5000",     // Hz 5000 > 4095
		"PAN 0,64",        // pan 64 > 63
		"PAN 0,-65",       // pan -65 < -64
		"ADSR 0,1,2,3,16", // nibble 16 > 15
	}
	for _, line := range cases {
		line := line
		t.Run(line, func(t *testing.T) {
			computer, step := bootClementinaToPrompt(t)
			typeLine(computer, step, line)
			x0 := peek(computer, addrCursorX)
			injectKeys(computer, 'Z')
			editorTickN(computer, step, 300_000)
			if peek(computer, addrCursorX) != x0+1 {
				t.Fatalf("%q should raise ILLEGAL QUANTITY and leave the prompt usable", line)
			}
		})
	}
}
