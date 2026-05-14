//go:build (linux && arm) || (linux && arm64)

package mia

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/warthog618/go-gpiocdev"
)

type pico_mia struct {
	gpioController *common.GPIOController

	addressBus *buses.BusConnector[uint8]
	dataBus    *buses.BusConnector[uint8]

	miaCS       buses.LineConnector
	reset       buses.LineConnector
	writeEnable buses.LineConnector
	irq         buses.LineConnector
}

func NewPicoMia(chipName string) (components.MiaChip, error) {
	gpioController, err := common.GetGPIOController(chipName)
	if err != nil {
		return nil, err
	}

	return &pico_mia{
		gpioController: gpioController,

		addressBus:  buses.NewBusConnector[uint8](),
		dataBus:     buses.NewBusConnector[uint8](),
		miaCS:       buses.NewConnectorEnabledHigh(),
		reset:       buses.NewConnectorEnabledLow(),
		writeEnable: buses.NewConnectorEnabledLow(),
		irq:         buses.NewConnectorEnabledLow(),
	}, nil
}

func (c *pico_mia) AddressBus() *buses.BusConnector[uint8] {
	return c.addressBus
}

func (c *pico_mia) DataBus() *buses.BusConnector[uint8] {
	return c.dataBus
}

func (c *pico_mia) MiaCS() buses.LineConnector {
	return c.miaCS
}

func (c *pico_mia) Reset() buses.LineConnector {
	return c.reset
}

func (c *pico_mia) WriteEnable() buses.LineConnector {
	return c.writeEnable
}

func (c *pico_mia) Irq() buses.LineConnector {
	return c.irq
}

func (c *pico_mia) Tick(context *common.StepContext) {
	if err := c.driveOutputLines(); err != nil {
		panic(err)
	}

	if err := c.driveInputLines(); err != nil {
		panic(err)
	}

	if err := c.driveBuses(); err != nil {
		panic(err)
	}
}

func (c *pico_mia) driveOutputLines() error {
	if err := driveGPIOLine(c.writeEnable.GetLine(), c.gpioController.WriteEnable()); err != nil {
		return err
	}

	if err := driveGPIOLine(c.miaCS.GetLine(), c.gpioController.MiaCS()); err != nil {
		return err
	}

	return nil
}

func (c *pico_mia) driveInputLines() error {
	if err := driveEmulatorLine(c.gpioController.Reset(), c.reset.GetLine()); err != nil {
		return err
	}

	if err := driveEmulatorLine(c.gpioController.Irq(), c.Irq().GetLine()); err != nil {
		return err
	}

	return nil
}

func (c *pico_mia) driveBuses() error {
	if err := driveGPIOBus(c.addressBus, c.gpioController.AddressBus()); err != nil {
		return err
	}

	if c.miaCS.Enabled() && !c.writeEnable.Enabled() {
		// Read mode -> emulation must drive GPIO bus for MIA to read data
		if err := driveGPIOBus(c.dataBus, c.gpioController.DataBus()); err != nil {
			return err
		}
	} else if c.miaCS.Enabled() && c.writeEnable.Enabled() {
		// Write mode -> emulation must read the bus to get data set by MIA
		if err := driveEmulatorBus(c.gpioController.DataBus(), c.dataBus); err != nil {
			return err
		}
	} else {
		if err := c.gpioController.DataBus().Reconfigure(gpiocdev.AsInput); err != nil {
			return err
		}
	}

	return nil
}

func driveGPIOLine(source buses.Line, dest *gpiocdev.Line) error {
	var err error = nil

	if source.Status() {
		err = dest.SetValue(1)
	} else {
		err = dest.SetValue(0)
	}

	return err
}

func driveEmulatorLine(source *gpiocdev.Line, dest buses.Line) error {
	value, err := source.Value()
	if err != nil {
		return err
	}

	if value == 1 {
		dest.Set(true)
	} else {
		dest.Set(false)
	}

	return nil
}

func driveGPIOBus(source *buses.BusConnector[uint8], dest *gpiocdev.Lines) error {
	buffer := make([]int, len(dest.Offsets()))
	value := source.Read()

	for i := range len(buffer) {
		if value&(1<<i) != 0 {
			buffer[i] = 1
		}
	}

	if err := dest.Reconfigure(gpiocdev.AsOutput(0)); err != nil {
		return err
	}

	return dest.SetValues(buffer)
}

func driveEmulatorBus(source *gpiocdev.Lines, dest *buses.BusConnector[uint8]) error {
	if err := source.Reconfigure(gpiocdev.AsInput); err != nil {
		return err
	}

	buffer := make([]int, len(source.Offsets()))
	var value uint8 = 0

	if err := source.Values(buffer); err != nil {
		return err
	}

	for i, bit := range buffer {
		if bit != 0 {
			value |= 1 << i
		}
	}

	dest.Write(value)

	return nil
}
