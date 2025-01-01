package terminal

import (
	"fmt"
	"strings"

	"github.com/fran150/clementina6502/pkg/components/cpu"
	"github.com/fran150/clementina6502/pkg/components/memory"
)

func DrawInstruction(processor *cpu.Cpu65C02S, ram *memory.Ram, address uint16, instruction *cpu.CpuInstructionData) string {
	addressModeDetails := cpu.GetAddressMode(instruction.AddressMode())

	size := addressModeDetails.MemSize() - 1

	sb := strings.Builder{}

	// Write current address
	sb.WriteString(fmt.Sprintf("$%04X: %s ", address, instruction.Mnemonic()))

	address = address & 0x7FFF

	// Write operands
	switch size {
	case 0:
	case 1:
		sb.WriteString(fmt.Sprintf(addressModeDetails.Format(), ram.Peek(address+1)))
	case 2:
		msb := uint16(ram.Peek(address+2)) << 8
		lsb := uint16(ram.Peek(address + 1))
		sb.WriteString(fmt.Sprintf(addressModeDetails.Format(), msb|lsb))
	default:
		sb.WriteString("Unrecognized Instruction or Address Mode")
	}

	sb.WriteString("\r\n")

	return sb.String()
}
