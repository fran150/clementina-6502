package tests

import (
	"testing"
	"time"

	"github.com/fran150/clementina6502/pkg/components/common"
	"github.com/fran150/clementina6502/pkg/computers/beneater"
)

func BenchmarkComputer(b *testing.B) {

	computer := beneater.CreateBenEaterComputer("/dev/ttys004")
	computer.Load("../assets/computer/beneater/eater.bin")

	context := common.CreateStepContext()

	var start = time.Now()

	for i := 0; i < b.N; i++ {
		context.NextCycle()
		computer.Step(&context)
	}

	context.Stop = true
	computer.Close()

	// Measure the elapsed time
	elapsed := time.Since(start)

	showExecutionSpeed(&context, elapsed)
}
