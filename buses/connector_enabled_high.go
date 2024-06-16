package buses

type ConnectorEnabledHigh struct {
	line Line
}

func CreateConnectorEnabledHigh() *ConnectorEnabledHigh {
	return &ConnectorEnabledHigh{}
}

func (cn *ConnectorEnabledHigh) Connect(line Line) {
	cn.line = line
}

func (cn *ConnectorEnabledHigh) Enabled() bool {
	return cn.line.Status()
}

func (cn *ConnectorEnabledHigh) SetEnable(value bool) {
	cn.line.Set(value)
}

func (cn *ConnectorEnabledHigh) GetLine() Line {
	return cn.line
}
