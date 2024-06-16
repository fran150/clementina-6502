package cpu

type ExtraCyclesType int

const (
	None            = 0
	PageBoundary    = 1
	BranchTaken     = 1
	ReadModifyWrite = 2
)

type AddressModeData struct {
	CycleCount  uint8
	Cycles      []CycleType
	ExtraCycles []ExtraCyclesType
	MemSize     uint8
}

type AddressMode string

const (
	Absolute                 = "a"
	AbsoluteIndexedIndirect  = "(a,x)"
	AbsoluteIndexedX         = "a,x"
	AbsoluteIndexedY         = "a,y"
	AbsoluteIndirect         = "(a)"
	Accumulator              = "A"
	Immediate                = "#"
	Implied                  = "i"
	ProgramCounterRelative   = "r"
	Stack                    = "s"
	ZeroPage                 = "zp"
	ZeroPageIndexedIndirect  = "(zp,x)"
	ZeroPageIndexedX         = "zp,x"
	ZeroPageIndexedY         = "zp,y"
	ZeroPageIndirect         = "(zp)"
	ZeroPageIndirectIndexedY = "(zp),y"
)

type CycleType string

const (
	CycleReadOpCode             = "ReadOpCode"
	CycleReadAddressLSB         = "ReadAddrLSB"
	CycleReadAddressMSB         = "ReadAddrMSB"
	CycleReadIndirectAddressLSB = "ReadIndLSB"
	CycleReadIndirectAddressMSB = "ReadIndMSB"
	CycleReadValue              = "ReadValue"
	CycleAction                 = "TakeAction"
	CycleExtraStep              = "ExtraSteps"
)

var AddressModes = map[AddressMode]AddressModeData{
	Absolute:                 {4, []CycleType{CycleReadOpCode, CycleReadAddressLSB, CycleReadAddressMSB, CycleAction}, []ExtraCyclesType{ReadModifyWrite}, 3},
	AbsoluteIndexedIndirect:  {6, []CycleType{CycleReadOpCode, CycleReadAddressLSB, CycleReadAddressMSB, CycleReadIndirectAddressLSB, CycleReadIndirectAddressMSB, CycleAction}, []ExtraCyclesType{None}, 3},
	AbsoluteIndexedX:         {4, []CycleType{CycleReadOpCode, CycleReadAddressLSB, CycleReadAddressMSB, CycleAction}, []ExtraCyclesType{PageBoundary, ReadModifyWrite}, 3},
	AbsoluteIndexedY:         {4, []CycleType{CycleReadOpCode, CycleReadAddressLSB, CycleReadAddressMSB, CycleAction}, []ExtraCyclesType{PageBoundary}, 3},
	AbsoluteIndirect:         {6, []CycleType{CycleReadOpCode, CycleReadAddressLSB, CycleReadAddressMSB, CycleReadIndirectAddressLSB, CycleReadIndirectAddressMSB, CycleAction}, []ExtraCyclesType{None}, 3},
	Accumulator:              {2, []CycleType{CycleReadOpCode, CycleAction}, []ExtraCyclesType{None}, 1},
	Immediate:                {2, []CycleType{CycleReadOpCode, CycleAction}, []ExtraCyclesType{None}, 2},
	Implied:                  {2, []CycleType{CycleReadOpCode, CycleAction}, []ExtraCyclesType{None}, 1},
	ProgramCounterRelative:   {2, []CycleType{CycleReadOpCode, CycleAction}, []ExtraCyclesType{PageBoundary, BranchTaken}, 2},
	Stack:                    {6, []CycleType{CycleReadOpCode, CycleAction, CycleReadIndirectAddressLSB, CycleReadIndirectAddressMSB, CycleAction, CycleAction}, []ExtraCyclesType{None}, 1},
	ZeroPage:                 {3, []CycleType{CycleReadOpCode, CycleReadAddressLSB}, []ExtraCyclesType{ReadModifyWrite}, 2},
	ZeroPageIndexedIndirect:  {6, []CycleType{CycleReadOpCode, CycleReadAddressLSB, CycleReadAddressMSB, CycleReadIndirectAddressLSB, CycleReadIndirectAddressMSB, CycleAction}, []ExtraCyclesType{None}, 2},
	ZeroPageIndexedX:         {4, []CycleType{CycleReadOpCode, CycleReadAddressLSB, CycleReadAddressMSB, CycleAction}, []ExtraCyclesType{ReadModifyWrite}, 2},
	ZeroPageIndexedY:         {4, []CycleType{CycleReadOpCode, CycleReadAddressLSB, CycleReadAddressMSB, CycleAction}, []ExtraCyclesType{None}, 2},
	ZeroPageIndirect:         {5, []CycleType{CycleReadOpCode, CycleReadAddressLSB, CycleReadIndirectAddressLSB, CycleReadIndirectAddressMSB, CycleAction}, []ExtraCyclesType{None}, 2},
	ZeroPageIndirectIndexedY: {5, []CycleType{CycleReadOpCode, CycleReadAddressLSB, CycleReadIndirectAddressLSB, CycleReadIndirectAddressMSB, CycleAction}, []ExtraCyclesType{PageBoundary}, 2},
}
