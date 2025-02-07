package ui

import (
	"fmt"
	"io"

	"github.com/fran150/clementina6502/pkg/components/cpu"
	"github.com/fran150/clementina6502/pkg/components/lcd"
	"github.com/fran150/clementina6502/pkg/components/via"
)

func getFlagStatusColor(status cpu.StatusRegister, bit cpu.StatusBit) string {
	if status.Flag(bit) {
		return "[green]"
	}

	return "[red]"
}

func ShowCPUState(writer io.Writer, processor *cpu.Cpu65C02S) {
	fmt.Fprintf(writer, "[yellow] A: [white]%5d [grey]($%02X)\n", processor.GetAccumulatorRegister(), processor.GetAccumulatorRegister())
	fmt.Fprintf(writer, "[yellow] X: [white]%5d [grey]($%02X)\n", processor.GetXRegister(), processor.GetXRegister())
	fmt.Fprintf(writer, "[yellow] Y: [white]%5d [grey]($%02X)\n", processor.GetYRegister(), processor.GetYRegister())
	fmt.Fprintf(writer, "[yellow]SP: [white]%5d [grey]($%02X)\n", processor.GetStackPointer(), processor.GetStackPointer())
	fmt.Fprintf(writer, "[yellow]PC: [white]$%04X [grey](%v)\n", processor.GetProgramCounter(), processor.GetProgramCounter())

	status := processor.GetProcessorStatusRegister()

	fmt.Fprint(writer, "[yellow]Flags: ")
	fmt.Fprintf(writer, "%sN", getFlagStatusColor(status, cpu.NegativeFlagBit))
	fmt.Fprintf(writer, "%sV", getFlagStatusColor(status, cpu.OverflowFlagBit))
	fmt.Fprintf(writer, "%s-", getFlagStatusColor(status, cpu.UnusedFlagBit))
	fmt.Fprintf(writer, "%sB", getFlagStatusColor(status, cpu.BreakCommandFlagBit))
	fmt.Fprintf(writer, "%sD", getFlagStatusColor(status, cpu.DecimalModeFlagBit))
	fmt.Fprintf(writer, "%sI", getFlagStatusColor(status, cpu.IrqDisableFlagBit))
	fmt.Fprintf(writer, "%sZ", getFlagStatusColor(status, cpu.ZeroFlagBit))
	fmt.Fprintf(writer, "%sC", getFlagStatusColor(status, cpu.CarryFlagBit))
	fmt.Fprint(writer, "\n")
}

func ShowVIAState(writer io.Writer, via *via.Via65C22S) {
	fmt.Fprintf(writer, "[yellow]VIA Registers:\n")
	fmt.Fprintf(writer, "[yellow] ORA:  [white]$%02X\n", via.GetOutputRegisterA())
	fmt.Fprintf(writer, "[yellow] ORB:  [white]$%02X\n", via.GetOutputRegisterB())
	fmt.Fprintf(writer, "[yellow] IRA:  [white]$%02X\n", via.GetInputRegisterA())
	fmt.Fprintf(writer, "[yellow] IRB:  [white]$%02X\n", via.GetInputRegisterB())
	fmt.Fprintf(writer, "[yellow] DDRA: [white]$%02X\n", via.GetDataDirectionRegisterA())
	fmt.Fprintf(writer, "[yellow] DDRB: [white]$%02X\n", via.GetDataDirectionRegisterB())
	fmt.Fprintf(writer, "[yellow] LL1:  [white]$%02X\n", via.GetLowLatches1())
	fmt.Fprintf(writer, "[yellow] HL1:  [white]$%02X\n", via.GetHighLatches1())
	fmt.Fprintf(writer, "[yellow] CTR1: [white]$%04X\n", via.GetCounter1())
	fmt.Fprintf(writer, "[yellow] LL2:  [white]$%02X\n", via.GetLowLatches2())
	fmt.Fprintf(writer, "[yellow] HL2:  [white]$%02X\n", via.GetHighLatches2())
	fmt.Fprintf(writer, "[yellow] CTR2: [white]$%04X\n", via.GetCounter2())
	fmt.Fprintf(writer, "[yellow] SR:   [white]$%02X\n", via.GetShiftRegister())
	fmt.Fprintf(writer, "[yellow] ACR:  [white]$%02X\n", via.GetAuxiliaryControl())
	fmt.Fprintf(writer, "[yellow] PCR:  [white]$%02X\n", via.GetPeripheralControl())
	fmt.Fprintf(writer, "[yellow] IFR:  [white]$%02X\n", via.GetInterruptFlagValue())
	fmt.Fprintf(writer, "[yellow] IER:  [white]$%02X\n", via.GetInterruptEnabledFlag())
	fmt.Fprintf(writer, "[yellow] Bus:  [white]$%04X\n", via.DataBus().Read())
}

