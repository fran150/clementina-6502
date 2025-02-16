package tests

import (
	"testing"
	"time"
)

func BenchmarkTimeNow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		time.Now()
	}
}

func BenchmarkTimeAdd(b *testing.B) {
	t := time.Now()

	for i := 0; i < b.N; i++ {
		duration := time.Since(t)
		t = t.Add(duration)
	}
}

func BenchmarkTimeInt64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		now()
	}
}

func BenchmarkTimeSince(b *testing.B) {
	for i := 0; i < b.N; i++ {
		time.Since(initTime)
	}
}

var initTime = time.Now()

func now() int64 {
	return int64(time.Since(initTime))
}
