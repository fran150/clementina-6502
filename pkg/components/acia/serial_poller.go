package acia

func (acia *Acia65C51N) writeBytes() {
	defer acia.wg.Done()

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
}

func (acia *Acia65C51N) readBytes() {
	defer acia.wg.Done()

	buff := make([]byte, 1)

	for acia.running {
		if acia.port != nil {
			n, err := acia.port.Read(buff)
			if err != nil {
				panic(err)
			}

			if n > 0 {
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
	}
}
