package via

type ViaIFR struct {
	value           uint8
	interruptEnable uint8
}

func (ifr *ViaIFR) getInterruptFlagValue() uint8 {
	return ifr.value
}

// If any of the bits 0 - 6 in the IFR is 1 then the bit 7 is 1
// If not, then the bit 7 is 0.
func (ifr *ViaIFR) setInterruptFlagValue(value uint8) {
	if (value & ifr.interruptEnable & 0x7F) > 0 {
		value |= 0x80
	} else {
		value &= 0x7F
	}

	ifr.value = value
}

func (ifr *ViaIFR) setInterruptFlagBit(flag viaIRQFlags) {
	ifr.setInterruptFlagValue(ifr.value | uint8(flag))
}

func (ifr *ViaIFR) clearInterruptFlagBit(flag viaIRQFlags) {
	ifr.setInterruptFlagValue(ifr.value & ^uint8(flag))
}

// The processor can read the contents of this register by placing the proper address
// on the register select and chip select inputs with the R/W line high. Bit 7 will
// read as a logic 0.
func (ifr *ViaIFR) getInterruptEnabledFlag() uint8 {
	return ifr.interruptEnable & 0x7F
}

// If bit 7 of the data placed on the system data bus during this write operation is a 0,
// each 1 in bits 6 through 0 clears the corresponding bit in the Interrupt Enable Register.
// Setting selected bits in the Interrupt Enable Register is accomplished by writing to
// the same address with bit 7 in the data word set to a logic 1.
// In this case, each 1 in bits 6 through 0 will set the corresponding bit. For each zero,
// the corresponding bit will be unaffected. T
func (ifr *ViaIFR) setInterruptEnabledFlag(value uint8) {
	mustSet := (value & 0x80) > 0
	value = value & 0x7F

	if mustSet {
		ifr.interruptEnable |= value
	} else {
		ifr.interruptEnable &= ^value
	}
}

func (ifr *ViaIFR) isInterruptTriggered() bool {
	return ifr.value&0x80 > 0
}
