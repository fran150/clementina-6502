; demo.s - DEMO.BAS rewritten in 6502 assembly.
;
; The same demo as demo.bas: five bouncing 8x8 balls in five colours over a
; two-voice background tune, 400 frames, then "DEMO DONE". The comments name
; the BASIC lines each part replaces. Line 350's DELAY 30 becomes FRAME_MS
; (10 ms): without BASIC's own time per frame, 30 ms is slower than needed.
;
; The program lives in extended RAM bank 1 at $8000, so loading it leaves
; BASIC untouched. demoprg.bas starts it with BLOAD "DEMO.PRG",32768, which
; loads the file and jumps to $8000 with bank 1 still selected. At the end it
; returns to BASIC's READY prompt through the kernel's warm start, which
; selects bank 0 again; the loader program is still there to RUN again.
;
; Build with cc65:
;   ca65 demo.s
;   ld65 -C demo.cfg -o demo.prg demo.o
; ca65 -D FRAME_MS=n demo.s builds it with a different delay (0 = none).

.setcpu "65C02"

; ---------------------------------------------------------------------------
; Machine interface (clementina-sdk specs/)

; Kernel jump table (kernel-abi.json)
KERN_WARMSTART  = $04BA         ; BASIC's READY prompt, bank 0 selected
KERN_CHROUT     = $04BD         ; prints A; keeps A, X and Y

CHR_FF          = $0C           ; clears the screen (what CLS prints)
CHR_CR          = $0D           ; newline

; MIA registers (mia-registers.json)
IDXA_PORT       = $FFE0
IDXA_SELECT     = $FFE1
CFG_SELECT      = $FFE2
CFG_PORT        = $FFE3
IDXB_PORT       = $FFE4
IDXB_SELECT     = $FFE5
CMD_PARAM1      = $FFE6
CMD_PARAM2      = $FFE7
CMD_PARAM3      = $FFE8
CMD_TRIGGER     = $FFE9
STATUS_L        = $FFEA
STATUS_BUSY     = %00001100     ; a command or a copy is running

; Index descriptor fields, through CFG_SELECT: window A's, then window B's
CFG_A_ADDR      = $00           ; low, middle, high
CFG_A_STEP      = $09           ; low, high
CFG_A_FLAGS     = $0B
CFG_B_ADDR      = $10
CFG_B_STEP      = $19
CFG_B_FLAGS     = $1B
STEP_ON_READ    = %00000001
STEP_ON_WRITE   = %00000010

; The descriptors this program uses: window A reads, window B writes. They
; are the pair the SDK runtime uses for the same job.
IDX_READ        = $40
IDX_WRITE       = $41

; MIA commands (mia-commands.json, audio.json)
CMD_CLOCK_SNAPSHOT = $55
CMD_AUDIO_ENABLE   = $60
CMD_AUDIO_STOP     = $61
CMD_SEQ_LOAD       = $63        ; PARAM1 = voice mask
CMD_SEQ_START      = $64
CMD_SEQ_STOP       = $65
CMD_SEQ_SET_BASE0  = $68        ; + voice; PARAM1-3 = track address

; MIA RAM (video.json, audio.json, docs/architecture/timing-and-interrupts.md)
RC_LAYERS       = $000021       ; bit 2: sprite layer
RC_SPRITE_BANK  = $00002C       ; CHR bank sprites draw from
RC_1BPP_MASK    = $00002D       ; bit n: CHR bank n is 1bpp
RC_1BPP_PLANES  = $00002E       ; background, sprite, overlay plane, 2 bits each
RC_OAM_LAST     = $000030       ; highest OAM index drawn
PALETTES        = $000100       ; 16 banks of 8 RGB565 colours, low byte first
CHR             = $000200       ; 8 banks of 6,144 bytes
OAM             = $010850       ; 5 bytes a sprite: tile, x, y, attr, ext
CLOCK_MS        = $011078       ; 32-bit milliseconds, after CLOCK_SNAPSHOT
AUDIO_VOLUME    = $012001       ; master volume 0-15
AUDIO_VOICE0    = $012010       ; 16 bytes a voice
VOICE_ADSR      = 3             ; ATTACK_DECAY, SUSTAIN_RELEASE, WAVEFORM, PAN
TRACK0          = $014000       ; where TRACK puts voice 0's track,
TRACK1          = $015000       ; and voice 1's

