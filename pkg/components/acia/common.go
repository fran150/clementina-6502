package acia

// Returns if the bit is enabled on the record
func isBitSet(register uint8, bit uint8) bool {
	return (register & bit) == bit
}

// Sets the bit to the the desired state in the specified record
func setRegisterBit(register *uint8, bit uint8, status bool) {
	if status {
		*register |= bit
	} else {
		*register &= ^bit
	}
}
