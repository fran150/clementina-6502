package via

import (
	"math"

	"github.com/fran150/clementina6502/buses"
)

type Via65C22SRegisters struct {
	outputRegisterA        uint8
	inputRegisterA         uint8
	outputRegisterB        uint8
	inputRegisterB         uint8
	dataDirectionRegisterA uint8
	dataDirectionRegisterB uint8
	t1LowLatches           uint8
	t1HighLatches          uint8
	t2LowLatches           uint8
	t2HighLatches          uint8
	t1Counter              uint16
	t2Counter              uint16
	shiftRegister          uint8
	auxiliaryControl       uint8
	peripheralControl      ViaPeripheralControlRegiter
	interruptFlag          uint8
	interruptEnable        uint8
}

type Via65C22S struct {
	peripheralAControlLines [2]*buses.ConnectorEnabledHigh
	peripheralBControlLines [2]*buses.ConnectorEnabledHigh
	chipSelect1             *buses.ConnectorEnabledHigh
	chipSelect2             *buses.ConnectorEnabledLow
	dataBus                 *buses.BusConnector[uint8]
	irqRequest              *buses.ConnectorEnabledLow
	peripheralPortA         *buses.BusConnector[uint8]
	peripheralPortB         *buses.BusConnector[uint8]
	reset                   *buses.ConnectorEnabledLow
	registerSelect          [4]*buses.ConnectorEnabledHigh
	readWrite               *buses.ConnectorEnabledLow

	previousCtrlA [2]bool
	previousCtrlB [2]bool

	caHandshakeInProgress bool
	cbHandshakeInProgress bool

	caHandshakeCycleCounter uint8
	cbHandshakeCycleCounter uint8

	registers Via65C22SRegisters

	registerReadHandlers  []func(*Via65C22S)
	registerWriteHandlers []func(*Via65C22S)
}

type viaRegisterCode uint8

const (
	regORBIRB            viaRegisterCode = 0x00
	regORAIRA            viaRegisterCode = 0x01
	regDDRB              viaRegisterCode = 0x02
	regDDRA              viaRegisterCode = 0x03
	regT1CL              viaRegisterCode = 0x04
	regT1CH              viaRegisterCode = 0x05
	regT1LL              viaRegisterCode = 0x06
	regT1HL              viaRegisterCode = 0x07
	regT2CL              viaRegisterCode = 0x08
	regT2CH              viaRegisterCode = 0x09
	regSR                viaRegisterCode = 0x0A
	regACR               viaRegisterCode = 0x0B
	regPCR               viaRegisterCode = 0x0C
	regIFR               viaRegisterCode = 0x0D
	regIER               viaRegisterCode = 0x0E
	regORAIRANoHandshake viaRegisterCode = 0x0F
)

type viaIRQFlags uint8

const (
	irqCA2 viaIRQFlags = 0x01
	irqCA1 viaIRQFlags = 0x02
	irqSR  viaIRQFlags = 0x04
	irqCB2 viaIRQFlags = 0x08
	irqCB1 viaIRQFlags = 0x10
	irqT2  viaIRQFlags = 0x20
	irqT1  viaIRQFlags = 0x40
	irqAny viaIRQFlags = 0x80
)

func CreateVia65C22() *Via65C22S {
	via := Via65C22S{
		peripheralAControlLines: [2]*buses.ConnectorEnabledHigh{
			buses.CreateConnectorEnabledHigh(),
			buses.CreateConnectorEnabledHigh(),
		},
		peripheralBControlLines: [2]*buses.ConnectorEnabledHigh{
			buses.CreateConnectorEnabledHigh(),
			buses.CreateConnectorEnabledHigh(),
		},
		chipSelect1:     buses.CreateConnectorEnabledHigh(),
		chipSelect2:     buses.CreateConnectorEnabledLow(),
		dataBus:         buses.CreateBusConnector[uint8](),
		irqRequest:      buses.CreateConnectorEnabledLow(),
		peripheralPortA: buses.CreateBusConnector[uint8](),
		peripheralPortB: buses.CreateBusConnector[uint8](),
		reset:           buses.CreateConnectorEnabledLow(),
		registerSelect: [4]*buses.ConnectorEnabledHigh{
			buses.CreateConnectorEnabledHigh(),
			buses.CreateConnectorEnabledHigh(),
			buses.CreateConnectorEnabledHigh(),
			buses.CreateConnectorEnabledHigh(),
		},
		readWrite:     buses.CreateConnectorEnabledLow(),
		previousCtrlA: [2]bool{false, false},
		previousCtrlB: [2]bool{false, false},

		registers: Via65C22SRegisters{},
	}

	via.populateRegisterReadHandlers()
	via.populateRegisterWriteHandlers()

	return &via
}

/************************************************************************************
* Handler configuration
*************************************************************************************/

