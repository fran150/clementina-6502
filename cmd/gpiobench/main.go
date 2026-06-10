//go:build (linux && arm) || (linux && arm64)

// gpiobench measures the per-call latency of the go-gpiocdev (chardev/ioctl)
// operations that the pico_mia bridge uses on every emulated clock cycle.
//
// Run it on the Raspberry Pi 5 WITH THE EMULATOR STOPPED:
//
//	go run ./cmd/gpiobench -chip gpiochip0 -n 100000
//
// It reports ns/op for:
//   - Line.Value()       single-line read   (used for phi2 / reset / irq)
//   - Line.SetValue()    single-line write  (used for WE / CS / resetRequest)
//   - Lines.Values()     batched read       (used for the data bus read-back)
//   - Lines.SetValues()  batched write      (used for the address / data bus)
//   - Reconfigure()      direction flip     (used when the data bus changes dir)
//
// The point is to confirm whether per-call cost is ~1 us (syscall-bound) or
// ~1 ms (PCIe/RP1-bound) so we know how much a memory-mapped GPIO rewrite buys.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/warthog618/go-gpiocdev"
)

// Pins taken from pkg/common/gpio_controller.go so the bench is representative.
var (
	dataLines    = []int{2, 3, 4, 17, 27, 22, 10, 9}
	phi2Pin      = 18 // input we read
	writeEnaPin  = 16 // output we write
	addressLines = []int{11, 5, 6, 13, 19}
)

func main() {
	chip := flag.String("chip", "gpiochip0", "GPIO chip name (Pi 5 RP1 is often gpiochip0 or gpiochip4)")
	n := flag.Int("n", 100000, "iterations per benchmark")
	flag.Parse()

	c, err := gpiocdev.NewChip(*chip, gpiocdev.WithConsumer("gpiobench"))
	if err != nil {
		fail("open chip %q: %v", *chip, err)
	}
	defer c.Close()

	fmt.Printf("chip=%s iterations=%d\n\n", *chip, *n)

	benchSingleRead(c, *n)
	benchSingleWrite(c, *n)
	benchBatchRead(c, *n)
	benchBatchWrite(c, *n)
	benchReconfigure(c, *n)
}

// benchSingleRead times Line.Value() — the call the poll loop spins on for phi2.
func benchSingleRead(c *gpiocdev.Chip, n int) {
	l, err := c.RequestLine(phi2Pin, gpiocdev.AsInput)
	if err != nil {
		fail("request phi2 read line: %v", err)
	}
	defer l.Close()

	start := time.Now()
	for i := 0; i < n; i++ {
		if _, err := l.Value(); err != nil {
			fail("Value(): %v", err)
		}
	}
	report("Line.Value()      (single read )", start, n)
}

// benchSingleWrite times Line.SetValue() — used for WE / CS / resetRequest.
func benchSingleWrite(c *gpiocdev.Chip, n int) {
	l, err := c.RequestLine(writeEnaPin, gpiocdev.AsOutput(0))
	if err != nil {
		fail("request WE write line: %v", err)
	}
	defer l.Close()

	start := time.Now()
	for i := 0; i < n; i++ {
		if err := l.SetValue(i & 1); err != nil {
			fail("SetValue(): %v", err)
		}
	}
	report("Line.SetValue()   (single write)", start, n)
}

// benchBatchRead times Lines.Values() — the batched data-bus read-back.
func benchBatchRead(c *gpiocdev.Chip, n int) {
	l, err := c.RequestLines(dataLines, gpiocdev.AsInput)
	if err != nil {
		fail("request data bus read: %v", err)
	}
	defer l.Close()

	buf := make([]int, len(dataLines))
	start := time.Now()
	for i := 0; i < n; i++ {
		if err := l.Values(buf); err != nil {
			fail("Values(): %v", err)
		}
	}
	report("Lines.Values()    (8-bit read )", start, n)
}

// benchBatchWrite times Lines.SetValues() — the batched address/data-bus write.
func benchBatchWrite(c *gpiocdev.Chip, n int) {
	l, err := c.RequestLines(addressLines, gpiocdev.AsOutput(0))
	if err != nil {
		fail("request address bus write: %v", err)
	}
	defer l.Close()

	buf := make([]int, len(addressLines))
	start := time.Now()
	for i := 0; i < n; i++ {
		buf[0] = i & 1
		if err := l.SetValues(buf); err != nil {
			fail("SetValues(): %v", err)
		}
	}
	report("Lines.SetValues() (5-bit write)", start, n)
}

// benchReconfigure times Reconfigure() flipping a bus between input and output,
// which prepareDataBus() does whenever the data-bus direction changes.
func benchReconfigure(c *gpiocdev.Chip, n int) {
	l, err := c.RequestLines(dataLines, gpiocdev.AsInput)
	if err != nil {
		fail("request data bus for reconfigure: %v", err)
	}
	defer l.Close()

	start := time.Now()
	for i := 0; i < n; i++ {
		var err error
		if i&1 == 0 {
			err = l.Reconfigure(gpiocdev.AsOutput(0))
		} else {
			err = l.Reconfigure(gpiocdev.AsInput)
		}
		if err != nil {
			fail("Reconfigure(): %v", err)
		}
	}
	report("Lines.Reconfigure (dir flip  )", start, n)
}

// report prints elapsed time and ns/op for a benchmark.
func report(name string, start time.Time, n int) {
	elapsed := time.Since(start)
	nsPerOp := float64(elapsed.Nanoseconds()) / float64(n)
	fmt.Printf("%-34s %10.0f ns/op   (%v total)\n", name, nsPerOp, elapsed)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "gpiobench: "+format+"\n", args...)
	os.Exit(1)
}
