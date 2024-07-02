package cpu

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
	AddressModeImplicit                  AddressMode = "IMP"
	AddressModeAccumulator               AddressMode = "ACC"
	AddressModeImmediate                 AddressMode = "IMM"
	AddressModeZeroPage                  AddressMode = "ZPP"
	AddressModeZeroPageRMW               AddressMode = "ZPPRMW"
	AddressModeZeroPageW                 AddressMode = "ZPPW"
	AddressModeZeroPageX                 AddressMode = "ZPX"
	AddressModeZeroPageXRMW              AddressMode = "ZPXRMW"
	AddressModeZeroPageXW                AddressMode = "ZPXW"
	AddressModeZeroPageY                 AddressMode = "ZPY"
	AddressModeAbsolute                  AddressMode = "ABS"
	AddressModeAbsoluteRMW               AddressMode = "ABSRMW"
	AddressModeAbsoluteW                 AddressMode = "ABSW"
	AddressModeAbsoluteX                 AddressMode = "ABX"
	AddressModeAbsoluteXRMW              AddressMode = "ABXRMW"
	AddressModeAbsoluteXW                AddressMode = "ABXW"
	AddressModeAbsoluteY                 AddressMode = "ABY"
	AddressModeAbsoluteYW                AddressMode = "ABYW"
	AddressModeRelative                  AddressMode = "REL"
	AddressModeIndirect                  AddressMode = "IND"
	AddressModeIndirectZeroPage          AddressMode = "INZ"
	AddressModeIndirectZeroPageW         AddressMode = "INZW"
	AddressModeZeroPageIndexedIndirectX  AddressMode = "IXN"
	AddressModeZeroPageIndexedIndirectXW AddressMode = "IXNW"
	AddressModeZeroPageIndirectIndexedY  AddressMode = "INY"
	AddressModeZeroPageIndirectIndexedYW AddressMode = "INYW"
	AddressModeAbsoluteIndexedIndirect   AddressMode = "AXI"
	AddressModePushStack                 AddressMode = "PHS"
	AddressModePullStack                 AddressMode = "PLS"
	AddressModeAbsoluteJump              AddressMode = "JMP"
	AddressModeJumpToSubroutine          AddressMode = "JSR"
	AddressModeReturnFromSubroutine      AddressMode = "RTS"
	AddressModeBreak                     AddressMode = "BRK"
	AddressModeReturnFromInterrupt       AddressMode = "RTI"
	AddressModeStop                      AddressMode = "STP"
	AddressModeWait                      AddressMode = "WAI"
)

// ----------------------------------------------------------------------

type AddressModeData struct {
	name              AddressMode
	text              string
	format            string
	microInstructions []cycleActions
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

func (data *AddressModeData) Cycle(index int) cycleActions {
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

	data := []AddressModeData{
		{AddressModeImplicit, "i", "", addressModeImplicitOrAccumulatorActions, 1},
		{AddressModeAccumulator, "A", "a", addressModeImplicitOrAccumulatorActions, 1},
		{AddressModeImmediate, "#", "#%02X", addressModeImmediateActions, 2},
		{AddressModeAbsoluteJump, "a", "$%04X", addressModeAbsoluteJumpActions, 3},
		{AddressModeAbsolute, "a", "$%04X", addressModeAbsoluteActions, 3},
		{AddressModeAbsoluteRMW, "a", "$%04X", addressModeAbsoluteRMWActions, 3},
		{AddressModeAbsoluteW, "a", "$%04X", addressModeAbsoluteWActions, 3},
		{AddressModeZeroPage, "zp", "$%02X", addressModeZeroPageActions, 2},
		{AddressModeZeroPageRMW, "zp", "$%02X", addressModeZeroPageRMWActions, 2},
		{AddressModeZeroPageW, "zp", "$%02X", addressModeZeroPageWActions, 2},
		{AddressModeZeroPageX, "zp,x", "$%02X, X", addressModeZeroPageXActions, 2},
		{AddressModeZeroPageXRMW, "zp,x", "$%02X, X", addressModeZeroPageXRMWActions, 2},
		{AddressModeZeroPageXW, "zp,x", "$%02X, X", addressModeZeroPageXWActions, 2},
		{AddressModeZeroPageY, "zp,y", "$%02X, Y", addressModeZeroPageYActions, 2},
		{AddressModeAbsoluteX, "a,x", "$%04X, X", addressModeAbsoluteXActions, 3},
		{AddressModeAbsoluteXRMW, "a,x", "$%04X, X", addressModeAbsoluteXRMWActions, 3},
		{AddressModeAbsoluteXW, "a,x", "$%04X, X", addressModeAbsoluteXWActions, 3},
		{AddressModeAbsoluteY, "a,y", "$%04X, Y", addressModeAbsoluteYActions, 3},
		{AddressModeAbsoluteYW, "a,y", "$%04X, Y", addressModeAbsoluteYWActions, 3},
		{AddressModeRelative, "r", "$%02X", addressModeRelativeActions, 2},
		{AddressModeZeroPageIndexedIndirectX, "(zp,x)", "($%02X, X)", addressModeZeroPageIndexedIndirectXActions, 2},
		{AddressModeZeroPageIndexedIndirectXW, "(zp,x)", "($%02X, X)", addressModeZeroPageIndexedIndirectXWActions, 2},
		{AddressModeZeroPageIndirectIndexedY, "(zp),y", "($%02X), Y", addressModeZeroPageIndirectIndexedYActions, 2},
		{AddressModeZeroPageIndirectIndexedYW, "(zp),y", "($%02X), Y", addressModeZeroPageIndirectIndexedYWActions, 2},
		{AddressModeIndirect, "(a)", "($%04X)", addressModeIndirectActions, 3},
		{AddressModeIndirectZeroPage, "(zp)", "($%02X)", addressModeIndirectZeroPageActions, 2},
		{AddressModeIndirectZeroPageW, "(zp)", "($%02X)", addressModeIndirectZeroPageWActions, 2},
		{AddressModeAbsoluteIndexedIndirect, "(a,x)", "($%04X, X)", addressModeAbsoluteIndexedIndirectActions, 3},
		{AddressModePushStack, "i", "", addressModePushStackActions, 1},
		{AddressModePullStack, "i", "", addressModePullStackActions, 1},
		{AddressModeBreak, "i", "", addressModeBreakActions, 2},
		{AddressModeReturnFromInterrupt, "", "", addressModeReturnFromInterruptActions, 1},
		{AddressModeJumpToSubroutine, "a", "$%04X", addressModeJumpToSubroutineActions, 3},
		{AddressModeReturnFromSubroutine, "i", "", addressModeReturnFromSubroutineActions, 1},

		// Pending implementation

		{AddressModeStop, "i", "", []cycleActions{}, 1},
		{AddressModeWait, "i", "", []cycleActions{}, 1},
	}

	for _, addressMode := range data {
		addressModeSet.nameIndex[addressMode.name] = &addressMode
	}

	return &addressModeSet
}
