package main

import (
	"fmt"
	"time"

	"github.com/fran150/clementina6502/pkg/computers/beneater"
)

func main() {
	computer := beneater.CreateBenEaterComputer("/dev/ttys004")
	defer computer.Close()

	computer.Load("./assets/computer/beneater/eater.bin")

	t := time.Now()

	context := computer.Run()

	elapsed := time.Since(t)
	total := (float64(context.Cycle) / elapsed.Seconds()) / 1_000_000

	fmt.Printf("Executed %v cycles in %v seconds\n", context.Cycle, elapsed)
	fmt.Printf("Computer ran at %v mhz\n", total)
}
