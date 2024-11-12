package acia

import "log"

func (acia *Acia65C51N) writeBytes() {
	for {
		if !acia.txRegisterEmpty {
			_, err := acia.port.Write([]byte{acia.txRegister})

			if err != nil {
				panic(err)
			}

			acia.txRegisterEmpty = true
		}
	}
}

func (acia *Acia65C51N) readBytes() {
	buff := make([]byte, 1)

	for {
		_, err := acia.port.Read(buff)
		if err != nil {
			log.Fatal(err)
			break
		}

		if acia.rxRegisterEmpty {
			acia.rxRegister = uint8(buff[0])
			acia.rxRegisterEmpty = false
		} else {
			acia.statusRegister |= (StatusOverrun | StatusIRQ)
		}
	}
}
