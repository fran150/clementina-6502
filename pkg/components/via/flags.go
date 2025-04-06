package via

// Auxiliary control masks for bits that controls if latching is enabled or not
type viaACRLatchingMasks uint8

const (
	acrMaskLatchingEnabledA viaACRLatchingMasks = 0x01 // Latching for port A is enabled
	acrMaskLatchingEnabledB viaACRLatchingMasks = 0x02 // Latching for port B is enabled
)

// Mask used to get the timer control mode from the ACR
type viaTimerControlMask uint8

const (
	acrT1ControlRunModeMask viaTimerControlMask = 0x40 // Mask used to get the run mode from the T1
	acrT1ControlOutputMask  viaTimerControlMask = 0x80 // Mask used to get the output mode from T1
	acrT2ControlRunModeMask viaTimerControlMask = 0x20 // Mask used to get the run mode from the T2
)

// Bits used to indetify the timer control mode from the ACR
type viaTimerRunningMode uint8

const (
	acrTxRunModeOneShot       viaTimerRunningMode = 0x00 // If run mode bits are zero for any mask (T1 or T2) then chip is in one-shot mode
	acrT1RunModeFree          viaTimerRunningMode = 0x40 // Bits used to determine free run mode on T1
	acrT2RunModePulseCounting viaTimerRunningMode = 0x20 // Bits used to determine pulse counting mode on T2
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

// All the different modes that output for control lines can be configured.
// This are used with the corresponding viaPCROutputMasks to set the PCR values
type viaPCROutputModes uint8

const (
	pcrCA2OutputModeHandshake viaPCROutputModes = 0x08 // Handshake mode for CA2
	pcrCA2OutputModePulse     viaPCROutputModes = 0x0A // Pulse mode for CA2
	pcrCA2OutputModeFixLow    viaPCROutputModes = 0x0C // Fixed Low output for CA2
	pcrCA2OutputModeFixHigh   viaPCROutputModes = 0x0E // Fixed High output for CA2
	pcrCB2OutputModeHandshake viaPCROutputModes = 0x80 // Handshake mode for CB2
	pcrCB2OutputModePulse     viaPCROutputModes = 0xA0 // Pulse mode for CB2
	pcrCB2OutputModeFixLow    viaPCROutputModes = 0xC0 // Fixed Low mode for CB2
	pcrCB2OutputModeFixHigh   viaPCROutputModes = 0xE0 // Fixed High mode for CB2
)
