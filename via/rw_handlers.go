package via

/************************************************************************************
* Input / Output Register B
*************************************************************************************/

func inputOutputRegisterBReadHandler(via *Via65C22S) {
	// Get the data direction register
	outputPins := via.registers.dataDirectionRegisterB
	inputPins := ^outputPins

	// MPU reads output register bit in ORB. Pin level has no effect.
	value := via.registers.outputRegisterB & outputPins

	if !via.latchesB.isLatchingEnabled() {
		// MPU reads input level on PB pin.
		value |= via.peripheralPortB.getConnector().Read() & inputPins
	} else {
		// MPU reads IRB bit
		value |= via.registers.inputRegisterB & inputPins
	}

	via.peripheralPortB.clearControlLinesInterruptFlagOnRW()

	via.dataBus.Write(value)
}

func inputOutputRegisterBWriteHandler(via *Via65C22S) {
	mode := via.latchesB.getOutputMode()

	if mode == pcrCB2OutputModeHandshake || mode == pcrCB2OutputModePulse {
		via.latchesB.initHandshake()
	}

	via.peripheralPortB.clearControlLinesInterruptFlagOnRW()

	// MPU writes to ORB
	via.registers.outputRegisterB = via.dataBus.Read()
}

/************************************************************************************
* Input / Output Register A
*************************************************************************************/

func inputOutputRegisterAReadHandler(via *Via65C22S) {
	var value uint8

	if !via.latchesA.isLatchingEnabled() {
		value = via.peripheralPortA.connector.Read()
	} else {
		value = via.registers.inputRegisterA
	}

	mode := via.latchesA.getOutputMode()

	if mode == pcrCA2OutputModeHandshake || mode == pcrCA2OutputModePulse {
		via.latchesA.initHandshake()
	}

	via.peripheralPortA.clearControlLinesInterruptFlagOnRW()

	via.dataBus.Write(value)
}

func inputOutputRegisterAWriteHandler(via *Via65C22S) {
	mode := via.latchesA.getOutputMode()

	if mode == pcrCA2OutputModeHandshake || mode == pcrCA2OutputModePulse {
		via.latchesA.initHandshake()
	}

	via.peripheralPortA.clearControlLinesInterruptFlagOnRW()

	// MPU writes to ORA
	via.registers.outputRegisterA = via.dataBus.Read()
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
	via.registers.highLatches1 = via.dataBus.Read()
	// Write into high order counter (first clear MSB and then assign)
	via.registers.counter1 = (via.registers.counter1 & 0x00FF) | high

	// Transfer low order latch to low order counter
	via.registers.counter1 = (via.registers.counter1 & 0xFF00) | uint16(via.registers.lowLatches1)

	// Enable the counter
	via.timer1.timerEnabled = true
	via.timer1.outputStatusWhenEnabled = false

	// Clears interrupt flags
	via.registers.interrupts.clearInterruptFlagBit(irqT1)
}

func writeT1HighOrderLatch(via *Via65C22S) {
	// Write into high order latch
	via.registers.highLatches1 = via.dataBus.Read()

	// Clear interrupt flags
	via.registers.interrupts.clearInterruptFlagBit(irqT1)
}

// Reads the LSB from the counter
func readT1LowOrderCounter(via *Via65C22S) {
	via.dataBus.Write(uint8(via.registers.counter1))

	// Clear interrupt flags
	via.registers.interrupts.clearInterruptFlagBit(irqT1)
}

// Reads the MSB from the counter
func readT1HighOrderCounter(via *Via65C22S) {
	// Makes 0 the LSB and moves the MSB to the lower byte
	value := (via.registers.counter1 & 0xFF00) >> 8
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
	via.registers.highLatches2 = via.dataBus.Read()
	// Write into high order counter (first clear MSB and then assign)
	via.registers.counter2 = (via.registers.counter2 & 0x00FF) | high

	// Transfer low order latch to low order counter
	via.registers.counter2 = (via.registers.counter2 & 0xFF00) | uint16(via.registers.lowLatches2)

	// Enable the counter
	via.timer2.timerEnabled = true
	via.timer2.outputStatusWhenEnabled = false

	// Clears interrupt flags
	via.registers.interrupts.clearInterruptFlagBit(irqT2)
}

// Reads the LSB from the counter
func readT2LowOrderCounter(via *Via65C22S) {
	via.dataBus.Write(uint8(via.registers.counter2))

	// Clears interrupt flags
	via.registers.interrupts.clearInterruptFlagBit(irqT2)
}

// Reads the MSB from the counter
func readT2HighOrderCounter(via *Via65C22S) {
	// Makes 0 the LSB and moves the MSB to the lower byte
	value := (via.registers.counter2 & 0xFF00) >> 8
	// Writes the value on the bus
	via.dataBus.Write(uint8(value))
}

/************************************************************************************
* Shift register handling
*************************************************************************************/

// Reads the LSB from the counter
func readShiftRegister(via *Via65C22S) {
	via.dataBus.Write(via.registers.shiftRegister)
	via.shifter.initCounter()
	via.shifter.shifterEnabled = true
}

// Reads the MSB from the counter
func writeShiftRegister(via *Via65C22S) {
	via.registers.shiftRegister = via.dataBus.Read()
	via.shifter.initCounter()
	via.shifter.shifterEnabled = true
}

/************************************************************************************
* Temporary
*************************************************************************************/

func dummyHandler(via *Via65C22S) {
}
