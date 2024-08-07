package cpu

import (
	"strings"
	"testing"
	"unicode"

	"github.com/fran150/clementina6502/memory"
)

func runInstructionTest(cpu *Cpu65C02S, ram *memory.Ram, cycles uint64) {
	for i := range cycles {
		cpu.Tick(i)
		ram.Tick(i)

		cpu.PostTick(i)
	}
}

func evaluateRegisterValue(t *testing.T, cpu *Cpu65C02S, name string, value uint8, expected uint8) {
	if value != expected {
		instruction := cpu.instructionSet.GetByOpCode(cpu.currentOpCode)
		addressMode := cpu.addressModeSet.GetByName(instruction.addressMode)

		t.Errorf("%s - %s - Current value of %s (%02X) doesnt match the expected value of (%02X)", instruction.Mnemonic(), addressMode.Text(), name, value, expected)
	}
}

func evaluateAddress(t *testing.T, cpu *Cpu65C02S, ram *memory.Ram, address uint16, expected uint8) {
	value := ram.Peek(address)

	if value != expected {
		instruction := cpu.instructionSet.GetByOpCode(cpu.currentOpCode)
		addressMode := cpu.addressModeSet.GetByName(instruction.addressMode)

		t.Errorf("%s - %s - Current value (%02X) of addres %04X doesnt match the expected value of (%02X)", instruction.Mnemonic(), addressMode.Text(), value, address, expected)
	}
}

func evaluateFlag(t *testing.T, cpu *Cpu65C02S, flagString string) {
	const flags string = "czidb-vn"

	instruction := cpu.instructionSet.GetByOpCode(cpu.currentOpCode)
	addressMode := cpu.addressModeSet.GetByName(instruction.addressMode)

	for i, flag := range flags {
		ucFlag := unicode.ToUpper(flag)
		lcFlag := unicode.ToLower(flag)

		if strings.Contains(flagString, string(ucFlag)) {
			if !cpu.processorStatusRegister.Flag(StatusBit(i)) {
				t.Errorf("%s - %s - Expected %s flag to be set", instruction.Mnemonic(), addressMode.Text(), string(ucFlag))
			}
		}

		if strings.Contains(flagString, string(lcFlag)) {
			if cpu.processorStatusRegister.Flag(StatusBit(i)) {
				t.Errorf("%s - %s - Expected %s flag NOT to be set", instruction.Mnemonic(), addressMode.Text(), string(ucFlag))
			}
		}
	}
}

func evaluateProgramCounter(t *testing.T, cpu *Cpu65C02S, expectedValue uint16) {
	if cpu.programCounter != expectedValue {
		instruction := cpu.instructionSet.GetByOpCode(cpu.currentOpCode)
		addressMode := cpu.addressModeSet.GetByName(instruction.addressMode)

		t.Errorf("%s - %s - Current value (%04X) of PC doesnt match the expected value of (%04X)", instruction.Mnemonic(), addressMode.Text(), cpu.programCounter, expectedValue)
	}
}

func evaluateAccumulatorInstruction(t *testing.T, cpu *Cpu65C02S, ram *memory.Ram, cycles uint64, flagString string, expectedAccumulatorValue uint8) {
	runInstructionTest(cpu, ram, cycles)
	evaluateRegisterValue(t, cpu, "accumulator", cpu.accumulatorRegister, expectedAccumulatorValue)
	evaluateFlag(t, cpu, flagString)
}

func evaluateXRegisterInstruction(t *testing.T, cpu *Cpu65C02S, ram *memory.Ram, cycles uint64, flagString string, expectedRegisterValue uint8) {
	runInstructionTest(cpu, ram, cycles)
	evaluateRegisterValue(t, cpu, "X Register", cpu.xRegister, expectedRegisterValue)
	evaluateFlag(t, cpu, flagString)
}

func evaluateYRegisterInstruction(t *testing.T, cpu *Cpu65C02S, ram *memory.Ram, cycles uint64, flagString string, expectedRegisterValue uint8) {
	runInstructionTest(cpu, ram, cycles)
	evaluateRegisterValue(t, cpu, "Y Register", cpu.yRegister, expectedRegisterValue)
	evaluateFlag(t, cpu, flagString)
}

func evaluateStackPointerInstruction(t *testing.T, cpu *Cpu65C02S, ram *memory.Ram, cycles uint64, flagString string, expectedRegisterValue uint8) {
	runInstructionTest(cpu, ram, cycles)
	evaluateRegisterValue(t, cpu, "Stack Pointer", cpu.stackPointer, expectedRegisterValue)
	evaluateFlag(t, cpu, flagString)
}

func evaluateRMWInstruction(t *testing.T, cpu *Cpu65C02S, ram *memory.Ram, cycles uint64, flagString string, address uint16, expectedValue uint8) {
	runInstructionTest(cpu, ram, cycles)
	evaluateAddress(t, cpu, ram, address, expectedValue)
	evaluateFlag(t, cpu, flagString)
}

func evaluateBranchInstruction(t *testing.T, cpu *Cpu65C02S, ram *memory.Ram, cycles uint64, flagString string, expectedProgramCounterValue uint16) {
	runInstructionTest(cpu, ram, cycles)
	evaluateProgramCounter(t, cpu, expectedProgramCounterValue)
	evaluateFlag(t, cpu, flagString)
}

func evaluateFlagInstruction(t *testing.T, cpu *Cpu65C02S, ram *memory.Ram, cycles uint64, flagString string) {
	runInstructionTest(cpu, ram, cycles)
	evaluateFlag(t, cpu, flagString)
}

func TestActionADC(t *testing.T) {
	cpu, ram := createComputer()

	cpu.xRegister = 0x05
	cpu.yRegister = 0x0A

	ram.Poke(0x0010, 0x02) // zp value $02
	ram.Poke(0x0015, 0xA0) // zp,x value $A0

	ram.Poke(0x00A5, 0x10) // (zp,x) redirect to $D110
	ram.Poke(0x00A6, 0xD1)

	ram.Poke(0x00B0, 0xFF) // (zp),y redirect to $D2FF
	ram.Poke(0x00B1, 0xD2)

	ram.Poke(0x00C0, 0x00) // (zp) redict to $E000
	ram.Poke(0x00C1, 0xE0)

	ram.Poke(0xD000, 0x10) // a value $10
	ram.Poke(0xD005, 0x11) // a,x value $11
	ram.Poke(0xD00A, 0x20) // a,y value $20

	ram.Poke(0xD110, 0x02) // (zp,x) value $02

	ram.Poke(0xD309, 0x10) // (zp),y value $10

	ram.Poke(0xE000, 0x01) // (zp) value $01

	// A = 0
	ram.Poke(0xC000, 0x69) // ADC #$0A ->  A + 0A = 0A
	ram.Poke(0xC001, 0x0A)
	ram.Poke(0xC002, 0x65) // ADC $10 -> A + 02 = 0C
	ram.Poke(0xC003, 0x10)
	ram.Poke(0xC004, 0x75) // ADC $10,X -> A + A0 = AC
	ram.Poke(0xC005, 0x10)
	ram.Poke(0xC006, 0x6D) // ADC $D000 -> A + 10 = BC
	ram.Poke(0xC007, 0x00)
	ram.Poke(0xC008, 0xD0)
	ram.Poke(0xC009, 0x7D) // ADC $D000,X -> A + 11 = CD
	ram.Poke(0xC00A, 0x00)
	ram.Poke(0xC00B, 0xD0)
	ram.Poke(0xC00C, 0x79) // ADC $D000,Y -> A + 20 = ED
	ram.Poke(0xC00D, 0x00)
	ram.Poke(0xC00E, 0xD0)
	ram.Poke(0xC00F, 0x61) // ADC ($A0,X) -> A + 02 = EF
	ram.Poke(0xC010, 0xA0)
	ram.Poke(0xC011, 0x71) // ADC ($B0),Y -> A + 10 = FF
	ram.Poke(0xC012, 0xB0)
	ram.Poke(0xC013, 0x72) // ADC ($C0) -> A + 01 = 00
	ram.Poke(0xC014, 0xC0)

	evaluateAccumulatorInstruction(t, cpu, ram, 2, "znc", 0x0A) // i -> 0 + 0A = 0A
	evaluateAccumulatorInstruction(t, cpu, ram, 3, "znc", 0x0C) // zp -> 0A + 02 = 0C
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zNc", 0xAC) // zp,x -> 0C + A0 = AC
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zNc", 0xBC) // a -> AC + 10 = BC
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zNc", 0xCD) // a,x -> BC + 11 = CD
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zNc", 0xED) // a,y -> CD + 20 = ED
	evaluateAccumulatorInstruction(t, cpu, ram, 6, "zNc", 0xEF) // (zp,x) -> ED + 02 = EF
	evaluateAccumulatorInstruction(t, cpu, ram, 6, "zNc", 0xFF) // (zp),y -> EF + 10 = FF
	evaluateAccumulatorInstruction(t, cpu, ram, 5, "ZnC", 0x00) // (zp) -> FF + 1 = 00
}

