package buses

// Line represents a single electrical connection or trace that allows interconnection
// between multiple pins or connectors of different chips. It can be in a high (true)
// or low (false) state.
type Line interface {
	// Returns the of the line, high (true) or low (false)
	Status() bool
	// Sets the status of the line high (true) or low (false)
	Set(value bool)
	// Toggles the status of the line
	Toggle()
}
