package cpu

type ExtraCyclesType int

const (
	None            = 0
	PageBoundary    = 1
	BranchTaken     = 1
	ReadModifyWrite = 2
)

type AddressModeData struct {
	Cycles      uint8
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

var AddressModes = map[AddressMode]AddressModeData{
	Absolute:                 {4, []ExtraCyclesType{ReadModifyWrite}, 3},
	AbsoluteIndexedIndirect:  {6, []ExtraCyclesType{None}, 3},
	AbsoluteIndexedX:         {4, []ExtraCyclesType{PageBoundary, ReadModifyWrite}, 3},
	AbsoluteIndexedY:         {4, []ExtraCyclesType{PageBoundary}, 3},
	AbsoluteIndirect:         {6, []ExtraCyclesType{None}, 3},
	Accumulator:              {2, []ExtraCyclesType{None}, 1},
	Immediate:                {2, []ExtraCyclesType{None}, 2},
	Implied:                  {2, []ExtraCyclesType{None}, 1},
	ProgramCounterRelative:   {2, []ExtraCyclesType{PageBoundary, BranchTaken}, 2},
	Stack:                    {6, []ExtraCyclesType{None}, 1},
	ZeroPage:                 {3, []ExtraCyclesType{ReadModifyWrite}, 2},
	ZeroPageIndexedIndirect:  {6, []ExtraCyclesType{None}, 2},
	ZeroPageIndexedX:         {4, []ExtraCyclesType{ReadModifyWrite}, 2},
	ZeroPageIndexedY:         {4, []ExtraCyclesType{None}, 2},
	ZeroPageIndirect:         {5, []ExtraCyclesType{None}, 2},
	ZeroPageIndirectIndexedY: {5, []ExtraCyclesType{PageBoundary}, 2},
}
