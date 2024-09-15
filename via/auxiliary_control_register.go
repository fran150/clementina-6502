package via

type viaACRLatchingMasks uint8

const (
	acrMaskLatchingA viaACRLatchingMasks = 0x01
	acrMaskLatchingB viaACRLatchingMasks = 0x02
)

type ViaAuxiliaryControlRegister uint8

func (acr *ViaAuxiliaryControlRegister) isLatchingEnabled(mask viaACRLatchingMasks) bool {
	return uint8(*acr)&uint8(mask) > 0x00
}
