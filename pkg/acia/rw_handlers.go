package acia

const (
	softResetStatusRegValue  uint8 = 0xFB
	softResetCommandRegValue uint8 = 0xE0
)

func writeTransmitData(acia *Acia65C51N) {
	acia.txMutex.Lock()
	defer acia.txMutex.Unlock()

	acia.txRegister = acia.dataBus.Read()
	acia.txRegisterEmpty = false
}

func programmedReset(acia *Acia65C51N) {
	acia.statusRegister &= softResetStatusRegValue
	acia.commandRegister &= softResetCommandRegValue
}

func writeCommand(acia *Acia65C51N) {
	acia.commandRegister = acia.dataBus.Read()

	if acia.port != nil {
		acia.setModemLines()
	}
}

func writeControl(acia *Acia65C51N) {
	acia.controlRegister = acia.dataBus.Read()

	if acia.port != nil {
		mode := acia.getMode()
		err := acia.port.SetMode(mode)
		if err != nil {
			panic(err)
		}
	}
}

func readReceiverData(acia *Acia65C51N) {
	acia.rxMutex.Lock()
	defer acia.rxMutex.Unlock()

	acia.dataBus.Write(acia.rxRegister)
	acia.rxRegisterEmpty = true
	acia.statusRegister &= ^(statusRDRF | statusParityError | statusFramingError | statusOverrun)
}

func readStatus(acia *Acia65C51N) {
	acia.dataBus.Write(acia.statusRegister)
	acia.statusRegister &= ^statusIRQ
}

func readCommand(acia *Acia65C51N) {
	acia.dataBus.Write(acia.commandRegister)
}

func readControl(acia *Acia65C51N) {
	acia.dataBus.Write(acia.controlRegister)
}
