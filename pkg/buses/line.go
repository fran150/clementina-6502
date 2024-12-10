package buses

// A line or trace allows to interconect multiple pins or connectors of different chips.
type Line interface {
	// Returns the of the line, high (true) or low (false)
	Status() bool
	// Sets the status of the line high (true) or low (false)
	Set(value bool)
	// Toggles the status of the line
	Toggle()
}
