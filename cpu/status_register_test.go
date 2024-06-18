package cpu

import "testing"

func TestGetFlag(t *testing.T) {
	status := StatusRegister(0x80)

	if !status.Flag(NegativeFlagBit) {
		t.Errorf("Negative was expected set")
	}

	status.SetFlag(ZeroFlagBit, true)

	if uint8(status) != 0x82 {
		t.Error("A value of 0x82 was expected in the status register")
	}

	if !status.Flag(ZeroFlagBit) || !status.Flag(NegativeFlagBit) {
		t.Errorf("Zero and Negative Flags were expected set")
	}

	status.SetFlag(ZeroFlagBit, false)

	if uint8(status) != 0x80 {
		t.Error("A value of 0x80 was expected in the status register")
	}

	if status.Flag(ZeroFlagBit) || !status.Flag(NegativeFlagBit) {
		t.Errorf("Zero flag was expected unset and Negative flags was expected set")
	}

}
