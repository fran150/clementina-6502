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

	// useFastGPIO is true when the RP1 mmap path was initialised successfully.
	// The hot path (Tick / PostTick) branches on this once-set flag so that the
	// chardev fallback remains available on hardware where /dev/gpiomem0 is absent.
	useFastGPIO bool

	// chardev-only fields — used only when useFastGPIO is false.
	addressBusConfigured bool
	currentDataDir       picoDataBusDirection
	busBuf               []int
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

		useFastGPIO: gpioController.HasFastGPIOWrites(),

		// chardev fallback fields
		currentDataDir: 0xFF,
		busBuf:         make([]int, 8),
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
func (c *pico_mia) Tick(context *common.StepContext) {
	if c.useFastGPIO {
		c.tickFast()
		return
	}

	if err := c.driveAddressBus(); err != nil {
		panic(err)
	}
	if err := c.prepareDataBus(); err != nil {
		panic(err)
	}
	if err := c.driveOutputLines(); err != nil {
		panic(err)
	}
}

// PostTick completes one MIA bus cycle by sampling GPIO inputs from the Pico.
func (c *pico_mia) PostTick(context *common.StepContext) {
	if c.useFastGPIO {
		c.postTickFast()
		return
	}

	if err := c.driveInputLines(); err != nil {
		panic(err)
	}
	if err := c.completeDataBus(); err != nil {
		panic(err)
	}
}

/*******************************************************************************************
* Fast (mmap) hot path
*
* Each operation is a direct RP1 register read or write — no system calls, no kernel
* transitions. Direction changes (data bus input ↔ output) are single register writes
* (~150 ns) instead of Reconfigure() ioctl calls (~39 µs).
********************************************************************************************/

// tickFast drives all emulator outputs onto the GPIO bus in the fast (mmap) path.
func (c *pico_mia) tickFast() {
	gc := c.gpioController

	// Drive the 5-bit address bus.
	gc.WriteAddressBusFast(c.addressBus.Read())

	// Drive the 3 output control lines.
	if c.writeEnable.GetLine() != nil {
		we := 0
		if c.writeEnable.GetLine().Status() {
			we = 1
		}
		gc.WriteWeFast(we)
	}
	if c.miaCS.GetLine() != nil {
		cs := 0
		if c.miaCS.GetLine().Status() {
			cs = 1
		}
		gc.WriteMiacsFast(cs)
	}
	if c.resetRequest.GetLine() != nil {
		rr := 0
		if c.resetRequest.GetLine().Status() {
			rr = 1
		}
		gc.WriteResetReqFast(rr)
	}

	// Set data bus direction and drive value when the CPU is writing.
	targetDir := picoDataBusDirectionForCycle(c.miaCS.Enabled(), c.writeEnable.Enabled())
	if targetDir == picoDataBusOutput {
		gc.SetDataBusOutputFast(c.dataBus.Read())
	} else {
		gc.SetDataBusInputFast()
	}
}

// postTickFast samples all GPIO inputs into the emulator in the fast (mmap) path.
func (c *pico_mia) postTickFast() {
	gc := c.gpioController

	// Sample the two input control lines.
	if c.reset.GetLine() != nil {
		c.reset.GetLine().Set(gc.ReadResetFast() != 0)
	}
	if c.irq.GetLine() != nil {
		c.irq.GetLine().Set(gc.ReadIrqFast() != 0)
	}

	// Sample the data bus when the CPU is reading from MIA.
	if picoDataBusDirectionForCycle(c.miaCS.Enabled(), c.writeEnable.Enabled()) == picoDataBusInput {
		c.dataBus.Write(gc.ReadDataBusFast())
	}
}

/*******************************************************************************************
* Chardev (gpiocdev) fallback — used when /dev/gpiomem0 is unavailable.
********************************************************************************************/

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

func (c *pico_mia) driveInputLines() error {
	if err := driveEmulatorLine(c.gpioController.Reset(), c.reset.GetLine()); err != nil {
		return err
	}
	if err := driveEmulatorLine(c.gpioController.Irq(), c.Irq().GetLine()); err != nil {
		return err
	}
	return nil
}

func (c *pico_mia) driveAddressBus() error {
	if !c.addressBusConfigured {
		if err := c.gpioController.AddressBus().Reconfigure(gpiocdev.AsOutput(0)); err != nil {
			return err
		}
		c.addressBusConfigured = true
	}
	return driveGPIOBus(c.addressBus, c.gpioController.AddressBus(), c.busBuf)
}

func (c *pico_mia) prepareDataBus() error {
	targetDir := picoDataBusDirectionForCycle(c.miaCS.Enabled(), c.writeEnable.Enabled())

	if targetDir != c.currentDataDir {
		var err error
		switch targetDir {
		case picoDataBusOutput:
			err = c.gpioController.DataBus().Reconfigure(gpiocdev.AsOutput(0))
		case picoDataBusInput, picoDataBusHighZ:
			err = c.gpioController.DataBus().Reconfigure(gpiocdev.AsInput)
		}
		if err != nil {
			return err
		}
		c.currentDataDir = targetDir
	}

	if targetDir == picoDataBusOutput {
		return driveGPIOBus(c.dataBus, c.gpioController.DataBus(), c.busBuf)
	}
	return nil
}

func (c *pico_mia) completeDataBus() error {
	if c.currentDataDir != picoDataBusInput {
		return nil
	}
	return driveEmulatorBus(c.gpioController.DataBus(), c.dataBus, c.busBuf)
}

/*******************************************************************************************
* GPIO line / bus helpers (shared by chardev fallback)
********************************************************************************************/

func driveGPIOLine(source buses.Line, dest *gpiocdev.Line) error {
	if source.Status() {
		return dest.SetValue(1)
	}
	return dest.SetValue(0)
}

func driveEmulatorLine(source *gpiocdev.Line, dest buses.Line) error {
	value, err := source.Value()
	if err != nil {
		return err
	}
	dest.Set(value == 1)
	return nil
}

func driveGPIOBus(source *buses.BusConnector[uint8], dest *gpiocdev.Lines, scratch []int) error {
	buffer := scratch[:len(dest.Offsets())]
	value := source.Read()
	for i := range buffer {
		if value&(1<<i) != 0 {
			buffer[i] = 1
		} else {
			buffer[i] = 0
		}
	}
	return dest.SetValues(buffer)
}

func driveEmulatorBus(source *gpiocdev.Lines, dest *buses.BusConnector[uint8], scratch []int) error {
	buffer := scratch[:len(source.Offsets())]
	var value uint8

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
