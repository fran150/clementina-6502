package common

import "time"

type StepContext struct {
	Cycle uint64
	T     time.Time
}
