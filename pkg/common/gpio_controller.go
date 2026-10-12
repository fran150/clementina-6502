//go:build (linux && arm) || (linux && arm64)

package common

import "github.com/warthog618/go-gpiocdev"

var addressLinesGPIO = []int{11, 5, 6, 13, 19}
var dataLinesGPIO = []int{2, 3, 4, 17, 27, 22, 10, 9}

var miaCSGPIO = 21
var resbGPIO = 20
var resetRequestGPIO = 26
var weGPIO = 16
var irqbGPIO = 12
var phi2GPIO = 18

// GPIOController owns the Raspberry Pi GPIO lines wired to the Clementina bus
// and MIA.
type GPIOController struct {
	addressBus   *gpiocdev.Lines
	dataBus      *gpiocdev.Lines
	miaCS        *gpiocdev.Line
	reset        *gpiocdev.Line
	resetRequest *gpiocdev.Line
	writeEnable  *gpiocdev.Line
	irq          *gpiocdev.Line
	phi2         *gpiocdev.Line
}

var gpioInterfaceInstance *GPIOController

// GetGPIOController requests the bus lines from the GPIO chip named chipName on
// first use and returns the shared controller afterwards.
func GetGPIOController(chipName string) (*GPIOController, error) {
	if gpioInterfaceInstance == nil {
		var chip *gpiocdev.Chip
		var addressBus *gpiocdev.Lines
		var dataBus *gpiocdev.Lines
		var miaCS *gpiocdev.Line
		var reset *gpiocdev.Line
		var resetRequest *gpiocdev.Line
		var writeEnable *gpiocdev.Line
		var irq *gpiocdev.Line
		var phi2 *gpiocdev.Line
		var err error

		chip, err = gpiocdev.NewChip(chipName)
		if err != nil {
			return nil, err
		}

		if addressBus, err = chip.RequestLines(addressLinesGPIO, gpiocdev.AsOutput(0)); err != nil {
			return nil, err
		}

		if dataBus, err = chip.RequestLines(dataLinesGPIO, gpiocdev.AsInput); err != nil {
			return nil, err
		}

		if miaCS, err = chip.RequestLine(miaCSGPIO, gpiocdev.AsOutput(0)); err != nil {
			return nil, err
		}

		if reset, err = chip.RequestLine(resbGPIO, gpiocdev.AsInput); err != nil {
			return nil, err
		}

		if resetRequest, err = chip.RequestLine(resetRequestGPIO, gpiocdev.AsOutput(1)); err != nil {
			return nil, err
		}

		if writeEnable, err = chip.RequestLine(weGPIO, gpiocdev.AsOutput(1)); err != nil {
			return nil, err
		}

		if irq, err = chip.RequestLine(irqbGPIO, gpiocdev.AsInput); err != nil {
			return nil, err
		}

		if phi2, err = chip.RequestLine(phi2GPIO, gpiocdev.AsInput); err != nil {
			return nil, err
		}

		gpioInterfaceInstance = &GPIOController{
			addressBus:   addressBus,
			dataBus:      dataBus,
			miaCS:        miaCS,
			reset:        reset,
			resetRequest: resetRequest,
			writeEnable:  writeEnable,
			irq:          irq,
			phi2:         phi2,
		}
	}

	return gpioInterfaceInstance, nil
}

// AddressBus returns the A0-A4 output lines.
func (g *GPIOController) AddressBus() *gpiocdev.Lines {
	return g.addressBus
}

// DataBus returns the D0-D7 lines.
func (g *GPIOController) DataBus() *gpiocdev.Lines {
	return g.dataBus
}

// MiaCS returns the MIA chip-select output line.
func (g *GPIOController) MiaCS() *gpiocdev.Line {
	return g.miaCS
}

// Reset returns the RESB input line.
func (g *GPIOController) Reset() *gpiocdev.Line {
	return g.reset
}

// ResetRequest returns the reset-request output line.
func (g *GPIOController) ResetRequest() *gpiocdev.Line {
	return g.resetRequest
}

// WriteEnable returns the write-enable output line.
func (g *GPIOController) WriteEnable() *gpiocdev.Line {
	return g.writeEnable
}

// Irq returns the IRQB input line.
func (g *GPIOController) Irq() *gpiocdev.Line {
	return g.irq
}

// Phi2 returns the PHI2 clock input line.
func (g *GPIOController) Phi2() *gpiocdev.Line {
	return g.phi2
}

// Close releases every GPIO line.
func (g *GPIOController) Close() {
	g.addressBus.Close()
	g.dataBus.Close()
	g.miaCS.Close()
	g.reset.Close()
	g.resetRequest.Close()
	g.writeEnable.Close()
	g.irq.Close()
	g.phi2.Close()
}
