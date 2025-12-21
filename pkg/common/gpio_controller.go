//go:build (linux && arm) || (linux && arm64)

package common

import "github.com/warthog618/go-gpiocdev"

type GPIOController struct {
	addressBus    [8]*gpiocdev.Line
	dataBus       [8]*gpiocdev.Line
	picoRAMEnable *gpiocdev.Line
	reset         *gpiocdev.Line
	writeEnable   *gpiocdev.Line
	outputEnable  *gpiocdev.Line
	hiRAMCS       *gpiocdev.Line
	io0CS         *gpiocdev.Line
	irqOut        *gpiocdev.Line
	clock         *gpiocdev.Line
}

var gpioInterfaceInstance *GPIOController

func GetGPIOInterface(chipName string) (*GPIOController, error) {
	if gpioInterfaceInstance == nil {
		var chip *gpiocdev.Chip
		var addressBus [8]*gpiocdev.Line
		var dataBus [8]*gpiocdev.Line
		var picoRAMEnable *gpiocdev.Line
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

		if addressBus, err = requestBusLines(chip, 0); err != nil {
			return nil, err
		}

		if dataBus, err = requestBusLines(chip, 8); err != nil {
			return nil, err
		}

		if picoRAMEnable, err = chip.RequestLine(16, gpiocdev.AsInput); err != nil {
			return nil, err
		}

		if reset, err = chip.RequestLine(17, gpiocdev.AsInput); err != nil {
			return nil, err
		}

		if writeEnable, err = chip.RequestLine(18, gpiocdev.AsInput); err != nil {
			return nil, err
		}

		if outputEnable, err = chip.RequestLine(19, gpiocdev.AsInput); err != nil {
			return nil, err
		}

		if hiRAMCS, err = chip.RequestLine(20, gpiocdev.AsInput); err != nil {
			return nil, err
		}

		if io0CS, err = chip.RequestLine(21, gpiocdev.AsInput); err != nil {
			return nil, err
		}

		if irqOut, err = chip.RequestLine(26, gpiocdev.AsInput); err != nil {
			return nil, err
		}

		if clock, err = chip.RequestLine(28, gpiocdev.AsInput); err != nil {
			return nil, err
		}

		gpioInterfaceInstance = &GPIOController{
			addressBus:    addressBus,
			dataBus:       dataBus,
			picoRAMEnable: picoRAMEnable,
			reset:         reset,
			writeEnable:   writeEnable,
			outputEnable:  outputEnable,
			hiRAMCS:       hiRAMCS,
			io0CS:         io0CS,
			irqOut:        irqOut,
			clock:         clock,
		}
	}

	return gpioInterfaceInstance, nil
}

func requestBusLines(chip *gpiocdev.Chip, baseGPIO int) ([8]*gpiocdev.Line, error) {
	var busLines [8]*gpiocdev.Line
	var err error

	for i := range 8 {
		if busLines[i], err = chip.RequestLine(baseGPIO+i, gpiocdev.AsInput); err != nil {
			return [8]*gpiocdev.Line{}, err
		}
	}

	return busLines, nil
}

func (g *GPIOController) AddressBus() [8]*gpiocdev.Line {
	return g.addressBus
}

func (g *GPIOController) DataBus() [8]*gpiocdev.Line {
	return g.dataBus
}

func (g *GPIOController) PicoRAMEnable() *gpiocdev.Line {
	return g.picoRAMEnable
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
