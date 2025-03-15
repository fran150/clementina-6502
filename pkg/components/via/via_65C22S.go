package via

import (
	"math"

	"github.com/fran150/clementina6502/pkg/common"
	"github.com/fran150/clementina6502/pkg/components/buses"
)

const numOfRSLines = 4

// Registers of the VIA chip
type via65C22SRegisters struct {
	outputRegisterA        uint8
	outputRegisterB        uint8
	inputRegisterA         uint8
	inputRegisterB         uint8
	dataDirectionRegisterA uint8
	dataDirectionRegisterB uint8
	lowLatches2            uint8
	lowLatches1            uint8
	highLatches2           uint8
	highLatches1           uint8
	counter2               uint16
	counter1               uint16
	shiftRegister          uint8
	auxiliaryControl       uint8
	peripheralControl      uint8
	interrupts             viaIFR
}

// Value of the register select lines for each register
type viaRegisterCode uint8

const (
	regORBIRB            viaRegisterCode = 0x00 // ORB and IRB are accessed using the same value depending on the R/W flag
	regORAIRA            viaRegisterCode = 0x01 // ORA and IRA are accessed using the same value depending on the R/W flag
	regDDRB              viaRegisterCode = 0x02 // Data direction register for port B
	regDDRA              viaRegisterCode = 0x03 // Data direction register for port A
	regT1CL              viaRegisterCode = 0x04 // Timer 1 low counter
	regT1CH              viaRegisterCode = 0x05 // Timer 1 high counter
	regT1LL              viaRegisterCode = 0x06 // Timer 1 low latch
	regT1HL              viaRegisterCode = 0x07 // Timer 1 high latch
	regT2CL              viaRegisterCode = 0x08 // Timer 2 low counter
	regT2CH              viaRegisterCode = 0x09 // Timer 2 high counter
	regSR                viaRegisterCode = 0x0A // Shift Register
	regACR               viaRegisterCode = 0x0B // Auxiliary control register
	regPCR               viaRegisterCode = 0x0C // Peripherial control register
	regIFR               viaRegisterCode = 0x0D // Interrupt Flags Register
	regIER               viaRegisterCode = 0x0E // Interrupt Enable Register
	regORAIRANoHandshake viaRegisterCode = 0x0F // Read / Write ORA / IRA without triggering handshake
)

// IFR flags or bits
type viaIRQFlags uint8

const (
	irqCA2 viaIRQFlags = 0x01 // Sets on CA2 transition, clears on R/W of IRA/ORA
	irqCA1 viaIRQFlags = 0x02 // Sets on CA1 transition, clears on R/W of IRA/ORA
	irqSR  viaIRQFlags = 0x04 // Sets when SR completed shifting 8 bits, clears on R/W of SR
	irqCB2 viaIRQFlags = 0x08 // Sets on CB2 transition, clears on R/W of IRB/ORB
	irqCB1 viaIRQFlags = 0x10 // Sets on CB1 transition, clears on R/W of IRB/ORB
	irqT2  viaIRQFlags = 0x20 // Sets on T2 count to zero, clears on R of T2 low or write T2 high
	irqT1  viaIRQFlags = 0x40 // Sets on T1 count to zero, clears on R T1 counter low or write T1 latch high
	irqAny viaIRQFlags = 0x80 // Sets when IRQ is enabled
)

// The W65C22 (W65C22N and W65C22S) Versatile Interface Adapter (VIA) is a flexible I/O device for use with
// the 65xx series microprocessor family. The W65C22 includes functions for programmed control of two
// peripheral ports (Ports A and B). Two program controlled 8-bit bidirectional peripheral I/O ports allow direct
// interfacing between the microprocessor and selected peripheral units. Each port has input data latching
// capability. Two programmable Data Direction Registers (A and B) allow selection of data direction (input or
// output) on an individual line basis. Also provided are two programmable 16-bit Interval Timer/Counters with
// latches. Timer 1 may be operated in a One-Shot Interrupt Mode with interrupts on each count to zero, or in a
// Free-Run Mode with a continuous series of evenly spaced interrupts. Timer 2 functions as both an interval
// and pulse counter. Serial Data transfers are provided by a serial to parallel/parallel to serial shift register.
// Application versatility is further increased by various control registers, including an Interrupt Flag Register, an
// Interrupt Enable Register and two Function Control Registers.
type Via65C22S struct {
	chipSelect1    *buses.ConnectorEnabledHigh
	chipSelect2    *buses.ConnectorEnabledLow
	dataBus        *buses.BusConnector[uint8]
	irqRequest     *buses.ConnectorEnabledLow
	reset          *buses.ConnectorEnabledLow
	registerSelect [numOfRSLines]*buses.ConnectorEnabledHigh
	readWrite      *buses.ConnectorEnabledLow

	registers *via65C22SRegisters

	peripheralPortA *viaPort
	peripheralPortB *viaPort
	latchesA        *viaLatches
	latchesB        *viaLatches
	timer1          *viaTimer
	timer2          *viaTimer
	controlLinesA   *viaControlLines
	controlLinesB   *viaControlLines
	shifter         *viaShifter

	registerReadHandlers  []func(*Via65C22S)
	registerWriteHandlers []func(*Via65C22S)
}