LAYER_SPRITES   = %00000100

; Sequencer opcodes (audio.json)
SEQ_NOTE        = $01           ; frequency (2 bytes), duration (3)
SEQ_REST        = $02           ; duration (3)
SEQ_JUMP        = $08           ; offset from the end of the record (3)

; ---------------------------------------------------------------------------
; The demo

BALLS           = 5
FRAMES          = 400           ; FOR F=1 TO 400
.ifndef FRAME_MS
FRAME_MS        = 10            ; DELAY 10 after each frame
.endif
.assert FRAME_MS < 256, error, "FRAME_MS must be below 256"
BALL_BANK       = 2             ; CHR bank holding the ball tile
MAX_X           = 311           ; 320 - 8 - 1
MAX_Y           = 191           ; 200 - 8 - 1

.segment "ZEROPAGE"             ; $98-$DF, free for programs: BASIC never uses it
ptr:            .res 2          ; source for copy and print

.segment "BSS"
mia:            .res 3          ; MIA address for the next seek
count:          .res 1
mask:           .res 1
frames:         .res 2          ; frames left
start_ms:       .res 2
pos_lo:         .res BALLS * 2  ; X(0)-X(4), then Y(0)-Y(4)
pos_hi:         .res BALLS * 2
speed:          .res BALLS * 2  ; DX(0)-DX(4), then DY(0)-DY(4)

; mia = a 24-bit MIA address
.macro set_mia address
        lda #<(address)
        sta mia
        lda #>(address)
        sta mia+1
        lda #^(address)
        sta mia+2
.endmacro

; Copies length (< 256) bytes from source to MIA RAM at destination.
.macro copy source, length, destination
        lda #<(source)
        sta ptr
        lda #>(source)
        sta ptr+1
        lda #(length)
        sta count
        set_mia destination
        jsr copy_to_mia
.endmacro

; PRINT "text"
.macro print text
        lda #<(text)
        sta ptr
        lda #>(text)
        sta ptr+1
        jsr print_line
.endmacro

.segment "HEADER"
        .addr start             ; PRG header: load address
        .byte 1                 ; and extended RAM bank

.segment "CODE"
start:                          ; BLOAD's run address
        cld

; 5 CLS
        lda #CHR_FF
        jsr KERN_CHROUT
; 15 PRINT "CLEMENTINA - COLOR SPRITES + MUSIC"
        print title

; 40 CHRMODE 2,1 : CHRPLANE 0,0,0 : SPRBANK 2 : SPRCOUNT 4
        set_mia RC_1BPP_MASK
        lda #1 << BALL_BANK
        jsr set_bits
        set_mia RC_1BPP_PLANES
        lda #0
        jsr poke
        set_mia RC_SPRITE_BANK
        lda #BALL_BANK
        jsr poke
        set_mia RC_OAM_LAST
        lda #BALLS - 1
        jsr poke

; 50-60 CHRREAD 2,0,8 : DATA ...
        copy ball_tile, 8, CHR + BALL_BANK * 6144

; 70-110 PALETTE p,1,r,g,b: colour 1 of each ball's palette
        ldx #0
@colour:
        lda ball_palette,x
        asl a
        asl a
        asl a
        asl a                   ; 16 bytes a palette
        ora #2                  ; colour 1
        sta mia
        lda #>PALETTES
        sta mia+1
        stz mia+2
        jsr seek_write
        lda colour_lo,x
        sta IDXB_PORT
        lda colour_hi,x
        sta IDXB_PORT
        inx
        cpx #BALLS
        bne @colour

; 120-200 FOR I=0 TO 4 : READ X(I),Y(I),DX(I),DY(I),P(I)
;         SPRITE I,0,X(I),Y(I),P(I),0 : NEXT I
        ldx #BALLS * 2 - 1
@start: lda start_lo,x
        sta pos_lo,x
        lda start_hi,x
        sta pos_hi,x
        lda start_speed,x
        sta speed,x
        dex
        bpl @start
        jsr show_balls

