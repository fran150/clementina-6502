package via

type viaACRLatchingMasks uint8

const (
	acrMaskLatchingEnabledA viaACRLatchingMasks = 0x01
	acrMaskLatchingEnabledB viaACRLatchingMasks = 0x02
)

type viaAuxiliaryControlRegister uint8

func (acr *viaAuxiliaryControlRegister) isLatchingEnabledForSide(side *ViaSide) bool {
	return uint8(*acr)&uint8(side.configuration.latchingEnabledMasks) > 0x00
}