func TestActionAND(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xFF
	cpu.xRegister = 0x05
	cpu.yRegister = 0x0A

	ram.Poke(0x0010, 0xFD) // zp value $FD
	ram.Poke(0x0015, 0xFB) // zp,x value $FB

	ram.Poke(0x00A5, 0x10) // (zp,x) redirect to $D110
	ram.Poke(0x00A6, 0xD1)

	ram.Poke(0x00B0, 0xFF) // (zp),y redirect to $D2FF
	ram.Poke(0x00B1, 0xD2)

	ram.Poke(0x00C0, 0x00) // (zp) redict to $E000
	ram.Poke(0x00C1, 0xE0)

	ram.Poke(0xD000, 0xF7) // a value $F7
	ram.Poke(0xD005, 0xEF) // a,x value $EF
	ram.Poke(0xD00A, 0xDF) // a,y value $DF

	ram.Poke(0xD110, 0x7F) // (zp,x) value $7F

	ram.Poke(0xD309, 0xBF) // (zp),y value $BF

	ram.Poke(0xE000, 0x08) // (zp) value $08

	// A = FF
	ram.Poke(0xC000, 0x29) // AND #$FE -> A & FE = FE
	ram.Poke(0xC001, 0xFE)
	ram.Poke(0xC002, 0x25) // AND $10 -> A & FD = FC
	ram.Poke(0xC003, 0x10)
	ram.Poke(0xC004, 0x35) // AND $10,X -> A & FB = F8
	ram.Poke(0xC005, 0x10)
	ram.Poke(0xC006, 0x2D) // AND $D000 -> A & F7 = F0
	ram.Poke(0xC007, 0x00)
	ram.Poke(0xC008, 0xD0)
	ram.Poke(0xC009, 0x3D) // AND $D000,X -> A & EF = E0
	ram.Poke(0xC00A, 0x00)
	ram.Poke(0xC00B, 0xD0)
	ram.Poke(0xC00C, 0x39) // AND $D000,Y -> A & DF = C0
	ram.Poke(0xC00D, 0x00)
	ram.Poke(0xC00E, 0xD0)
	ram.Poke(0xC00F, 0x21) // AND ($A0,X) -> A & 7F = 40
	ram.Poke(0xC010, 0xA0)
	ram.Poke(0xC011, 0x31) // AND ($B0),Y -> A & BF = 00
	ram.Poke(0xC012, 0xB0)
	ram.Poke(0xC013, 0x32) // AND ($C0) -> A & 08 = 08
	ram.Poke(0xC014, 0xC0)

	evaluateAccumulatorInstruction(t, cpu, ram, 2, "zN", 0xFE) // i -> FF & FE = FE
	evaluateAccumulatorInstruction(t, cpu, ram, 3, "zN", 0xFC) // zp -> FE & FC = FC
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zN", 0xF8) // zp,x -> FC & FB = F8
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zN", 0xF0) // a -> F8 & F7 = F0
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zN", 0xE0) // a,x -> F0 & EF = E0
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zN", 0xC0) // a,y -> E0 & DF = C0
	evaluateAccumulatorInstruction(t, cpu, ram, 6, "zn", 0x40) // (zp,x) -> C0 & 7F = 40
	evaluateAccumulatorInstruction(t, cpu, ram, 6, "Zn", 0x00) // (zp),y -> 40 & BF = 00
	cpu.accumulatorRegister = 0xFF
	evaluateAccumulatorInstruction(t, cpu, ram, 5, "zn", 0x08) // (zp) -> FF & 08 = 08
}

func TestActionASL(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xAA
	cpu.xRegister = 0x05

	ram.Poke(0x0010, 0x08) // zp value $08
	ram.Poke(0x0015, 0x04) // zp,x value $04

	ram.Poke(0xD000, 0x40) // a value $40
	ram.Poke(0xD005, 0x80) // a,x value $80

	// A = AA
	ram.Poke(0xC000, 0x0A) // ASL a <- AA
	ram.Poke(0xC001, 0x06) // ASL $10 <- 08
	ram.Poke(0xC002, 0x10)
	ram.Poke(0xC003, 0x16) // ASL $10,X <- 04
	ram.Poke(0xC004, 0x10)
	ram.Poke(0xC005, 0x0E) // ASL $D000 <- 40
	ram.Poke(0xC006, 0x00)
	ram.Poke(0xC007, 0xD0)
	ram.Poke(0xC008, 0x1E) // ASL $D000,X <- 80
	ram.Poke(0xC009, 0x00)
	ram.Poke(0xC00A, 0xD0)

	evaluateAccumulatorInstruction(t, cpu, ram, 2, "znC", 0x54) // i -> AA << 1 = 55
	evaluateRMWInstruction(t, cpu, ram, 5, "znc", 0x0010, 0x10) // zp -> 08 << 1 = 11
	evaluateRMWInstruction(t, cpu, ram, 6, "znc", 0x0015, 0x08) // zp,x -> 04 << 1 = 08
	evaluateRMWInstruction(t, cpu, ram, 6, "zNc", 0xD000, 0x80) // a -> 40 << 1 = 80
	evaluateRMWInstruction(t, cpu, ram, 7, "ZnC", 0xD005, 0x00) // a,x -> 80 << 1 = 00
}

func TestActionBCC(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0x90) // BCC $10
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC012, 0x90) // BCC $DF (backwards)
	ram.Poke(0xC013, 0xDF)

	ram.Poke(0xBFF3, 0x90) // BCC $FF (not taken)
	ram.Poke(0xBFF4, 0xFF)
	ram.Poke(0xBFF5, 0xEA) // NOP

	evaluateBranchInstruction(t, cpu, ram, 3, "c", 0xC012)
	evaluateBranchInstruction(t, cpu, ram, 4, "c", 0xBFF3)
	cpu.processorStatusRegister.SetFlag(CarryFlagBit, true)
	evaluateBranchInstruction(t, cpu, ram, 2, "C", 0xBFF5)
	evaluateBranchInstruction(t, cpu, ram, 2, "z", 0xBFF6)
}

func TestActionBCS(t *testing.T) {
	cpu, ram := createComputer()

	cpu.processorStatusRegister.SetFlag(CarryFlagBit, true)

	ram.Poke(0xC000, 0xB0) // BCS $10
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC012, 0xB0) // BCS $DF (backwards)
	ram.Poke(0xC013, 0xDF)

	ram.Poke(0xBFF3, 0xB0) // BCS $FF (not taken)
	ram.Poke(0xBFF4, 0xFF)
	ram.Poke(0xBFF5, 0xEA) // NOP

	evaluateBranchInstruction(t, cpu, ram, 3, "C", 0xC012)
	evaluateBranchInstruction(t, cpu, ram, 4, "C", 0xBFF3)
	cpu.processorStatusRegister.SetFlag(CarryFlagBit, false)
	evaluateBranchInstruction(t, cpu, ram, 2, "c", 0xBFF5)
	evaluateBranchInstruction(t, cpu, ram, 2, "z", 0xBFF6)
}

func TestActionBEQ(t *testing.T) {
	cpu, ram := createComputer()

	cpu.processorStatusRegister.SetFlag(ZeroFlagBit, true)

	ram.Poke(0xC000, 0xF0) // BEQ $10
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC012, 0xF0) // BEQ $DF (backwards)
	ram.Poke(0xC013, 0xDF)

	ram.Poke(0xBFF3, 0xF0) // BEQ $FF (not taken)
	ram.Poke(0xBFF4, 0xFF)
	ram.Poke(0xBFF5, 0xEA) // NOP

	evaluateBranchInstruction(t, cpu, ram, 3, "Z", 0xC012)
	evaluateBranchInstruction(t, cpu, ram, 4, "Z", 0xBFF3)
	cpu.processorStatusRegister.SetFlag(ZeroFlagBit, false)
	evaluateBranchInstruction(t, cpu, ram, 2, "z", 0xBFF5)
	evaluateBranchInstruction(t, cpu, ram, 2, "z", 0xBFF6)
}

func TestActionBIT(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xFF
	cpu.xRegister = 0x05

	ram.Poke(0x0010, 0x7F) // zp value $7F
	ram.Poke(0x0015, 0xBF) // zp,x value $BF

	ram.Poke(0xD000, 0x00) // a value $00
	ram.Poke(0xD005, 0x3F) // a,x value $EF

	// A = FF
	ram.Poke(0xC000, 0x89) // BIT #$FE -> FF & FE = FE
	ram.Poke(0xC001, 0xFE)
	ram.Poke(0xC002, 0x24) // BIT $10 -> FF & 7F = 7F
	ram.Poke(0xC003, 0x10)
	ram.Poke(0xC004, 0x34) // BIT $10,X -> FF & BF = BF
	ram.Poke(0xC005, 0x10)
	ram.Poke(0xC006, 0x2C) // AND $D000 -> FF & 00 = 00
	ram.Poke(0xC007, 0x00)
	ram.Poke(0xC008, 0xD0)
	ram.Poke(0xC009, 0x3C) // AND $D000,X -> F8 & 3F = 38
	ram.Poke(0xC00A, 0x00)
	ram.Poke(0xC00B, 0xD0)

	evaluateAccumulatorInstruction(t, cpu, ram, 2, "zNV", 0xFF) // i -> FF & FE = FE
	evaluateAccumulatorInstruction(t, cpu, ram, 3, "znV", 0xFF) // zp -> FF & 7F = 7F
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zNv", 0xFF) // zp,x -> FF & BF = BF
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "Znv", 0xFF) // a -> FF & 00 = 00
	cpu.accumulatorRegister = 0xF8
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "znv", 0xF8) // a,x -> F8 & 3F = 38
}

func TestActionBMI(t *testing.T) {
	cpu, ram := createComputer()

	cpu.processorStatusRegister.SetFlag(NegativeFlagBit, true)

	ram.Poke(0xC000, 0x30) // BMI $10
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC012, 0x30) // BMI $DF (backwards)
	ram.Poke(0xC013, 0xDF)

	ram.Poke(0xBFF3, 0x30) // BMI $FF (not taken)
	ram.Poke(0xBFF4, 0xFF)
	ram.Poke(0xBFF5, 0xEA) // NOP

	evaluateBranchInstruction(t, cpu, ram, 3, "N", 0xC012)
	evaluateBranchInstruction(t, cpu, ram, 4, "N", 0xBFF3)
	cpu.processorStatusRegister.SetFlag(NegativeFlagBit, false)
	evaluateBranchInstruction(t, cpu, ram, 2, "n", 0xBFF5)
	evaluateBranchInstruction(t, cpu, ram, 2, "n", 0xBFF6)
}

func TestActionBNE(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0xD0) // BNE $10
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC012, 0xD0) // BNE $F0
	ram.Poke(0xC013, 0xDF)

	ram.Poke(0xBFF3, 0xD0) // BNE $FF (not taken)
	ram.Poke(0xBFF4, 0xFF)
	ram.Poke(0xBFF5, 0xEA) // NOP

	evaluateBranchInstruction(t, cpu, ram, 3, "z", 0xC012)
	evaluateBranchInstruction(t, cpu, ram, 4, "z", 0xBFF3)
	cpu.processorStatusRegister.SetFlag(ZeroFlagBit, true)
	evaluateBranchInstruction(t, cpu, ram, 2, "Z", 0xBFF5)
	evaluateBranchInstruction(t, cpu, ram, 2, "Z", 0xBFF6)
}

func TestActionBPL(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0x10) // BPL $10
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC012, 0x10) // BPL $F0
	ram.Poke(0xC013, 0xDF)

	ram.Poke(0xBFF3, 0x10) // BPL $FF (not taken)
	ram.Poke(0xBFF4, 0xFF)
	ram.Poke(0xBFF5, 0xEA) // NOP

	evaluateBranchInstruction(t, cpu, ram, 3, "n", 0xC012)
	evaluateBranchInstruction(t, cpu, ram, 4, "n", 0xBFF3)
	cpu.processorStatusRegister.SetFlag(NegativeFlagBit, true)
	evaluateBranchInstruction(t, cpu, ram, 2, "N", 0xBFF5)
	evaluateBranchInstruction(t, cpu, ram, 2, "N", 0xBFF6)
}