// Creates a VIA65C22S chip
func NewVia65C22() *Via65C22S {
	via := Via65C22S{
		chipSelect1: buses.NewConnectorEnabledHigh(),
		chipSelect2: buses.NewConnectorEnabledLow(),
		dataBus:     buses.NewBusConnector[uint8](),
		irqRequest:  buses.NewConnectorEnabledLow(),
		reset:       buses.NewConnectorEnabledLow(),
		registerSelect: [numOfRSLines]*buses.ConnectorEnabledHigh{
			buses.NewConnectorEnabledHigh(),
			buses.NewConnectorEnabledHigh(),
			buses.NewConnectorEnabledHigh(),
			buses.NewConnectorEnabledHigh(),
		},
		readWrite: buses.NewConnectorEnabledLow(),

		registers: &via65C22SRegisters{},
	}

	via.controlLinesA = newViaControlLines(&via, &viaControlLineConfiguration{
		transitionConfigurationMasks: [2]viaPCRTransitionMasks{
			pcrMaskCA1TransitionType,
			pcrMaskCA2TransitionType,
		},
		controlLinesIRQBits: [2]viaIRQFlags{
			irqCA1,
			irqCA2,
		},
	})

	via.controlLinesB = newViaControlLines(&via, &viaControlLineConfiguration{
		transitionConfigurationMasks: [2]viaPCRTransitionMasks{
			pcrMaskCB1TransitionType,
			pcrMaskCB2TransitionType,
		},
		controlLinesIRQBits: [2]viaIRQFlags{
			irqCB1,
			irqCB2,
		},
	})

	via.peripheralPortA = newViaPort(&via, &viaPortConfiguration{
		clearC2OnRWMask: pcrMaskCA2ClearOnRW,
		controlLinesIRQBits: [2]viaIRQFlags{
			irqCA1,
			irqCA2,
		},
		inputRegister:         &via.registers.inputRegisterA,
		outputRegister:        &via.registers.outputRegisterA,
		dataDirectionRegister: &via.registers.dataDirectionRegisterA,
		controlLines:          via.controlLinesA,
	})

	via.peripheralPortB = newViaPort(&via, &viaPortConfiguration{
		clearC2OnRWMask: pcrMaskCB2ClearOnRW,
		controlLinesIRQBits: [2]viaIRQFlags{
			irqCB1,
			irqCB2,
		},
		inputRegister:         &via.registers.inputRegisterB,
		outputRegister:        &via.registers.outputRegisterB,
		dataDirectionRegister: &via.registers.dataDirectionRegisterB,
		controlLines:          via.controlLinesB,
	})

	via.latchesA = newViaLatches(&via, &viaLatchesConfiguration{
		latchingEnabledMasks:    acrMaskLatchingEnabledA,
		outputConfigurationMask: pcrMaskCAOutputMode,
		handshakeMode:           pcrCA2OutputModeHandshake,
		pulseMode:               pcrCA2OutputModePulse,
		fixedModeLow:            pcrCA2OutputModeFixLow,
		fixedModeHigh:           pcrCA2OutputModeFixHigh,
		inputRegister:           &via.registers.inputRegisterA,
		port:                    via.peripheralPortA,
		controlLines:            via.controlLinesA,
	})

	via.latchesB = newViaLatches(&via, &viaLatchesConfiguration{
		latchingEnabledMasks:    acrMaskLatchingEnabledB,
		outputConfigurationMask: pcrMaskCBOutputMode,
		handshakeMode:           pcrCB2OutputModeHandshake,
		pulseMode:               pcrCB2OutputModePulse,
		fixedModeLow:            pcrCB2OutputModeFixLow,
		fixedModeHigh:           pcrCB2OutputModeFixHigh,
		inputRegister:           &via.registers.inputRegisterB,
		port:                    via.peripheralPortB,
		controlLines:            via.controlLinesB,
	})

	via.timer1 = newViaTimer(&via, &viaTimerConfiguration{
		timerInterruptBit: irqT1,
		timerRunModeMask:  acrT1ControlRunModeMask,
		timerOutputMask:   acrT1ControlOutputMask,
		lowLatches:        &via.registers.lowLatches1,
		highLatches:       &via.registers.highLatches1,
		counter:           &via.registers.counter1,
		port:              via.peripheralPortB,
	})

	via.timer2 = newViaTimer(&via, &viaTimerConfiguration{
		timerInterruptBit: irqT2,
		timerRunModeMask:  acrT2ControlRunModeMask,
		lowLatches:        &via.registers.lowLatches2,
		highLatches:       &via.registers.highLatches2,
		counter:           &via.registers.counter2,
		port:              via.peripheralPortA,
	})

	via.shifter = newViaShifter(&via, &viaShifterConfiguration{
		timer:        via.timer2,
		controlLines: via.controlLinesB,
	})

	via.populateRegisterReadHandlers()
	via.populateRegisterWriteHandlers()

	return &via
}

