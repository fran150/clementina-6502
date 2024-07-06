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

func evaluateAccumulator(t *testing.T, cpu *Cpu65C02S, expected uint8) {
	if cpu.accumulatorRegister != expected {
		instruction := cpu.instructionSet.GetByOpCode(cpu.currentOpCode)
		addressMode := cpu.addressModeSet.GetByName(instruction.addressMode)

		t.Errorf("%s - %s - Current value of accumulator (%02X) doesnt match the expected value of (%02X)", instruction.Mnemonic(), addressMode.Text(), cpu.accumulatorRegister, expected)
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
	evaluateAccumulator(t, cpu, expectedAccumulatorValue)
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

	evaluateAccumulatorInstruction(t, cpu, ram, 2, "zn", 0x0A) // i -> 0 + 0A = 0A
	evaluateAccumulatorInstruction(t, cpu, ram, 3, "zn", 0x0C) // zp -> 0A + 02 = 0C
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zN", 0xAC) // zp,x -> 0C + A0 = AC
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zN", 0xBC) // a -> AC + 10 = BC
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zN", 0xCD) // a,x -> BC + 11 = CD
	evaluateAccumulatorInstruction(t, cpu, ram, 4, "zN", 0xED) // a,y -> CD + 20 = ED
	evaluateAccumulatorInstruction(t, cpu, ram, 6, "zN", 0xEF) // (zp,x) -> ED + 02 = EF
	evaluateAccumulatorInstruction(t, cpu, ram, 6, "zN", 0xFF) // (zp),y -> EF + 10 = FF
	evaluateAccumulatorInstruction(t, cpu, ram, 5, "Zn", 0x00) // (zp) -> FF + 1 = 00
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

	ram.Poke(0x00A5, 0x10) // (zp,x) redirect to $D110
	ram.Poke(0x00A6, 0xD1)

	ram.Poke(0x00B0, 0xFF) // (zp),y redirect to $D2FF
	ram.Poke(0x00B1, 0xD2)

	ram.Poke(0x00C0, 0x00) // (zp) redict to $E000
	ram.Poke(0x00C1, 0xE0)

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
	ram.Poke(0xC012, 0x90) // BCC $F0
	ram.Poke(0xC013, 0xF0)
	ram.Poke(0xC104, 0x90) // BCC $FF (not taken)
	ram.Poke(0xC105, 0xFF)
	ram.Poke(0xC106, 0xEA) // NOP

	evaluateBranchInstruction(t, cpu, ram, 3, "c", 0xC012)
	evaluateBranchInstruction(t, cpu, ram, 4, "c", 0xC104)
	cpu.processorStatusRegister.SetFlag(CarryFlagBit, true)
	evaluateBranchInstruction(t, cpu, ram, 2, "C", 0xC106)
	evaluateBranchInstruction(t, cpu, ram, 2, "z", 0xC107)
}

func TestActionBCS(t *testing.T) {
	cpu, ram := createComputer()

	cpu.processorStatusRegister.SetFlag(CarryFlagBit, true)

	ram.Poke(0xC000, 0xB0) // BCS $10
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC012, 0xB0) // BCS $F0
	ram.Poke(0xC013, 0xF0)
	ram.Poke(0xC104, 0xB0) // BCS $FF (not taken)
	ram.Poke(0xC105, 0xFF)
	ram.Poke(0xC106, 0xEA) // NOP

	evaluateBranchInstruction(t, cpu, ram, 3, "C", 0xC012)
	evaluateBranchInstruction(t, cpu, ram, 4, "C", 0xC104)
	cpu.processorStatusRegister.SetFlag(CarryFlagBit, false)
	evaluateBranchInstruction(t, cpu, ram, 2, "c", 0xC106)
	evaluateBranchInstruction(t, cpu, ram, 2, "z", 0xC107)
}

