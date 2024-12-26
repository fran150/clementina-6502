package via

// The interrupt flags register (IFR) contains information that can be
// used to determine the source of an interruption.
type viaIFR struct {
	value           uint8 // Current value of the register
	interruptEnable uint8 // Each bit of the Interrupt Enable Register controls if a flag of the IFR will cause an IRQ
}

// Returns the IFR value
func (ifr *viaIFR) getInterruptFlagValue() uint8 {
	return ifr.value
}

// Sets the value of the IFR
// If any of the bits 0 - 6 in the IFR is 1 then the bit 7 is 1
// If not, then the bit 7 is 0.
func (ifr *viaIFR) setInterruptFlagValue(value uint8) {
	if (value & ifr.interruptEnable & 0x7F) > 0 {
		value |= 0x80
	} else {
		value &= 0x7F
	}

	ifr.value = value
}

// Sets a specific bit of the IFR
func (ifr *viaIFR) setInterruptFlagBit(flag viaIRQFlags) {
	ifr.setInterruptFlagValue(ifr.value | uint8(flag))
}

// Clears a specific bit of the IFR
func (ifr *viaIFR) clearInterruptFlagBit(flag viaIRQFlags) {
	ifr.setInterruptFlagValue(ifr.value & ^uint8(flag))
}

// The processor can read the contents of this register by placing the proper address
// on the register select and chip select inputs with the R/W line high. Bit 7 will
// read as a logic 0.
func (ifr *viaIFR) getInterruptEnabledFlag() uint8 {
	return ifr.interruptEnable & 0x7F
}

// If bit 7 of the data placed on the system data bus during this write operation is a 0,
// each 1 in bits 6 through 0 clears the corresponding bit in the Interrupt Enable Register.
// Setting selected bits in the Interrupt Enable Register is accomplished by writing to
// the same address with bit 7 in the data word set to a logic 1.
// In this case, each 1 in bits 6 through 0 will set the corresponding bit. For each zero,
// the corresponding bit will be unaffected. T
func (ifr *viaIFR) setInterruptEnabledFlag(value uint8) {
	mustSet := (value & 0x80) > 0
	value = value & 0x7F

	if mustSet {
		ifr.interruptEnable |= value
	} else {
		ifr.interruptEnable &= ^value
	}
}

// Returns true if bit 7 of the IFR is set meaning that an IRQ should be triggered
func (ifr *viaIFR) isInterruptTriggered() bool {
	return ifr.value&0x80 > 0
}
