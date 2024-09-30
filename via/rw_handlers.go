package via

/************************************************************************************
* Input / Output Register B
*************************************************************************************/

func inputOutputRegisterBReadHandler(via *Via65C22S) {
	// Get the data direction register
	outputPins := via.sideB.registers.dataDirectionRegister
	inputPins := ^outputPins

	// MPU reads output register bit in ORB. Pin level has no effect.
	value := via.sideB.registers.outputRegister & outputPins

	if !via.sideB.peripheralPort.isLatchingEnabled() {
		// MPU reads input level on PB pin.
		value |= via.sideB.peripheralPort.getConnector().Read() & inputPins
	} else {
		// MPU reads IRB bit
		value |= via.sideB.registers.inputRegister & inputPins
	}

	via.sideB.peripheralPort.clearControlLinesInterruptFlagOnRW()

	via.dataBus.Write(value)
}

func inputOutputRegisterBWriteHandler(via *Via65C22S) {
	mode := via.sideB.controlLines.getOutputMode()

	if mode == pcrCB2OutputModeHandshake || mode == pcrCB2OutputModePulse {
		via.sideB.controlLines.initHandshake()
	}

	via.sideB.peripheralPort.clearControlLinesInterruptFlagOnRW()

	// MPU writes to ORB
	via.sideB.registers.outputRegister = via.dataBus.Read()
}

/************************************************************************************
* Input / Output Register A
*************************************************************************************/

func inputOutputRegisterAReadHandler(via *Via65C22S) {
	var value uint8

	if !via.sideA.peripheralPort.isLatchingEnabled() {
		value = via.sideA.peripheralPort.connector.Read()
	} else {
		value = via.sideA.registers.inputRegister
	}

	mode := via.sideA.controlLines.getOutputMode()

	if mode == pcrCA2OutputModeHandshake || mode == pcrCA2OutputModePulse {
		via.sideA.controlLines.initHandshake()
	}

	via.sideA.peripheralPort.clearControlLinesInterruptFlagOnRW()

	via.dataBus.Write(value)
}

func inputOutputRegisterAWriteHandler(via *Via65C22S) {
	mode := via.sideA.controlLines.getOutputMode()

	if mode == pcrCA2OutputModeHandshake || mode == pcrCA2OutputModePulse {
		via.sideA.controlLines.initHandshake()
	}

	via.sideA.peripheralPort.clearControlLinesInterruptFlagOnRW()

	// MPU writes to ORA
	via.sideA.registers.outputRegister = via.dataBus.Read()
}

/************************************************************************************
* These handlers directly updates the value of the record
*************************************************************************************/

func readFromRecord(register *uint8) func(via *Via65C22S) {
	return func(via *Via65C22S) {
		via.dataBus.Write(*register)
	}
}

func writeToRecord(register *uint8) func(via *Via65C22S) {
	return func(via *Via65C22S) {
		*register = via.dataBus.Read()
	}
}

/************************************************************************************
* Reads and writes the Interrupt Flag Register
*************************************************************************************/

func readnterruptFlagHandler(via *Via65C22S) {
	via.dataBus.Write(via.registers.interrupts.getInterruptFlagValue())
}

func writeInterruptFlagHandler(via *Via65C22S) {
	via.registers.interrupts.setInterruptFlagValue(via.dataBus.Read())
}

/************************************************************************************
* Reads and writes the Interrupt Enable Register
*************************************************************************************/

// The processor can read the contents of this register by placing the proper address
// on the register select and chip select inputs with the R/W line high. Bit 7 will
// read as a logic 0.
func readInterruptEnableHandler(via *Via65C22S) {
	via.dataBus.Write(via.registers.interrupts.getInterruptEnabledFlag())
}

// If bit 7 of the data placed on the system data bus during this write operation is a 0,
// each 1 in bits 6 through 0 clears the corresponding bit in the Interrupt Enable Register.
// Setting selected bits in the Interrupt Enable Register is accomplished by writing to
// the same address with bit 7 in the data word set to a logic 1.
// In this case, each 1 in bits 6 through 0 will set the corresponding bit. For each zero,
// the corresponding bit will be unaffected. T
func writeInterruptEnableHandler(via *Via65C22S) {
	via.registers.interrupts.setInterruptEnabledFlag(via.dataBus.Read())
}

/************************************************************************************
* Writes / Reads to T1 Low and High order counters and latches
*************************************************************************************/

func writeT1HighOrderCounter(via *Via65C22S) {
	// MSB value for the current value in the bus
	var high uint16 = uint16(via.dataBus.Read()) << 8

	// Write into high order latch
	via.sideB.registers.highLatches = via.dataBus.Read()
	// Write into high order counter (first clear MSB and then assign)
	via.sideB.registers.counter = (via.sideB.registers.counter & 0x00FF) | high

	// Transfer low order latch to low order counter
	via.sideB.registers.counter = (via.sideB.registers.counter & 0xFF00) | uint16(via.sideB.registers.lowLatches)

	// Enable the counter
	via.sideB.timer.timerEnabled = true
	via.sideB.timer.outputStatusWhenEnabled = false

	// Clears interrupt flags
	via.registers.interrupts.clearInterruptFlagBit(irqT1)
}

func writeT1HighOrderLatch(via *Via65C22S) {
	// Write into high order latch
	via.sideB.registers.highLatches = via.dataBus.Read()

	// Clear interrupt flags
	via.registers.interrupts.clearInterruptFlagBit(irqT1)
}

// Reads the LSB from the counter
func readT1LowOrderCounter(via *Via65C22S) {
	via.dataBus.Write(uint8(via.sideB.registers.counter))

	// Clear interrupt flags
	via.registers.interrupts.clearInterruptFlagBit(irqT1)
}

// Reads the MSB from the counter
func readT1HighOrderCounter(via *Via65C22S) {
	// Makes 0 the LSB and moves the MSB to the lower byte
	value := (via.sideB.registers.counter & 0xFF00) >> 8
	// Writes the value on the bus
	via.dataBus.Write(uint8(value))
}

/************************************************************************************
* Writes / Reads to T2 Low and High order counters and latches
*************************************************************************************/

func writeT2HighOrderCounter(via *Via65C22S) {
	// MSB value for the current value in the bus
	var high uint16 = uint16(via.dataBus.Read()) << 8

	// Write into high order latch
	via.sideA.registers.highLatches = via.dataBus.Read()
	// Write into high order counter (first clear MSB and then assign)
	via.sideA.registers.counter = (via.sideA.registers.counter & 0x00FF) | high

	// Transfer low order latch to low order counter
	via.sideA.registers.counter = (via.sideA.registers.counter & 0xFF00) | uint16(via.sideA.registers.lowLatches)

	// Enable the counter
	via.sideA.timer.timerEnabled = true
	via.sideA.timer.outputStatusWhenEnabled = false

	// Clears interrupt flags
	via.registers.interrupts.clearInterruptFlagBit(irqT2)
}

// Reads the LSB from the counter
func readT2LowOrderCounter(via *Via65C22S) {
	via.dataBus.Write(uint8(via.sideA.registers.counter))

	// Clears interrupt flags
	via.registers.interrupts.clearInterruptFlagBit(irqT2)
}

// Reads the MSB from the counter
func readT2HighOrderCounter(via *Via65C22S) {
	// Makes 0 the LSB and moves the MSB to the lower byte
	value := (via.sideA.registers.counter & 0xFF00) >> 8
	// Writes the value on the bus
	via.dataBus.Write(uint8(value))
}

/************************************************************************************
* Temporary
*************************************************************************************/

func dummyHandler(via *Via65C22S) {
}