; 210 SPRON
        set_mia RC_LAYERS
        lda #LAYER_SPRITES
        jsr set_bits

; 220 SNDON : VOL 10
        lda #CMD_AUDIO_ENABLE
        ldx #0
        jsr command
        set_mia AUDIO_VOLUME
        lda #10
        jsr poke

; 230-240 WAVE, ADSR and PAN for voices 0 and 1
        copy voice0, 4, AUDIO_VOICE0 + VOICE_ADSR
        copy voice1, 4, AUDIO_VOICE0 + 16 + VOICE_ADSR

; 250-255 TRACK 0,... : TRACK 1,...
        copy track0, track0_end - track0, TRACK0
        ldx #0
        jsr load_track
        copy track1, track1_end - track1, TRACK1
        ldx #1
        jsr load_track

; 260 BAND 1 (BASIC starts every voice that has a track; only 0 and 1 do)
        lda #CMD_SEQ_START
        ldx #%0011
        jsr command

; 270 FOR F=1 TO 400
        lda #<FRAMES
        sta frames
        lda #>FRAMES
        sta frames+1
frame:
; 290-340 FOR I=0 TO 4: move ball I, bounce it off the edges, SPRX/SPRY
        ldx #0
@axis:  ldy #0                  ; X(I) for entries 0-4,
        cpx #BALLS
        bcc @move
        iny                     ; Y(I) for entries 5-9
@move:  jsr move
        inx
        cpx #BALLS * 2
        bne @axis
        jsr show_balls
; 350 DELAY 30, here FRAME_MS
.if FRAME_MS > 0
        jsr delay
.endif
; 360 NEXT F
        lda frames
        bne @low
        dec frames+1
@low:   dec frames
        lda frames
        ora frames+1
        bne frame

; 370 BAND 0
        lda #CMD_SEQ_STOP
        ldx #%1111
        jsr command
; 380 SPROFF : SNDOFF
        set_mia RC_LAYERS
        lda #LAYER_SPRITES
        jsr clear_bits
        lda #CMD_AUDIO_STOP
        ldx #0
        jsr command
; 390 PRINT "DEMO DONE"
        print done_text

; Return to READY.
        jmp KERN_WARMSTART

; ---------------------------------------------------------------------------
; Moves one coordinate of one ball (lines 300-320): adds its speed, and if
; that takes it outside 0..maximum, reverses the speed and adds it again.
; X = entry (ball, or ball + 5 for y), Y = 0 for x, 1 for y. Keeps X and Y.
move:
        jsr add_speed
        lda pos_hi,x
        bmi @bounce             ; below 0
        lda max_lo,y            ; above the maximum?
        cmp pos_lo,x
        lda max_hi,y
        sbc pos_hi,x
        bcs @done
@bounce:
        lda speed,x
        eor #$FF
        inc a
        sta speed,x
        jmp add_speed
@done:  rts

; Adds entry X's signed speed to its 16-bit position.
add_speed:
        lda speed,x
        clc
        adc pos_lo,x
        sta pos_lo,x
        lda #0
        bit speed,x
        bpl @high
        lda #$FF                ; a negative speed's high byte
@high:  adc pos_hi,x
        sta pos_hi,x
        rts

; Writes all five OAM records (lines 140 and 330): tile 0, the x and y low
; bytes, the palette, then x bits 8-9 and y bit 8. Flags are 0: shown, no
; flip, no priority.
show_balls:
        set_mia OAM
        jsr seek_write
        ldx #0
@ball:  stz IDXB_PORT           ; tile
        lda pos_lo,x
        sta IDXB_PORT           ; x
        lda pos_lo+BALLS,x
        sta IDXB_PORT           ; y
        lda ball_palette,x
        sta IDXB_PORT           ; attr
        lda pos_hi+BALLS,x
        and #1
        asl a
        asl a
        sta mask
        lda pos_hi,x
        and #3
        ora mask
        sta IDXB_PORT           ; ext
        inx
        cpx #BALLS
        bne @ball
        rts

; Waits FRAME_MS milliseconds of MIA wall time, the clock BASIC's DELAY reads.
delay:
        jsr read_ms
        sta start_ms
        stx start_ms+1
