package lcd

type lcdTimingConfig struct {
	clearDisplayMicro int64
	returnHomeMicro   int64
	instructionMicro  int64
	blinkingMicro     int64
}
