package controllers

// DefaultSpeedController manages emulation speed with non-linear scaling.
type DefaultSpeedController struct {
	targetSpeedMhz float64
	// Cache the reciprocal for performance (1/speed = nanoseconds per cycle)
	cachedNanosPerCycle float64
	speedChanged        bool
}

// NewSpeedController creates a new speed controller with the specified initial speed.
//
// Parameters:
//   - initialSpeedMhz: The initial target speed in MHz
//
// Returns:
//   - A pointer to the initialized DefaultSpeedController
func NewSpeedController(initialSpeedMhz float64) *DefaultSpeedController {
	sc := &DefaultSpeedController{
		targetSpeedMhz: initialSpeedMhz,
		speedChanged:   true,
	}
	sc.updateCache()
	return sc
}

// SpeedUp increases the emulation speed of the computer.
// It uses a non-linear scale for speeds below 0.5 MHz and a linear scale above.
func (s *DefaultSpeedController) SpeedUp() {
	if s.targetSpeedMhz < 0.5 {
		// Non-linear increase below 0.5 MHz
		// Increase by 20% of current speed
		increase := s.targetSpeedMhz * 0.2
		if increase < 0.000001 {
			// Ensure minimum increase to avoid tiny increments
			increase = 0.000001
		}
		s.targetSpeedMhz += increase
	} else {
		// Linear increase above 0.5 MHz
		s.targetSpeedMhz += 0.1
	}
	s.speedChanged = true
	s.updateCache()
}

// SpeedDown decreases the emulation speed of the computer.
// It uses a linear scale for speeds above 0.5 MHz and a non-linear scale below,
// ensuring the speed never goes below a minimum threshold.
func (s *DefaultSpeedController) SpeedDown() {
	if s.targetSpeedMhz > 0.5 {
		// Linear reduction above 0.5 MHz
		s.targetSpeedMhz -= 0.1
	} else {
		// Non-linear reduction below 0.5 MHz to avoid reaching 0
		// This will reduce by a fraction of the current speed
		reduction := s.targetSpeedMhz * 0.2
		if reduction < 0.000001 {
			// Ensure minimum reduction to avoid tiny decrements
			reduction = 0.000001
		}
		s.targetSpeedMhz -= reduction
	}
	s.speedChanged = true
	s.updateCache()
}

// GetTargetSpeed returns the current target speed in MHz.
func (s *DefaultSpeedController) GetTargetSpeed() float64 {
	return s.targetSpeedMhz
}

// SetTargetSpeed sets the target speed in MHz.
func (s *DefaultSpeedController) SetTargetSpeed(speedMhz float64) {
	if speedMhz > 0 {
		s.targetSpeedMhz = speedMhz
		s.speedChanged = true
		s.updateCache()
	}
}

// GetTargetSpeedPtr returns a pointer to the current target speed in MHz.
// This is useful for UI components that need to monitor speed changes.
func (s *DefaultSpeedController) GetTargetSpeedPtr() *float64 {
	return &s.targetSpeedMhz
}

// updateCache updates the cached nanoseconds per cycle calculation
func (s *DefaultSpeedController) updateCache() {
	if s.targetSpeedMhz > 0 {
		s.cachedNanosPerCycle = 1000.0 / s.targetSpeedMhz // nanoseconds per cycle
	}
}

// GetNanosPerCycle returns the cached nanoseconds per cycle for performance
func (s *DefaultSpeedController) GetNanosPerCycle() float64 {
	if s.speedChanged {
		s.updateCache()
		s.speedChanged = false
	}
	return s.cachedNanosPerCycle
}