func TestActionBRA(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0x80) // BRA $10
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC012, 0x80) // BRA $F0
	ram.Poke(0xC013, 0xDF)

	evaluateBranchInstruction(t, cpu, ram, 3, "n", 0xC012)
	evaluateBranchInstruction(t, cpu, ram, 4, "n", 0xBFF3)
}

func TestActionBRKandRTI(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xFFFE, 0x00)
	ram.Poke(0xFFFF, 0xD0)

	ram.Poke(0xD000, 0xA9) // LDA #$FF
	ram.Poke(0xD001, 0xFF)
	ram.Poke(0xD002, 0x40) // RTI

	ram.Poke(0xC000, 0x00) // BRK (takes 2 bytes even if the second is not used)
	ram.Poke(0xC001, 0x00) //
	ram.Poke(0xC002, 0xA9) // LDA #$77
	ram.Poke(0xC003, 0x77)

	evaluateBranchInstruction(t, cpu, ram, 7, "I", 0xD000)        // Executes BRK
	evaluateAddress(t, cpu, ram, 0x01FB, 0x34)                    // Validates Stack address
	evaluateAccumulatorInstruction(t, cpu, ram, 2, "INvzc", 0xFF) // Executes LDA
	evaluateBranchInstruction(t, cpu, ram, 6, "I", 0xC002)        // Executes RTI
	evaluateAccumulatorInstruction(t, cpu, ram, 2, "Invzc", 0x77) // Executes LDA
}

func TestActionBVC(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0x50) // BVC $10
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC012, 0x50) // BVC $F0
	ram.Poke(0xC013, 0xDF)

	ram.Poke(0xBFF3, 0x50) // BVC $FF (not taken)
	ram.Poke(0xBFF4, 0xFF)
	ram.Poke(0xBFF5, 0xEA) // NOP

	evaluateBranchInstruction(t, cpu, ram, 3, "v", 0xC012)
	evaluateBranchInstruction(t, cpu, ram, 4, "v", 0xBFF3)
	cpu.processorStatusRegister.SetFlag(OverflowFlagBit, true)
	evaluateBranchInstruction(t, cpu, ram, 2, "V", 0xBFF5)
	evaluateBranchInstruction(t, cpu, ram, 2, "V", 0xBFF6)
}

func TestActionBVS(t *testing.T) {
	cpu, ram := createComputer()

	cpu.processorStatusRegister.SetFlag(OverflowFlagBit, true)

	ram.Poke(0xC000, 0x70) // BVS $10
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC012, 0x70) // BVS $F0
	ram.Poke(0xC013, 0xDF)

	ram.Poke(0xBFF3, 0x70) // BVS $FF (not taken)
	ram.Poke(0xBFF4, 0xFF)
	ram.Poke(0xBFF5, 0xEA) // NOP

	evaluateBranchInstruction(t, cpu, ram, 3, "V", 0xC012)
	evaluateBranchInstruction(t, cpu, ram, 4, "V", 0xBFF3)
	cpu.processorStatusRegister.SetFlag(OverflowFlagBit, false)
	evaluateBranchInstruction(t, cpu, ram, 2, "v", 0xBFF5)
	evaluateBranchInstruction(t, cpu, ram, 2, "v", 0xBFF6)
}

func TestActionCLC(t *testing.T) {
	cpu, ram := createComputer()

	cpu.processorStatusRegister.SetFlag(CarryFlagBit, true)

	ram.Poke(0xC000, 0x18) // CLC
	ram.Poke(0xC001, 0xEA) // NOP

	evaluateFlagInstruction(t, cpu, ram, 2, "c")
}

func TestActionCLD(t *testing.T) {
	cpu, ram := createComputer()

	cpu.processorStatusRegister.SetFlag(DecimalModeFlagBit, true)

	ram.Poke(0xC000, 0xD8) // CLD
	ram.Poke(0xC001, 0xEA) // NOP

	evaluateFlagInstruction(t, cpu, ram, 2, "d")
}

func TestActionCLI(t *testing.T) {
	cpu, ram := createComputer()

	cpu.processorStatusRegister.SetFlag(IrqDisableFlagBit, true)

	ram.Poke(0xC000, 0x58) // CLI
	ram.Poke(0xC001, 0xEA) // NOP

	evaluateFlagInstruction(t, cpu, ram, 2, "i")
}

func TestActionCLV(t *testing.T) {
	cpu, ram := createComputer()

	cpu.processorStatusRegister.SetFlag(OverflowFlagBit, true)

	ram.Poke(0xC000, 0xB8) // CLI
	ram.Poke(0xC001, 0xEA) // NOP

	evaluateFlagInstruction(t, cpu, ram, 2, "v")
}

func TestActionCMP(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0x0F

	cpu.xRegister = 0x05
	cpu.yRegister = 0x0A

	ram.Poke(0x0010, 0x02) // zp value $02
	ram.Poke(0x0015, 0xA0) // zp,x value $A0

	ram.Poke(0x00A5, 0x10) // (zp,x) redirect to $D110
	ram.Poke(0x00A6, 0xD1)

	ram.Poke(0x00B0, 0xFF) // (zp),y redirect to $D2FF
	ram.Poke(0x00B1, 0xD2)

	ram.Poke(0x00C0, 0x00) // (zp) redict to $E000
	ram.Poke(0x00C1, 0xE0)

	ram.Poke(0xD000, 0x10) // a value $10
	ram.Poke(0xD005, 0x0F) // a,x value $0F
	ram.Poke(0xD00A, 0x02) // a,y value $02

	ram.Poke(0xD110, 0xA0) // (zp,x) value $A0

	ram.Poke(0xD309, 0x10) // (zp),y value $10

	ram.Poke(0xE000, 0x0F) // (zp) value $0F

	// A = 0F
	ram.Poke(0xC000, 0xC9) // CMP #$0F
	ram.Poke(0xC001, 0x0F)
	ram.Poke(0xC002, 0xC5) // CMP $10
	ram.Poke(0xC003, 0x10)
	ram.Poke(0xC004, 0xD5) // CMP $10,X
	ram.Poke(0xC005, 0x10)
	ram.Poke(0xC006, 0xCD) // CMP $D000
	ram.Poke(0xC007, 0x00)
	ram.Poke(0xC008, 0xD0)
	ram.Poke(0xC009, 0xDD) // CMP $D000,X
	ram.Poke(0xC00A, 0x00)
	ram.Poke(0xC00B, 0xD0)
	ram.Poke(0xC00C, 0xD9) // ADC $D000,Y
	ram.Poke(0xC00D, 0x00)
	ram.Poke(0xC00E, 0xD0)
	ram.Poke(0xC00F, 0xC1) // ADC ($A0,X)
	ram.Poke(0xC010, 0xA0)
	ram.Poke(0xC011, 0xD1) // ADC ($B0),Y
	ram.Poke(0xC012, 0xB0)
	ram.Poke(0xC013, 0xD2) // ADC ($C0)
	ram.Poke(0xC014, 0xC0)

	evaluateAccumulatorInstruction(t, cpu, ram, 2, "vZnC", 0x0F) // 0F - 0F = 00
	evaluateAccumulatorInstruction(t, cpu, ram, 3, "vznC", 0x0F) // 0F - 02 = 0D
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "vznc", 0x0F) // 0F - A0 = 6F
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "vzNc", 0x0F) // 0F - 10 = FF
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "vZnC", 0x0F) // 0F - 0F = 00
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "vznC", 0x0F) // 0F - 02 = 0D
	evaluateAccumulatorInstruction(t, cpu, ram, 6, "vznc", 0x0F) // 0F - A0 = 6F
	evaluateAccumulatorInstruction(t, cpu, ram, 6, "vzNc", 0x0F) // 0F - 10 = FF
	evaluateAccumulatorInstruction(t, cpu, ram, 5, "vZnC", 0x0F) // 0F - 0F = 00
}

func TestActionCPX(t *testing.T) {
	cpu, ram := createComputer()

	cpu.xRegister = 0x0F

	ram.Poke(0x0010, 0x02) // zp value $02

	ram.Poke(0xD000, 0x10) // a value $10

	// A = 0F
	ram.Poke(0xC000, 0xE0) // CPX #$0F
	ram.Poke(0xC001, 0x0F)
	ram.Poke(0xC002, 0xE4) // CPX $10
	ram.Poke(0xC003, 0x10)
	ram.Poke(0xC004, 0xEC) // CPX $D000
	ram.Poke(0xC005, 0x00)
	ram.Poke(0xC006, 0xD0)

	evaluateAccumulatorInstruction(t, cpu, ram, 2, "vZnC", 0x00) // 0F - 0F = 00
	evaluateAccumulatorInstruction(t, cpu, ram, 3, "vznC", 0x00) // 0F - 02 = 0D
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "vzNc", 0x00) // 0F - 10 = FF
}

func TestActionCPY(t *testing.T) {
	cpu, ram := createComputer()

	cpu.yRegister = 0x0F

	ram.Poke(0x0010, 0x02) // zp value $02

	ram.Poke(0xD000, 0x10) // a value $10

	// A = 0F
	ram.Poke(0xC000, 0xC0) // CPY #$0F
	ram.Poke(0xC001, 0x0F)
	ram.Poke(0xC002, 0xC4) // CPY $10
	ram.Poke(0xC003, 0x10)
	ram.Poke(0xC004, 0xCC) // CPY $D000
	ram.Poke(0xC005, 0x00)
	ram.Poke(0xC006, 0xD0)

	evaluateAccumulatorInstruction(t, cpu, ram, 2, "vZnC", 0x00) // 0F - 0F = 00
	evaluateAccumulatorInstruction(t, cpu, ram, 3, "vznC", 0x00) // 0F - 02 = 0D
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "vzNc", 0x00) // 0F - 10 = FF
}

