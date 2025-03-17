package acia

// writeBytes is a goroutine that handles the transmission of bytes through the ACIA.
// It continuously monitors the transmit register and sends data through the serial port
// when the following conditions are met:
// - A port is configured
// - The transmit register is not empty
// - CTS (Clear To Send) is enabled (if CTS control is being used)
// The goroutine runs until acia.running is set to false.
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

// readBytes is a goroutine that handles the reception of bytes through the ACIA.
// It continuously reads from the serial port and processes incoming data as follows:
// - Reads one byte at a time from the configured port
// - If the receive register is not empty when new data arrives, sets the overrun flag
// - Stores the received byte in the receive register
// - If echo mode is enabled, copies the received byte to the transmit register
// The goroutine runs until acia.running is set to false.
//
// The function uses mutexes to ensure thread-safe access to shared registers:
// - rxMutex for protecting the receive register and status
// - txMutex for protecting the transmit register when in echo mode
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
