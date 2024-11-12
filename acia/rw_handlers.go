package acia

func writeTransmitData(acia *Acia65C51N) {
	acia.txRegister = acia.dataBus.Read()
	acia.txRegisterEmpty = false
}

func programmedReset(acia *Acia65C51N) {
}

func writeCommand(acia *Acia65C51N) {
	acia.commandRegister = acia.dataBus.Read()

	if acia.commandRegister&0x01 == 0x01 {
		acia.port.SetDTR(true)
	} else {
		acia.port.SetDTR(false)
	}

	if acia.commandRegister&0x0C == 0x00 {
		acia.port.SetRTS(false)
	} else {
		acia.port.SetRTS(true)
	}
}

func writeControl(acia *Acia65C51N) {
	acia.controlRegister = acia.dataBus.Read()

	mode := acia.getMode()
	err := acia.port.SetMode(mode)

	if err != nil {
		panic(err)
	}
}

func readReceiverData(acia *Acia65C51N) {
	acia.dataBus.Write(acia.rxRegister)
	acia.rxRegisterEmpty = true
	acia.rxRegister = 0x00

	acia.statusRegister &= ^(StatusRDRF | StatusParityError | StatusFramingError | StatusOverrun)
}

func readStatus(acia *Acia65C51N) {
	acia.dataBus.Write(acia.statusRegister)
	acia.statusRegister &= ^StatusIRQ
}

func readCommand(acia *Acia65C51N) {
	acia.dataBus.Write(acia.commandRegister)
}

func readControl(acia *Acia65C51N) {
	acia.dataBus.Write(acia.controlRegister)
}