func (via *Via65C22S) populateRegisterReadHandlers() {
	via.registerReadHandlers = []func(*Via65C22S){
		inputOutputRegisterBReadHandler,                       // 0x00
		inputOutputRegisterAReadHandler,                       // 0x01
		readFromRecord(&via.registers.dataDirectionRegisterB), // 0x02
		readFromRecord(&via.registers.dataDirectionRegisterA), // 0x03
		dummyHandler, // 0x04
		dummyHandler, // 0x05
		dummyHandler, // 0x06
		dummyHandler, // 0x07
		dummyHandler, // 0x08
		dummyHandler, // 0x09
		dummyHandler, // 0x0A
		readFromRecord(&via.registers.auxiliaryControl),            // 0x0B
		readFromRecord((*uint8)(&via.registers.peripheralControl)), // 0x0C
		readFromRecord(&via.registers.interruptFlag),               // 0x0D
		readInterruptEnableHandler,                                 // 0x0E
		dummyHandler,                                               // 0x0F
	}
}

func (via *Via65C22S) populateRegisterWriteHandlers() {
	via.registerWriteHandlers = []func(*Via65C22S){
		inputOutputRegisterBWriteHandler,                     // 0x00
		inputOutputRegisterAWriteHandler,                     // 0x01
		writeToRecord(&via.registers.dataDirectionRegisterB), // 0x02
		writeToRecord(&via.registers.dataDirectionRegisterA), // 0x03
		dummyHandler, // 0x04
		dummyHandler, // 0x05
		dummyHandler, // 0x06
		dummyHandler, // 0x07
		dummyHandler, // 0x08
		dummyHandler, // 0x09
		dummyHandler, // 0x0A
		writeToRecord(&via.registers.auxiliaryControl),            // 0x0B
		writeToRecord((*uint8)(&via.registers.peripheralControl)), // 0x0C
		writeInterruptFlagHandler,                                 // 0x0D
		writeInterruptEnableHandler,                               // 0x0E
		dummyHandler,                                              // 0x0F
	}
}

/************************************************************************************
* Getters / Setters
*************************************************************************************/

func (via *Via65C22S) PeripheralAControlLines(num uint8) *buses.ConnectorEnabledHigh {
	return via.peripheralAControlLines[num]
}

func (via *Via65C22S) PeripheralBControlLines(num uint8) *buses.ConnectorEnabledHigh {
	return via.peripheralBControlLines[num]
}

func (via *Via65C22S) ChipSelect1() *buses.ConnectorEnabledHigh {
	return via.chipSelect1
}

func (via *Via65C22S) ChipSelect2() *buses.ConnectorEnabledLow {
	return via.chipSelect2
}

func (via *Via65C22S) DataBus() *buses.BusConnector[uint8] {
	return via.dataBus
}

func (via *Via65C22S) IrqRequest() *buses.ConnectorEnabledLow {
	return via.irqRequest
}

func (via *Via65C22S) PeripheralPortA() *buses.BusConnector[uint8] {
	return via.peripheralPortA
}

func (via *Via65C22S) PeripheralPortB() *buses.BusConnector[uint8] {
	return via.peripheralPortB
}

func (via *Via65C22S) Reset() *buses.ConnectorEnabledLow {
	return via.reset
}

func (via *Via65C22S) RegisterSelect(num uint8) *buses.ConnectorEnabledHigh {
	return via.registerSelect[num]
}

func (via *Via65C22S) ReadWrite() *buses.ConnectorEnabledLow {
	return via.readWrite
}

func (via *Via65C22S) ConnectRegisterSelectLines(lines [4]buses.Line) {
	for i := range 4 {
		via.registerSelect[i].Connect(lines[i])
	}
}

/************************************************************************************
* Internal functions
*************************************************************************************/

func (via *Via65C22S) getRegisterSelectValue() viaRegisterCode {
	var value uint8

	for i := range 4 {
		if via.registerSelect[i].Enabled() {
			value += uint8(math.Pow(2, float64(i)))
		}
	}

	return viaRegisterCode(value)
}

func (via *Via65C22S) setCAOutputMode() {
	switch via.registers.peripheralControl.getOutputMode(pcrMaskCAOutputMode) {
	case pcrCA2OutputModeHandshake:
		transitionedCA1 := via.checkControlLineTransitioned(pcrMaskCA1TransitionType, via.peripheralAControlLines[0], via.previousCtrlA[0])

		if via.caHandshakeInProgress && transitionedCA1 {
			via.caHandshakeInProgress = false
		}

		if via.caHandshakeInProgress {
			via.peripheralAControlLines[1].SetEnable(false)
		} else {
			via.peripheralAControlLines[1].SetEnable(true)
		}
	case pcrCA2OutputModePulse:
		if via.caHandshakeInProgress {
			via.caHandshakeCycleCounter += 1
		}

		if via.caHandshakeCycleCounter > 2 && !via.peripheralAControlLines[1].Enabled() {
			via.caHandshakeInProgress = false
		}

		if via.caHandshakeInProgress {
			via.peripheralAControlLines[1].SetEnable(false)
		} else {
			via.peripheralAControlLines[1].SetEnable(true)
		}
	case pcrCA2OutputModeFix:
		if via.registers.peripheralControl.isTransitionPositive(pcrMaskCA2TransitionType) {
			via.peripheralAControlLines[1].SetEnable(true)
		} else {
			via.peripheralAControlLines[1].SetEnable(false)
		}
	}
}