func TestActionDEC(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xFF
	cpu.xRegister = 0x05

	ram.Poke(0x0010, 0x80) // zp value $80
	ram.Poke(0x0015, 0x01) // zp,x value $01

	ram.Poke(0xD000, 0x00) // a value $00
	ram.Poke(0xD005, 0xF1) // a,x value $F1

	// A = FF
	ram.Poke(0xC000, 0x3A) // DEC a -> FF - 1 = $FD
	ram.Poke(0xC001, 0xC6) // DEC $10 -> $80 - 1 = $7F
	ram.Poke(0xC002, 0x10)
	ram.Poke(0xC003, 0xD6) // DEC $10,X -> $01 - 1 = $00
	ram.Poke(0xC004, 0x10)
	ram.Poke(0xC005, 0xCE) // DEC $D000 -> $00 - 1 = $FF
	ram.Poke(0xC006, 0x00)
	ram.Poke(0xC007, 0xD0)
	ram.Poke(0xC008, 0xDE) // DEC $D000,X -> F1 - 1 = $F0
	ram.Poke(0xC009, 0x00)
	ram.Poke(0xC00A, 0xD0)

	evaluateAccumulatorInstruction(t, cpu, ram, 2, "zN", 0xFE) // a -> $FE - 1 = $FD
	evaluateRMWInstruction(t, cpu, ram, 5, "zn", 0x0010, 0x7F) // zp -> $80 - 1 = $7F
	evaluateRMWInstruction(t, cpu, ram, 6, "Zn", 0x0015, 0x00) // zp,x -> $01 - 1 = $00
	evaluateRMWInstruction(t, cpu, ram, 6, "zN", 0xD000, 0xFF) // a -> $00 - 1 = $FF
	evaluateRMWInstruction(t, cpu, ram, 7, "zN", 0xD005, 0xF0) // a,x -> $F1 - 1 = $F0
}

func TestActionDEX(t *testing.T) {
	cpu, ram := createComputer()

	cpu.xRegister = 0x02

	ram.Poke(0xC000, 0xCA) // DEX
	ram.Poke(0xC001, 0xCA) // DEX
	ram.Poke(0xC002, 0xCA) // DEX
	ram.Poke(0xC003, 0xEA) // NOP

	evaluateXRegisterInstruction(t, cpu, ram, 2, "zn", 0x01)
	evaluateXRegisterInstruction(t, cpu, ram, 2, "Zn", 0x00)
	evaluateXRegisterInstruction(t, cpu, ram, 2, "zN", 0xFF)
}

func TestActionDEY(t *testing.T) {
	cpu, ram := createComputer()

	cpu.yRegister = 0x02

	ram.Poke(0xC000, 0x88) // DEY
	ram.Poke(0xC001, 0x88) // DEY
	ram.Poke(0xC002, 0x88) // DEY
	ram.Poke(0xC003, 0xEA) // NOP

	evaluateYRegisterInstruction(t, cpu, ram, 2, "zn", 0x01)
	evaluateYRegisterInstruction(t, cpu, ram, 2, "Zn", 0x00)
	evaluateYRegisterInstruction(t, cpu, ram, 2, "zN", 0xFF)
}

func TestActionEOR(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xFF
	cpu.xRegister = 0x05
	cpu.yRegister = 0x0A

	ram.Poke(0x0010, 0xFD) // zp value $FD
	ram.Poke(0x0015, 0xFB) // zp,x value $FB

	ram.Poke(0x00A5, 0x10) // (zp,x) redirect to $D110
	ram.Poke(0x00A6, 0xD1)

	ram.Poke(0x00B0, 0xFF) // (zp),y redirect to $D2FF
	ram.Poke(0x00B1, 0xD2)

	ram.Poke(0x00C0, 0x00) // (zp) redict to $E000
	ram.Poke(0x00C1, 0xE0)

	ram.Poke(0xD000, 0xF7) // a value $F7
	ram.Poke(0xD005, 0xEF) // a,x value $EF
	ram.Poke(0xD00A, 0xDF) // a,y value $DF

	ram.Poke(0xD110, 0x7F) // (zp,x) value $7F

	ram.Poke(0xD309, 0xBF) // (zp),y value $BF

	ram.Poke(0xE000, 0x08) // (zp) value $08

	// A = FF
	ram.Poke(0xC000, 0x49) // EOR #$FE -> A ^ FE = 01
	ram.Poke(0xC001, 0xFE)
	ram.Poke(0xC002, 0x45) // EOR $10 -> A ^ FD = FC
	ram.Poke(0xC003, 0x10)
	ram.Poke(0xC004, 0x55) // EOR $10,X -> A ^ FB = 07
	ram.Poke(0xC005, 0x10)
	ram.Poke(0xC006, 0x4D) // EOR $D000 -> A ^ F7 = F0
	ram.Poke(0xC007, 0x00)
	ram.Poke(0xC008, 0xD0)
	ram.Poke(0xC009, 0x5D) // EOR $D000,X -> A ^ EF = 1F
	ram.Poke(0xC00A, 0x00)
	ram.Poke(0xC00B, 0xD0)
	ram.Poke(0xC00C, 0x59) // EOR $D000,Y -> A ^ DF = C0
	ram.Poke(0xC00D, 0x00)
	ram.Poke(0xC00E, 0xD0)
	ram.Poke(0xC00F, 0x41) // EOR ($A0,X) -> A ^ 7F = BF
	ram.Poke(0xC010, 0xA0)
	ram.Poke(0xC011, 0x51) // EOR ($B0),Y -> A ^ BF = 00
	ram.Poke(0xC012, 0xB0)
	ram.Poke(0xC013, 0x52) // EOR ($C0) -> A ^ 08 = F7
	ram.Poke(0xC014, 0xC0)

	evaluateAccumulatorInstruction(t, cpu, ram, 2, "zn", 0x01) // i -> FF ^ FE = 01
	evaluateAccumulatorInstruction(t, cpu, ram, 3, "zN", 0xFC) // zp -> 01 ^ FC = FC
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zn", 0x07) // zp,x -> FC ^ FB = 07
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zN", 0xF0) // a -> 07 ^ F7 = F0
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zn", 0x1F) // a,x -> F0 ^ EF = 1F
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zN", 0xC0) // a,y -> 1F ^ DF = C0
	evaluateAccumulatorInstruction(t, cpu, ram, 6, "zN", 0xBF) // (zp,x) -> C0 ^ 7F = BF
	evaluateAccumulatorInstruction(t, cpu, ram, 6, "Zn", 0x00) // (zp),y -> BF ^ BF = 00
	cpu.accumulatorRegister = 0xFF
	evaluateAccumulatorInstruction(t, cpu, ram, 5, "zN", 0xF7) // (zp) -> FF ^ 08 = F7
}

func TestActionINC(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xFF
	cpu.xRegister = 0x05

	ram.Poke(0x0010, 0x7F) // zp value $7F
	ram.Poke(0x0015, 0x00) // zp,x value $00

	ram.Poke(0xD000, 0x84) // a value $84
	ram.Poke(0xD005, 0xF0) // a,x value $F0

	// A = FF
	ram.Poke(0xC000, 0x1A) // INC a -> FF + 1 = $00
	ram.Poke(0xC001, 0xE6) // INC $10 -> $7F + 1 = $80
	ram.Poke(0xC002, 0x10)
	ram.Poke(0xC003, 0xF6) // INC $10,X -> $00 + 1 = $01
	ram.Poke(0xC004, 0x10)
	ram.Poke(0xC005, 0xEE) // INC $D000 -> $84 + 1 = $85
	ram.Poke(0xC006, 0x00)
	ram.Poke(0xC007, 0xD0)
	ram.Poke(0xC008, 0xFE) // INC $D000,X -> F0 + 1 = $F1
	ram.Poke(0xC009, 0x00)
	ram.Poke(0xC00A, 0xD0)

	evaluateAccumulatorInstruction(t, cpu, ram, 2, "Zn", 0x00) // A -> $FF + 1 = $00
	evaluateRMWInstruction(t, cpu, ram, 5, "zN", 0x0010, 0x80) // zp -> $7F + 1 = $80
	evaluateRMWInstruction(t, cpu, ram, 6, "zn", 0x0015, 0x01) // zp,x -> $00 + 1 = $01
	evaluateRMWInstruction(t, cpu, ram, 6, "zN", 0xD000, 0x85) // a -> $84 + 1 = $85
	evaluateRMWInstruction(t, cpu, ram, 7, "zN", 0xD005, 0xF1) // a,x -> $F0 + 1 = $F1
}

func TestActionINX(t *testing.T) {
	cpu, ram := createComputer()

	cpu.xRegister = 0xFE

	ram.Poke(0xC000, 0xE8) // INX
	ram.Poke(0xC001, 0xE8) // INX
	ram.Poke(0xC002, 0xE8) // INX
	ram.Poke(0xC003, 0xEA) // NOP

	evaluateXRegisterInstruction(t, cpu, ram, 2, "zN", 0xFF)
	evaluateXRegisterInstruction(t, cpu, ram, 2, "Zn", 0x00)
	evaluateXRegisterInstruction(t, cpu, ram, 2, "zn", 0x01)
}

func TestActionINY(t *testing.T) {
	cpu, ram := createComputer()

	cpu.yRegister = 0xFE

	ram.Poke(0xC000, 0xC8) // INY
	ram.Poke(0xC001, 0xC8) // INY
	ram.Poke(0xC002, 0xC8) // INY
	ram.Poke(0xC003, 0xEA) // NOP

	evaluateYRegisterInstruction(t, cpu, ram, 2, "zN", 0xFF)
	evaluateYRegisterInstruction(t, cpu, ram, 2, "Zn", 0x00)
	evaluateYRegisterInstruction(t, cpu, ram, 2, "zn", 0x01)
}

func TestActionJMP(t *testing.T) {
	cpu, ram := createComputer()

	cpu.xRegister = 0x50

	ram.Poke(0x00FF, 0x00)
	ram.Poke(0x0100, 0xE0)

	ram.Poke(0xC000, 0x4C) // JMP $D000
	ram.Poke(0xC001, 0x00)
	ram.Poke(0xC002, 0xD0)

	ram.Poke(0xC050, 0x00)
	ram.Poke(0xC051, 0xF0)

	ram.Poke(0xD000, 0x6C) // JMP ($00FF)
	ram.Poke(0xD001, 0xFF)
	ram.Poke(0xD002, 0x00)

	ram.Poke(0xE000, 0x7C) // JMP ($C000, X)
	ram.Poke(0xE001, 0x00)
	ram.Poke(0xE002, 0xC0)

	ram.Poke(0xF000, 0xEA)

	evaluateBranchInstruction(t, cpu, ram, 3, "", 0xD000)
	evaluateBranchInstruction(t, cpu, ram, 6, "", 0xE000)
	evaluateBranchInstruction(t, cpu, ram, 6, "", 0xF000)
}

