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
	ReadFromStackPointer        MicroInstruction = 0x0004 // 0100
	ReadFromNextAddressInBus    MicroInstruction = 0x0008 // 1000

	IntoOpCode                 MicroInstruction = 0x0010 // 0001
	IntoDataRegister           MicroInstruction = 0x0020 // 0010
	IntoInstructionRegisterLSB MicroInstruction = 0x0040 // 0100
	IntoInstructionRegisterMSB MicroInstruction = 0x0080 // 1000

	AddXToInstructionRegister    MicroInstruction = 0x0200 // 0010
	AddYToInstructionRegister    MicroInstruction = 0x0300 // 0011
	AddXToInstructionRegisterLSB MicroInstruction = 0x0400 // 0100
	AddYToInstructionRegisterLSB MicroInstruction = 0x0500 // 0101

	CycleSkip                           MicroInstruction = 0x0000
	CycleAction                         MicroInstruction = 0x1000
	CycleExtra                          MicroInstruction = 0x2000
	CycleWriteDataToInstructionRegister MicroInstruction = 0x4000
	AddDataRegisterToProgramCounter     MicroInstruction = 0x8000
)

/*
	See http://www.6502.org/users/obelisk/65C02/addressing.html for details on the address modes.

	RMW modes:
	According to the official documentation: https://www.westerndesigncenter.com/wdc/documentation/w65c02s.pdf
	Read-Modify-Write (RMW) instructions should add 2 cycles.
	The original 6502 made 2 write cycles, one with the unchaged value (internally modifies the value) and then a write with the updated values.
	According to the official doc for the 65C02S the processor does one extra read and one write on effective address.
	This is explained in the 65C02 wiki: https://en.wikipedia.org/wiki/WDC_65C02#Bug_fixes

	It seems that for RMW instructions the old 6502 has a bug in where the extra cycle is used even if there is no page boundary crossing
    This was fixed in 65C02S for ROR, ROL, ASL, LSR instructions but not for INC and DEC
    In 65C02 ROR, ROL, ASL, LSR has all 6 cycles when no boundary is crossed
    INC and DEC has always 7 cycles
    See http://forum.6502.org/viewtopic.php?p=38895#p38895
	Address modes suffixed with RMWM (Mandatory) removes the "optional" from the extra cycle. Executing it even when no page boundary is crossed

	Absolute indexed modes:
	When performing indexed addressing, if indexing crosses a page boundary all NMOS variants will read from an invalid address before accessing
	the correct address. The 65C02 fixed this problem by performing a dummy read of the instruction opcode when indexing crosses a page boundary.
	A dummy read is performed on the base address prior to indexing, such that LDA $1200,X will do a dummy read on $1200 prior to the value of X
	being added to $1200
*/

type AddressMode string

