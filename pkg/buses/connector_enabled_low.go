package buses

// Provides a connection point from a chip to a line.
// This can be used in chip emulations as the interface between the chip and lines that connect to other chips.
// For example a line can connect from the CPU R/W to the RAM R/W line. The CPU can set the value of the line controlling if
// it will read or write the value in memory.
// This connector should be used on chip pins that are enabled when the line are low. Typically in datasheets these pins are represented
// with a line above the pin name or ends with a B (for BAR). For example CS1B is "chip select 1" and is enabled when the line is low
type ConnectorEnabledLow struct {
	line Line
}

// Creates and returns a connector that is enabled when the line is low
func CreateConnectorEnabledLow() *ConnectorEnabledLow {
	return &ConnectorEnabledLow{}
}

// Connects to the specified line
func (cn *ConnectorEnabledLow) Connect(line Line) {
	cn.line = line
}

// Returns if the pin is enabled. In this connector type this happens when the line is low.
// If the pin is not connected it always returns false.
func (cn *ConnectorEnabledLow) Enabled() bool {
	if cn.line != nil {
		return !cn.line.Status()
	}

	return false
}

// Changes the value of the line connected to the pin (if any)
func (cn *ConnectorEnabledLow) SetEnable(value bool) {
	if cn.line != nil {
		cn.line.Set(!value)
	}
}

// Gets the reference to connected line. Returns nil if connector is not wired to any line
func (cn *ConnectorEnabledLow) GetLine() Line {
	return cn.line
}
