//go:build (linux && arm) || (linux && arm64)

package pico

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/warthog618/go-gpiocdev"
)

var addressBusGPIO = []int{0, 1, 2, 3, 4, 5, 6, 7}
var dataBusGPIO = []int{8, 9, 10, 11, 12, 13, 14, 15}

const picoRAMEnableGPIO = 16
const resetOutGPIO = 17
const writeEnableGPIO = 18
const outputEnableGPIO = 19

const hiRAMCSGPIO = 20
const io0CSGPIO = 21

const irqOutGPIO = 26

type MiaConnector struct {
	addressBus *buses.BusConnector[uint8]
	dataBus    *buses.BusConnector[uint8]

	hiRAMEnable  buses.LineConnector
	reset        buses.LineConnector
	writeEnable  buses.LineConnector
	outputEnable buses.LineConnector
	romCS        buses.LineConnector
	videoCS      buses.LineConnector
	genCS        buses.LineConnector
	irqOut       buses.LineConnector
}

func NewMiaConnector() *MiaConnector {
	// Initialize GPIO pins as inputs with no pull-up/pull-down
	allPins := append(addressBusGPIO, dataBusGPIO...)
	allPins = append(allPins, picoRAMEnableGPIO, resetOutGPIO, writeEnableGPIO, outputEnableGPIO, hiRAMCSGPIO, io0CSGPIO, irqOutGPIO)

	for _, pin := range allPins {
		gpiocdev.RequestLine("gpiochip0", pin, gpiocdev.AsInput)
	}

	return &MiaConnector{
		addressBus:   buses.NewBusConnector[uint8](),
		dataBus:      buses.NewBusConnector[uint8](),
		hiRAMEnable:  buses.NewConnectorEnabledLow(),
		reset:        buses.NewConnectorEnabledLow(),
		writeEnable:  buses.NewConnectorEnabledLow(),
		outputEnable: buses.NewConnectorEnabledLow(),
		romCS:        buses.NewConnectorEnabledLow(),
		videoCS:      buses.NewConnectorEnabledLow(),
		genCS:        buses.NewConnectorEnabledLow(),
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

func (c *MiaConnector) RomCS() buses.LineConnector {
	return c.romCS
}

func (c *MiaConnector) VideoCS() buses.LineConnector {
	return c.videoCS
}

func (c *MiaConnector) GenCS() buses.LineConnector {
	return c.genCS
}

func (c *MiaConnector) IrqOut() buses.LineConnector {
	return c.irqOut
}

func (c *MiaConnector) Tick(context *common.StepContext) {

}
