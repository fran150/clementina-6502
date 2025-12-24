//go:build (linux && arm) || (linux && arm64)

package pico

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/warthog618/go-gpiocdev"
)

type MiaConnector struct {
	gpioController *common.GPIOController

	addressBus *buses.BusConnector[uint8]
	dataBus    *buses.BusConnector[uint8]

	hiRAMEnable  buses.LineConnector
	reset        buses.LineConnector
	writeEnable  buses.LineConnector
	outputEnable buses.LineConnector
	hiRAMCS      buses.LineConnector
	io0CS        buses.LineConnector
	irqOut       buses.LineConnector
}

func NewMiaConnector(chipName string) *MiaConnector {
	gpioController, err := common.GetGPIOController(chipName)
	// TODO: Might want to handle this error better
	if err != nil {
		panic(err)
	}

	return &MiaConnector{
		gpioController: gpioController,

		addressBus:   buses.NewBusConnector[uint8](),
		dataBus:      buses.NewBusConnector[uint8](),
		hiRAMEnable:  buses.NewConnectorEnabledLow(),
		reset:        buses.NewConnectorEnabledLow(),
		writeEnable:  buses.NewConnectorEnabledLow(),
		outputEnable: buses.NewConnectorEnabledLow(),
		hiRAMCS:      buses.NewConnectorEnabledLow(),
		io0CS:        buses.NewConnectorEnabledLow(),
		irqOut:       buses.NewConnectorEnabledLow(),
	}
}

func (c *MiaConnector) AddressBus() *buses.BusConnector[uint8] {
	return c.addressBus
}

func (c *MiaConnector) DataBus() *buses.BusConnector[uint8] {
	return c.dataBus
}

func (c *MiaConnector) HiRAMEnable() buses.LineConnector {
	return c.hiRAMEnable
}

func (c *MiaConnector) Reset() buses.LineConnector {
	return c.reset
}

func (c *MiaConnector) WriteEnable() buses.LineConnector {
	return c.writeEnable
}

func (c *MiaConnector) OutputEnable() buses.LineConnector {
	return c.outputEnable
}

func (c *MiaConnector) HiRAMCS() buses.LineConnector {
	return c.hiRAMCS
}

func (c *MiaConnector) IO0CS() buses.LineConnector {
	return c.io0CS
}

func (c *MiaConnector) IrqOut() buses.LineConnector {
	return c.irqOut
}

func (c *MiaConnector) Tick(context *common.StepContext) {
	c.gpioController.WriteEnable().SetValue(getLineStatusForGPIO(c.writeEnable.GetLine()))
	c.gpioController.OutputEnable().SetValue(getLineStatusForGPIO(c.outputEnable.GetLine()))
	c.gpioController.HiRAMCS().SetValue(getLineStatusForGPIO(c.hiRAMCS.GetLine()))
	c.gpioController.Io0CS().SetValue(getLineStatusForGPIO(c.io0CS.GetLine()))

	c.hiRAMEnable.GetLine().Set(getLineStatusForEmulator(c.gpioController.HiRAMEnable()))
	c.reset.GetLine().Set(getLineStatusForEmulator(c.gpioController.Reset()))

	// Open collector logic. The line will be pulled low, the MIA will not drive the line high at any point.
	if !getLineStatusForEmulator(c.gpioController.IrqOut()) {
		c.irqOut.GetLine().Set(false)
	}

	common.WriteGPIOBus(c.gpioController.AddressBus(), c.addressBus.Read())

	if c.outputEnable.Enabled() && !c.writeEnable.Enabled() && (c.hiRAMCS.Enabled() || c.io0CS.Enabled()) {
		common.SetBusDirection(c.gpioController.DataBus(), false)
		dataValue := common.ReadGPIOBus(c.gpioController.DataBus())
		c.dataBus.Write(dataValue)
	} else if !c.outputEnable.Enabled() && c.writeEnable.Enabled() && (c.hiRAMCS.Enabled() || c.io0CS.Enabled()) {
		common.SetBusDirection(c.gpioController.DataBus(), true)
		dataValue := c.dataBus.Read()
		common.WriteGPIOBus(c.gpioController.DataBus(), dataValue)
	} else {
		common.SetBusDirection(c.gpioController.DataBus(), false)
	}
}

func getLineStatusForGPIO(line buses.Line) int {
	if line.Status() {
		return 1
	} else {
		return 0
	}
}

func getLineStatusForEmulator(line *gpiocdev.Line) bool {
	value, err := line.Value()
	// TODO: Might want to handle this error better
	if err != nil {
		panic(err)
	}

	if value == 0 {
		return false
	} else {
		return true
	}
}