func (via *Via65C22S) setCBOutputMode() {
	switch via.registers.peripheralControl.getOutputMode(pcrMaskCBOutputMode) {
	case pcrCB2OutputModeHandshake:
		transitionedCB1 := via.checkControlLineTransitioned(pcrMaskCB1TransitionType, via.peripheralBControlLines[0], via.previousCtrlB[0])

		if via.cbHandshakeInProgress && transitionedCB1 {
			via.cbHandshakeInProgress = false
		}

		if via.cbHandshakeInProgress {
			via.peripheralBControlLines[1].SetEnable(false)
		} else {
			via.peripheralBControlLines[1].SetEnable(true)
		}
	case pcrCB2OutputModePulse:
		if via.cbHandshakeInProgress {
			via.cbHandshakeCycleCounter += 1
		}

		if via.cbHandshakeCycleCounter > 2 && !via.peripheralBControlLines[1].Enabled() {
			via.cbHandshakeInProgress = false
		}

		if via.cbHandshakeInProgress {
			via.peripheralBControlLines[1].SetEnable(false)
		} else {
			via.peripheralBControlLines[1].SetEnable(true)
		}
	case pcrCB2OutputModeFix:
		if via.registers.peripheralControl.isTransitionPositive(pcrMaskCB2TransitionType) {
			via.peripheralBControlLines[1].SetEnable(true)
		} else {
			via.peripheralBControlLines[1].SetEnable(false)
		}
	}
}

func (via *Via65C22S) checkControlLineTransitioned(mask viaPCRTranstitionMasks, controlLine *buses.ConnectorEnabledHigh, previousLineState bool) bool {
	caCtrlPositive := via.registers.peripheralControl.isTransitionPositive(mask)

	currentCrl := controlLine.Enabled()
	previousCtrl := previousLineState

	return (caCtrlPositive && !previousCtrl && currentCrl) || (!caCtrlPositive && previousCtrl && !currentCrl)
}

type viaACRLatchingMasks uint8

const (
	acrMaskLatchingA viaACRLatchingMasks = 0x01
	acrMaskLatchingB viaACRLatchingMasks = 0x02
)

func (via *Via65C22S) isLatchingEnabled(mask viaACRLatchingMasks) bool {
	return via.registers.auxiliaryControl&uint8(mask) > 0x00
}

func (via *Via65C22S) latchPortA() {
	// Read pin levels on port A
	value := via.peripheralPortA.Read()

	// Read pins are all the ones with 0 in the DDR
	readPins := ^via.registers.dataDirectionRegisterA

	if via.isLatchingEnabled(acrMaskLatchingA) {
		transitioned := via.checkControlLineTransitioned(pcrMaskCA1TransitionType, via.peripheralAControlLines[0], via.previousCtrlA[0])

		// If latching is enabled value is the one at the time of CB transition
		if transitioned {
			via.registers.inputRegisterA = value & readPins
		}
	}
}

func (via *Via65C22S) latchPortB() {
	// Read pin levels on port B
	value := via.peripheralPortB.Read()

	// Read pins are all the ones with 0 in the DDR
	readPins := ^via.registers.dataDirectionRegisterB

	if via.isLatchingEnabled(acrMaskLatchingB) {
		transitioned := via.checkControlLineTransitioned(pcrMaskCB1TransitionType, via.peripheralBControlLines[0], via.previousCtrlB[0])

		// If latching is enabled value is the one at the time of CB transition
		if transitioned {
			via.registers.inputRegisterB = value & readPins
		}
	}
}

func (via *Via65C22S) isByteSet(value uint8, bitNumber uint8) bool {
	mask := uint8(math.Pow(2, float64(bitNumber)))

	return (value & mask) > 0
}

func (via *Via65C22S) writePortA() {
	for i := range uint8(8) {
		if via.isByteSet(via.registers.dataDirectionRegisterA, i) {
			via.PeripheralPortA().GetLine(i).Set(via.isByteSet(via.registers.outputRegisterA, i))
		}
	}
}

func (via *Via65C22S) writePortB() {
	for i := range uint8(8) {
		if via.isByteSet(via.registers.dataDirectionRegisterB, i) {
			via.PeripheralPortB().GetLine(i).Set(via.isByteSet(via.registers.outputRegisterB, i))
		}
	}
}

