package via

import (
	"math"

	"github.com/fran150/clementina6502/buses"
)

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

	registers map[viaRegisterCode]*uint8
}

type viaRegisterCode uint8

const (
	outputRegisterB            viaRegisterCode = 0x00
	inputRegisterB             viaRegisterCode = 0x10
	outputRegisterA            viaRegisterCode = 0x01
	inputRegisterA             viaRegisterCode = 0x11
	dataDirectionRegisterB     viaRegisterCode = 0x02
	dataDirectionRegisterA     viaRegisterCode = 0x03
	t1LowLatches               viaRegisterCode = 0x04
	t1LowCounter               viaRegisterCode = 0x14
	t1HighCounter              viaRegisterCode = 0x05
	t1LowLatches2              viaRegisterCode = 0x06
	t1HighLatches              viaRegisterCode = 0x07
	t2LowLatches               viaRegisterCode = 0x08
	t2LowCounter               viaRegisterCode = 0x18
	t2HighCounter              viaRegisterCode = 0x09
	shiftRegister              viaRegisterCode = 0x0A
	auxiliaryControl           viaRegisterCode = 0x0B
	peripheralControl          viaRegisterCode = 0x0C
	interruptFlag              viaRegisterCode = 0x0D
	interruptEnable            viaRegisterCode = 0x0E
	outputRegisterBNoHandshake viaRegisterCode = 0x0F
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

		registers: map[viaRegisterCode]*uint8{
			outputRegisterB:            new(uint8),
			inputRegisterB:             new(uint8),
			outputRegisterA:            new(uint8),
			inputRegisterA:             new(uint8),
			dataDirectionRegisterB:     new(uint8),
			dataDirectionRegisterA:     new(uint8),
			t1LowLatches:               new(uint8),
			t1LowCounter:               new(uint8),
			t1HighCounter:              new(uint8),
			t1LowLatches2:              new(uint8),
			t1HighLatches:              new(uint8),
			t2LowLatches:               new(uint8),
			t2LowCounter:               new(uint8),
			t2HighCounter:              new(uint8),
			shiftRegister:              new(uint8),
			auxiliaryControl:           new(uint8),
			peripheralControl:          new(uint8),
			interruptFlag:              new(uint8),
			interruptEnable:            new(uint8),
			outputRegisterBNoHandshake: new(uint8),
		},
	}

	return &via
}

/************************************************************************************
* Handler configuration
*************************************************************************************/

var registerReadHandlers = []func(*Via65C22S, *uint8){
	inputOutuputRegisterBReadHandler, // 0x00
	inputOutuputRegisterAReadHandler, // 0x01
	readFromRecord,                   // 0x02
	readFromRecord,                   // 0x03
	dummyHandler,                     // 0x04
	dummyHandler,                     // 0x05
	dummyHandler,                     // 0x06
	dummyHandler,                     // 0x07
	dummyHandler,                     // 0x08
	dummyHandler,                     // 0x09
	dummyHandler,                     // 0x0A
	readFromRecord,                   // 0x0B
	readFromRecord,                   // 0x0C
	readInterruptFlagHandler,         // 0x0D
	readInterruptEnableHandler,       // 0x0E
	dummyHandler,                     // 0x0F
}

