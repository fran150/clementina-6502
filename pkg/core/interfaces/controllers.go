package interfaces

// SpeedController defines the interface for managing emulation speed.
type SpeedController interface {
	// SpeedUp increases the emulation speed.
	SpeedUp()

	// SpeedDown decreases the emulation speed.
	SpeedDown()

	// GetTargetSpeed returns the current target speed in MHz.
	GetTargetSpeed() float64

	// SetTargetSpeed sets the target speed in MHz.
	SetTargetSpeed(speedMhz float64)

	// GetNanosPerCycle returns the nanoseconds per cycle for the current speed.
	GetNanosPerCycle() float64
}
