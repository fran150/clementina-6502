package via

import (
	"math"

	"github.com/fran150/clementina6502/buses"
)

type Via65C22SRegisters struct {
	shiftRegister     uint8
	auxiliaryControl  uint8
	peripheralControl uint8
	interrupts        ViaIFR
}

type Via65C22S struct {
	sideA ViaSide
	sideB ViaSide

	chipSelect1    *buses.ConnectorEnabledHigh
	chipSelect2    *buses.ConnectorEnabledLow
	dataBus        *buses.BusConnector[uint8]
	irqRequest     *buses.ConnectorEnabledLow
	reset          *buses.ConnectorEnabledLow
	registerSelect [4]*buses.ConnectorEnabledHigh
	readWrite      *buses.ConnectorEnabledLow

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
	sideAConfiguration := ViaSideConfiguration{
		transitionConfigurationMasks: [2]viaPCRTranstitionMasks{
			pcrMaskCA1TransitionType,
			pcrMaskCA2TransitionType,
		},

		latchingEnabledMasks:    acrMaskLatchingEnabledA,
		outputConfigurationMask: pcrMaskCAOutputMode,
		handshakeMode:           pcrCA2OutputModeHandshake,
		pulseMode:               pcrCA2OutputModePulse,
		fixedMode:               pcrCA2OutputModeFix,
		clearC2OnRWMask:         pcrMaskCA2ClearOnRW,

		timerRunModeMask:  t2ControlRunModeMask,
		timerInterruptBit: irqT2,

		controlLinesIRQBits: [2]viaIRQFlags{
			irqCA1,
			irqCA2,
		},
	}

	sideBConfiguration := ViaSideConfiguration{
		transitionConfigurationMasks: [2]viaPCRTranstitionMasks{
			pcrMaskCB1TransitionType,
			pcrMaskCB2TransitionType,
		},

		latchingEnabledMasks:    acrMaskLatchingEnabledB,
		outputConfigurationMask: pcrMaskCBOutputMode,
		handshakeMode:           pcrCB2OutputModeHandshake,
		pulseMode:               pcrCB2OutputModePulse,
		fixedMode:               pcrCB2OutputModeFix,

		timerRunModeMask:  t1ControlRunModeMask,
		timerOutputMask:   t1ControlOutputMask,
		timerInterruptBit: irqT1,

		controlLinesIRQBits: [2]viaIRQFlags{
			irqCB1,
			irqCB2,
		},
	}

	via := Via65C22S{
		chipSelect1: buses.CreateConnectorEnabledHigh(),
		chipSelect2: buses.CreateConnectorEnabledLow(),
		dataBus:     buses.CreateBusConnector[uint8](),
		irqRequest:  buses.CreateConnectorEnabledLow(),
		reset:       buses.CreateConnectorEnabledLow(),
		registerSelect: [4]*buses.ConnectorEnabledHigh{
			buses.CreateConnectorEnabledHigh(),
			buses.CreateConnectorEnabledHigh(),
			buses.CreateConnectorEnabledHigh(),
			buses.CreateConnectorEnabledHigh(),
		},
		readWrite: buses.CreateConnectorEnabledLow(),

		registers: Via65C22SRegisters{},
	}

	via.sideA = createViaSide(&via, sideAConfiguration)
	via.sideB = createViaSide(&via, sideBConfiguration)

	via.populateRegisterReadHandlers()
	via.populateRegisterWriteHandlers()

	return &via
}

/************************************************************************************
* Handler configuration
*************************************************************************************/

