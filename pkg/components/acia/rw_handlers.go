package acia

// Sets the value in the data bus into the tx register. This will be picked
// up by the pollers and sent through the serial port.
func writeTransmitData(acia *Acia65C51N) {
	acia.txMutex.Lock()
	defer acia.txMutex.Unlock()

	acia.txRegister = acia.dataBus.Read()
	acia.txRegisterEmpty = false
}

// Writing any value to the status register causes a soft reset
func programmedReset(acia *Acia65C51N) {
	acia.statusRegister &= softResetStatusRegMask
	acia.commandRegister &= softResetCommandRegMask
}

// Sets the value in the data bus into the command register
func writeCommand(acia *Acia65C51N) {
	acia.commandRegister = acia.dataBus.Read()

	if !isBitSet(acia.commandRegister, commandTICRTSBit) && isBitSet(acia.commandRegister, commandTICTXBit) {
		panic("ACIA: Command TIC bits should never have RTS disabled and TX IRQ enabled (0x04). See page 10 in datasheet.")
	}

	if isBitSet(acia.commandRegister, commandPMEBit) {
		panic("ACIA: Command register must not enable parity. See page 13 in the datasheet")
	}
}

// Sets the value in the data bus into the control register
func writeControl(acia *Acia65C51N) {
	acia.controlRegister = acia.dataBus.Read()

	// If the chip is connected to serial port, updates the
	// mode based on the new values in the control register
	if acia.port != nil {
		mode := acia.getMode()
		err := acia.port.SetMode(mode)
		if err != nil {
			panic(err)
		}
	}
}

// Sets the value of the RX register in the data bus.
// Reading RX value also clears error flags in the status register
func readReceiverData(acia *Acia65C51N) {
	acia.rxMutex.Lock()
	defer acia.rxMutex.Unlock()

	acia.dataBus.Write(acia.rxRegister)
	acia.rxRegisterEmpty = true
	acia.statusRegister &= ^(statusRDRF | statusParityError | statusFramingError | statusOverrun)
}

// Sets the value of the status register in the data bus
// Reading the status register resets the IRQ flag
func readStatus(acia *Acia65C51N) {
	acia.dataBus.Write(acia.statusRegister)
	acia.statusRegister &= ^statusIRQ
}

// Sets the value of the command register in the data bus
func readCommand(acia *Acia65C51N) {
	acia.dataBus.Write(acia.commandRegister)
}

// Sets the value of the control register in the data bus
func readControl(acia *Acia65C51N) {
	acia.dataBus.Write(acia.controlRegister)
}
