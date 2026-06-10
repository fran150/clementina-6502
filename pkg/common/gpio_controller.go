//go:build (linux && arm) || (linux && arm64)

package common

import (
	"log"

	"github.com/warthog618/go-gpiocdev"
)

var addressLinesGPIO = []int{11, 5, 6, 13, 19}
var dataLinesGPIO = []int{2, 3, 4, 17, 27, 22, 10, 9}

var miaCSGPIO = 21
var resbGPIO = 20
var resetRequestGPIO = 26
var weGPIO = 16
var irqbGPIO = 12
var phi2GPIO = 18

type GPIOController struct {
	addressBus   *gpiocdev.Lines
	dataBus      *gpiocdev.Lines
	miaCS        *gpiocdev.Line
	reset        *gpiocdev.Line
	resetRequest *gpiocdev.Line
	writeEnable  *gpiocdev.Line
	irq          *gpiocdev.Line
	phi2         *gpiocdev.Line

	// mmap provides direct RP1 register access for the hot-path operations
	// (PHI2 polling, address/data bus reads/writes, control line writes).
	// Initialised after gpiocdev has set up FUNCSEL for all pins; nil if
	// /dev/gpiomem0 is not available (falls back to gpiocdev).
	mmap *rp1MmapGPIO
}

var gpioInterfaceInstance *GPIOController

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

		// Attempt to open the RP1 mmap path. gpiocdev has already programmed FUNCSEL
		// for every pin; we read those values and precompute the fast CTRL values.
		// If /dev/gpiomem0 is unavailable the chardev path remains fully functional.
		if m, merr := newRp1MmapGPIO(); merr != nil {
			log.Printf("gpio: mmap fast path unavailable (%v); using chardev", merr)
		} else {
			gpioInterfaceInstance.mmap = m
			gpioInterfaceInstance.initMmapPins()
		}
	}

	return gpioInterfaceInstance, nil
}

// initMmapPins calls InitPin for every GPIO used by the emulator.
// Must be called after gpiocdev has set up all pins so FUNCSEL is already correct.
func (g *GPIOController) initMmapPins() {
	for _, pin := range addressLinesGPIO {
		g.mmap.InitPin(pin, true, 0)
	}
	for _, pin := range dataLinesGPIO {
		g.mmap.InitPin(pin, false, 0)
	}
	g.mmap.InitPin(miaCSGPIO, true, 0)
	g.mmap.InitPin(resbGPIO, false, 0)
	g.mmap.InitPin(resetRequestGPIO, true, 1)
	g.mmap.InitPin(weGPIO, true, 1)
	g.mmap.InitPin(irqbGPIO, false, 0)
	g.mmap.InitPin(phi2GPIO, false, 0)
}

// HasFastGPIO returns true when the RP1 mmap path is available.
func (g *GPIOController) HasFastGPIO() bool {
	return g.mmap != nil
}

/*******************************************************************************************
* Fast (mmap) bulk read/write methods — hot-path replacements for chardev ioctl calls.
* Each method is a direct register operation: no system calls, no kernel transitions.
* Callers should check HasFastGPIO() at startup; after that, panic on nil mmap means
* the caller forgot the check.
********************************************************************************************/

// ReadPhi2Fast reads GPIO 18 (PHI2) via mmap (~100–200 ns vs ~1.5 µs chardev).
func (g *GPIOController) ReadPhi2Fast() int {
	return g.mmap.ReadPin(phi2GPIO)
}

// ReadAddressBusFast reads all 5 address-bus pins and packs them into a uint8 (bits 0–4).
// Replaces Lines.Values() (~8 µs chardev) with 5 sequential register reads.
func (g *GPIOController) ReadAddressBusFast() uint8 {
	var v uint8
	for i, pin := range addressLinesGPIO {
		if g.mmap.ReadPin(pin) != 0 {
			v |= 1 << i
		}
	}
	return v
}

