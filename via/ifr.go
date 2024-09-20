package via

type ViaIFR struct {
	value uint8

	interruptEnable *uint8
}

func (ifr *ViaIFR) getValue() uint8 {
	return ifr.value
}

// If any of the bits 0 - 6 in the IFR is 1 then the bit 7 is 1
// If not, then the bit 7 is 0.
func (ifr *ViaIFR) setValue(value uint8) {
	if (value & *ifr.interruptEnable & 0x7F) > 0 {
		value |= 0x80
	} else {
		value &= 0x7F
	}

	ifr.value = value
}

func (ifr *ViaIFR) setBit(flag viaIRQFlags) {
	ifr.setValue(ifr.value | uint8(flag))
}

func (ifr *ViaIFR) clearBit(flag viaIRQFlags) {
	ifr.setValue(ifr.value & ^uint8(flag))
}
