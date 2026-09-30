package modules

import "github.com/fran150/clementina-6502/pkg/components/buses"

// net joins several chip pins into one node, the way a trace does on the board.
// Connecting it to a line connects every pin to that line. Reads and writes go
// through the first pin.
type net struct {
	pins []buses.LineConnector
}

// newNet creates a node that joins the given pins.
func newNet(pins ...buses.LineConnector) *net {
	return &net{pins: pins}
}

// Connect connects every pin of the node to the line.
func (n *net) Connect(line buses.Line) {
	for _, pin := range n.pins {
		pin.Connect(line)
	}
}

// Enabled returns the state of the first pin.
func (n *net) Enabled() bool {
	return n.pins[0].Enabled()
}

// SetEnable drives the line through the first pin.
func (n *net) SetEnable(value bool) {
	n.pins[0].SetEnable(value)
}

// GetLine returns the line the node is connected to.
func (n *net) GetLine() buses.Line {
	return n.pins[0].GetLine()
}
