//go:build (linux && arm) || (linux && arm64)

package mia

import (
	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/fran150/clementina-6502/pkg/components/buses"
	"github.com/warthog618/go-gpiocdev"
)

/*******************************************************************************************
* Structs definition
********************************************************************************************/

// pico_mia represents a physical Pico running the Clementina MIA firmware.
// It bridges the emulator buses and control lines to Raspberry Pi GPIO pins.
// See GPIO_PIN_MAP.md for the Pico to Raspberry Pi 5 wiring map.
type pico_mia struct {
	gpioController *common.GPIOController

	addressBus *buses.BusConnector[uint8]
	dataBus    *buses.BusConnector[uint8]

	miaCS        buses.LineConnector
	reset        buses.LineConnector
	resetRequest buses.LineConnector
	writeEnable  buses.LineConnector
	irq          buses.LineConnector
}

/*******************************************************************************************
* Constructor
********************************************************************************************/

// NewPicoMia creates a MIA component backed by the Raspberry Pi GPIO interface.
//
// Parameters:
//   - chipName: The Linux GPIO chip to use
//
// Returns:
//   - A MIA chip connected to the GPIO controller
//   - An error if the GPIO controller cannot be initialized
func NewPicoMia(chipName string) (components.MiaChip, error) {
	gpioController, err := common.GetGPIOController(chipName)
	if err != nil {
		return nil, err
	}

	return &pico_mia{
		gpioController: gpioController,

		addressBus:   buses.NewBusConnector[uint8](),
		dataBus:      buses.NewBusConnector[uint8](),
		miaCS:        buses.NewConnectorEnabledHigh(),
		reset:        buses.NewConnectorEnabledLow(),
		resetRequest: buses.NewConnectorEnabledLow(),
		writeEnable:  buses.NewConnectorEnabledLow(),
		irq:          buses.NewConnectorEnabledLow(),
	}, nil
}

/*******************************************************************************************
* MiaChip Interface methods
********************************************************************************************/

// AddressBus returns the 5-bit MIA register address bus connector.
func (c *pico_mia) AddressBus() *buses.BusConnector[uint8] {
	return c.addressBus
}

// DataBus returns the shared 8-bit data bus connector.
func (c *pico_mia) DataBus() *buses.BusConnector[uint8] {
	return c.dataBus
}

// MiaCS returns the active-high MIA chip select connector.
func (c *pico_mia) MiaCS() buses.LineConnector {
	return c.miaCS
}

// Reset returns the active-low reset line driven by the MIA.
func (c *pico_mia) Reset() buses.LineConnector {
	return c.reset
}

// ResetRequest returns the active-low input that asks MIA to reset.
func (c *pico_mia) ResetRequest() buses.LineConnector {
	return c.resetRequest
}

// WriteEnable returns the active-low CPU write enable connector.
func (c *pico_mia) WriteEnable() buses.LineConnector {
	return c.writeEnable
}

// Irq returns the active-low IRQ connector driven by MIA.
func (c *pico_mia) Irq() buses.LineConnector {
	return c.irq
}

/*******************************************************************************************
* Ticker methods
********************************************************************************************/

// Tick starts one MIA bus cycle by driving GPIO outputs for the Pico to sample.
//
// Parameters:
//   - context: The current step context
func (c *pico_mia) Tick(context *common.StepContext) {
	if err := c.driveOutputLines(); err != nil {
		panic(err)
	}

	if err := c.driveAddressBus(); err != nil {
		panic(err)
	}

	if err := c.prepareDataBus(); err != nil {
		panic(err)
	}
}

// PostTick completes one MIA bus cycle by sampling GPIO inputs from the Pico.
//
// Parameters:
//   - context: The current step context
func (c *pico_mia) PostTick(context *common.StepContext) {
	if err := c.driveInputLines(); err != nil {
		panic(err)
	}

	if err := c.completeDataBus(); err != nil {
		panic(err)
	}
}

/*******************************************************************************************
* GPIO line helpers
********************************************************************************************/

