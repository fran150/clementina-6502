package tests

import (
	"testing"
	"time"

	"github.com/fran150/clementina6502/pkg/common"
	"github.com/fran150/clementina6502/pkg/computers/beneater"
)

func BenchmarkComputer(b *testing.B) {
	computer, _ := beneater.NewBenEaterComputer("/dev/ttys004", false)
	if err := computer.LoadRom("../assets/computer/beneater/eater.bin"); err != nil {
		b.Fatal(err)
	}

	context := common.NewStepContext()

	var start = time.Now()

	for i := 0; i < 100_000_000; i++ {
		context.NextCycle()
		computer.Tick(&context)
	}

	context.Stop = true
	computer.Close()

	// Measure the elapsed time
	elapsed := time.Since(start)

	showExecutionSpeed(&context, elapsed)
}
