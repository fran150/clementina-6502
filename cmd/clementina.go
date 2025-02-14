package main

import (
	"fmt"
	"time"

	"github.com/fran150/clementina6502/pkg/computers"
	"github.com/fran150/clementina6502/pkg/computers/beneater"
)

func main() {
	computer := beneater.CreateBenEaterComputer("/dev/ttys004")
	computer.Load("./assets/computer/beneater/eater.bin")

	executor := computers.CreateExecutor(computer, &computers.ExecutorConfig{
		TargetSpeedMhz: 80.0,
		DisplayFps:     15,
	})

	t := time.Now()

	context := executor.Run()

	elapsed := time.Since(t)
	total := (float64(context.Cycle) / elapsed.Seconds()) / 1_000_000

	fmt.Printf("Executed %v cycles in %v seconds\n", context.Cycle, elapsed)
	fmt.Printf("Computer ran at %v mhz\n", total)

	computer.Close()
}
