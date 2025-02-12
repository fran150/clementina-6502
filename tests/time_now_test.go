package tests

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkTimeNow(b *testing.B) {
	t := time.Now()

	for i := 0; i < b.N; i++ {
		t = time.Now()
	}

	fmt.Println(t)
}

func BenchmarkTimeAdd(b *testing.B) {
	t := time.Now()

	for i := 0; i < b.N; i++ {
		duration := time.Since(t)
		t = t.Add(duration)
	}

	fmt.Println(t)
}

func BenchmarkTimeInt64(b *testing.B) {
	var t int64

	for i := 0; i < b.N; i++ {
		t = now()
	}

	fmt.Println(t)
}

func BenchmarkTimeSince(b *testing.B) {
	var t time.Duration

	for i := 0; i < b.N; i++ {
		t = time.Since(initTime)
	}

	fmt.Println(t.Microseconds())
}

var initTime = time.Now()

func now() int64 {
	return int64(time.Since(initTime))
}