const (
	AddressModeImplicit    AddressMode = "IMP"
	AddressModeAccumulator AddressMode = "ACC"
	// Tested here
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

	// TODO: Also, STA a,x and BRK seems to have special handling according to official manual

	data := []AddressModeData{
		{AddressModeImplicit, "i", "", []MicroInstruction{ReadFromNextAddressInBus + CycleAction}, 1},
		{AddressModeAccumulator, "A", "a", []MicroInstruction{ReadFromNextAddressInBus + CycleAction}, 1},
		{AddressModeImmediate, "#", "#%#x", []MicroInstruction{ReadFromProgramCounter | IntoDataRegister | CycleAction}, 2},
		{AddressModeAbsoluteJump, "a", "$%#x", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromProgramCounter | IntoInstructionRegisterMSB | CycleAction}, 3},
		{AddressModeAbsolute, "a", "$%#x", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromProgramCounter | IntoInstructionRegisterMSB, ReadFromInstructionRegister | IntoDataRegister | CycleAction}, 3},
		{AddressModeAbsoluteRMW, "a", "$%#x", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromProgramCounter | IntoInstructionRegisterMSB, ReadFromInstructionRegister | IntoDataRegister, ReadFromInstructionRegister | IntoDataRegister, CycleAction}, 3},
		{AddressModeZeroPage, "zp", "$%#x", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromInstructionRegister | IntoDataRegister | CycleAction}, 2},
		{AddressModeZeroPageRMW, "zp", "$%#x", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromInstructionRegister | IntoDataRegister, ReadFromInstructionRegister | IntoDataRegister, CycleAction}, 2},
		{AddressModeZeroPageX, "zp,x", "$%#x, X", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromInstructionRegister | IntoDataRegister | AddXToInstructionRegisterLSB, ReadFromInstructionRegister | IntoDataRegister | CycleAction}, 2},
		{AddressModeZeroPageXRMW, "zp,x", "$%#x, X", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromInstructionRegister | IntoDataRegister | AddXToInstructionRegisterLSB, ReadFromInstructionRegister | IntoDataRegister, CycleAction}, 2},
		{AddressModeZeroPageY, "zp,y", "$%#x, Y", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromInstructionRegister | IntoDataRegister | AddYToInstructionRegisterLSB, ReadFromInstructionRegister | IntoDataRegister | CycleAction}, 2},
		{AddressModeAbsoluteX, "a,x", "$%#x, X", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromProgramCounter | IntoInstructionRegisterMSB | AddXToInstructionRegister, CycleExtra, ReadFromInstructionRegister | IntoDataRegister | CycleAction}, 3},
		{AddressModeAbsoluteXRMW, "a,x", "$%#x, X", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromProgramCounter | IntoInstructionRegisterMSB | AddXToInstructionRegister, CycleExtra, ReadFromInstructionRegister | IntoDataRegister, ReadFromInstructionRegister | IntoDataRegister, CycleAction}, 3},
		{AddressModeAbsoluteY, "a,y", "$%#x, Y", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromProgramCounter | IntoInstructionRegisterMSB | AddYToInstructionRegister, CycleExtra, ReadFromInstructionRegister | IntoDataRegister | CycleAction}, 3},
		// Reviewed here
		{AddressModeRelative, "r", "%#x", []MicroInstruction{ReadFromProgramCounter | IntoDataRegister | CycleAction, AddDataRegisterToProgramCounter | CycleExtra, CycleExtra}, 2},
		{AddressModeIndirect, "(a)", "($%#x)", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromProgramCounter | IntoInstructionRegisterMSB, ReadFromInstructionRegister | IntoInstructionRegisterLSB, ReadFromNextAddressInBus | IntoInstructionRegisterMSB | CycleAction}, 3},
		{AddressModeIndirectZeroPage, "(zp)", "($%#x)", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromInstructionRegister | IntoInstructionRegisterLSB, ReadFromNextAddressInBus | IntoInstructionRegisterMSB, ReadFromInstructionRegister | IntoDataRegister, CycleAction}, 2},
		{AddressModeZeroPageIndexedIndirectX, "(zp,x)", "($%#x, X)", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB | AddXToInstructionRegister, ReadFromInstructionRegister | IntoInstructionRegisterLSB, ReadFromNextAddressInBus | IntoInstructionRegisterMSB, ReadFromInstructionRegister | IntoDataRegister, CycleAction}, 2},
		{AddressModeZeroPageIndirectIndexedY, "(zp),y", "($%#x), Y", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromInstructionRegister | IntoInstructionRegisterLSB, ReadFromNextAddressInBus | IntoInstructionRegisterMSB | AddYToInstructionRegister, ReadFromInstructionRegister | IntoDataRegister | CycleExtra, ReadFromInstructionRegister | IntoDataRegister | CycleAction}, 2},
		{AddressModeAbsoluteIndexedIndirect, "(a,x)", "($%#x, X)", []MicroInstruction{ReadFromProgramCounter | IntoInstructionRegisterLSB, ReadFromProgramCounter | IntoInstructionRegisterMSB | AddXToInstructionRegister, ReadFromInstructionRegister | IntoInstructionRegisterLSB, ReadFromNextAddressInBus | IntoInstructionRegisterMSB, ReadFromInstructionRegister | IntoDataRegister | CycleAction}, 3},
		{AddressModePushStack, "(a,x)", "($%#x, X)", []MicroInstruction{CycleAction, CycleWriteDataToInstructionRegister}, 1},
		{AddressModePullStack, "(a,x)", "($%#x, X)", []MicroInstruction{CycleSkip, ReadFromStackPointer | IntoDataRegister, CycleAction}, 1},
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
