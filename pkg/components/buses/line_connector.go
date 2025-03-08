package buses

// This allows the chip to connect to a line, it basically represents individual pins
// in a chip
type LineConnector interface {
	// Connect associates this connector with the specified line
	// The line parameter is the bus line to connect to
	Connect(line Line)

	// Returns true if the connector is enabled. Depending on the chip some
	// connectors might be enabled when the line is high or low.
	// Pins that are enabled when line is low are usually marked with a bar on top of their name
	// or a B at then end of the name. For example CS1B is Chip Select 1 (enabled on low)
	Enabled() bool

	// SetEnable enables or disables the pin by setting the line to low or high depending on the type of pin
	// The value parameter determines whether to enable (true) or disable (false) the pin
	SetEnable(value bool)

	// Gets a reference to the line connected to this or nil if no line is connected
	GetLine() Line
}
