package buses

// ConnectorEnabledHigh provides a connection point from a chip to a line.
// This can be used in chip emulations as the interface between the chip and lines that connect to other chips.
// For example a line can connect from the CPU R/W to the RAM R/W line. The CPU can set the value of the line controlling if
// it will read or write the value in memory.
// This connector should be used on chip pins that are enabled when the line are high.
type ConnectorEnabledHigh struct {
	line Line
}

// NewConnectorEnabledHigh creates and returns a connector that is enabled when the line is high.
// This type of connector is typically used for pins that are active-high.
func NewConnectorEnabledHigh() *ConnectorEnabledHigh {
	return &ConnectorEnabledHigh{}
}

// Connects to the specified line
func (cn *ConnectorEnabledHigh) Connect(line Line) {
	cn.line = line
}

// Returns if the pin is enabled. In this connector type this happens when the line is high.
// If the pin is not connected it always returns false.
func (cn *ConnectorEnabledHigh) Enabled() bool {
	if cn.line != nil {
		return cn.line.Status()
	}

	return false
}

// Changes the value of the line connected to the pin (if any)
func (cn *ConnectorEnabledHigh) SetEnable(value bool) {
	if cn.line != nil {
		cn.line.Set(value)
	}
}

// Gets the reference to connected line. Returns nil if connector is not wired to any line
func (cn *ConnectorEnabledHigh) GetLine() Line {
	return cn.line
}
