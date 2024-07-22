package buses

type ConnectorEnabledLow struct {
	line Line
}

func CreateConnectorEnabledLow() *ConnectorEnabledLow {
	return &ConnectorEnabledLow{}
}

func (cn *ConnectorEnabledLow) Connect(line Line) {
	cn.line = line
}

// TODO: Handle not connected lines, now is throwing null pointer exception
func (cn *ConnectorEnabledLow) Enabled() bool {
	return !cn.line.Status()
}

func (cn *ConnectorEnabledLow) SetEnable(value bool) {
	cn.line.Set(!value)
}

func (cn *ConnectorEnabledLow) GetLine() Line {
	return cn.line
}