func TestActionJSRandRTS(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xD000, 0xA9) // LDA #$FF
	ram.Poke(0xD001, 0xFF)
	ram.Poke(0xD002, 0x60) // RTS

	ram.Poke(0xC000, 0x20) // JSR $D000
	ram.Poke(0xC001, 0x00)
	ram.Poke(0xC002, 0xD0)
	ram.Poke(0xC003, 0xA9) // LDA #$77
	ram.Poke(0xC004, 0x77)

	evaluateBranchInstruction(t, cpu, ram, 6, "", 0xD000)        // Executes JSR
	evaluateAccumulatorInstruction(t, cpu, ram, 2, "Nvzc", 0xFF) // Executes LDA
	evaluateBranchInstruction(t, cpu, ram, 6, "", 0xC003)        // Executes RTS
	evaluateAccumulatorInstruction(t, cpu, ram, 2, "nvzc", 0x77) // Executes LDA
}

func TestActionLDA(t *testing.T) {
	cpu, ram := createComputer()

	cpu.xRegister = 0x05
	cpu.yRegister = 0x0A

	ram.Poke(0x0010, 0x02) // zp value $02
	ram.Poke(0x0015, 0xA0) // zp,x value $A0

	ram.Poke(0x00A5, 0x10) // (zp,x) redirect to $D110
	ram.Poke(0x00A6, 0xD1)

	ram.Poke(0x00B0, 0xFF) // (zp),y redirect to $D2FF
	ram.Poke(0x00B1, 0xD2)

	ram.Poke(0x00C0, 0x00) // (zp) redict to $E000
	ram.Poke(0x00C1, 0xE0)

	ram.Poke(0xD000, 0x10) // a value $10
	ram.Poke(0xD005, 0x11) // a,x value $11
	ram.Poke(0xD00A, 0x20) // a,y value $20

	ram.Poke(0xD110, 0x00) // (zp,x) value $02

	ram.Poke(0xD309, 0x10) // (zp),y value $10

	ram.Poke(0xE000, 0x01) // (zp) value $01

	// A = 0
	ram.Poke(0xC000, 0xA9) // LDA #$0A -> 0A
	ram.Poke(0xC001, 0x0A)
	ram.Poke(0xC002, 0xA5) // LDA $10 -> 02
	ram.Poke(0xC003, 0x10)
	ram.Poke(0xC004, 0xB5) // LDA $10,X -> A0
	ram.Poke(0xC005, 0x10)
	ram.Poke(0xC006, 0xAD) // LDA $D000 -> 10
	ram.Poke(0xC007, 0x00)
	ram.Poke(0xC008, 0xD0)
	ram.Poke(0xC009, 0xBD) // LDA $D000,X -> 11
	ram.Poke(0xC00A, 0x00)
	ram.Poke(0xC00B, 0xD0)
	ram.Poke(0xC00C, 0xB9) // LDA $D000,Y -> 20
	ram.Poke(0xC00D, 0x00)
	ram.Poke(0xC00E, 0xD0)
	ram.Poke(0xC00F, 0xA1) // LDA ($A0,X) -> 00
	ram.Poke(0xC010, 0xA0)
	ram.Poke(0xC011, 0xB1) // LDA ($B0),Y -> 10
	ram.Poke(0xC012, 0xB0)
	ram.Poke(0xC013, 0xB2) // LDA ($C0) -> 01
	ram.Poke(0xC014, 0xC0)

	evaluateAccumulatorInstruction(t, cpu, ram, 2, "zn", 0x0A) // i -> 0A
	evaluateAccumulatorInstruction(t, cpu, ram, 3, "zn", 0x02) // zp -> 02
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zN", 0xA0) // zp,x -> A0
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zn", 0x10) // a -> 10
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zn", 0x11) // a,x -> 11
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zn", 0x20) // a,y -> 20
	evaluateAccumulatorInstruction(t, cpu, ram, 6, "Zn", 0x00) // (zp,x) -> 00
	evaluateAccumulatorInstruction(t, cpu, ram, 6, "zn", 0x10) // (zp),y -> 10
	evaluateAccumulatorInstruction(t, cpu, ram, 5, "zn", 0x01) // (zp) -> 01
}

func TestActionLDX(t *testing.T) {
	cpu, ram := createComputer()

	cpu.yRegister = 0x05

	ram.Poke(0x0010, 0x7F) // zp value $7F
	ram.Poke(0x0015, 0xBF) // zp,y value $BF

	ram.Poke(0xD000, 0x00) // a value $00
	ram.Poke(0xD005, 0x3F) // a,y value $EF

	ram.Poke(0xC000, 0xA2) // LDX #$FE -> $FE
	ram.Poke(0xC001, 0xFE)
	ram.Poke(0xC002, 0xA6) // LDX $10 -> 7F
	ram.Poke(0xC003, 0x10)
	ram.Poke(0xC004, 0xB6) // LDX $10,y -> BF
	ram.Poke(0xC005, 0x10)
	ram.Poke(0xC006, 0xAE) // LDX $D000 -> 00
	ram.Poke(0xC007, 0x00)
	ram.Poke(0xC008, 0xD0)
	ram.Poke(0xC009, 0xBE) // LDX $D000,y -> 3F
	ram.Poke(0xC00A, 0x00)
	ram.Poke(0xC00B, 0xD0)

	evaluateXRegisterInstruction(t, cpu, ram, 2, "zN", 0xFE) // i -> FE
	evaluateXRegisterInstruction(t, cpu, ram, 3, "zn", 0x7F) // zp -> 7F
	evaluateXRegisterInstruction(t, cpu, ram, 4, "zN", 0xBF) // zp,x -> BF
	evaluateXRegisterInstruction(t, cpu, ram, 4, "Zn", 0x00) // a -> 00
	evaluateXRegisterInstruction(t, cpu, ram, 4, "zn", 0x3F) // a,x -> 3F
}

func TestActionLDY(t *testing.T) {
	cpu, ram := createComputer()

	cpu.xRegister = 0x05

	ram.Poke(0x0010, 0x7F) // zp value $7F
	ram.Poke(0x0015, 0xBF) // zp,x value $BF

	ram.Poke(0xD000, 0x00) // a value $00
	ram.Poke(0xD005, 0x3F) // a,x value $EF

	ram.Poke(0xC000, 0xA0) // LDY #$FE -> $FE
	ram.Poke(0xC001, 0xFE)
	ram.Poke(0xC002, 0xA4) // LDY $10 -> 7F
	ram.Poke(0xC003, 0x10)
	ram.Poke(0xC004, 0xB4) // LDY $10,x -> BF
	ram.Poke(0xC005, 0x10)
	ram.Poke(0xC006, 0xAC) // LDY $D000 -> 00
	ram.Poke(0xC007, 0x00)
	ram.Poke(0xC008, 0xD0)
	ram.Poke(0xC009, 0xBC) // LDY $D000,x -> 3F
	ram.Poke(0xC00A, 0x00)
	ram.Poke(0xC00B, 0xD0)

	evaluateYRegisterInstruction(t, cpu, ram, 2, "zN", 0xFE) // i -> FE
	evaluateYRegisterInstruction(t, cpu, ram, 3, "zn", 0x7F) // zp -> 7F
	evaluateYRegisterInstruction(t, cpu, ram, 4, "zN", 0xBF) // zp,x -> BF
	evaluateYRegisterInstruction(t, cpu, ram, 4, "Zn", 0x00) // a -> 00
	evaluateYRegisterInstruction(t, cpu, ram, 4, "zn", 0x3F) // a,x -> 3F
}

func TestActionLSR(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xAA
	cpu.xRegister = 0x05

	ram.Poke(0x0010, 0x08) // zp value $08
	ram.Poke(0x0015, 0x55) // zp,x value $55

	ram.Poke(0xD000, 0x55) // a value $55
	ram.Poke(0xD005, 0x01) // a,x value $01

	// A = AA
	ram.Poke(0xC000, 0x4A) // LSR a <- AA
	ram.Poke(0xC001, 0x46) // LSR $10 <- 08
	ram.Poke(0xC002, 0x10)
	ram.Poke(0xC003, 0x56) // LSR $10,X <- 55
	ram.Poke(0xC004, 0x10)
	ram.Poke(0xC005, 0x4E) // LSR $D000 <- 55
	ram.Poke(0xC006, 0x00)
	ram.Poke(0xC007, 0xD0)
	ram.Poke(0xC008, 0x5E) // LSR $D000,X <- 01
	ram.Poke(0xC009, 0x00)
	ram.Poke(0xC00A, 0xD0)

	evaluateAccumulatorInstruction(t, cpu, ram, 2, "znc", 0x55) // i -> AA >> 1 = 55
	evaluateRMWInstruction(t, cpu, ram, 5, "znc", 0x0010, 0x04) // zp -> 08 >> 1 = 04
	evaluateRMWInstruction(t, cpu, ram, 6, "znC", 0x0015, 0x2A) // zp,x -> 55 >> 1 = 2A
	evaluateRMWInstruction(t, cpu, ram, 6, "znC", 0xD000, 0x2A) // a -> 55 >> 1 = 2A
	evaluateRMWInstruction(t, cpu, ram, 7, "ZnC", 0xD005, 0x00) // a,x -> 01 << 1 = 00
}

func TestActionNOP(t *testing.T) {
	cpu, ram := createComputer()

	cpu.yRegister = 0xFE

	ram.Poke(0xC000, 0xEA) // NOP
	ram.Poke(0xC001, 0xEA) // NOP
	ram.Poke(0xC002, 0xC8) // INY

	evaluateYRegisterInstruction(t, cpu, ram, 2, "zn", 0xFE)
	evaluateYRegisterInstruction(t, cpu, ram, 2, "zn", 0xFE)
	evaluateYRegisterInstruction(t, cpu, ram, 2, "zN", 0xFF)
}

