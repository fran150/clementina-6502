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
	auxiliaryControl       ViaAuxiliaryControlRegister
	peripheralControl      ViaPeripheralControlRegister
	interruptFlag          uint8
	interruptEnable        uint8
}

type Via65C22S struct {
	peripheralAControlLines *ViaControlLines
	peripheralBControlLines *ViaControlLines
	chipSelect1             *buses.ConnectorEnabledHigh
	chipSelect2             *buses.ConnectorEnabledLow
	dataBus                 *buses.BusConnector[uint8]
	irqRequest              *buses.ConnectorEnabledLow
	peripheralPortA         *ViaPort
	peripheralPortB         *ViaPort
	reset                   *buses.ConnectorEnabledLow
	registerSelect          [4]*buses.ConnectorEnabledHigh
	readWrite               *buses.ConnectorEnabledLow

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

var caTransitionConfigurationMasks [2]viaPCRTranstitionMasks = [2]viaPCRTranstitionMasks{
	pcrMaskCA1TransitionType,
	pcrMaskCA2TransitionType,
}

var caOutputConfigurationModes [3]viaPCROutputModes = [3]viaPCROutputModes{
	pcrCA2OutputModeHandshake,
	pcrCA2OutputModePulse,
	pcrCA2OutputModeFix,
}

var cbTransitionConfigurationMasks [2]viaPCRTranstitionMasks = [2]viaPCRTranstitionMasks{
	pcrMaskCB1TransitionType,
	pcrMaskCB2TransitionType,
}

var cbOutputConfigurationModes [3]viaPCROutputModes = [3]viaPCROutputModes{
	pcrCB2OutputModeHandshake,
	pcrCB2OutputModePulse,
	pcrCB2OutputModeFix,
}

func CreateVia65C22() *Via65C22S {
	via := Via65C22S{
		peripheralAControlLines: createControlLines(caTransitionConfigurationMasks, pcrMaskCAOutputMode, caOutputConfigurationModes),
		peripheralBControlLines: createControlLines(cbTransitionConfigurationMasks, pcrMaskCBOutputMode, cbOutputConfigurationModes),
		chipSelect1:             buses.CreateConnectorEnabledHigh(),
		chipSelect2:             buses.CreateConnectorEnabledLow(),
		dataBus:                 buses.CreateBusConnector[uint8](),
		irqRequest:              buses.CreateConnectorEnabledLow(),
		reset:                   buses.CreateConnectorEnabledLow(),
		registerSelect: [4]*buses.ConnectorEnabledHigh{
			buses.CreateConnectorEnabledHigh(),
			buses.CreateConnectorEnabledHigh(),
			buses.CreateConnectorEnabledHigh(),
			buses.CreateConnectorEnabledHigh(),
		},
		readWrite: buses.CreateConnectorEnabledLow(),

		registers: Via65C22SRegisters{},
	}

	via.peripheralPortA = createViaPort(&via.registers.auxiliaryControl, &via.registers.peripheralControl, via.peripheralAControlLines)
	via.peripheralPortB = createViaPort(&via.registers.auxiliaryControl, &via.registers.peripheralControl, via.peripheralBControlLines)

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
		readFromRecord((*uint8)(&via.registers.auxiliaryControl)),  // 0x0B
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
		writeToRecord((*uint8)(&via.registers.auxiliaryControl)),  // 0x0B
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
	return via.peripheralAControlLines.GetLine(num)
}

func (via *Via65C22S) PeripheralBControlLines(num uint8) *buses.ConnectorEnabledHigh {
	return via.peripheralBControlLines.GetLine(num)
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
	return via.peripheralPortA.getConnector()
}

func (via *Via65C22S) PeripheralPortB() *buses.BusConnector[uint8] {
	return via.peripheralPortB.getConnector()
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

func (via *Via65C22S) setInterruptFlagOnControlLinesTransition() {

	if via.peripheralAControlLines.checkControlLineTransitioned(&via.registers.peripheralControl, 0) {
		via.setInterruptFlag(irqCA1)
	}

	if via.peripheralAControlLines.checkControlLineTransitioned(&via.registers.peripheralControl, 1) {
		via.setInterruptFlag(irqCA2)
	}

	if via.peripheralBControlLines.checkControlLineTransitioned(&via.registers.peripheralControl, 0) {
		via.setInterruptFlag(irqCB1)
	}

	if via.peripheralBControlLines.checkControlLineTransitioned(&via.registers.peripheralControl, 1) {
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
	via.peripheralPortA.latchPort(via.registers.dataDirectionRegisterA, &via.registers.inputRegisterA, acrMaskLatchingA)
	via.peripheralPortB.latchPort(via.registers.dataDirectionRegisterB, &via.registers.inputRegisterB, acrMaskLatchingB)

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
		via.peripheralPortA.writePort(via.registers.dataDirectionRegisterA, via.registers.outputRegisterA)
		via.peripheralPortB.writePort(via.registers.dataDirectionRegisterB, via.registers.outputRegisterB)
	}

	if via.registers.peripheralControl.isSetForOutput(pcrMaskCA2OutputEnabled) {
		via.peripheralAControlLines.setOutputMode(&via.registers.peripheralControl)
	}

	if via.registers.peripheralControl.isSetForOutput(pcrMaskCB2OutputEnabled) {
		via.peripheralBControlLines.setOutputMode(&via.registers.peripheralControl)
	}

	via.setInterruptFlagOnControlLinesTransition()

	via.peripheralAControlLines.storePreviousControlLinesValues()
	via.peripheralBControlLines.storePreviousControlLinesValues()

	via.handleIRQLine()
}