// If any of the bits 0 - 6 in the IFR is 1 then the bit 7 is 1
// If not, then the bit 7 is 0.
func (via *Via65C22S) writeInterruptFlagRegister(value uint8) {
	if (value & via.registers.interruptEnable & 0x7F) > 0 {
		value |= 0x80
	} else {
		value &= 0x7F
	}

	via.registers.interruptFlag = value
}

func (via *Via65C22S) setInterruptFlag(flag viaIRQFlags) {
	via.writeInterruptFlagRegister(via.registers.interruptFlag | uint8(flag))
}

func (via *Via65C22S) clearInterruptFlag(flag viaIRQFlags) {
	via.writeInterruptFlagRegister(via.registers.interruptFlag & ^uint8(flag))
}

func (via *Via65C22S) storePreviousControlLinesValues() {
	via.previousCtrlA[0] = via.peripheralAControlLines[0].Enabled()
	via.previousCtrlB[0] = via.peripheralBControlLines[0].Enabled()
	via.previousCtrlA[1] = via.peripheralAControlLines[1].Enabled()
	via.previousCtrlB[1] = via.peripheralBControlLines[1].Enabled()
}

func (via *Via65C22S) setInterruptFlagOnControlLinesTransition() {
	if via.checkControlLineTransitioned(pcrMaskCA1TransitionType, via.peripheralAControlLines[0], via.previousCtrlA[0]) {
		via.setInterruptFlag(irqCA1)
	}

	if via.checkControlLineTransitioned(pcrMaskCA2TransitionType, via.peripheralAControlLines[1], via.previousCtrlA[1]) {
		via.setInterruptFlag(irqCA2)
	}

	if via.checkControlLineTransitioned(pcrMaskCB1TransitionType, via.peripheralBControlLines[0], via.previousCtrlB[0]) {
		via.setInterruptFlag(irqCB1)
	}

	if via.checkControlLineTransitioned(pcrMaskCB2TransitionType, via.peripheralBControlLines[1], via.previousCtrlB[1]) {
		via.setInterruptFlag(irqCB2)
	}
}

func (via *Via65C22S) clearControlLinesInterruptFlagOnRWPortA() {
	via.clearInterruptFlag(irqCA1)

	if via.registers.peripheralControl.isSetToClearOnRW(pcrMaskCA2ClearOnRW) {
		via.clearInterruptFlag(irqCA2)
	}
}

func (via *Via65C22S) clearControlLinesInterruptFlagOnRWPortB() {
	via.clearInterruptFlag(irqCB1)

	if via.registers.peripheralControl.isSetToClearOnRW(pcrMaskCB2ClearOnRW) {
		via.clearInterruptFlag(irqCB2)
	}
}

func (via *Via65C22S) handleIRQLine() {
	if (via.registers.interruptFlag & via.registers.interruptEnable & 0x7F) > 0 {
		via.IrqRequest().SetEnable(true)
	} else {
		via.IrqRequest().SetEnable(false)
	}
}

/************************************************************************************
* Tick methods
*************************************************************************************/

func (via *Via65C22S) Tick(t uint64) {
	// From https://lateblt.tripod.com/bit67.txt:
	// The ORs are also never transparent Whereas an input bus which has input latching turned off can change with its
	// input without the Enable pin even being cycled, outputting to an OR will not take effect until the Enable pin has made
	// a transition to low or high.
	via.latchPortA()
	via.latchPortB()

	if via.chipSelect1.Enabled() && via.chipSelect2.Enabled() {
		selectedRegisterValue := via.getRegisterSelectValue()

		if !via.readWrite.Enabled() {
			via.registerReadHandlers[uint8(selectedRegisterValue)](via)
		} else {
			via.registerWriteHandlers[uint8(selectedRegisterValue)](via)
		}
	}
}

func (via *Via65C22S) PostTick(t uint64) {
	// From https://lateblt.tripod.com/bit67.txt:
	// The ORs are also never transparent Whereas an input bus which has input latching turned off can change with its
	// input without the Enable pin even being cycled, outputting to an OR will not take effect until the Enable pin has made
	// a transition to low or high.
	if via.chipSelect1.Enabled() && via.chipSelect2.Enabled() {
		via.writePortA()
		via.writePortB()
	}

	if via.registers.peripheralControl.isSetForOutput(pcrMaskCA2OutputEnabled) {
		via.setCAOutputMode()
	}

	if via.registers.peripheralControl.isSetForOutput(pcrMaskCB2OutputEnabled) {
		via.setCBOutputMode()
	}

	via.setInterruptFlagOnControlLinesTransition()
	via.storePreviousControlLinesValues()
	via.handleIRQLine()
}