@wait:  jsr read_ms
        sec
        sbc start_ms
        tay
        txa
        sbc start_ms+1
        bne @done               ; 256 ms or more
        cpy #FRAME_MS
        bcc @wait
@done:  rts

; A (low) and X (high) = the bottom 16 bits of MIA's millisecond clock.
read_ms:
        lda #CMD_CLOCK_SNAPSHOT
        ldx #0
        jsr command
        jsr wait_idle           ; the snapshot is written when it finishes
        set_mia CLOCK_MS
        jsr seek_read
        lda IDXA_PORT
        ldx IDXA_PORT
        rts

; Points voice X's sequencer at the track just copied to `mia`, as TRACK
; does: AUDIO_SEQ_SET_BASE, then AUDIO_SEQ_LOAD.
load_track:
        jsr wait_idle
        lda mia
        sta CMD_PARAM1
        lda mia+1
        sta CMD_PARAM2
        lda mia+2
        sta CMD_PARAM3
        txa
        clc
        adc #CMD_SEQ_SET_BASE0
        sta CMD_TRIGGER
        lda voice_bit,x
        tax
        lda #CMD_SEQ_LOAD
        jmp command

; Prints the zero-terminated text at (ptr), then a newline.
print_line:
        ldy #0
@char:  lda (ptr),y
        beq @end
        jsr KERN_CHROUT
        iny
        bne @char
@end:   lda #CHR_CR
        jmp KERN_CHROUT

; Copies `count` bytes from (ptr) to MIA RAM at `mia`.
copy_to_mia:
        jsr seek_write
        ldy #0
@byte:  lda (ptr),y
        sta IDXB_PORT
        iny
        cpy count
        bne @byte
        rts

; Writes A to MIA RAM at `mia`. Keeps X and Y.
poke:   pha
        jsr seek_write
        pla
        sta IDXB_PORT
        rts

; Sets A's bits in the byte at `mia`. Keeps X and Y.
set_bits:
        sta mask
        jsr seek_read
        lda IDXA_PORT
        ora mask
        bra poke

; Clears A's bits in the byte at `mia`. Keeps X and Y.
clear_bits:
        eor #$FF
        sta mask
        jsr seek_read
        lda IDXA_PORT
        and mask
        bra poke

; Points window B at `mia`, stepping one byte after each write. The address
; goes last, high byte first, since each address byte reloads the port.
; Keeps X and Y.
seek_write:
        jsr wait_idle
        lda #IDX_WRITE
        sta IDXB_SELECT
        lda #CFG_B_FLAGS
        sta CFG_SELECT
        lda #STEP_ON_WRITE
        sta CFG_PORT
        lda #CFG_B_STEP
        sta CFG_SELECT
        lda #1
        sta CFG_PORT
        lda #CFG_B_STEP+1
        sta CFG_SELECT
        stz CFG_PORT
        lda #CFG_B_ADDR+2
        sta CFG_SELECT
        lda mia+2
        sta CFG_PORT
        lda #CFG_B_ADDR+1
        sta CFG_SELECT
        lda mia+1
        sta CFG_PORT
        lda #CFG_B_ADDR
        sta CFG_SELECT
        lda mia
        sta CFG_PORT
        rts

; Points window A at `mia`, stepping one byte after each read. Keeps X and Y.
seek_read:
        jsr wait_idle
        lda #IDX_READ
        sta IDXA_SELECT
        lda #CFG_A_FLAGS
        sta CFG_SELECT
        lda #STEP_ON_READ
        sta CFG_PORT
        lda #CFG_A_STEP
        sta CFG_SELECT
        lda #1
        sta CFG_PORT
        lda #CFG_A_STEP+1
        sta CFG_SELECT
        stz CFG_PORT
        lda #CFG_A_ADDR+2
        sta CFG_SELECT
        lda mia+2
        sta CFG_PORT
        lda #CFG_A_ADDR+1
        sta CFG_SELECT
        lda mia+1
        sta CFG_PORT
        lda #CFG_A_ADDR
        sta CFG_SELECT
        lda mia
        sta CFG_PORT
        rts

; Sends MIA command A with PARAM1 = X once the previous one has finished.
command:
        jsr wait_idle
        stx CMD_PARAM1
        stz CMD_PARAM2
        stz CMD_PARAM3
        sta CMD_TRIGGER
        rts