// WriteAddressBusFast drives all 5 address-bus pins from bits 0–4 of value.
// Replaces Lines.SetValues() (~1.9 µs chardev) with 5 sequential register writes.
func (g *GPIOController) WriteAddressBusFast(value uint8) {
	for i, pin := range addressLinesGPIO {
		g.mmap.SetOutput(pin, int((value>>i)&1))
	}
}

// ReadDataBusFast reads all 8 data-bus pins and packs them into a uint8.
func (g *GPIOController) ReadDataBusFast() uint8 {
	var v uint8
	for i, pin := range dataLinesGPIO {
		if g.mmap.ReadPin(pin) != 0 {
			v |= 1 << i
		}
	}
	return v
}

// WriteDataBusFast drives all 8 data-bus pins from bits 0–7 of value.
func (g *GPIOController) WriteDataBusFast(value uint8) {
	for i, pin := range dataLinesGPIO {
		g.mmap.SetOutput(pin, int((value>>i)&1))
	}
}

// SetDataBusInputFast switches all 8 data-bus pins to input mode.
// Replaces Lines.Reconfigure(AsInput) (~39 µs chardev) with 8 register writes.
func (g *GPIOController) SetDataBusInputFast() {
	for _, pin := range dataLinesGPIO {
		g.mmap.SetInput(pin)
	}
}

// SetDataBusOutputFast switches all 8 data-bus pins to output and drives value.
// Replaces Lines.Reconfigure(AsOutput) (~39 µs chardev) with 8 register writes.
func (g *GPIOController) SetDataBusOutputFast(value uint8) {
	for i, pin := range dataLinesGPIO {
		g.mmap.SetOutput(pin, int((value>>i)&1))
	}
}

// ReadResetFast reads GPIO 20 (RESB) via mmap.
func (g *GPIOController) ReadResetFast() int {
	return g.mmap.ReadPin(resbGPIO)
}

// ReadIrqFast reads GPIO 12 (IRQB) via mmap.
func (g *GPIOController) ReadIrqFast() int {
	return g.mmap.ReadPin(irqbGPIO)
}

// WriteWeFast drives GPIO 16 (WENB / write-enable) to value.
func (g *GPIOController) WriteWeFast(value int) {
	g.mmap.SetOutput(weGPIO, value)
}

// WriteMiacsFast drives GPIO 21 (MIA chip select) to value.
func (g *GPIOController) WriteMiacsFast(value int) {
	g.mmap.SetOutput(miaCSGPIO, value)
}

// WriteResetReqFast drives GPIO 26 (reset request) to value.
func (g *GPIOController) WriteResetReqFast(value int) {
	g.mmap.SetOutput(resetRequestGPIO, value)
}

/*******************************************************************************************
* Original chardev accessors — kept for non-hot-path use and graceful fallback.
********************************************************************************************/

func (g *GPIOController) AddressBus() *gpiocdev.Lines {
	return g.addressBus
}

func (g *GPIOController) DataBus() *gpiocdev.Lines {
	return g.dataBus
}

func (g *GPIOController) MiaCS() *gpiocdev.Line {
	return g.miaCS
}

func (g *GPIOController) Reset() *gpiocdev.Line {
	return g.reset
}

func (g *GPIOController) ResetRequest() *gpiocdev.Line {
	return g.resetRequest
}

func (g *GPIOController) WriteEnable() *gpiocdev.Line {
	return g.writeEnable
}

func (g *GPIOController) Irq() *gpiocdev.Line {
	return g.irq
}

func (g *GPIOController) Phi2() *gpiocdev.Line {
	return g.phi2
}

func (g *GPIOController) Close() {
	if g.mmap != nil {
		_ = g.mmap.close()
	}
	g.addressBus.Close()
	g.dataBus.Close()
	g.miaCS.Close()
	g.reset.Close()
	g.resetRequest.Close()
	g.writeEnable.Close()
	g.irq.Close()
	g.phi2.Close()
}
