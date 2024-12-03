package acia

var counter int

func (acia *Acia65C51N) writeBytes() {
	for acia.running {
		if acia.port != nil && !acia.txRegisterEmpty {
			if acia.isCTSEnabled() {
				acia.txRegisterEmpty = true

				_, err := acia.port.Write([]byte{acia.txRegister})
				if err != nil {
					panic(err)
				}
			}
		}
	}

	acia.wg.Done()
}

func (acia *Acia65C51N) readBytes() {
	buff := make([]byte, 1)

	for acia.running {
		if acia.port != nil {
			_, err := acia.port.Read(buff)
			if err != nil {
				panic(err)
			}

			acia.rxMutex.Lock()

			if !acia.rxRegisterEmpty {
				acia.statusRegister |= statusOverrun
			}

			acia.rxRegisterEmpty = false
			acia.rxRegister = uint8(buff[0])

			acia.rxMutex.Unlock()

			if acia.isReceiverEchoModeEnabled() {
				acia.txMutex.Lock()

				acia.txRegister = acia.rxRegister
				acia.txRegisterEmpty = false

				acia.txMutex.Unlock()
			}
		}
	}

	acia.wg.Done()
}
