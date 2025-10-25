package controllers

import "github.com/fran150/clementina-6502/pkg/core"

// speedController manages emulation speed with non-linear scaling.
type speedController struct {
	targetSpeedMhz float64
	// Cache the reciprocal for performance (1/speed = nanoseconds per cycle)
	cachedNanosPerCycle float64
}

// newSpeedController creates a new DefaultSpeedController with the specified initial speed.
// This is an internal constructor function.
//
// Parameters:
//   - initialSpeedMhz: The initial target speed in MHz
//
// Returns:
//   - A pointer to the initialized DefaultSpeedController
func newSpeedController(initialSpeedMhz float64) *speedController {
	sc := &speedController{
		targetSpeedMhz: initialSpeedMhz,
	}
	sc.updateCache()
	return sc
}

// NewSpeedController creates a new speed controller with the specified initial speed.
//
// Parameters:
//   - initialSpeedMhz: The initial target speed in MHz
//
// Returns:
//   - A pointer to the initialized SpeedController
func NewSpeedController(initialSpeedMhz float64) core.SpeedController {
	return newSpeedController(initialSpeedMhz)
}

// SpeedUp increases the emulation speed of the computer.
// Uses progressive scaling: 0.1 for speeds ≥1, 0.01 for 0.1-0.99, 0.001 for 0.01-0.099, etc.
func (s *speedController) SpeedUp() {
	increment := s.getSpeedIncrement()
	s.targetSpeedMhz += increment
	s.updateCache()
}

// SpeedDown decreases the emulation speed of the computer.
// Uses progressive scaling: 0.1 for speeds >1, 0.01 for 0.1-1.0, 0.001 for 0.01-0.1, etc.
func (s *speedController) SpeedDown() {
	increment := s.getSpeedIncrement()
	s.targetSpeedMhz -= increment
	s.updateCache()
}

// GetTargetSpeed returns the current target speed in MHz.
func (s *speedController) GetTargetSpeed() float64 {
	return s.targetSpeedMhz
}

// SetTargetSpeed sets the target speed in MHz.
// The speed must be greater than 0, otherwise the request is ignored.
func (s *speedController) SetTargetSpeed(speedMhz float64) {
	if speedMhz > 0 {
		s.targetSpeedMhz = speedMhz
		s.updateCache()
	}
}

// updateCache updates the cached nanoseconds per cycle calculation
func (s *speedController) updateCache() {
	if s.targetSpeedMhz > 0 {
		// Convert MHz to nanoseconds per cycle: (1 second / 1 microsecond) / speedMhz
		s.cachedNanosPerCycle = 1000.0 / s.targetSpeedMhz // nanoseconds per cycle
	}
}

// GetNanosPerCycle returns the cached nanoseconds per cycle for performance.
func (s *speedController) GetNanosPerCycle() float64 {
	return s.cachedNanosPerCycle
}

// getSpeedIncrement calculates the appropriate increment/decrement based on current speed.
// Returns 0.1 for speeds ≥1, 0.01 for 0.1-0.99, 0.001 for 0.01-0.099, etc.
func (s *speedController) getSpeedIncrement() float64 {
	if s.targetSpeedMhz >= 1.0 {
		return 0.1
	}

	// For speeds < 1.0, find the appropriate decimal place
	// Start with 0.01 for the 0.1-0.99 range, then scale down
	increment := 0.01
	rangeThreshold := 0.1

	for s.targetSpeedMhz < rangeThreshold {
		increment /= 10
		rangeThreshold /= 10
	}

	return increment
}
