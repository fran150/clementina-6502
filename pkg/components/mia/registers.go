package mia

const (
	miaRAMSize       = 128 * 1024
	miaRAMMask       = miaRAMSize - 1
	miaRegisterCount = 32
	miaRegisterMask  = 0x1F
	miaIndexCount    = 256
	miaAddressMask   = 0x00FFFFFF

	miaKernelTargetAddress = 0x4000
	miaCPUResetPulseCycles = 4
	miaDefaultPhi2Hz       = 2000
)

const (
	miaCfgSpeedL uint8 = 0x20
	miaCfgSpeedM uint8 = 0x21
	miaCfgSpeedH uint8 = 0x22
)

const (
	miaRegIdxAPort uint8 = iota
	miaRegIdxASelector
	miaRegCfgSelector
	miaRegCfgPort
	miaRegIdxBPort
	miaRegIdxBSelector
	miaRegCmdParam1
	miaRegCmdParam2
	miaRegCmdParam3
	miaRegCmdTrigger
	miaRegStatusLSB
	miaRegStatusMSB
	miaRegErrorLSB
	miaRegErrorMSB
	miaRegIRQMaskLSB
	miaRegIRQMaskMSB
	miaRegIRQStatusLSB
	miaRegIRQStatusMSB
	miaRegReserved12
	miaRegReserved13
	miaRegReserved14
	miaRegReserved15
	miaRegReserved16
	miaRegReserved17
	miaRegReserved18
	miaRegReserved19
	miaRegNMIVectorLSB
	miaRegNMIVectorMSB
	miaRegResetVectorLSB
	miaRegResetVectorMSB
	miaRegIRQVectorLSB
	miaRegIRQVectorMSB
)

type miaState uint8

const (
	miaStateLoader miaState = iota
	miaStateNormal
)

const (
	miaStatusMasterMode     uint16 = 1 << 0
	miaStatusErrors         uint16 = 1 << 1
	miaStatusCmdRunning     uint16 = 1 << 2
	miaStatusDMARunning     uint16 = 1 << 3
	miaStatusSpeedChanging  uint16 = 1 << 4
	miaStatusVideoRequested uint16 = 1 << 5
	miaStatusVideoSent      uint16 = 1 << 6
)

const (
	miaIRQError        uint16 = 1 << 0
	miaIRQIdxAWrap     uint16 = 1 << 1
	miaIRQIdxBWrap     uint16 = 1 << 2
	miaIRQCommand      uint16 = 1 << 3
	miaIRQSpeedChanged uint16 = 1 << 4
	miaIRQVideoRequest uint16 = 1 << 5
	miaIRQVideoSent    uint16 = 1 << 6
	miaIRQVideoAcked   uint16 = 1 << 7
	miaIRQTriggered    uint16 = 1 << 15
)

// MIA ROM bootstrap installed at $4000. It enables video, initializes palette
// bank 0 with a few RGB565 colors, then polls the latched video ACK event before
// moving the backdrop color to the next palette entry.
var miaKernelData = [...]uint8{
	0xA9, 0x40, 0x8D, 0xE9, 0xFF, // lda #VIDEO_ENABLE; sta CMD_TRIGGER
	0xA9, 0x90, 0x8D, 0xE1, 0xFF, // lda #PALETTE_BANK0; sta IDXA_SELECT

	0xA9, 0x00, 0x8D, 0xE0, 0xFF, 0xA9, 0x00, 0x8D, 0xE0, 0xFF, // black
	0xA9, 0x1F, 0x8D, 0xE0, 0xFF, 0xA9, 0x00, 0x8D, 0xE0, 0xFF, // blue
	0xA9, 0xE0, 0x8D, 0xE0, 0xFF, 0xA9, 0x07, 0x8D, 0xE0, 0xFF, // green
	0xA9, 0x00, 0x8D, 0xE0, 0xFF, 0xA9, 0xF8, 0x8D, 0xE0, 0xFF, // red
	0xA9, 0xE0, 0x8D, 0xE0, 0xFF, 0xA9, 0xFF, 0x8D, 0xE0, 0xFF, // yellow
	0xA9, 0x1F, 0x8D, 0xE0, 0xFF, 0xA9, 0xF8, 0x8D, 0xE0, 0xFF, // magenta
	0xA9, 0xFF, 0x8D, 0xE0, 0xFF, 0xA9, 0x07, 0x8D, 0xE0, 0xFF, // cyan
	0xA9, 0xFF, 0x8D, 0xE0, 0xFF, 0xA9, 0xFF, 0x8D, 0xE0, 0xFF, // white

	0xA9, 0x01, 0x8D, 0xE6, 0xFF, // lda #VIDEO_MODE_ENABLE; sta CMD_PARAM1
	0xA9, 0x43, 0x8D, 0xE9, 0xFF, // lda #VIDEO_SET_MODE; sta CMD_TRIGGER
	0xA9, 0x00, 0x85, 0x00, // lda #0; sta $00
	0xA9, 0x87, 0x8D, 0xE1, 0xFF, // lda #BACKDROP_COLOR; sta IDXA_SELECT
	0xA9, 0x00, 0x8D, 0xE0, 0xFF, // lda #0; sta IDXA_PORT

	0xAD, 0xF0, 0xFF, 0x29, 0x80, 0xF0, 0xF9, // wait_ack: poll IRQ_VIDEO_ACKED
	0xAD, 0xF0, 0xFF, 0x29, 0x7F, 0x8D, 0xF0, 0xFF, // clear ACK event
	0xE6, 0x00, 0xA5, 0x00, 0x29, 0x07, 0x85, 0x00,
	0x8D, 0xE0, 0xFF, // sta IDXA_PORT
	0x4C, 0x72, 0x40, // jmp wait_ack
}
