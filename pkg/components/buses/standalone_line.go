package buses

// StandaloneLine represents an electrical line or trace in a circuit that can be used to connect 
// chip pins or connectors. For example, a line can be created and then wired
// to the R/W line of the CPU, RAM and VIA chips. Then whoever drives the line,
// typically the CPU, will use to signal a read or write operation to the selected chip.
type StandaloneLine struct {
	// Status indicates whether the line is high (true) or low (false)
	status bool
}

// NewStandaloneLine creates an electrical line or trace in a circuit that can be used to connect
// multiple chips. The status parameter sets the initial state of the line (high=true, low=false).
func NewStandaloneLine(status bool) *StandaloneLine {
	return &StandaloneLine{
		status: status,
	}
}

// Returns if the line is high (true) or low (false)
func (line *StandaloneLine) Status() bool {
	return line.status
}

// Sets if the line is high (true) or low (false)
func (line *StandaloneLine) Set(value bool) {
	line.status = value
}

// Toggles the status of the line
func (line *StandaloneLine) Toggle() {
	line.Set(!line.Status())
}