func TestActionORA(t *testing.T) {
	cpu, ram := createComputer()

	cpu.xRegister = 0x05
	cpu.yRegister = 0x0A

	ram.Poke(0x0010, 0x02) // zp value $02
	ram.Poke(0x0015, 0x04) // zp,x value $04

	ram.Poke(0x00A5, 0x10) // (zp,x) redirect to $D110
	ram.Poke(0x00A6, 0xD1)

	ram.Poke(0x00B0, 0xFF) // (zp),y redirect to $D2FF
	ram.Poke(0x00B1, 0xD2)

	ram.Poke(0x00C0, 0x00) // (zp) redict to $E000
	ram.Poke(0x00C1, 0xE0)

	ram.Poke(0xD000, 0x08) // a value $08
	ram.Poke(0xD005, 0x10) // a,x value $10
	ram.Poke(0xD00A, 0x20) // a,y value $20

	ram.Poke(0xD110, 0x40) // (zp,x) value $40

	ram.Poke(0xD309, 0x80) // (zp),y value $80

	ram.Poke(0xE000, 0x00) // (zp) value $00

	// A = 00
	ram.Poke(0xC000, 0x09) // ORA #$FE -> A | 01 = 01
	ram.Poke(0xC001, 0x01)
	ram.Poke(0xC002, 0x05) // ORA $10 -> A | 02 = 03
	ram.Poke(0xC003, 0x10)
	ram.Poke(0xC004, 0x15) // ORA $10,X -> A | 04 = 07
	ram.Poke(0xC005, 0x10)
	ram.Poke(0xC006, 0x0D) // ORA $D000 -> A | 08 = 0F
	ram.Poke(0xC007, 0x00)
	ram.Poke(0xC008, 0xD0)
	ram.Poke(0xC009, 0x1D) // ORA $D000,X -> A | 10 = 1F
	ram.Poke(0xC00A, 0x00)
	ram.Poke(0xC00B, 0xD0)
	ram.Poke(0xC00C, 0x19) // ORA $D000,Y -> A | 20 = 3F
	ram.Poke(0xC00D, 0x00)
	ram.Poke(0xC00E, 0xD0)
	ram.Poke(0xC00F, 0x01) // ORA ($A0,X) -> A | 40 = 7F
	ram.Poke(0xC010, 0xA0)
	ram.Poke(0xC011, 0x11) // ORA ($B0),Y -> A | 80 = FF
	ram.Poke(0xC012, 0xB0)
	ram.Poke(0xC013, 0x12) // ORA ($C0) -> A | AA = FF
	ram.Poke(0xC014, 0xC0)

	evaluateAccumulatorInstruction(t, cpu, ram, 2, "zn", 0x01) // i -> 00 | 01 = 01
	evaluateAccumulatorInstruction(t, cpu, ram, 3, "zn", 0x03) // zp -> 01 | 02 = 03
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zn", 0x07) // zp,x -> 02 | 04 = 07
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zn", 0x0F) // a -> 04 | 08 = 0F
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zn", 0x1F) // a,x -> 0F | 10 = 1F
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zn", 0x3F) // a,y -> 1F | 20 = 3F
	evaluateAccumulatorInstruction(t, cpu, ram, 6, "zn", 0x7F) // (zp,x) -> 3F | 40 = 7F
	evaluateAccumulatorInstruction(t, cpu, ram, 6, "zN", 0xFF) // (zp),y -> 7F | 80 = FF
	cpu.accumulatorRegister = 0x00
	evaluateAccumulatorInstruction(t, cpu, ram, 5, "Zn", 0x00) // (zp) -> 00 | 00 = 00
}

func TestActionPHAandPLA(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0x48) // PHA
	ram.Poke(0xC001, 0x48) // PHA
	ram.Poke(0xC002, 0x48) // PHA
	ram.Poke(0xC003, 0x68) // PLA
	ram.Poke(0xC004, 0x68) // PLA
	ram.Poke(0xC005, 0x68) // PLA

	cpu.accumulatorRegister = 0x00
	evaluateRMWInstruction(t, cpu, ram, 3, "", 0x01FD, 0x00)
	cpu.accumulatorRegister = 0xFF
	evaluateRMWInstruction(t, cpu, ram, 3, "", 0x01FC, 0xFF)
	cpu.accumulatorRegister = 0x70
	evaluateRMWInstruction(t, cpu, ram, 3, "", 0x01FB, 0x70)
	cpu.accumulatorRegister = 0x00
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "nz", 0x70)
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "Nz", 0xFF)
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "nZ", 0x00)
}

func TestActionPHP(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0x08) // PHP
	ram.Poke(0xC001, 0x08) // PHP
	ram.Poke(0xC002, 0x28) // PLP
	ram.Poke(0xC003, 0x28) // PLP

	cpu.processorStatusRegister.SetFlag(CarryFlagBit, true)
	evaluateRMWInstruction(t, cpu, ram, 3, "", 0x01FD, 0x35)
	cpu.processorStatusRegister.SetFlag(NegativeFlagBit, true)
	evaluateRMWInstruction(t, cpu, ram, 3, "", 0x01FC, 0xB5)
	cpu.processorStatusRegister.SetFlag(NegativeFlagBit, false)
	cpu.processorStatusRegister.SetFlag(CarryFlagBit, false)
	evaluateFlagInstruction(t, cpu, ram, 4, "CNB")
	evaluateFlagInstruction(t, cpu, ram, 4, "CB")
}

func TestActionPHXandPLX(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0xDA) // PHX
	ram.Poke(0xC001, 0xDA) // PHX
	ram.Poke(0xC002, 0xDA) // PHX
	ram.Poke(0xC003, 0xFA) // PLX
	ram.Poke(0xC004, 0xFA) // PLX
	ram.Poke(0xC005, 0xFA) // PLX

	cpu.xRegister = 0x00
	evaluateRMWInstruction(t, cpu, ram, 3, "", 0x01FD, 0x00)
	cpu.xRegister = 0xFF
	evaluateRMWInstruction(t, cpu, ram, 3, "", 0x01FC, 0xFF)
	cpu.xRegister = 0x70
	evaluateRMWInstruction(t, cpu, ram, 3, "", 0x01FB, 0x70)
	cpu.xRegister = 0x00
	evaluateXRegisterInstruction(t, cpu, ram, 4, "nz", 0x70)
	evaluateXRegisterInstruction(t, cpu, ram, 4, "Nz", 0xFF)
	evaluateXRegisterInstruction(t, cpu, ram, 4, "nZ", 0x00)
}

func TestActionPHYandPLY(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0x5A) // PHY
	ram.Poke(0xC001, 0x5A) // PHY
	ram.Poke(0xC002, 0x5A) // PHY
	ram.Poke(0xC003, 0x7A) // PLY
	ram.Poke(0xC004, 0x7A) // PLY
	ram.Poke(0xC005, 0x7A) // PLY

	cpu.yRegister = 0x00
	evaluateRMWInstruction(t, cpu, ram, 3, "", 0x01FD, 0x00)
	cpu.yRegister = 0xFF
	evaluateRMWInstruction(t, cpu, ram, 3, "", 0x01FC, 0xFF)
	cpu.yRegister = 0x70
	evaluateRMWInstruction(t, cpu, ram, 3, "", 0x01FB, 0x70)
	cpu.yRegister = 0x00
	evaluateYRegisterInstruction(t, cpu, ram, 4, "nz", 0x70)
	evaluateYRegisterInstruction(t, cpu, ram, 4, "Nz", 0xFF)
	evaluateYRegisterInstruction(t, cpu, ram, 4, "nZ", 0x00)
}

func TestActionROL(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xAA
	cpu.xRegister = 0x05

	ram.Poke(0x0010, 0x08) // zp value $08
	ram.Poke(0x0015, 0x04) // zp,x value $04

	ram.Poke(0xD000, 0xC0) // a value $C0
	ram.Poke(0xD005, 0x80) // a,x value $80

	// A = AA
	ram.Poke(0xC000, 0x2A) // ROL a <- AA
	ram.Poke(0xC001, 0x26) // ROL $10 <- 08
	ram.Poke(0xC002, 0x10)
	ram.Poke(0xC003, 0x36) // ROL $10,X <- 04
	ram.Poke(0xC004, 0x10)
	ram.Poke(0xC005, 0x2E) // ROL $D000 <- C0
	ram.Poke(0xC006, 0x00)
	ram.Poke(0xC007, 0xD0)
	ram.Poke(0xC008, 0x3E) // ROL $D000,X <- 80
	ram.Poke(0xC009, 0x00)
	ram.Poke(0xC00A, 0xD0)

	evaluateAccumulatorInstruction(t, cpu, ram, 2, "znC", 0x54) // i -> AA << 1 = 54
	evaluateRMWInstruction(t, cpu, ram, 5, "znc", 0x0010, 0x11) // zp -> 08 << 1 = 11
	evaluateRMWInstruction(t, cpu, ram, 6, "znc", 0x0015, 0x08) // zp,x -> 04 << 1 = 08
	evaluateRMWInstruction(t, cpu, ram, 6, "zNC", 0xD000, 0x80) // a -> C0 << 1 = 80
	evaluateRMWInstruction(t, cpu, ram, 7, "znC", 0xD005, 0x01) // a,x -> 80 << 1 = 00
}

func TestActionROR(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xAA
	cpu.xRegister = 0x05

	ram.Poke(0x0010, 0x08) // zp value $08
	ram.Poke(0x0015, 0x55) // zp,x value $55

	ram.Poke(0xD000, 0x55) // a value $55
	ram.Poke(0xD005, 0x01) // a,x value $01

	// A = AA
	ram.Poke(0xC000, 0x6A) // ROR a <- AA
	ram.Poke(0xC001, 0x66) // ROR $10 <- 08
	ram.Poke(0xC002, 0x10)
	ram.Poke(0xC003, 0x76) // ROR $10,X <- 55
	ram.Poke(0xC004, 0x10)
	ram.Poke(0xC005, 0x6E) // ROR $D000 <- 55
	ram.Poke(0xC006, 0x00)
	ram.Poke(0xC007, 0xD0)
	ram.Poke(0xC008, 0x7E) // ROR $D000,X <- 01
	ram.Poke(0xC009, 0x00)
	ram.Poke(0xC00A, 0xD0)

	evaluateAccumulatorInstruction(t, cpu, ram, 2, "znc", 0x55) // i -> AA >> 1 = 55
	evaluateRMWInstruction(t, cpu, ram, 5, "znc", 0x0010, 0x04) // zp -> 08 >> 1 = 04
	evaluateRMWInstruction(t, cpu, ram, 6, "znC", 0x0015, 0x2A) // zp,x -> 55 >> 1 = 2A
	evaluateRMWInstruction(t, cpu, ram, 6, "zNC", 0xD000, 0xAA) // a -> 55 >> 1 = 2A
	cpu.processorStatusRegister.SetFlag(CarryFlagBit, false)
	evaluateRMWInstruction(t, cpu, ram, 7, "ZnC", 0xD005, 0x00) // a,x -> 01 << 1 = 00
}

