package acia

import "fmt"

func (acia *Acia65C51N) writeBytes() {
	for !acia.stop {
		if !acia.txRegisterEmpty {
			acia.txRegisterEmpty = true

			_, err := acia.port.Write([]byte{acia.txRegister})

			if err != nil {
				panic(err)
			}
		}
	}
}

func (acia *Acia65C51N) readBytes() {
	buff := make([]byte, 1)

	for !acia.stop {
		_, err := acia.port.Read(buff)
		if err != nil {
			panic(err)
		}

		acia.mu.Lock()
		if !acia.rxRegisterEmpty {
			fmt.Printf("\tChip Buffer Overrun: %v\n", string(buff[0]))
			acia.statusRegister |= (StatusOverrun | StatusIRQ)
		}
		acia.rxRegisterEmpty = false
		fmt.Printf("\tWritten Into Chip Buffer: %v\n", string(buff[0]))
		acia.rxRegister = uint8(buff[0])
		acia.mu.Unlock()
	}
}
