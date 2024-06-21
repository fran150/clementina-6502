package cpu

const (
	MicroInstructionTypeSource      = 0x000F
	MicroInstructionTypeDestination = 0x00F0
	MicroInstructionTypeArithmetic  = 0x0F00
	MicroInstructionTypeAction      = 0xF000
)

type MicroInstruction uint16

const (
	ReadFromProgramCounter      MicroInstruction = 0x0001 // 0001
	ReadFromInstructionRegister MicroInstruction = 0x0002 // 0010
	ReadFromNextAddressOnBus    MicroInstruction = 0x0004 // 0100
	ReadFromStackPointer        MicroInstruction = 0x0008 // 1000

	IntoOpCode                 MicroInstruction = 0x0010 // 0001
	IntoDataRegister           MicroInstruction = 0x0020 // 0010
	IntoInstructionRegisterLSB MicroInstruction = 0x0040 // 0100
	IntoInstructionRegisterMSB MicroInstruction = 0x0080 // 1000

	IncrementAddressBus             MicroInstruction = 0x0100 // 0001
	AddXToInstructionRegister       MicroInstruction = 0x0200 // 0010
	AddYToInstructionRegister       MicroInstruction = 0x0400 // 0100
	AddXToInstructionRegisterLSB    MicroInstruction = 0x0900 // 1001
	AddYToInstructionRegisterLSB    MicroInstruction = 0x0A00 // 1010
	AddDataRegisterToProgramCounter MicroInstruction = 0x0C00 // 1100

	CycleAction     MicroInstruction = 0x1000 // 0001
	CycleExtra      MicroInstruction = 0x2000 // 0010
	CycleWriteToBus MicroInstruction = 0x4000 // 0100
	CycleSkip       MicroInstruction = 0x8000 // 1000
)

/*
	See http://www.6502.org/users/obelisk/65C02/addressing.html for details on the address modes.

	According to the official documentation: https://www.westerndesigncenter.com/wdc/documentation/w65c02s.pdf
	Read-Modify-Write (RMW) instructions should add 2 cycles.

	It seems that what the chip internally does is double write (first unchanged value and then incremented)
    See: http://forum.6502.org/viewtopic.php?f=8&t=3648&view=previous
	Address modes suffixed with RMW contains these extra cycle for the unchanged write (CycleWriteToBus) and the update and write.
	(nitpick: according to the post above this happens differently: Write Unchaged + Internal Add and then Write to bus.
	For the emulation we do Write Unchanged and then Internal Add + Write. This doesn't change the outcome as seen from outside the cpu)

	It seems that for RMW instructions the old 6502 has a bug in where the extra cycle is used even if there is no page boundary crossing
    This was fixed in 65C02S for ROR, ROL, ASL, LSR instructions but not for INC and DEC
    In 65C02 ROR, ROL, ASL, LSR has all 6 cycles when no boundary is crossed
    INC and DEC has always 7 cycles
    See http://forum.6502.org/viewtopic.php?p=38895#p38895
	Address modes suffixed with RMWM (Mandatory) removes the "optional" from the extra cycle. Executing it even when no page boundary is crossed
*/

type AddressMode string

const (
	AddressModeImplicit                 AddressMode = "IMP"
	AddressModeAccumulator              AddressMode = "ACC"
	AddressModeImmediate                AddressMode = "IMM"
	AddressModeZeroPage                 AddressMode = "ZPP"
	AddressModeZeroPageRMW              AddressMode = "ZPPRMW"
	AddressModeZeroPageX                AddressMode = "ZPX"
	AddressModeZeroPageXRMW             AddressMode = "ZPXRMW"
	AddressModeZeroPageY                AddressMode = "ZPY"
	AddressModeRelative                 AddressMode = "REL"
	AddressModeAbsolute                 AddressMode = "ABS"
	AddressModeAbsoluteRMW              AddressMode = "ABSRMW"
	AddressModeAbsoluteX                AddressMode = "ABX"
	AddressModeAbsoluteXRMW             AddressMode = "ABXRMW"
	AddressModeAbsoluteXRMWM            AddressMode = "ABXRMWM"
	AddressModeAbsoluteY                AddressMode = "ABY"
	AddressModeIndirect                 AddressMode = "IND"
	AddressModeIndirectZeroPage         AddressMode = "INZ"
	AddressModeZeroPageIndexedIndirectX AddressMode = "IXN"
	AddressModeZeroPageIndirectIndexedY AddressMode = "INX"
	AddressModeAbsoluteIndexedIndirect  AddressMode = "AXI"
	AddressModePushStack                AddressMode = "PHS"
	AddressModePullStack                AddressMode = "PLS"
	AddressModeAbsoluteJump             AddressMode = "JMP"
	AddressModeReturnFromSubroutine     AddressMode = "RTS"
	AddressModeReturnFromInterrupt      AddressMode = "RTI"
	AddressModeReturnFromBreak          AddressMode = "BRK"
)