func TestActionSBC(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xFF
	cpu.xRegister = 0x05
	cpu.yRegister = 0x0A

	ram.Poke(0x0010, 0x02) // zp value $02
	ram.Poke(0x0015, 0x20) // zp,x value $20

	ram.Poke(0x00A5, 0x10) // (zp,x) redirect to $D110
	ram.Poke(0x00A6, 0xD1)

	ram.Poke(0x00B0, 0xFF) // (zp),y redirect to $D2FF
	ram.Poke(0x00B1, 0xD2)

	ram.Poke(0x00C0, 0x00) // (zp) redict to $E000
	ram.Poke(0x00C1, 0xE0)

	ram.Poke(0xD000, 0x11) // a value $11
	ram.Poke(0xD005, 0x10) // a,x value $10
	ram.Poke(0xD00A, 0xA0) // a,y value $A0

	ram.Poke(0xD110, 0x02) // (zp,x) value $02

	ram.Poke(0xD309, 0x0A) // (zp),y value $0A

	ram.Poke(0xE000, 0x01) // (zp) value $01

	// A = FF
	ram.Poke(0xC000, 0xE9) // SBC #$0A -> A - 10 = EF
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC002, 0xE5) // SBC $10 -> EF - 02 = ED
	ram.Poke(0xC003, 0x10)
	ram.Poke(0xC004, 0xF5) // SBC $10,X -> ED - 20 = CD
	ram.Poke(0xC005, 0x10)
	ram.Poke(0xC006, 0xED) // SBC $D000 -> CD - 11 = BC
	ram.Poke(0xC007, 0x00)
	ram.Poke(0xC008, 0xD0)
	ram.Poke(0xC009, 0xFD) // SBC $D000,X -> BC - 10 = AC
	ram.Poke(0xC00A, 0x00)
	ram.Poke(0xC00B, 0xD0)
	ram.Poke(0xC00C, 0xF9) // SBC $D000,Y -> AC - A0 = 0C
	ram.Poke(0xC00D, 0x00)
	ram.Poke(0xC00E, 0xD0)
	ram.Poke(0xC00F, 0xE1) // SBC ($A0,X) -> 0C - 02 = 0A
	ram.Poke(0xC010, 0xA0)
	ram.Poke(0xC011, 0xF1) // SBC ($B0),Y -> 0A - 0A = 00
	ram.Poke(0xC012, 0xB0)
	ram.Poke(0xC013, 0xF2) // SBC ($C0) -> 00 - 01 = FF
	ram.Poke(0xC014, 0xC0)

	cpu.processorStatusRegister.SetFlag(CarryFlagBit, true)
	evaluateAccumulatorInstruction(t, cpu, ram, 2, "zNC", 0xEF) // i -> FF - 10 = EF
	evaluateAccumulatorInstruction(t, cpu, ram, 3, "zNC", 0xED) // zp -> EF - 02 = ED
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zNC", 0xCD) // zp,x -> ED - 20 = CD
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zNC", 0xBC) // a -> CD - 11 = BC
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zNC", 0xAC) // a,x -> BC - 10 = AC
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "znC", 0x0C) // a,y -> AC - A0 = 0C
	evaluateAccumulatorInstruction(t, cpu, ram, 6, "znC", 0x0A) // (zp,x) -> 0C - 02 = 0A
	evaluateAccumulatorInstruction(t, cpu, ram, 6, "ZnC", 0x00) // (zp),y -> 0A - 0A = 00
	evaluateAccumulatorInstruction(t, cpu, ram, 5, "zNc", 0xFF) // (zp) -> 00 - 1 = FF
}

func TestActionSEC(t *testing.T) {
	cpu, ram := createComputer()

	cpu.processorStatusRegister.SetFlag(CarryFlagBit, false)

	ram.Poke(0xC000, 0x38) // SEC
	ram.Poke(0xC001, 0xEA) // NOP

	evaluateFlagInstruction(t, cpu, ram, 2, "C")
}

func TestActionSED(t *testing.T) {
	cpu, ram := createComputer()

	cpu.processorStatusRegister.SetFlag(DecimalModeFlagBit, false)

	ram.Poke(0xC000, 0xF8) // SED
	ram.Poke(0xC001, 0xEA) // NOP

	evaluateFlagInstruction(t, cpu, ram, 2, "D")
}

func TestActionSEI(t *testing.T) {
	cpu, ram := createComputer()

	cpu.processorStatusRegister.SetFlag(IrqDisableFlagBit, false)

	ram.Poke(0xC000, 0x78) // SEI
	ram.Poke(0xC001, 0xEA) // NOP

	evaluateFlagInstruction(t, cpu, ram, 2, "I")
}

func TestActionSTA(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xFF
	cpu.xRegister = 0x05
	cpu.yRegister = 0x0A

	ram.Poke(0x00A5, 0x10) // (zp,x) redirect to $D110
	ram.Poke(0x00A6, 0xD1)

	ram.Poke(0x00B0, 0xFF) // (zp),y redirect to $D2FF
	ram.Poke(0x00B1, 0xD2)

	ram.Poke(0x00C0, 0x00) // (zp) redict to $E000
	ram.Poke(0x00C1, 0xE0)

	// A = FF
	ram.Poke(0xC000, 0x85) // STA $10
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC002, 0x95) // STA $10,X
	ram.Poke(0xC003, 0x10)
	ram.Poke(0xC004, 0x8D) // STA $D000
	ram.Poke(0xC005, 0x00)
	ram.Poke(0xC006, 0xD0)
	ram.Poke(0xC007, 0x9D) // STA $D000,X
	ram.Poke(0xC008, 0x00)
	ram.Poke(0xC009, 0xD0)
	ram.Poke(0xC00A, 0x99) // STA $D000,Y
	ram.Poke(0xC00B, 0x00)
	ram.Poke(0xC00C, 0xD0)
	ram.Poke(0xC00D, 0x81) // STA ($A0,X)
	ram.Poke(0xC00E, 0xA0)
	ram.Poke(0xC00F, 0x91) // STA ($B0),Y
	ram.Poke(0xC010, 0xB0)
	ram.Poke(0xC011, 0x92) // STA ($C0)
	ram.Poke(0xC012, 0xC0)

	evaluateRMWInstruction(t, cpu, ram, 3, "", 0x0010, 0xFF)
	evaluateRMWInstruction(t, cpu, ram, 4, "", 0x0015, 0xFF)
	evaluateRMWInstruction(t, cpu, ram, 4, "", 0xD000, 0xFF)
	evaluateRMWInstruction(t, cpu, ram, 5, "", 0xD005, 0xFF)
	evaluateRMWInstruction(t, cpu, ram, 5, "", 0xD00A, 0xFF)
	evaluateRMWInstruction(t, cpu, ram, 6, "", 0xD110, 0xFF)
	evaluateRMWInstruction(t, cpu, ram, 6, "", 0xD309, 0xFF)
	evaluateRMWInstruction(t, cpu, ram, 5, "", 0xE000, 0xFF)
}

func TestActionSTX(t *testing.T) {
	cpu, ram := createComputer()

	cpu.xRegister = 0xFF
	cpu.yRegister = 0x05

	// X = FF
	ram.Poke(0xC000, 0x86) // STX $10
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC002, 0x96) // STX $10,Y
	ram.Poke(0xC003, 0x10)
	ram.Poke(0xC004, 0x8E) // STX $D000
	ram.Poke(0xC005, 0x00)
	ram.Poke(0xC006, 0xD0)

	evaluateRMWInstruction(t, cpu, ram, 3, "", 0x0010, 0xFF)
	evaluateRMWInstruction(t, cpu, ram, 4, "", 0x0015, 0xFF)
	evaluateRMWInstruction(t, cpu, ram, 4, "", 0xD000, 0xFF)
}

func TestActionSTY(t *testing.T) {
	cpu, ram := createComputer()

	cpu.xRegister = 0x05
	cpu.yRegister = 0xFF

	// X = FF
	ram.Poke(0xC000, 0x84) // STY $10
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC002, 0x94) // STY $10,X
	ram.Poke(0xC003, 0x10)
	ram.Poke(0xC004, 0x8C) // STY $D000
	ram.Poke(0xC005, 0x00)
	ram.Poke(0xC006, 0xD0)

	evaluateRMWInstruction(t, cpu, ram, 3, "", 0x0010, 0xFF)
	evaluateRMWInstruction(t, cpu, ram, 4, "", 0x0015, 0xFF)
	evaluateRMWInstruction(t, cpu, ram, 4, "", 0xD000, 0xFF)
}

func TestActionSTZ(t *testing.T) {
	cpu, ram := createComputer()

	cpu.xRegister = 0x05

	ram.Poke(0x0010, 0xFF) // zp value $FF
	ram.Poke(0x0015, 0xFF) // zp,x value $FF

	ram.Poke(0xD000, 0xFF) // a value $FF
	ram.Poke(0xD005, 0xFF) // a,x value $FF

	ram.Poke(0xC000, 0x64) // STZ $10
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC002, 0x74) // STZ $10,X
	ram.Poke(0xC003, 0x10)
	ram.Poke(0xC004, 0x9C) // STZ $D000
	ram.Poke(0xC005, 0x00)
	ram.Poke(0xC006, 0xD0)
	ram.Poke(0xC007, 0x9E) // STZ $D000,X
	ram.Poke(0xC008, 0x00)
	ram.Poke(0xC009, 0xD0)

	evaluateRMWInstruction(t, cpu, ram, 3, "", 0x0010, 0x00) // zp
	evaluateRMWInstruction(t, cpu, ram, 4, "", 0x0015, 0x00) // zp,x
	evaluateRMWInstruction(t, cpu, ram, 4, "", 0xD000, 0x00) // a
	evaluateRMWInstruction(t, cpu, ram, 5, "", 0xD005, 0x00) // a,x
}

func TestActionTAX(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xAA
	cpu.xRegister = 0x05

	ram.Poke(0xC000, 0xAA) // TAX
	ram.Poke(0xC001, 0xAA) // TAX
	ram.Poke(0xC002, 0xEA) // NOP

	evaluateXRegisterInstruction(t, cpu, ram, 2, "Nz", 0xAA)
	cpu.accumulatorRegister = 0x00
	evaluateXRegisterInstruction(t, cpu, ram, 2, "Zn", 0x00)
}

