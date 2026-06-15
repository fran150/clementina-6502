package mia

import "github.com/fran150/clementina-6502/assets"

const (
	miaRAMSize       = 128 * 1024
	miaRAMMask       = miaRAMSize - 1
	miaRegisterCount = 32
	miaRegisterMask  = 0x1F
	miaIndexCount    = 256
	miaAddressMask   = 0x00FFFFFF

	miaKernelTargetAddress = 0x4000
	miaCPUResetPulseCycles = 4
	miaDefaultPhi2Hz       = 1200000
	miaMinPhi2Hz           = 1
	miaMaxPhi2Hz           = 8000000
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
	miaRegInputStatus
	miaRegInputChar
	miaRegInputCharCount
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
	miaStatusExecPaused     uint16 = 1 << 7
	miaStatusAudioActive    uint16 = 1 << 8
	miaStatusSDPresent      uint16 = 1 << 9
	miaStatusSDBusy         uint16 = 1 << 10
	miaStatusFSMounted      uint16 = 1 << 11
)

const (
	miaIRQError         uint16 = 1 << 0
	miaIRQIdxAWrap      uint16 = 1 << 1
	miaIRQIdxBWrap      uint16 = 1 << 2
	miaIRQCommand       uint16 = 1 << 3
	miaIRQSpeedChanged  uint16 = 1 << 4
	miaIRQVideoRequest  uint16 = 1 << 5
	miaIRQVideoSent     uint16 = 1 << 6
	miaIRQVideoAcked    uint16 = 1 << 7
	miaIRQInputKeyboard uint16 = 1 << 8
	miaIRQInputMouse    uint16 = 1 << 9
	miaIRQInputGamepad  uint16 = 1 << 10
	miaIRQSDDone        uint16 = 1 << 11
	miaIRQSDError       uint16 = 1 << 12
	miaIRQFSEvent       uint16 = 1 << 13
	miaIRQTriggered     uint16 = 1 << 15
)

// MIA ROM bootstrap installed at $4000. Embedded at build time so it does not
// depend on the working directory or external files at runtime.
var miaKernelData = assets.MiaKernel
