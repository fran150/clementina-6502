package via

// Auxiliary control masks for bits that controls if latching is enabled or not
type viaACRLatchingMasks uint8

const (
	acrMaskLatchingEnabledA viaACRLatchingMasks = 0x01 // Latching for port A is enabled
	acrMaskLatchingEnabledB viaACRLatchingMasks = 0x02 // Latching for port B is enabled
)

// Peripheral control register masks for bits that controls if the chip actions on a positive
// or negative edge transition of each line
type viaPCRTransitionMasks uint8

const (
	pcrMaskCA1TransitionType viaPCRTransitionMasks = 0x01 // Transition configuration for CA1
	pcrMaskCA2TransitionType viaPCRTransitionMasks = 0x0C // Transition configuration for CA2
	pcrMaskCB1TransitionType viaPCRTransitionMasks = 0x10 // Transition configuration for CB1
	pcrMaskCB2TransitionType viaPCRTransitionMasks = 0xC0 // Transition configuration for CB2
)

// Peripheral control register masks for bits that controls if the IRQ flag must be cleared
// when reading or writing ORA/ORB.
// In the manual it describes this as "independent interrupt"
type viaPCRInterruptClearMasks uint8

const (
	pcrMaskCA2ClearOnRW viaPCRInterruptClearMasks = 0x02 // Bit that controls if clear IRQ when R/W CA2
	pcrMaskCB2ClearOnRW viaPCRInterruptClearMasks = 0x20 // Bit that controls if clear IRQ when R/W CB2
)

// Peripheral control register masks for bits that controls output modes of the control lines
type viaPCROutputMasks uint8

const (
	pcrMaskCAOutputMode viaPCROutputMasks = 0x0E // Mask to control output modes for CA lines
	pcrMaskCBOutputMode viaPCROutputMasks = 0xE0 // Mask to control output modes for CB lines
)

// All the diferent modes that output for control lines can be configured.
// This are used with the corresponding viaPCROutputMasks to set the PCR values
type viaPCROutputModes uint8

const (
	pcrCA2OutputModeHandshake viaPCROutputModes = 0x08 // Handshake mode for CA2
	pcrCA2OutputModePulse     viaPCROutputModes = 0x0A // Pulse mode for CA2
	pcrCA2OutputModeFix       viaPCROutputModes = 0x0C // Fixed output for CA2
	pcrCB2OutputModeHandshake viaPCROutputModes = 0x80 // Handshake mode for CB2
	pcrCB2OutputModePulse     viaPCROutputModes = 0xA0 // Pulse mode for CB2
	pcrCB2OutputModeFix       viaPCROutputModes = 0xC0 // Fixed mode for CB2
)