; Waits until no MIA command or copy is running. Keeps A, X and Y.
wait_idle:
        pha
@busy:  lda STATUS_L
        and #STATUS_BUSY
        bne @busy
        pla
        rts

; ---------------------------------------------------------------------------
.segment "RODATA"

title:          .byte "CLEMENTINA - COLOR SPRITES + MUSIC", 0
done_text:      .byte "DEMO DONE", 0

; 60 DATA 60,126,255,255,255,255,126,60
ball_tile:
        .byte %00111100
        .byte %01111110
        .byte %11111111
        .byte %11111111
        .byte %11111111
        .byte %11111111
        .byte %01111110
        .byte %00111100

; 70-110 PALETTE 1-5,1,r,g,b, and P(I): ball I uses palette I+1
.define RGB565(r, g, b) ((r) << 11 | (g) << 5 | (b))
RED             = RGB565(31, 0, 0)
GREEN           = RGB565(0, 63, 0)
BLUE            = RGB565(0, 0, 31)
YELLOW          = RGB565(31, 63, 0)
MAGENTA         = RGB565(31, 0, 31)
colour_lo:      .lobytes RED, GREEN, BLUE, YELLOW, MAGENTA
colour_hi:      .hibytes RED, GREEN, BLUE, YELLOW, MAGENTA
ball_palette:   .byte 1, 2, 3, 4, 5

; 160-200 DATA: x of balls 0-4, then their y; then dx, then dy
.define START 20, 100, 180, 60, 220,  20, 40, 60, 120, 150
start_lo:       .lobytes START
start_hi:       .hibytes START
start_speed:    .lobytes 3, -2, 2, -3, 1,  2, 3, -2, -1, 3

max_lo:         .lobytes MAX_X, MAX_Y
max_hi:         .hibytes MAX_X, MAX_Y

voice_bit:      .byte %0001, %0010, %0100, %1000

; 230-240: ATTACK_DECAY, SUSTAIN_RELEASE, WAVEFORM and PAN of one voice
.macro voice wave, attack, decay, sustain, release, pan
        .byte (attack) << 4 | (decay), (sustain) << 4 | (release), wave, <(pan)
.endmacro
voice0: voice 1, 0, 8, 10, 6, -30       ; WAVE 0,1 : ADSR 0,0,8,10,6 : PAN 0,-30
voice1: voice 2, 0, 6, 10, 6, 30        ; WAVE 1,2 : ADSR 1,0,6,10,6 : PAN 1,30

; 250-255: the two tracks, as the sequencer bytes TRACK compiles them to.
; T90 is 90 ticks of 1/160 s a quarter note, 150 sequencer ticks (24 kHz)
; each; pitches are Hz * 16, TRACK's octave-0 table shifted by the octave.
QUARTER         = 90 * 150
EIGHTH          = QUARTER / 2
DOTTED_HALF     = QUARTER * 3
WHOLE           = QUARTER * 4

C3              = 262 << 3
G3              = 392 << 3
C4              = 262 << 4
E4              = 330 << 4
G4              = 392 << 4
C5              = 262 << 5

.macro note pitch, length
        .byte SEQ_NOTE
        .word pitch
        .faraddr length
.endmacro
.macro rest length
        .byte SEQ_REST
        .faraddr length
.endmacro
.macro jump_to target           ; relative, so the track can sit anywhere
        .byte SEQ_JUMP
        .faraddr ((target) - (* + 3)) & $FFFFFF
.endmacro

track0:                         ; TRACK 0,"T90 L8 O4 | C E G > C < G E R1"
track0_loop:
        note C4, EIGHTH
        note E4, EIGHTH
        note G4, EIGHTH
        note C5, EIGHTH
        note G4, EIGHTH
        note E4, EIGHTH
        rest WHOLE
        jump_to track0_loop
track0_end:

track1:                         ; TRACK 1,"T90 L4 O3 | R2. C G C G"
track1_loop:
        rest DOTTED_HALF
        note C3, QUARTER
        note G3, QUARTER
        note C3, QUARTER
        note G3, QUARTER
        jump_to track1_loop
track1_end:
