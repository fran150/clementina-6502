package cpu

// In 6502 family of processors each instruction has different address modes. This type represents the internal name of each
// address mode. Each address mode has different number of cycles depending on other factors, for this emulation these has been
// identified with different names.
//
//   - Read / Modify / Write: Instructions such as INC or DEC take 3 extra cycles to do the updates. These are marked with the RMW suffix.
//     For example: AddressModeZeroPageRMW is the RMW version of the Zero Page address mode. The original 6502 made 2 write cycles, one with
//     the unchaged value (internally modifies the value) and then a write with the updated values. According to the official doc for the 65C02S the
//     processor does one extra read and one write on effective address.
//     This is explained in the 65C02 wiki: https://en.wikipedia.org/wiki/WDC_65C02#Bug_fixes
//   - Write instructions: This emulation uses 2 functions to emulate each cycle, one "tick" function to allow set up the state of the buses and lines
//     and a second "PostTick" function to allow the compoenent react to these values. In normal "Read" instructions the bus is set to read in the
//     tick function and the registers updates are performed in the post tick. In "write" instructions such as STA, STX and STY, the bus must be set
//     on the initial tick to let the ram update the value on the post tick function. These are marked by a W suffix. For example,
//     AddressModeZeroPageW is the zero page address mode used for STA.
//   - Specific types: Instructions that handle stack pointer such as PHP, PLA, etc have specific cycles and are handled by different address modes.
//     The same applies to instructions that handle storing the program counter before jumping to a specific address and returning such as JSR and RTS.
//     These are marked with specific names for their address modes although in official doc are marked as SP, Absolute modes or similar.
//   - Relative Extended mode: Although marked as relative address mode in the official doc, the new 65C02's BBS and BBR instructions have an specific
//     cycle type. See discussions about BBR and BBS correct timing here:
//     https://www.reddit.com/r/beneater/comments/1cac3ly/clarification_of_65c02_instruction_execution_times/
//   - Non instructions address modes: Finally, in this emulation we have marked as "address modes" the cycles executed when a RESET, NMI or IRQ
//     interrupt is triggered.
//
// For more information about the cycles for regular 6502 processor see: https://www.atarihq.com/danb/files/64doc.txt
//
// Some notes for 65C02 processor differences with respect to regular 6502:
//   - When performing indexed addressing, if indexing crosses a page boundary all NMOS variants will read from an invalid address before accessing
//     the correct address. The 65C02 fixed this problem by performing a dummy read of the instruction opcode when indexing crosses a page boundary.
//     A dummy read is performed on the base address prior to indexing, such that LDA $1200,X will do a dummy read on $1200 prior to the value of X
//     being added to $1200.
//   - The original 6502 had a bug with indirect addressing when there was a page boundary crossing. For example, JMP ($12FF) will read the LSB of
//     the target address from $12FF correctly but will fetch the MSB from $1200. The 65C02 corrected this bug reading from $1300 instead as expected.
//   - On 6502 RMW instructions execute an extra write of the unmodified value, before writing the updated value in the next cycle. The 65C02 corrected
//     this problem by replacing the extra write with an extra read. On 6502, these instructions take additional cycle even if there is no page boundary
//     crossed. In the 65C02 these instructions take 1 less cycle if the target is on the same page.
//   - When performing indexed addressing, if indexing crosses a page boundary original 6502 will read from an invalid address before accessing the
//     correct address. For example,
type AddressMode int

const (
	AddressModeImplicit AddressMode = iota
	AddressModeAccumulator
	AddressModeImmediate
	AddressModeZeroPage
	AddressModeZeroPageRMW
	AddressModeZeroPageW
	AddressModeZeroPageX
	AddressModeZeroPageXRMW
	AddressModeZeroPageXW
	AddressModeZeroPageY
	AddressModeZeroPageYW
	AddressModeAbsolute
	AddressModeAbsoluteRMW
	AddressModeAbsoluteW
	AddressModeAbsoluteX
	AddressModeAbsoluteXRMW
	AddressModeAbsoluteXW
	AddressModeAbsoluteY
	AddressModeAbsoluteYW
	AddressModeRelative
	AddressModeIndirect
	AddressModeIndirectZeroPage
	AddressModeIndirectZeroPageW
	AddressModeZeroPageIndexedIndirectX
	AddressModeZeroPageIndexedIndirectXW
	AddressModeZeroPageIndirectIndexedY
	AddressModeZeroPageIndirectIndexedYW
	AddressModeAbsoluteIndexedIndirect
	AddressModePushStack
	AddressModePullStack
	AddressModeAbsoluteJump
	AddressModeJumpToSubroutine
	AddressModeReturnFromSubroutine
	AddressModeBreak
	AddressModeReturnFromInterrupt
	AddressModeRelativeExtended

	// Non instructions address modes
	AddressModeIRQ
	AddressModeNMI
	AddressModeReset
)

// ----------------------------------------------------------------------

// Stores useful data for the different address modes
type AddressModeData struct {
	name              AddressMode
	text              string
	format            string
	microInstructions []cycleActions
	memSize           uint8
}

// Returns the name of the address mode
func (data *AddressModeData) Name() AddressMode {
	return data.name
}

// Returns the typical text abbreviation of the address mode found on manuals or books.
func (data *AddressModeData) Text() string {
	return data.text
}

// Returns an string usable with fmt.Printf function to write the assembler version of the
// instruction being executed.
func (data *AddressModeData) Format() string {
	return data.format
}

// Returns data about the current cycle.
func (data *AddressModeData) cycle(index int) cycleActions {
	return data.microInstructions[index]
}

// Returns the number of cycles of the current instruction.
func (data *AddressModeData) Cycles() int {
	return len(data.microInstructions) + 1
}

// Returns the number of bytes required to read to execute the instruction in this address mode.
func (data *AddressModeData) MemSize() uint8 {
	return data.memSize
}

// ----------------------------------------------------------------------

// Set of address modes supported by the processor.
type AddressModeSet struct {
	nameIndex [40]*AddressModeData
}

// Gets address mode data of an specific mode by it's name.
func (addressModeSet *AddressModeSet) GetByName(name AddressMode) *AddressModeData {
	return addressModeSet.nameIndex[name]
}

// Creates and returns the list of address modes supported by this processor.
func CreateAddressModesSet() *AddressModeSet {
	addressModeSet := AddressModeSet{
		nameIndex: [40]*AddressModeData{},
	}

	data := []AddressModeData{

		// 6502 Standard address modes and variants

		{AddressModeImplicit, "i", "", addressModeImplicitOrAccumulatorActions, 1},
		{AddressModeAccumulator, "A", "a", addressModeImplicitOrAccumulatorActions, 1},
		{AddressModeImmediate, "#", "#$%02X", addressModeImmediateActions, 2},
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
		{AddressModeZeroPageYW, "zp,y", "$%02X, Y", addressModeZeroPageYWActions, 2},
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

		// Non Instruction address modes. These are used to handle interrupt request and are similar to BRK
		{AddressModeIRQ, "irq", "", addressModeIRQActions, 0},
		{AddressModeNMI, "nmi", "", addressModeNMIActions, 0},
		{AddressModeReset, "reset", "", addressModeResetActions, 0},

		// See discussions about BBR and BBS correct timing here:
		// https://www.reddit.com/r/beneater/comments/1cac3ly/clarification_of_65c02_instruction_execution_times/
		{AddressModeRelativeExtended, "zp, r", "$%02x, $%02X", addressModeRelativeExtendedActions, 3},
	}

	for _, addressMode := range data {
		addressModeSet.nameIndex[addressMode.name] = &addressMode
	}

	return &addressModeSet
}

// Returns details about the specified address mode
func GetAddressMode(name AddressMode) *AddressModeData {
	return addressModeSet.GetByName(name)
}
