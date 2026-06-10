//go:build (linux && arm) || (linux && arm64)

package common

// Direct register access for Raspberry Pi 5 RP1 GPIO (IO_BANK0).
//
// The RP1 GPIO IO_BANK0 is memory-mapped via /dev/gpiomem0.
// Each of the 28 GPIO pins has two consecutive 32-bit registers:
//
//	GPIO_STATUS[n]  at byte offset n*8     (read-only)
//	GPIO_CTRL[n]    at byte offset n*8+4   (read/write)
//
// Register field reference — RP1-TRM Rev 1.0, Section 9 (GPIO):
//
//	STATUS bit 17: INFROMPAD — current logic level driven by the pad.
//	  Always readable regardless of pin direction or function.
//
//	CTRL bits [4:0]:  FUNCSEL — peripheral function (5 = SIO/GPIO).
//	CTRL bits [9:8]:  OUTOVER — output override:
//	  0 = normal (follow FUNCSEL output), 2 = force low, 3 = force high.
//	CTRL bits [11:10]: OEOVER  — output-enable override:
//	  0 = normal (follow FUNCSEL OE), 2 = force OE=0 (input), 3 = force OE=1 (output).
//
// These positions match the RP2040 GPIO_CTRL layout (RP1 is register-compatible).
// If a pin behaves incorrectly the most likely cause is OUTOVER/OEOVER at wrong
// positions; compare the raw CTRL value against your RP1-TRM revision to verify.

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	// rp1GPIOMapSize is the number of bytes to map from /dev/gpiomem0.
	// 28 GPIO pins × 8 bytes = 224 bytes, but we round up to one page (4 KB)
	// for mmap alignment. The STATUS register for GPIO 27 is at offset 220 (0xDC),
	// which fits comfortably within a single page.
	rp1GPIOMapSize = 0x1000 // 4 KB

	// RP1 GPIO_CTRL register field masks and shifts (RP1-TRM Rev 1.0, Section 9).
	rp1FuncselShift = 0
	rp1FuncselMask  = uint32(0x1F) // bits [4:0]

	rp1OutoverShift = 8
	rp1OutoverMask  = uint32(0x3) << rp1OutoverShift // bits [9:8]
	rp1OutoverLow   = uint32(2) << rp1OutoverShift    // force pad low
	rp1OutoverHigh  = uint32(3) << rp1OutoverShift    // force pad high

	rp1OeoverShift  = 10
	rp1OeoverMask   = uint32(0x3) << rp1OeoverShift // bits [11:10]
	rp1OeoverInput  = uint32(2) << rp1OeoverShift    // force OE=0  (input)
	rp1OeoverOutput = uint32(3) << rp1OeoverShift    // force OE=1  (output)

	// RP1 GPIO_STATUS register: bit 17 = INFROMPAD.
	rp1InFromPadBit  = 17
	rp1InFromPadMask = uint32(1) << rp1InFromPadBit

	// Mask that clears the OUTOVER and OEOVER fields together.
	rp1OverrideMask = rp1OutoverMask | rp1OeoverMask
)

// rp1MmapGPIO provides direct register access to RP1 IO_BANK0 GPIO via /dev/gpiomem0.
// All hot-path operations are single 32-bit loads or stores with no system calls.
//
// For each pin, three CTRL values are precomputed at init time (InitPin) so that
// hot-path writes are a single store: pinCtrlInput, pinCtrlLow, pinCtrlHigh.
type rp1MmapGPIO struct {
	mem          []byte
	pinCtrlInput [28]uint32
	pinCtrlLow   [28]uint32
	pinCtrlHigh  [28]uint32
}

// newRp1MmapGPIO opens /dev/gpiomem0 and maps the RP1 IO_BANK0 register page.
// The caller must ensure that any gpiocdev (chardev) setup for the same pins is
// completed before calling InitPin, so that FUNCSEL is already programmed by the kernel.
func newRp1MmapGPIO() (*rp1MmapGPIO, error) {
	f, err := os.OpenFile("/dev/gpiomem0", os.O_RDWR|os.O_SYNC, 0)
	if err != nil {
		return nil, fmt.Errorf("open /dev/gpiomem0: %w (user must be in the gpio group)", err)
	}
	defer f.Close()

	mem, err := syscall.Mmap(
		int(f.Fd()), 0, rp1GPIOMapSize,
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED,
	)
	if err != nil {
		return nil, fmt.Errorf("mmap /dev/gpiomem0: %w", err)
	}

	return &rp1MmapGPIO{mem: mem}, nil
}

func (m *rp1MmapGPIO) close() error {
	return syscall.Munmap(m.mem)
}

// statusReg returns a pointer to GPIO_STATUS[pin].
func (m *rp1MmapGPIO) statusReg(pin int) *uint32 {
	return (*uint32)(unsafe.Pointer(&m.mem[pin*8]))
}

// ctrlReg returns a pointer to GPIO_CTRL[pin].
func (m *rp1MmapGPIO) ctrlReg(pin int) *uint32 {
	return (*uint32)(unsafe.Pointer(&m.mem[pin*8+4]))
}

// InitPin reads the current CTRL value set by the kernel (via gpiocdev) and uses it to
// precompute the three CTRL variants needed in the hot path. Call this once per pin
// after the gpiocdev RequestLine / RequestLines setup has completed.
//
// isOutput and initialValue set the initial hardware state.
func (m *rp1MmapGPIO) InitPin(pin int, isOutput bool, initialValue int) {
	// Preserve FUNCSEL (set by gpiocdev) and clear the override fields we control.
	base := *m.ctrlReg(pin) & ^rp1OverrideMask

	m.pinCtrlInput[pin] = base | rp1OeoverInput
	m.pinCtrlLow[pin] = base | rp1OeoverOutput | rp1OutoverLow
	m.pinCtrlHigh[pin] = base | rp1OeoverOutput | rp1OutoverHigh

	if isOutput {
		m.SetOutput(pin, initialValue)
	} else {
		m.SetInput(pin)
	}
}

// ReadPin returns the current logic level (0 or 1) driven on the pad.
// Reads STATUS.INFROMPAD (bit 17), which is valid regardless of pin direction.
func (m *rp1MmapGPIO) ReadPin(pin int) int {
	return int((*m.statusReg(pin) >> rp1InFromPadBit) & 1)
}

// SetOutput drives an output pin to the given value (0 or 1) using a single register write.
func (m *rp1MmapGPIO) SetOutput(pin, value int) {
	if value != 0 {
		*m.ctrlReg(pin) = m.pinCtrlHigh[pin]
	} else {
		*m.ctrlReg(pin) = m.pinCtrlLow[pin]
	}
}

// SetInput switches a pin to input mode using a single register write.
func (m *rp1MmapGPIO) SetInput(pin int) {
	*m.ctrlReg(pin) = m.pinCtrlInput[pin]
}