// driveOutputLines drives emulator-owned control lines onto Raspberry Pi GPIO.
//
// Returns:
//   - An error if any GPIO write fails
func (c *pico_mia) driveOutputLines() error {
	if err := driveGPIOLine(c.writeEnable.GetLine(), c.gpioController.WriteEnable()); err != nil {
		return err
	}

	if err := driveGPIOLine(c.miaCS.GetLine(), c.gpioController.MiaCS()); err != nil {
		return err
	}

	if err := driveGPIOLine(c.resetRequest.GetLine(), c.gpioController.ResetRequest()); err != nil {
		return err
	}

	return nil
}

// driveInputLines samples Pico-owned control lines into the emulator.
//
// Returns:
//   - An error if any GPIO read fails
func (c *pico_mia) driveInputLines() error {
	if err := driveEmulatorLine(c.gpioController.Reset(), c.reset.GetLine()); err != nil {
		return err
	}

	if err := driveEmulatorLine(c.gpioController.Irq(), c.Irq().GetLine()); err != nil {
		return err
	}

	return nil
}

/*******************************************************************************************
* GPIO bus helpers
********************************************************************************************/

// driveAddressBus drives the low MIA address bits onto Raspberry Pi GPIO.
//
// Returns:
//   - An error if the GPIO bus write fails
func (c *pico_mia) driveAddressBus() error {
	return driveGPIOBus(c.addressBus, c.gpioController.AddressBus())
}

// prepareDataBus sets the Pi data bus direction for the current MIA cycle.
//
// Returns:
//   - An error if GPIO direction or data access fails
func (c *pico_mia) prepareDataBus() error {
	switch picoDataBusDirectionForCycle(c.miaCS.Enabled(), c.writeEnable.Enabled()) {
	case picoDataBusOutput:
		// CPU writes to MIA, Pi drives the data bus.
		if err := driveGPIOBus(c.dataBus, c.gpioController.DataBus()); err != nil {
			return err
		}
	case picoDataBusInput:
		// CPU reads from MIA, Pi samples the Pico-driven data bus.
		if err := driveEmulatorBus(c.gpioController.DataBus(), c.dataBus); err != nil {
			return err
		}
	case picoDataBusHighZ:
		if err := c.gpioController.DataBus().Reconfigure(gpiocdev.AsInput); err != nil {
			return err
		}
	}

	return nil
}

// completeDataBus samples the Pico data bus after a CPU read cycle.
//
// Returns:
//   - An error if the GPIO bus read fails
func (c *pico_mia) completeDataBus() error {
	if picoDataBusDirectionForCycle(c.miaCS.Enabled(), c.writeEnable.Enabled()) != picoDataBusInput {
		return nil
	}

	return driveEmulatorBus(c.gpioController.DataBus(), c.dataBus)
}

// driveGPIOLine writes one emulator line value to one GPIO line.
//
// Parameters:
//   - source: The emulator line to read
//   - dest: The GPIO line to write
//
// Returns:
//   - An error if the GPIO write fails
func driveGPIOLine(source buses.Line, dest *gpiocdev.Line) error {
	var err error = nil

	if source.Status() {
		err = dest.SetValue(1)
	} else {
		err = dest.SetValue(0)
	}

	return err
}

// driveEmulatorLine writes one GPIO line value to one emulator line.
//
// Parameters:
//   - source: The GPIO line to read
//   - dest: The emulator line to write
//
// Returns:
//   - An error if the GPIO read fails
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

// driveGPIOBus writes one emulator bus value to a GPIO bus.
//
// Parameters:
//   - source: The emulator bus to read
//   - dest: The GPIO lines to write
//
// Returns:
//   - An error if GPIO direction or value writes fail
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

// driveEmulatorBus writes one GPIO bus value to an emulator bus.
//
// Parameters:
//   - source: The GPIO lines to read
//   - dest: The emulator bus to write
//
// Returns:
//   - An error if GPIO direction or value reads fail
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