var registerWriteHandlers = []func(*Via65C22S, *uint8){
	inputOutuputRegisterBWriteHandler, // 0x00
	inputOutuputRegisterAWriteHandler, // 0x01
	writeToRecord,                     // 0x02
	writeToRecord,                     // 0x03
	dummyHandler,                      // 0x04
	dummyHandler,                      // 0x05
	dummyHandler,                      // 0x06
	dummyHandler,                      // 0x07
	dummyHandler,                      // 0x08
	dummyHandler,                      // 0x09
	dummyHandler,                      // 0x0A
	writeToRecord,                     // 0x0B
	writeToRecord,                     // 0x0C
	writeInterruptFlagHandler,         // 0x0D
	writeInterruptEnableHandler,       // 0x0E
	dummyHandler,                      // 0x0F
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

func (via *Via65C22S) getRegisterSelectValue() uint8 {
	var value uint8

	for i := range 4 {
		if via.registerSelect[i].Enabled() {
			value += uint8(math.Pow(2, float64(i)))
		}
	}

	return value
}

func (via *Via65C22S) isCA1ConfiguredForPositiveTransition() bool {
	// Get the peripheral control register
	pcr := via.registers[peripheralControl]

	// Bit 0 in PCR determines CA 1 control. 1 is latch on positive edge, 0 is latch on negative edge
	return (*pcr & 0x01) > 0
}

func (via *Via65C22S) isCA2ConfiguredForPositiveTransition() bool {
	// Get the peripheral control register
	pcr := via.registers[peripheralControl]

	// Bit 1, 2 and 3 in PCR determines CA 2 control.
	// 000 and 001 transition on negative edge
	// 010 and 011 transition on positive edge
	return ((*pcr & 0x0E) == 0x4) || ((*pcr & 0x0E) == 0x6)
}

func (via *Via65C22S) isCA2ConfiguredForClearOnRW() bool {
	// Get the peripheral control register
	pcr := via.registers[peripheralControl]

	// Bit 1, 2 and 3 in PCR determines CA 2 control.
	// 000 and 010 allows clearing CA2 interrupt when Reading or Writing ORA/IRA
	return ((*pcr & 0x0E) == 0x00) || ((*pcr & 0x0E) == 0x04)
}

func (via *Via65C22S) isCB1ConfiguredForPositiveTransition() bool {
	// Get the peripheral control register
	pcr := via.registers[peripheralControl]

	// Bit 4 in PCR determines CB 1 control. 1 is latch on positive edge, 0 is latch on negative edge
	return (*pcr & 0x10) > 0
}

func (via *Via65C22S) isCB2ConfiguredForPositiveTransition() bool {
	// Get the peripheral control register
	pcr := via.registers[peripheralControl]

	// Bit 7, 6 and 5 in PCR determines CA B control.
	// 000 and 001 transition on negative edge
	// 010 and 011 transition on positive edge
	return ((*pcr & 0xE0) == 0x40) || ((*pcr & 0xE0) == 0x60)
}

func (via *Via65C22S) isCB2ConfiguredForClearOnRW() bool {
	// Get the peripheral control register
	pcr := via.registers[peripheralControl]

	// Bit 7, 6 and 5 in PCR determines CA 2 control.
	// 000 and 010 allows clearing CA2 interrupt when Reading or Writing ORA/IRA
	return ((*pcr & 0xE0) == 0x00) || ((*pcr & 0xE0) == 0x40)
}

func (via *Via65C22S) isCA2ConfiguredForOutput() bool {
	// Get the peripheral control register
	pcr := via.registers[peripheralControl]

	// Bit 3 determines if output mode is active
	return (*pcr & 0x08) > 0x00
}

func (via *Via65C22S) isCB2ConfiguredForOutput() bool {
	// Get the peripheral control register
	pcr := via.registers[peripheralControl]

	// Bit 7 determines if output mode is active
	return (*pcr & 0x80) > 0x00
}

func (via *Via65C22S) isCA2ConfiguredForHandshake() bool {
	// Get the peripheral control register
	pcr := via.registers[peripheralControl]

	return (*pcr & 0x0C) == 0x08
}

func (via *Via65C22S) isCB2ConfiguredForHandshake() bool {
	// Get the peripheral control register
	pcr := via.registers[peripheralControl]

	return (*pcr & 0xC0) > 0x80
}

func (via *Via65C22S) setCAOutputMode() {
	// Get the peripheral control register
	pcr := via.registers[peripheralControl]

	switch *pcr & 0x0E {
	case 0x08:
		if via.caHandshakeInProgress && via.checkTransitionedCA1() {
			via.caHandshakeInProgress = false
		}

		if via.caHandshakeInProgress {
			via.peripheralAControlLines[1].SetEnable(false)
		} else {
			via.peripheralAControlLines[1].SetEnable(true)
		}
	case 0x0A:
		if via.caHandshakeInProgress && !via.peripheralAControlLines[1].Enabled() {
			via.caHandshakeInProgress = false
		}

		if via.caHandshakeInProgress {
			via.peripheralAControlLines[1].SetEnable(false)
		} else {
			via.peripheralAControlLines[1].SetEnable(true)
		}
	case 0x0C:
		via.peripheralAControlLines[1].SetEnable(false)
	case 0x0E:
		via.peripheralAControlLines[1].SetEnable(true)
	}
}

func (via *Via65C22S) setCBOutputMode() {
	// Get the peripheral control register
	pcr := via.registers[peripheralControl]

	switch *pcr & 0xE0 {
	case 0x80:
		if via.cbHandshakeInProgress && via.checkTransitionedCB1() {
			via.cbHandshakeInProgress = false
		}

		if via.cbHandshakeInProgress {
			via.peripheralBControlLines[1].SetEnable(false)
		} else {
			via.peripheralBControlLines[1].SetEnable(true)
		}
	case 0xA0:
		if via.cbHandshakeInProgress && !via.peripheralBControlLines[1].Enabled() {
			via.cbHandshakeInProgress = false
		}

		if via.cbHandshakeInProgress {
			via.peripheralBControlLines[1].SetEnable(false)
		} else {
			via.peripheralBControlLines[1].SetEnable(true)
		}
	case 0xC0:
		via.peripheralBControlLines[1].SetEnable(false)
	case 0xE0:
		via.peripheralBControlLines[1].SetEnable(true)
	}
}

func (via *Via65C22S) checkTransitionedCA1() bool {
	caCtrlPositive := via.isCA1ConfiguredForPositiveTransition()

	currentCrl := via.peripheralAControlLines[0].Enabled()
	previousCtrl := via.previousCtrlA[0]

	// If transition is configured on positive edge then transition happened if previous CA value was low and now is high
	// If transition is configured on negative edge then transition happened if previous CA value was high nad now is low
	return (caCtrlPositive && !previousCtrl && currentCrl) || (!caCtrlPositive && previousCtrl && !currentCrl)
}

func (via *Via65C22S) checkTransitionedCA2() bool {
	caCtrlPositive := via.isCA2ConfiguredForPositiveTransition()

	currentCrl := via.peripheralAControlLines[1].Enabled()
	previousCtrl := via.previousCtrlA[1]

	// If transition is configured on positive edge then transition happened if previous CA value was low and now is high
	// If transition is configured on negative edge then transition happened if previous CA value was high nad now is low
	return (caCtrlPositive && !previousCtrl && currentCrl) || (!caCtrlPositive && previousCtrl && !currentCrl)
}

func (via *Via65C22S) checkTransitionedCB1() bool {
	cbCtrlPositive := via.isCB1ConfiguredForPositiveTransition()

	currentCrl := via.peripheralBControlLines[0].Enabled()
	previousCtrl := via.previousCtrlB[0]

	// If transition is configured on positive edge then transition happened if previous CB value was low and now is high
	// If transition is configured on negative edge then transition happened if previous CB value was high nad now is low
	return (cbCtrlPositive && !previousCtrl && currentCrl) || (!cbCtrlPositive && previousCtrl && !currentCrl)

}

func (via *Via65C22S) checkTransitionedCB2() bool {
	cbCtrlPositive := via.isCB2ConfiguredForPositiveTransition()

	currentCrl := via.peripheralBControlLines[1].Enabled()
	previousCtrl := via.previousCtrlB[1]

	// If transition is configured on positive edge then transition happened if previous CB value was low and now is high
	// If transition is configured on negative edge then transition happened if previous CB value was high nad now is low
	return (cbCtrlPositive && !previousCtrl && currentCrl) || (!cbCtrlPositive && previousCtrl && !currentCrl)
}

func (via *Via65C22S) isLatchingEnabledPortA() bool {
	// Get the auxiliary control register
	acr := via.registers[auxiliaryControl]

	// Bit 0 of ACR determines if latching is enabled for port A
	return (*acr & 0x01) > 0
}

func (via *Via65C22S) isLatchingEnabledPortB() bool {
	// Get the auxiliary control register
	acr := via.registers[auxiliaryControl]

	// Bit 1 of ACR determines if latching is enabled for port B
	return (*acr & 0x02) > 0
}

func (via *Via65C22S) latchPortA() {
	// Read pin levels on port A
	value := via.peripheralPortA.Read()

	// Get the IRA register
	register := via.registers[inputRegisterA]

	// Get the data direction register
	ddr := via.registers[dataDirectionRegisterA]

	// Read pins are all the ones with 0 in the DDR
	readPins := ^*ddr

	if via.isLatchingEnabledPortA() {
		// If latching is enabled value is the one at the time of CB transition
		if via.checkTransitionedCA1() {
			*register = value & readPins
		}
	}
}

func (via *Via65C22S) latchPortB() {
	// Read pin levels on port B
	value := via.peripheralPortB.Read()

	// Get the ORB / IRB register
	register := via.registers[inputRegisterB]

	// Get the data direction register
	ddr := via.registers[dataDirectionRegisterB]

	// Read pins are all the ones with 0 in the DDR
	readPins := ^*ddr

	if via.isLatchingEnabledPortB() {
		// If latching is enabled value is the one at the time of CB transition
		if via.checkTransitionedCB1() {
			*register = value & readPins
		}
	}
}

func (via *Via65C22S) isByteSet(value uint8, bitNumber uint8) bool {
	mask := uint8(math.Pow(2, float64(bitNumber)))

	return (value & mask) > 0
}

func (via *Via65C22S) writePortA() {
	register := via.registers[outputRegisterA]

	// Bytes in 1 are the output pins
	outputPins := via.registers[dataDirectionRegisterA]

	for i := range uint8(8) {
		if via.isByteSet(*outputPins, i) {
			via.PeripheralPortA().GetLine(i).Set(via.isByteSet(*register, i))
		}
	}
}

func (via *Via65C22S) writePortB() {
	register := via.registers[outputRegisterB]

	// Bytes in 1 are the output pins
	outputPins := via.registers[dataDirectionRegisterB]

	for i := range uint8(8) {
		if via.isByteSet(*outputPins, i) {
			via.PeripheralPortB().GetLine(i).Set(via.isByteSet(*register, i))
		}
	}
}

// If any of the bits 0 - 6 in the IFR is 1 then the bit 7 is 1
// If not, then the bit 7 is 0.
func (via *Via65C22S) writeInterruptFlagRegister(value uint8) {
	ier := via.registers[interruptEnable]

	if (value & *ier & 0x7F) > 0 {
		value |= 0x80
	} else {
		value &= 0x7F
	}

	ifr := via.registers[interruptFlag]
	*ifr = value
}

func (via *Via65C22S) setInterruptFlag(flag viaIRQFlags) {
	register := via.registers[interruptFlag]
	via.writeInterruptFlagRegister(*register | uint8(flag))
}

func (via *Via65C22S) clearInterruptFlag(flag viaIRQFlags) {
	register := via.registers[interruptFlag]
	via.writeInterruptFlagRegister(*register & ^uint8(flag))
}

func (via *Via65C22S) storePreviousControlLinesValues() {
	via.previousCtrlA[0] = via.peripheralAControlLines[0].Enabled()
	via.previousCtrlB[0] = via.peripheralBControlLines[0].Enabled()
	via.previousCtrlA[1] = via.peripheralAControlLines[1].Enabled()
	via.previousCtrlB[1] = via.peripheralBControlLines[1].Enabled()
}

func (via *Via65C22S) setInterruptFlagOnControlLinesTransition() {
	if via.checkTransitionedCA1() {
		via.setInterruptFlag(irqCA1)
	}

	if via.checkTransitionedCA2() {
		via.setInterruptFlag(irqCA2)
	}

	if via.checkTransitionedCB1() {
		via.setInterruptFlag(irqCB1)
	}

	if via.checkTransitionedCB2() {
		via.setInterruptFlag(irqCB2)
	}
}

func (via *Via65C22S) clearControlLinesInterruptFlagOnRWPortA() {
	via.clearInterruptFlag(irqCA1)

	if via.isCA2ConfiguredForClearOnRW() {
		via.clearInterruptFlag(irqCA2)
	}
}

func (via *Via65C22S) clearControlLinesInterruptFlagOnRWPortB() {
	via.clearInterruptFlag(irqCB1)

	if via.isCB2ConfiguredForClearOnRW() {
		via.clearInterruptFlag(irqCB2)
	}
}

func (via *Via65C22S) handleIRQLine() {
	ifr := via.registers[interruptFlag]
	ier := via.registers[interruptEnable]

	if (*ifr & *ier & 0x7F) > 0 {
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
		var selectedRegister *uint8

		selectedRegisterValue := via.getRegisterSelectValue()

		if !via.readWrite.Enabled() {
			var ok bool
			selectedRegisterValue += 0x10

			selectedRegister, ok = via.registers[viaRegisterCode(selectedRegisterValue)]

			if !ok {
				selectedRegister = via.registers[viaRegisterCode(selectedRegisterValue-0x10)]
			}
		} else {
			selectedRegister = via.registers[viaRegisterCode(selectedRegisterValue)]
		}

		if !via.readWrite.Enabled() {
			registerReadHandlers[selectedRegisterValue&0x0F](via, selectedRegister)
		} else {
			registerWriteHandlers[selectedRegisterValue&0x0F](via, selectedRegister)
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

	if via.isCA2ConfiguredForOutput() {
		via.setCAOutputMode()
	}

	if via.isCB2ConfiguredForOutput() {
		via.setCBOutputMode()
	}

	via.setInterruptFlagOnControlLinesTransition()
	via.storePreviousControlLinesValues()
	via.handleIRQLine()
}

/************************************************************************************
* Input / Output Register B
*************************************************************************************/

func inputOutuputRegisterBReadHandler(via *Via65C22S, register *uint8) {
	// Get the data direction register
	outputPins := via.registers[dataDirectionRegisterB]
	inputPins := ^*outputPins

	irb := via.registers[inputRegisterB]
	orb := via.registers[outputRegisterB]

	// MPU reads output register bit in ORB. Pin level has no effect.
	value := *orb & *outputPins

	if !via.isLatchingEnabledPortB() {
		// MPU reads input level on PB pin.
		value |= via.peripheralPortB.Read() & inputPins
	} else {
		// MPU reads IRB bit
		value |= *irb & inputPins
	}

	via.clearControlLinesInterruptFlagOnRWPortB()

	via.dataBus.Write(value)
}

func inputOutuputRegisterBWriteHandler(via *Via65C22S, register *uint8) {
	if via.isCB2ConfiguredForHandshake() {
		via.cbHandshakeInProgress = true
	}

	via.clearControlLinesInterruptFlagOnRWPortB()

	// MPU writes to ORB
	*register = via.dataBus.Read()
}

/************************************************************************************
* Input / Output Register A
*************************************************************************************/

func inputOutuputRegisterAReadHandler(via *Via65C22S, register *uint8) {
	var value uint8

	if !via.isLatchingEnabledPortA() {
		value = via.peripheralPortA.Read()
	} else {
		value = *register
	}

	if via.isCA2ConfiguredForHandshake() {
		via.caHandshakeInProgress = true
	}

	via.clearControlLinesInterruptFlagOnRWPortA()

	via.dataBus.Write(value)
}

func inputOutuputRegisterAWriteHandler(via *Via65C22S, register *uint8) {
	if via.isCA2ConfiguredForHandshake() {
		via.caHandshakeInProgress = true
	}

	via.clearControlLinesInterruptFlagOnRWPortA()

	// MPU writes to ORA
	*register = via.dataBus.Read()
}

/************************************************************************************
* These handlers directly updates the value of the record
*************************************************************************************/

func readFromRecord(via *Via65C22S, register *uint8) {
	via.dataBus.Write(*register)
}

func writeToRecord(via *Via65C22S, register *uint8) {
	*register = via.dataBus.Read()
}

/************************************************************************************
* Reads and writes the Interrupt Flag Register
*************************************************************************************/

func readInterruptFlagHandler(via *Via65C22S, register *uint8) {
	via.dataBus.Write(*register)
}

func writeInterruptFlagHandler(via *Via65C22S, _ *uint8) {
	via.writeInterruptFlagRegister(via.dataBus.Read())
}

/************************************************************************************
* Reads and writes the Interrupt Enable Register
*************************************************************************************/

// The processor can read the contents of this register by placing the proper address
// on the register select and chip select inputs with the R/W line high. Bit 7 will
// read as a logic 0.
func readInterruptEnableHandler(via *Via65C22S, register *uint8) {
	via.dataBus.Write(*register & 0x7F)
}

// If bit 7 of the data placed on the system data bus during this write operation is a 0,
// each 1 in bits 6 through 0 clears the corresponding bit in the Interrupt Enable Register.
// Setting selected bits in the Interrupt Enable Register is accomplished by writing to
// the same address with bit 7 in the data word set to a logic 1.
// In this case, each 1 in bits 6 through 0 will set the corresponding bit. For each zero,
// the corresponding bit will be unaffected. T
func writeInterruptEnableHandler(via *Via65C22S, register *uint8) {
	mustSet := (via.dataBus.Read() & 0x80) > 0
	value := via.dataBus.Read() & 0x7F

	if mustSet {
		*register = *register | value
	} else {
		*register = *register & ^value
	}
}

/************************************************************************************
* Temporary
*************************************************************************************/

func dummyHandler(via *Via65C22S, register *uint8) {
}
