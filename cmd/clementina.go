package main

import (
	"fmt"
	"time"

	"github.com/fran150/clementina6502/pkg/computers/beneater"
	"github.com/fran150/clementina6502/pkg/terminal"
)

func main() {
	computer := beneater.NewBenEaterComputer("/dev/ttys004")
	defer computer.Close()

	computer.LoadRom("./assets/computer/beneater/eater.bin")

	app := terminal.NewApplication(computer)

	t := time.Now()

	context := app.Run()

	elapsed := time.Since(t)
	total := (float64(context.Cycle) / elapsed.Seconds()) / 1_000_000

	fmt.Printf("Executed %v cycles in %v seconds\n", context.Cycle, elapsed)
	fmt.Printf("Computer ran at %v mhz\n", total)
}