/************************************************************************************
* Handler configuration
*************************************************************************************/

// Populate the handler functions when reading for each of the RS values
func (via *Via65C22S) populateRegisterReadHandlers() {
	via.registerReadHandlers = []func(*Via65C22S){
		inputOutputRegisterBReadHandler,                            // 0x00
		inputOutputRegisterAReadHandler,                            // 0x01
		readFromRecord(&via.registers.dataDirectionRegisterB),      // 0x02
		readFromRecord(&via.registers.dataDirectionRegisterA),      // 0x03
		readT1LowOrderCounter,                                      // 0x04
		readT1HighOrderCounter,                                     // 0x05
		readFromRecord(&via.registers.lowLatches1),                 // 0x06
		readFromRecord(&via.registers.highLatches1),                // 0x07
		readT2LowOrderCounter,                                      // 0x08
		readT2HighOrderCounter,                                     // 0x09
		readShiftRegister,                                          // 0x0A
		readFromRecord((*uint8)(&via.registers.auxiliaryControl)),  // 0x0B
		readFromRecord((*uint8)(&via.registers.peripheralControl)), // 0x0C
		readnterruptFlagHandler,                                    // 0x0D
		readInterruptEnableHandler,                                 // 0x0E
		inputOutputRegisterAReadHandlerNoHandshake,                 // 0x0F
	}
}

// Populate the handler functions when writing each of the RS values
func (via *Via65C22S) populateRegisterWriteHandlers() {
	via.registerWriteHandlers = []func(*Via65C22S){
		inputOutputRegisterBWriteHandler,                          // 0x00
		inputOutputRegisterAWriteHandler,                          // 0x01
		writeToRecord(&via.registers.dataDirectionRegisterB),      // 0x02
		writeToRecord(&via.registers.dataDirectionRegisterA),      // 0x03
		writeToRecord(&via.registers.lowLatches1),                 // 0x04
		writeT1HighOrderCounter,                                   // 0x05
		writeToRecord(&via.registers.lowLatches1),                 // 0x06
		writeT1HighOrderLatch,                                     // 0x07
		writeToRecord(&via.registers.lowLatches2),                 // 0x08
		writeT2HighOrderCounter,                                   // 0x09
		writeShiftRegister,                                        // 0x0A
		writeToRecord((*uint8)(&via.registers.auxiliaryControl)),  // 0x0B
		writeToRecord((*uint8)(&via.registers.peripheralControl)), // 0x0C
		writeInterruptFlagHandler,                                 // 0x0D
		writeInterruptEnableHandler,                               // 0x0E
		inputOutputRegisterAWriteHandlerNoHandshake,               // 0x0F
	}
}

/************************************************************************************
* Pin Getters / Setters
*************************************************************************************/

// Retuns a reference to the specified peripherial control line A (CA1 and CA2).
// Line is zero based so CA1 is 0 and CA2 is 1
// Returns nil if an invalid line number is specified
func (via *Via65C22S) PeripheralAControlLines(num int) *buses.ConnectorEnabledHigh {
	return via.controlLinesA.getLine(num)
}

// Retuns a reference to the specified peripherial control line B (CB1 and CB2).
// Line is zero based so CB1 is 0 and CB2 is 1
// Returns nil if an invalid line number is specified
func (via *Via65C22S) PeripheralBControlLines(num int) *buses.ConnectorEnabledHigh {
	return via.controlLinesB.getLine(num)
}

// Chip select line 1 CS1
func (via *Via65C22S) ChipSelect1() *buses.ConnectorEnabledHigh {
	return via.chipSelect1
}

// Chip select line 2 CS2B
func (via *Via65C22S) ChipSelect2() *buses.ConnectorEnabledLow {
	return via.chipSelect2
}

// Returns a reference to the data bus connector.
func (via *Via65C22S) DataBus() *buses.BusConnector[uint8] {
	return via.dataBus
}

// Returns a reference to the IRQ line
func (via *Via65C22S) IrqRequest() *buses.ConnectorEnabledLow {
	return via.irqRequest
}

// Returns a reference to port A connector
func (via *Via65C22S) PeripheralPortA() *buses.BusConnector[uint8] {
	return via.peripheralPortA.getConnector()
}

// Returns a reference to port B connector
func (via *Via65C22S) PeripheralPortB() *buses.BusConnector[uint8] {
	return via.peripheralPortB.getConnector()
}

