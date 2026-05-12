package mia

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/fran150/clementina-6502/pkg/components/buses"
)

type emulated_mia struct {
	addressBus *buses.BusConnector[uint8]
	dataBus    *buses.BusConnector[uint8]

	miaCS       buses.LineConnector
	reset       buses.LineConnector
	writeEnable buses.LineConnector
	irq         buses.LineConnector
}

func NewEmulatedMia() components.MiaChip {
	return &emulated_mia{
		addressBus:  buses.NewBusConnector[uint8](),
		dataBus:     buses.NewBusConnector[uint8](),
		miaCS:       buses.NewConnectorEnabledLow(),
		reset:       buses.NewConnectorEnabledLow(),
		writeEnable: buses.NewConnectorEnabledLow(),
		irq:         buses.NewConnectorEnabledLow(),
	}
}

func (c *emulated_mia) AddressBus() *buses.BusConnector[uint8] {
	return c.addressBus
}

func (c *emulated_mia) DataBus() *buses.BusConnector[uint8] {
	return c.dataBus
}

func (c *emulated_mia) MiaCS() buses.LineConnector {
	return c.miaCS
}

func (c *emulated_mia) Reset() buses.LineConnector {
	return c.reset
}

func (c *emulated_mia) WriteEnable() buses.LineConnector {
	return c.writeEnable
}

func (c *emulated_mia) Irq() buses.LineConnector {
	return c.irq
}

func (c *emulated_mia) Tick(context *common.StepContext) {

}