func drawLcdDDRAM(writer io.Writer, displayStatus lcd.DisplayStatus) {
	const itemsPerLine = 10

	for i, data := range displayStatus.DDRAM {
		fmt.Fprintf(writer, "[yellow]%02v: [white]%s ", i, string(data))

		if i%itemsPerLine == (itemsPerLine - 1) {
			fmt.Fprintf(writer, "\n")
		}
	}
}

func drawLcdLine(writer io.Writer, lineStart uint8, displayStatus lcd.DisplayStatus, cursorStatus lcd.CursorStatus, min uint8, max uint8) {
	var count uint8 = 0
	var index uint8 = lineStart

	for count < (max - min) {
		if index >= max {
			index = min
		}

		if index == cursorStatus.CursorPosition && cursorStatus.BlinkStatusShowing {
			fmt.Fprintf(writer, "*")
		} else {
			fmt.Fprint(writer, string(displayStatus.DDRAM[index]))
		}

		index++
		count++
	}
}

func ShowLCD(sb io.Writer, lcd *lcd.LcdHD44780U) {
	const line1MinIndex, line1MaxIndex = 0, 40
	const line2MinIndex, line2MaxIndex = 40, 80

	displayStatus := lcd.GetDisplayStatus()
	cursorStatus := lcd.GetCursorStatus()

	fmt.Fprintf(sb, "LCD Screen: \n")
	drawLcdLine(sb, lcd.GetDisplayStatus().Line1Start, displayStatus, cursorStatus, line1MinIndex, line1MaxIndex)
	fmt.Fprint(sb, "\n")
	drawLcdLine(sb, lcd.GetDisplayStatus().Line2Start, displayStatus, cursorStatus, line2MinIndex, line2MaxIndex)
	fmt.Fprint(sb, "\n\n")
}

func ShowLCDState(sb io.Writer, lcd *lcd.LcdHD44780U) {
	cursorStatus := lcd.GetCursorStatus()
	displayStatus := lcd.GetDisplayStatus()

	fmt.Fprintf(sb, "LCD Memory:\n")
	drawLcdDDRAM(sb, displayStatus)

	fmt.Fprintf(sb, "Display ON: %v\n", displayStatus.DisplayOn)
	fmt.Fprintf(sb, "8 Bit Mode: %v\n", displayStatus.Is8BitMode)
	fmt.Fprintf(sb, "Line 2 display: %v\n", displayStatus.Is2LineDisplay)
	fmt.Fprintf(sb, "Cursor Position: %v\n", cursorStatus.CursorPosition)
	fmt.Fprintf(sb, "Bus: %v\n", lcd.DataBus().Read())
	fmt.Fprintf(sb, "E: %v\n", lcd.Enable().Enabled())
	fmt.Fprintf(sb, "RW: %v\n", lcd.ReadWrite().Enabled())
	fmt.Fprintf(sb, "RS: %v\n", lcd.RegisterSelect().Enabled())
}

func ShowCurrentInstruction(sb io.Writer, programCounter uint16, instruction *cpu.CpuInstructionData, potentialOperands [2]uint8) {
	addressModeDetails := cpu.GetAddressMode(instruction.AddressMode())

	size := addressModeDetails.MemSize() - 1

	// Write current address
	fmt.Fprintf(sb, "[blue]$%04X: [red]%s [white]", programCounter, instruction.Mnemonic())

	// Write operands
	switch size {
	case 0:
	case 1:
		fmt.Fprintf(sb, addressModeDetails.Format(), potentialOperands[0])
	case 2:
		msb := uint16(potentialOperands[1]) << 8
		lsb := uint16(potentialOperands[0])
		fmt.Fprintf(sb, addressModeDetails.Format(), msb|lsb)
	default:
		fmt.Fprintf(sb, "Unrecognized Instruction or Address Mode")
	}

	fmt.Fprint(sb, "\r\n")
}
