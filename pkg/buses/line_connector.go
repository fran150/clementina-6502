package buses

type LineConnector interface {
	Connect(line Line)
	Enabled() bool
	SetEnable(value bool)
	GetLine() Line
}
