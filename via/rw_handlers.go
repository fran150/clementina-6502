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

	via.clearControlLinesInterruptFlagOnRWPortB()

	via.dataBus.Write(value)
}

func inputOutputRegisterBWriteHandler(via *Via65C22S) {
	mode := via.sideB.controlLines.getOutputMode()

	if mode == pcrCB2OutputModeHandshake || mode == pcrCB2OutputModePulse {
		via.sideB.controlLines.initHandshake()
	}

	via.clearControlLinesInterruptFlagOnRWPortB()

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

	via.clearControlLinesInterruptFlagOnRWPortA()

	via.dataBus.Write(value)
}

func inputOutputRegisterAWriteHandler(via *Via65C22S) {
	mode := via.sideA.controlLines.getOutputMode()

	if mode == pcrCA2OutputModeHandshake || mode == pcrCA2OutputModePulse {
		via.sideA.controlLines.initHandshake()
	}

	via.clearControlLinesInterruptFlagOnRWPortA()

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

func writeInterruptFlagHandler(via *Via65C22S) {
	via.writeInterruptFlagRegister(via.dataBus.Read())
}

/************************************************************************************
* Reads and writes the Interrupt Enable Register
*************************************************************************************/

// The processor can read the contents of this register by placing the proper address
// on the register select and chip select inputs with the R/W line high. Bit 7 will
// read as a logic 0.
func readInterruptEnableHandler(via *Via65C22S) {
	via.dataBus.Write(via.registers.interruptEnable & 0x7F)
}

// If bit 7 of the data placed on the system data bus during this write operation is a 0,
// each 1 in bits 6 through 0 clears the corresponding bit in the Interrupt Enable Register.
// Setting selected bits in the Interrupt Enable Register is accomplished by writing to
// the same address with bit 7 in the data word set to a logic 1.
// In this case, each 1 in bits 6 through 0 will set the corresponding bit. For each zero,
// the corresponding bit will be unaffected. T
func writeInterruptEnableHandler(via *Via65C22S) {
	mustSet := (via.dataBus.Read() & 0x80) > 0
	value := via.dataBus.Read() & 0x7F

	if mustSet {
		via.registers.interruptEnable |= value
	} else {
		via.registers.interruptEnable &= ^value
	}
}

/************************************************************************************
* Temporary
*************************************************************************************/

func dummyHandler(via *Via65C22S) {
}
