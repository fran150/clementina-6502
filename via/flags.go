package via

type viaACRLatchingMasks uint8

const (
	acrMaskLatchingEnabledA viaACRLatchingMasks = 0x01
	acrMaskLatchingEnabledB viaACRLatchingMasks = 0x02
)

type viaPCRTranstitionMasks uint8

const (
	pcrMaskCA1TransitionType viaPCRTranstitionMasks = 0x01
	pcrMaskCA2TransitionType viaPCRTranstitionMasks = 0x0C
	pcrMaskCB1TransitionType viaPCRTranstitionMasks = 0x10
	pcrMaskCB2TransitionType viaPCRTranstitionMasks = 0xC0
)

type viaPCRInterruptClearMasks uint8

const (
	pcrMaskCA2ClearOnRW viaPCRInterruptClearMasks = 0x02
	pcrMaskCB2ClearOnRW viaPCRInterruptClearMasks = 0x20
)

type viaPCROutputMasks uint8

const (
	pcrMaskCAOutputMode viaPCROutputMasks = 0x0E
	pcrMaskCBOutputMode viaPCROutputMasks = 0xE0
)

type viaPCROutputModes uint8

const (
	pcrCA2OutputModeHandshake viaPCROutputModes = 0x08
	pcrCA2OutputModePulse     viaPCROutputModes = 0x0A
	pcrCA2OutputModeFix       viaPCROutputModes = 0x0C
	pcrCB2OutputModeHandshake viaPCROutputModes = 0x80
	pcrCB2OutputModePulse     viaPCROutputModes = 0xA0
	pcrCB2OutputModeFix       viaPCROutputModes = 0xC0
)
