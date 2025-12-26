//go:build (linux && arm) || (linux && arm64)

package common

import "github.com/warthog618/go-gpiocdev"

var addressLinesGPIO = [8]int{2, 3, 4, 17, 27, 22, 10, 9}
var dataLinesGPIO = [8]int{11, 5, 6, 13, 19, 26, 20, 21}

var hiRAMEnableGPIO = 16
var resetGPIO = 12
var writeEnableGPIO = 7
var outputEnableGPIO = 8
var hiRAMCSGPIO = 25
var io0CSGPIO = 24
var irqOutGPIO = 18
var clockGPIO = 14

type GPIOController struct {
	addressBus   *gpiocdev.Lines
	dataBus      *gpiocdev.Lines
	hiRAMEnable  *gpiocdev.Line
	reset        *gpiocdev.Line
	writeEnable  *gpiocdev.Line
	outputEnable *gpiocdev.Line
	hiRAMCS      *gpiocdev.Line
	io0CS        *gpiocdev.Line
	irqOut       *gpiocdev.Line
	clock        *gpiocdev.Line
}

var gpioInterfaceInstance *GPIOController

func GetGPIOController(chipName string) (*GPIOController, error) {
	if gpioInterfaceInstance == nil {
		var chip *gpiocdev.Chip
		var addressBus *gpiocdev.Lines
		var dataBus *gpiocdev.Lines
		var hiRAMEnable *gpiocdev.Line
		var reset *gpiocdev.Line
		var writeEnable *gpiocdev.Line
		var outputEnable *gpiocdev.Line
		var hiRAMCS *gpiocdev.Line
		var io0CS *gpiocdev.Line
		var irqOut *gpiocdev.Line
		var clock *gpiocdev.Line
		var err error

		chip, err = gpiocdev.NewChip(chipName)
		if err != nil {
			return nil, err
		}

		if addressBus, err = gpiocdev.RequestLines(chipName, addressLinesGPIO[:], gpiocdev.AsOutput(0)); err != nil {
			return nil, err
		}

		if dataBus, err = gpiocdev.RequestLines(chipName, dataLinesGPIO[:], gpiocdev.AsInput); err != nil {
			return nil, err
		}

		if hiRAMEnable, err = chip.RequestLine(hiRAMEnableGPIO, gpiocdev.AsInput); err != nil {
			return nil, err
		}

		if reset, err = chip.RequestLine(resetGPIO, gpiocdev.AsInput); err != nil {
			return nil, err
		}

		if writeEnable, err = chip.RequestLine(writeEnableGPIO, gpiocdev.AsOutput(1)); err != nil {
			return nil, err
		}

		if outputEnable, err = chip.RequestLine(outputEnableGPIO, gpiocdev.AsOutput(1)); err != nil {
			return nil, err
		}

		if hiRAMCS, err = chip.RequestLine(hiRAMCSGPIO, gpiocdev.AsOutput(1)); err != nil {
			return nil, err
		}

		if io0CS, err = chip.RequestLine(io0CSGPIO, gpiocdev.AsOutput(1)); err != nil {
			return nil, err
		}

		if irqOut, err = chip.RequestLine(irqOutGPIO, gpiocdev.AsInput); err != nil {
			return nil, err
		}

		if clock, err = chip.RequestLine(clockGPIO, gpiocdev.AsInput); err != nil {
			return nil, err
		}

		gpioInterfaceInstance = &GPIOController{
			addressBus:   addressBus,
			dataBus:      dataBus,
			hiRAMEnable:  hiRAMEnable,
			reset:        reset,
			writeEnable:  writeEnable,
			outputEnable: outputEnable,
			hiRAMCS:      hiRAMCS,
			io0CS:        io0CS,
			irqOut:       irqOut,
			clock:        clock,
		}
	}

	return gpioInterfaceInstance, nil
}

func (g *GPIOController) AddressBus() *gpiocdev.Lines {
	return g.addressBus
}

func (g *GPIOController) DataBus() *gpiocdev.Lines {
	return g.dataBus
}

func (g *GPIOController) HiRAMEnable() *gpiocdev.Line {
	return g.hiRAMEnable
}

func (g *GPIOController) Reset() *gpiocdev.Line {
	return g.reset
}

func (g *GPIOController) WriteEnable() *gpiocdev.Line {
	return g.writeEnable
}

func (g *GPIOController) OutputEnable() *gpiocdev.Line {
	return g.outputEnable
}

func (g *GPIOController) HiRAMCS() *gpiocdev.Line {
	return g.hiRAMCS
}

func (g *GPIOController) Io0CS() *gpiocdev.Line {
	return g.io0CS
}

func (g *GPIOController) IrqOut() *gpiocdev.Line {
	return g.irqOut
}

func (g *GPIOController) Clock() *gpiocdev.Line {
	return g.clock
}

func SetBusDirection(bus [8]*gpiocdev.Line, asOutput bool) error {
	var err error
	for i := range 8 {
		if asOutput {
			err = bus[i].Reconfigure(gpiocdev.AsOutput(0))
		} else {
			err = bus[i].Reconfigure(gpiocdev.AsInput)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func WriteGPIOBus(bus *gpiocdev.Lines, value uint8) {
	values := make([]int, 8)

	for i := range 8 {
		values[i] = (int(value & (1 << i)))
	}

	bus.SetValues(values)
}

func ReadGPIOBus(bus *gpiocdev.Lines) uint8 {
	var value uint8 = 0
	values := make([]int, 8)

	err := bus.Values(values)
	if err != nil {
		panic(err)
	}

	for i := range 8 {
		if values[i] != 0 {
			value |= (1 << i)
		}
	}

	return value
}

func (g *GPIOController) Close() {
	g.addressBus.Close()
	g.dataBus.Close()
	g.hiRAMEnable.Close()
	g.reset.Close()
	g.writeEnable.Close()
	g.outputEnable.Close()
	g.hiRAMCS.Close()
	g.io0CS.Close()
	g.irqOut.Close()
	g.clock.Close()
}
