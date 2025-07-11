package lcd

import "time"

type lcdTimingConfig struct {
	clearDisplayMicro time.Duration
	returnHomeMicro   time.Duration
	instructionMicro  time.Duration
	blinkingMicro     time.Duration
}
