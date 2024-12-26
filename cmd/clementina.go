package main

import (
	"github.com/fran150/clementina6502/pkg/computers/beneater"
)

func main() {
	computer := beneater.CreateBenEaterComputer("/dev/ttys004")

	computer.Load("./assets/computer/beneater/wozmon.bin")
	computer.Run()
}