func TestActionTAY(t *testing.T) {
	cpu, ram := createComputer()

	cpu.accumulatorRegister = 0xAA
	cpu.yRegister = 0x05

	ram.Poke(0xC000, 0xA8) // TAY
	ram.Poke(0xC001, 0xA8) // TAY
	ram.Poke(0xC002, 0xEA) // NOP

	evaluateYRegisterInstruction(t, cpu, ram, 2, "Nz", 0xAA)
	cpu.accumulatorRegister = 0x00
	evaluateYRegisterInstruction(t, cpu, ram, 2, "Zn", 0x00)
}

func TestActionTSX(t *testing.T) {
	cpu, ram := createComputer()

	cpu.xRegister = 0x05

	ram.Poke(0xC000, 0xBA) // TSX
	ram.Poke(0xC001, 0xBA) // TSX
	ram.Poke(0xC002, 0xEA) // NOP

	evaluateXRegisterInstruction(t, cpu, ram, 2, "Nz", 0xFD)
	cpu.stackPointer = 0x00
	evaluateXRegisterInstruction(t, cpu, ram, 2, "nZ", 0x00)
}

func TestActionTXA(t *testing.T) {
	cpu, ram := createComputer()

	cpu.xRegister = 0xAA
	cpu.accumulatorRegister = 0x05

	ram.Poke(0xC000, 0x8A) // TXA
	ram.Poke(0xC001, 0x8A) // TXA
	ram.Poke(0xC002, 0xEA) // NOP

	evaluateAccumulatorInstruction(t, cpu, ram, 2, "Nz", 0xAA)
	cpu.xRegister = 0x00
	evaluateAccumulatorInstruction(t, cpu, ram, 2, "Zn", 0x00)
}

func TestActionTXS(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0x9A) // TXS
	ram.Poke(0xC001, 0x9A) // TXS
	ram.Poke(0xC002, 0xEA) // NOP

	cpu.xRegister = 0xFA
	evaluateStackPointerInstruction(t, cpu, ram, 2, "", 0xFA)
	cpu.xRegister = 0x00
	evaluateStackPointerInstruction(t, cpu, ram, 2, "", 0x00)
}

func TestActionTYA(t *testing.T) {
	cpu, ram := createComputer()

	cpu.yRegister = 0xAA
	cpu.accumulatorRegister = 0x05

	ram.Poke(0xC000, 0x98) // TYA
	ram.Poke(0xC001, 0x98) // TYA
	ram.Poke(0xC002, 0xEA) // NOP

	evaluateAccumulatorInstruction(t, cpu, ram, 2, "Nz", 0xAA)
	cpu.yRegister = 0x00
	evaluateAccumulatorInstruction(t, cpu, ram, 2, "Zn", 0x00)
}

func TestActionBBR(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0x0010, 0xFD)
	ram.Poke(0x0015, 0x7F)

	ram.Poke(0xC000, 0x1F) // BBR1 $10, $07
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC002, 0x07)

	ram.Poke(0xC00A, 0x1F) // BBR1 $10, $D3
	ram.Poke(0xC00B, 0x10)
	ram.Poke(0xC00C, 0xD3)

	ram.Poke(0xBFE0, 0x1F) // BBR1 $15, $20  (won't be taken)
	ram.Poke(0xBFE1, 0x15)
	ram.Poke(0xBFE2, 0x20)

	ram.Poke(0xBFE3, 0xEA) // NOP

	ram.Poke(0xBFE4, 0x7F) // BBR7 $15, $20  (won't be taken)
	ram.Poke(0xBFE5, 0x15)
	ram.Poke(0xBFE6, 0x20)

	evaluateBranchInstruction(t, cpu, ram, 6, "", 0xC00A)
	evaluateBranchInstruction(t, cpu, ram, 7, "", 0xBFE0)
	evaluateBranchInstruction(t, cpu, ram, 5, "", 0xBFE3)
	// Not Taken but check NOP executed normally
	evaluateBranchInstruction(t, cpu, ram, 2, "", 0xBFE4)
	// Test another bit and branch taken
	evaluateBranchInstruction(t, cpu, ram, 6, "", 0xBFE7)
}

func TestActionBBS(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0x0010, 0x02)
	ram.Poke(0x0015, 0x80)

	ram.Poke(0xC000, 0x9F) // BBS1 $10, $07
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC002, 0x07)

	ram.Poke(0xC00A, 0x9F) // BBS1 $10, $D3
	ram.Poke(0xC00B, 0x10)
	ram.Poke(0xC00C, 0xD3)

	ram.Poke(0xBFE0, 0x9F) // BBS1 $15, $20  (won't be taken)
	ram.Poke(0xBFE1, 0x15)
	ram.Poke(0xBFE2, 0x20)

	ram.Poke(0xBFE3, 0xEA) // NOP

	ram.Poke(0xBFE4, 0xFF) // BBS7 $15, $20  (won't be taken)
	ram.Poke(0xBFE5, 0x15)
	ram.Poke(0xBFE6, 0x20)

	evaluateBranchInstruction(t, cpu, ram, 6, "", 0xC00A)
	evaluateBranchInstruction(t, cpu, ram, 7, "", 0xBFE0)
	evaluateBranchInstruction(t, cpu, ram, 5, "", 0xBFE3)
	// Not Taken but check NOP executed normally
	evaluateBranchInstruction(t, cpu, ram, 2, "", 0xBFE4)
	// Test another bit and branch taken
	evaluateBranchInstruction(t, cpu, ram, 6, "", 0xBFE7)
}

func TestActionRMB(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0x0010, 0xFF)

	ram.Poke(0xC000, 0x07) // RMB0 $10
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC002, 0x17) // RMB1 $10
	ram.Poke(0xC003, 0x10)
	ram.Poke(0xC004, 0x27) // RMB2 $10
	ram.Poke(0xC005, 0x10)
	ram.Poke(0xC006, 0x37) // RMB3 $10
	ram.Poke(0xC007, 0x10)
	ram.Poke(0xC008, 0x47) // RMB4 $10
	ram.Poke(0xC009, 0x10)
	ram.Poke(0xC00A, 0x57) // RMB5 $10
	ram.Poke(0xC00B, 0x10)
	ram.Poke(0xC00C, 0x67) // RMB6 $10
	ram.Poke(0xC00D, 0x10)
	ram.Poke(0xC00E, 0x77) // RMB7 $10
	ram.Poke(0xC00F, 0x10)

	evaluateRMWInstruction(t, cpu, ram, 5, "", 0x0010, 0xFE)
	evaluateRMWInstruction(t, cpu, ram, 5, "", 0x0010, 0xFC)
	evaluateRMWInstruction(t, cpu, ram, 5, "", 0x0010, 0xF8)
	evaluateRMWInstruction(t, cpu, ram, 5, "", 0x0010, 0xF0)
	evaluateRMWInstruction(t, cpu, ram, 5, "", 0x0010, 0xE0)
	evaluateRMWInstruction(t, cpu, ram, 5, "", 0x0010, 0xC0)
	evaluateRMWInstruction(t, cpu, ram, 5, "", 0x0010, 0x80)
	evaluateRMWInstruction(t, cpu, ram, 5, "", 0x0010, 0x00)
}

func TestActionSMB(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0x0010, 0x00)

	ram.Poke(0xC000, 0x87) // RMB0 $10
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC002, 0x97) // RMB1 $10
	ram.Poke(0xC003, 0x10)
	ram.Poke(0xC004, 0xA7) // RMB2 $10
	ram.Poke(0xC005, 0x10)
	ram.Poke(0xC006, 0xB7) // RMB3 $10
	ram.Poke(0xC007, 0x10)
	ram.Poke(0xC008, 0xC7) // RMB4 $10
	ram.Poke(0xC009, 0x10)
	ram.Poke(0xC00A, 0xD7) // RMB5 $10
	ram.Poke(0xC00B, 0x10)
	ram.Poke(0xC00C, 0xE7) // RMB6 $10
	ram.Poke(0xC00D, 0x10)
	ram.Poke(0xC00E, 0xF7) // RMB7 $10
	ram.Poke(0xC00F, 0x10)

	evaluateRMWInstruction(t, cpu, ram, 5, "", 0x0010, 0x01)
	evaluateRMWInstruction(t, cpu, ram, 5, "", 0x0010, 0x03)
	evaluateRMWInstruction(t, cpu, ram, 5, "", 0x0010, 0x07)
	evaluateRMWInstruction(t, cpu, ram, 5, "", 0x0010, 0x0F)
	evaluateRMWInstruction(t, cpu, ram, 5, "", 0x0010, 0x1F)
	evaluateRMWInstruction(t, cpu, ram, 5, "", 0x0010, 0x3F)
	evaluateRMWInstruction(t, cpu, ram, 5, "", 0x0010, 0x7F)
	evaluateRMWInstruction(t, cpu, ram, 5, "", 0x0010, 0xFF)
}

func TestActionTRB(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0x0010, 0xAA)

	ram.Poke(0xD000, 0x2A)

	ram.Poke(0xC000, 0x14) // TRB $10
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC002, 0x14) // TRB $10
	ram.Poke(0xC003, 0x10)
	ram.Poke(0xC004, 0x1C) // TRB $D000
	ram.Poke(0xC005, 0x00)
	ram.Poke(0xC006, 0xD0)

	cpu.accumulatorRegister = 0x80
	evaluateRMWInstruction(t, cpu, ram, 5, "Z", 0x0010, 0x2A)
	cpu.accumulatorRegister = 0x40
	evaluateRMWInstruction(t, cpu, ram, 5, "z", 0x0010, 0x2A)
	cpu.accumulatorRegister = 0x28
	evaluateRMWInstruction(t, cpu, ram, 6, "Z", 0xD000, 0x02)
}

func TestActionTSB(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0x0010, 0x00)

	ram.Poke(0xD000, 0xAE)

	ram.Poke(0xC000, 0x04) // TSB $10
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC002, 0x04) // TSB $10
	ram.Poke(0xC003, 0x10)
	ram.Poke(0xC004, 0x0C) // TSB $D000
	ram.Poke(0xC005, 0x00)
	ram.Poke(0xC006, 0xD0)

	cpu.accumulatorRegister = 0xAA
	evaluateRMWInstruction(t, cpu, ram, 5, "z", 0x0010, 0xAA)
	cpu.accumulatorRegister = 0x84
	evaluateRMWInstruction(t, cpu, ram, 5, "Z", 0x0010, 0xAE)
	cpu.accumulatorRegister = 0x51
	evaluateRMWInstruction(t, cpu, ram, 6, "z", 0xD000, 0xFF)
}
