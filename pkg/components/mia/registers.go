package mia

const (
	miaRAMSize       = 256 * 1024
	miaRegisterCount = 32
	miaRegisterMask  = 0x1F
	miaIndexCount    = 256
	miaAddressMask   = 0x00FFFFFF

	miaKernelTargetAddress = 0x4000
)

const (
	miaRegIdxAPort uint8 = iota
	miaRegIdxASelector
	miaRegCfgPort
	miaRegCfgSelector
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
	miaStatusMasterMode uint16 = 1 << 0
	miaStatusErrors     uint16 = 1 << 1
	miaStatusCmdRunning uint16 = 1 << 2
	miaStatusDMARunning uint16 = 1 << 3
)

const (
	miaIRQError     uint16 = 1 << 0
	miaIRQIdxAWrap  uint16 = 1 << 1
	miaIRQIdxBWrap  uint16 = 1 << 2
	miaIRQCommand   uint16 = 1 << 3
	miaIRQTriggered uint16 = 1 << 15
)

var miaKernelData = [...]uint8{
	0xA9, 0x01, 0x1A, 0xAD, 0x01, 0xC0, 0xAE, 0x03,
	0xC0, 0xAC, 0x00, 0xC0, 0x4C, 0x02, 0x40,
}