func TestActionBEQ(t *testing.T) {
	cpu, ram := createComputer()

	cpu.processorStatusRegister.SetFlag(ZeroFlagBit, true)

	ram.Poke(0xC000, 0xF0) // BEQ $10
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC012, 0xF0) // BEQ $F0
	ram.Poke(0xC013, 0xF0)
	ram.Poke(0xC104, 0xF0) // BEQ $FF (not taken)
	ram.Poke(0xC105, 0xFF)
	ram.Poke(0xC106, 0xEA) // NOP

	evaluateBranchInstruction(t, cpu, ram, 3, "Z", 0xC012)
	evaluateBranchInstruction(t, cpu, ram, 4, "Z", 0xC104)
	cpu.processorStatusRegister.SetFlag(ZeroFlagBit, false)
	evaluateBranchInstruction(t, cpu, ram, 2, "z", 0xC106)
	evaluateBranchInstruction(t, cpu, ram, 2, "z", 0xC107)
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
	ram.Poke(0xC012, 0x30) // BMI $F0
	ram.Poke(0xC013, 0xF0)
	ram.Poke(0xC104, 0x30) // BMI $FF (not taken)
	ram.Poke(0xC105, 0xFF)
	ram.Poke(0xC106, 0xEA) // NOP

	evaluateBranchInstruction(t, cpu, ram, 3, "N", 0xC012)
	evaluateBranchInstruction(t, cpu, ram, 4, "N", 0xC104)
	cpu.processorStatusRegister.SetFlag(NegativeFlagBit, false)
	evaluateBranchInstruction(t, cpu, ram, 2, "n", 0xC106)
	evaluateBranchInstruction(t, cpu, ram, 2, "n", 0xC107)
}

func TestActionBNE(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0xD0) // BNE $10
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC012, 0xD0) // BNE $F0
	ram.Poke(0xC013, 0xF0)
	ram.Poke(0xC104, 0xD0) // BNE $FF (not taken)
	ram.Poke(0xC105, 0xFF)
	ram.Poke(0xC106, 0xEA) // NOP

	evaluateBranchInstruction(t, cpu, ram, 3, "z", 0xC012)
	evaluateBranchInstruction(t, cpu, ram, 4, "z", 0xC104)
	cpu.processorStatusRegister.SetFlag(ZeroFlagBit, true)
	evaluateBranchInstruction(t, cpu, ram, 2, "Z", 0xC106)
	evaluateBranchInstruction(t, cpu, ram, 2, "Z", 0xC107)
}

func TestActionBPL(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0x10) // BPL $10
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC012, 0x10) // BPL $F0
	ram.Poke(0xC013, 0xF0)
	ram.Poke(0xC104, 0x10) // BPL $FF (not taken)
	ram.Poke(0xC105, 0xFF)
	ram.Poke(0xC106, 0xEA) // NOP

	evaluateBranchInstruction(t, cpu, ram, 3, "n", 0xC012)
	evaluateBranchInstruction(t, cpu, ram, 4, "n", 0xC104)
	cpu.processorStatusRegister.SetFlag(NegativeFlagBit, true)
	evaluateBranchInstruction(t, cpu, ram, 2, "N", 0xC106)
	evaluateBranchInstruction(t, cpu, ram, 2, "N", 0xC107)
}

func TestActionBRA(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xC000, 0x80) // BRA $10
	ram.Poke(0xC001, 0x10)
	ram.Poke(0xC012, 0x80) // BRA $F0
	ram.Poke(0xC013, 0xF0)

	evaluateBranchInstruction(t, cpu, ram, 3, "n", 0xC012)
	evaluateBranchInstruction(t, cpu, ram, 4, "n", 0xC104)
}

func TestActionBRK(t *testing.T) {
	cpu, ram := createComputer()

	ram.Poke(0xFFFE, 0x00)
	ram.Poke(0xFFFF, 0xD0)

	ram.Poke(0xD000, 0xA9) // LDA #$FF
	ram.Poke(0xD001, 0xFF)
	ram.Poke(0xD002, 0x40) // RTI

	ram.Poke(0xC000, 0x00) // BRK
	ram.Poke(0xC001, 0xA9) // LDA #$77
	ram.Poke(0xC002, 0x77)

	evaluateBranchInstruction(t, cpu, ram, 7, "B", 0xD000)
	//evaluateBranchInstruction(t, cpu, ram, 4, "n", 0xC104)
}