func (via *Via65C22S) populateRegisterReadHandlers() {
	via.registerReadHandlers = []func(*Via65C22S){
		inputOutputRegisterBReadHandler,                            // 0x00
		inputOutputRegisterAReadHandler,                            // 0x01
		readFromRecord(&via.sideB.registers.dataDirectionRegister), // 0x02
		readFromRecord(&via.sideA.registers.dataDirectionRegister), // 0x03
		readT1LowOrderCounter,                                      // 0x04
		readT1HighOrderCounter,                                     // 0x05
		readFromRecord(&via.sideB.registers.lowLatches),            // 0x06
		readFromRecord(&via.sideB.registers.highLatches),           // 0x07
		readT2LowOrderCounter,                                      // 0x08
		readT2HighOrderCounter,                                     // 0x09
		dummyHandler,                                               // 0x0A
		readFromRecord((*uint8)(&via.registers.auxiliaryControl)),  // 0x0B
		readFromRecord((*uint8)(&via.registers.peripheralControl)), // 0x0C
		readnterruptFlagHandler,                                    // 0x0D
		readInterruptEnableHandler,                                 // 0x0E
		dummyHandler,                                               // 0x0F
	}
}

func (via *Via65C22S) populateRegisterWriteHandlers() {
	via.registerWriteHandlers = []func(*Via65C22S){
		inputOutputRegisterBWriteHandler,                          // 0x00
		inputOutputRegisterAWriteHandler,                          // 0x01
		writeToRecord(&via.sideB.registers.dataDirectionRegister), // 0x02
		writeToRecord(&via.sideA.registers.dataDirectionRegister), // 0x03
		writeToRecord(&via.sideB.registers.lowLatches),            // 0x04
		writeT1HighOrderCounter,                                   // 0x05
		writeToRecord(&via.sideB.registers.lowLatches),            // 0x06
		writeT1HighOrderLatch,                                     // 0x07
		writeToRecord(&via.sideA.registers.lowLatches),            // 0x08
		writeT2HighOrderCounter,                                   // 0x09
		dummyHandler,                                              // 0x0A
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
	return via.sideA.controlLines.getLine(num)
}

func (via *Via65C22S) PeripheralBControlLines(num uint8) *buses.ConnectorEnabledHigh {
	return via.sideB.controlLines.getLine(num)
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
	return via.sideA.peripheralPort.getConnector()
}

func (via *Via65C22S) PeripheralPortB() *buses.BusConnector[uint8] {
	return via.sideB.peripheralPort.getConnector()
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

func (via *Via65C22S) handleIRQLine() {
	if via.registers.interrupts.isInterruptTriggered() {
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
	via.sideA.peripheralPort.latchPort()
	via.sideB.peripheralPort.latchPort()

	via.sideA.timer.tick()
	via.sideB.timer.tick()
	// Count down pulse only enabled in timer 2 which in this case is on Side A
	via.sideA.peripheralPort.countDownPulseIfEnabled()

	if via.chipSelect1.Enabled() && via.chipSelect2.Enabled() {
		selectedRegisterValue := via.getRegisterSelectValue()

		if !via.readWrite.Enabled() {
			via.registerReadHandlers[uint8(selectedRegisterValue)](via)
		} else {
			via.registerWriteHandlers[uint8(selectedRegisterValue)](via)
		}
	}

	// From https://lateblt.tripod.com/bit67.txt:
	// The ORs are also never transparent Whereas an input bus which has input latching turned off can change with its
	// input without the Enable pin even being cycled, outputting to an OR will not take effect until the Enable pin has made
	// a transition to low or high.
	if via.chipSelect1.Enabled() && via.chipSelect2.Enabled() {
		via.sideA.peripheralPort.writePortOutputRegister()
		via.sideB.peripheralPort.writePortOutputRegister()
	}

	via.sideA.peripheralPort.writeTimerOutput()
	via.sideB.peripheralPort.writeTimerOutput()

	via.sideA.controlLines.setOutput()
	via.sideB.controlLines.setOutput()

	via.sideA.controlLines.setInterruptFlagOnControlLinesTransition()
	via.sideB.controlLines.setInterruptFlagOnControlLinesTransition()

	via.sideA.controlLines.storePreviousControlLinesValues()
	via.sideB.controlLines.storePreviousControlLinesValues()

	via.handleIRQLine()
}