// ----------------------------------------------------------------------

type AddressModeData struct {
	name              AddressMode
	text              string
	format            string
	microInstructions []MicroInstruction
	memSize           uint8
}

func (data *AddressModeData) Name() AddressMode {
	return data.name
}

func (data *AddressModeData) Text() string {
	return data.text
}

func (data *AddressModeData) Format() string {
	return data.format
}

func (data *AddressModeData) MicroInstruction(index int) MicroInstruction {
	return data.microInstructions[index]
}

func (data *AddressModeData) Cycles() int {
	return len(data.microInstructions) + 1
}

func (data *AddressModeData) MemSize() uint8 {
	return data.memSize
}

// ----------------------------------------------------------------------

type AddressModeSet struct {
	nameIndex map[AddressMode]*AddressModeData
}

func (addressModeSet *AddressModeSet) GetByName(name AddressMode) *AddressModeData {
	return addressModeSet.nameIndex[name]
}

func CreateAddressModesSet() *AddressModeSet {
	addressModeSet := AddressModeSet{
		nameIndex: map[AddressMode]*AddressModeData{},
	}

	// TODO: Also, STA a,x and BRK seems to have special handling according to offical manual

	data := []AddressModeData{
		{AddressModeImplicit, "i", "", []MicroInstruction{CycleAction}, 1},
		{AddressModeAccumulator, "A", "a", []MicroInstruction{CycleAction}, 1},
		{AddressModeImmediate, "#", "#%#x", []MicroInstruction{ReadFromProgramCounter | IntoDataRegister | CycleAction}, 2},
		{AddressModeZeroPage, "zp", "$%#x", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromInstructionRegister | IntoDataRegister | CycleAction}, 2},
		{AddressModeZeroPageRMW, "zp", "$%#x", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromInstructionRegister | IntoDataRegister, CycleWriteToBus, CycleAction}, 2},
		{AddressModeZeroPageX, "zp,x", "$%#x, X", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB | AddXToInstructionRegisterLSB, ReadFromInstructionRegister | IntoDataRegister, CycleAction}, 2},
		{AddressModeZeroPageXRMW, "zp,x", "$%#x, X", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB | AddXToInstructionRegisterLSB, ReadFromInstructionRegister | IntoDataRegister, ReadFromInstructionRegister | IntoDataRegister, CycleWriteToBus, CycleAction}, 2},
		{AddressModeZeroPageY, "zp,y", "$%#x, Y", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB | AddYToInstructionRegisterLSB, ReadFromInstructionRegister | IntoDataRegister, CycleAction}, 2},
		{AddressModeRelative, "r", "%#x", []MicroInstruction{ReadFromProgramCounter | IntoDataRegister | CycleAction, AddDataRegisterToProgramCounter | CycleExtra, CycleExtra}, 2},
		{AddressModeAbsolute, "a", "$%#x", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromProgramCounter | IntoInstructionRegisterMSB, ReadFromInstructionRegister | IntoDataRegister | CycleAction}, 3},
		{AddressModeAbsoluteRMW, "a", "$%#x", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromProgramCounter | IntoInstructionRegisterMSB, ReadFromInstructionRegister | IntoDataRegister, CycleWriteToBus, CycleAction}, 3},
		{AddressModeAbsoluteX, "a,x", "$%#x, X", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromProgramCounter | IntoInstructionRegisterMSB | AddXToInstructionRegister, ReadFromInstructionRegister | IntoDataRegister | CycleExtra, ReadFromInstructionRegister | IntoDataRegister | CycleAction}, 3},
		{AddressModeAbsoluteXRMW, "a,x", "$%#x, X", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromProgramCounter | IntoInstructionRegisterMSB | AddXToInstructionRegister, ReadFromInstructionRegister | IntoDataRegister | CycleExtra, CycleWriteToBus, CycleAction}, 3},
		{AddressModeAbsoluteXRMWM, "a,x", "$%#x, X", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromProgramCounter | IntoInstructionRegisterMSB | AddXToInstructionRegister, ReadFromInstructionRegister | IntoDataRegister, CycleWriteToBus, CycleAction}, 3},
		{AddressModeAbsoluteY, "a,y", "$%#x, Y", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromProgramCounter | IntoInstructionRegisterMSB | AddYToInstructionRegister, ReadFromInstructionRegister | IntoDataRegister | CycleExtra, ReadFromInstructionRegister | IntoDataRegister | CycleAction}, 3},
		{AddressModeIndirect, "(a)", "($%#x)", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromProgramCounter | IntoInstructionRegisterMSB, ReadFromInstructionRegister | IntoInstructionRegisterLSB, ReadFromNextAddressOnBus | IntoInstructionRegisterMSB | CycleAction}, 3},
		{AddressModeIndirectZeroPage, "(zp)", "($%#x)", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromInstructionRegister | IntoInstructionRegisterLSB, ReadFromNextAddressOnBus | IntoInstructionRegisterMSB, ReadFromInstructionRegister | IntoDataRegister, CycleAction}, 2},
		{AddressModeZeroPageIndexedIndirectX, "(zp,x)", "($%#x, X)", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB | AddXToInstructionRegister, ReadFromInstructionRegister | IntoInstructionRegisterLSB, ReadFromNextAddressOnBus | IntoInstructionRegisterMSB, ReadFromInstructionRegister | IntoDataRegister, CycleAction}, 2},
		{AddressModeZeroPageIndirectIndexedY, "(zp),y", "($%#x), Y", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromInstructionRegister | IntoInstructionRegisterLSB, ReadFromNextAddressOnBus | IntoInstructionRegisterMSB | AddYToInstructionRegister, ReadFromInstructionRegister | IntoDataRegister | CycleExtra, ReadFromInstructionRegister | IntoDataRegister | CycleAction}, 2},
		{AddressModeAbsoluteIndexedIndirect, "(a,x)", "($%#x, X)", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromProgramCounter | IntoInstructionRegisterMSB | AddXToInstructionRegister, ReadFromInstructionRegister | IntoInstructionRegisterLSB, ReadFromNextAddressOnBus | IntoInstructionRegisterMSB, ReadFromInstructionRegister | IntoDataRegister | CycleAction}, 3},
		{AddressModePushStack, "(a,x)", "($%#x, X)", []MicroInstruction{CycleAction, CycleWriteToBus}, 1},
		{AddressModePullStack, "(a,x)", "($%#x, X)", []MicroInstruction{CycleSkip, ReadFromStackPointer | IntoDataRegister, CycleAction}, 1},
		{AddressModeAbsoluteJump, "a", "$%#x", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromProgramCounter | IntoInstructionRegisterMSB | CycleAction}, 3},
		{AddressModeReturnFromInterrupt, "a", "$%#x", []MicroInstruction{CycleSkip, ReadFromStackPointer | IntoInstructionRegisterLSB, ReadFromStackPointer | IntoInstructionRegisterMSB | CycleAction, ReadFromStackPointer | IntoInstructionRegisterLSB, ReadFromStackPointer | IntoInstructionRegisterMSB | CycleAction}, 1},
		{AddressModeReturnFromSubroutine, "a", "$%#x", []MicroInstruction{CycleSkip, ReadFromStackPointer | IntoInstructionRegisterLSB, ReadFromStackPointer | IntoInstructionRegisterMSB | CycleAction, ReadFromStackPointer | IntoInstructionRegisterLSB, ReadFromStackPointer | IntoInstructionRegisterMSB | CycleAction}, 1},
		// RTI, RTS 6
		// STP 3
		// WAI 3
		// BRK 7
		//

	}

	for _, addressMode := range data {
		addressModeSet.nameIndex[addressMode.name] = &addressMode
	}

	return &addressModeSet
}