// Returns a reference to the reset line
func (via *Via65C22S) Reset() *buses.ConnectorEnabledLow {
	return via.reset
}

// Returns a reference to the register select lines.
// It's zero based so RS1 is 0, RS 2 is 1, etc.
func (via *Via65C22S) RegisterSelect(num uint8) *buses.ConnectorEnabledHigh {
	if num >= numOfRSLines {
		panic("Register select line number out of range")
	}

	return via.registerSelect[num]
}

// Returns a reference to the R/W line.
func (via *Via65C22S) ReadWrite() *buses.ConnectorEnabledLow {
	return via.readWrite
}

// Connects the register select lines to the specified lines
func (via *Via65C22S) ConnectRegisterSelectLines(lines [numOfRSLines]buses.Line) {
	for i := range numOfRSLines {
		via.registerSelect[i].Connect(lines[i])
	}
}

/************************************************************************************
* Internal functions
*************************************************************************************/

// Returns the register select value based on the status of the RS lines
func (via *Via65C22S) getRegisterSelectValue() viaRegisterCode {
	var value uint8

	for i := range 4 {
		if via.registerSelect[i].Enabled() {
			value += uint8(math.Pow(2, float64(i)))
		}
	}

	return viaRegisterCode(value)
}

// Sets or clears the IRQ line based on the chip status
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

// Executes one emulation step
func (via *Via65C22S) Tick(context *common.StepContext) {
	// From https://lateblt.tripod.com/bit67.txt:
	// The ORs are also never transparent Whereas an input bus which has input latching turned off can change with its
	// input without the Enable pin even being cycled, outputting to an OR will not take effect until the Enable pin has made
	// a transition to low or high.
	via.latchesA.latchPort()
	via.latchesB.latchPort()

	var pbLine6Status bool = false
	pbLine6 := via.peripheralPortB.connector.GetLine(6)
	if pbLine6 != nil {
		pbLine6Status = pbLine6.Status()
	}

	via.timer1.tick(true)
	via.timer2.tick(pbLine6Status)

	via.shifter.tick()

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
		via.peripheralPortA.writePortOutputRegister()
		via.peripheralPortB.writePortOutputRegister()
	}

	via.timer1.setControlLinesBasedOnTimerStatus()
	via.timer2.setControlLinesBasedOnTimerStatus()

	via.shifter.setControlLinesBasedOnShifterStatus()

	via.latchesA.setOutput()
	via.latchesB.setOutput()

	via.controlLinesA.setInterruptFlagOnControlLinesTransition()
	via.controlLinesB.setInterruptFlagOnControlLinesTransition()

	via.controlLinesA.storePreviousControlLinesValues()
	via.controlLinesB.storePreviousControlLinesValues()

	via.handleIRQLine()
}

/************************************************************************************
* Internal Registers Getters
*************************************************************************************/

func (via *Via65C22S) GetOutputRegisterA() uint8 {
	return via.registers.outputRegisterA
}

func (via *Via65C22S) GetOutputRegisterB() uint8 {
	return via.registers.outputRegisterB
}

func (via *Via65C22S) GetInputRegisterA() uint8 {
	return via.registers.inputRegisterA
}

func (via *Via65C22S) GetInputRegisterB() uint8 {
	return via.registers.inputRegisterB
}

func (via *Via65C22S) GetDataDirectionRegisterA() uint8 {
	return via.registers.dataDirectionRegisterA
}

func (via *Via65C22S) GetDataDirectionRegisterB() uint8 {
	return via.registers.dataDirectionRegisterB
}

func (via *Via65C22S) GetLowLatches2() uint8 {
	return via.registers.lowLatches2
}

func (via *Via65C22S) GetLowLatches1() uint8 {
	return via.registers.lowLatches1
}

func (via *Via65C22S) GetHighLatches2() uint8 {
	return via.registers.highLatches2
}

func (via *Via65C22S) GetHighLatches1() uint8 {
	return via.registers.highLatches1
}

func (via *Via65C22S) GetCounter2() uint16 {
	return via.registers.counter2
}

func (via *Via65C22S) GetCounter1() uint16 {
	return via.registers.counter1
}

func (via *Via65C22S) GetShiftRegister() uint8 {
	return via.registers.shiftRegister
}

func (via *Via65C22S) GetAuxiliaryControl() uint8 {
	return via.registers.auxiliaryControl
}

func (via *Via65C22S) GetPeripheralControl() uint8 {
	return via.registers.peripheralControl
}

func (via *Via65C22S) GetInterruptFlagValue() uint8 {
	return via.registers.interrupts.getInterruptFlagValue()
}

func (via *Via65C22S) GetInterruptEnabledFlag() uint8 {
	return via.registers.interrupts.getInterruptEnabledFlag()
}
