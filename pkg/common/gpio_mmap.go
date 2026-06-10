//go:build (linux && arm) || (linux && arm64)

package common

// Direct register access for Raspberry Pi 5 RP1 GPIO (IO_BANK0).
//
// The RP1 GPIO IO_BANK0 is memory-mapped via /dev/gpiomem0.
// Each of the 28 GPIO pins has two consecutive 32-bit registers:
//
//	GPIO_STATUS[n]  at byte offset n*8     (read-only hardware)
//	GPIO_CTRL[n]    at byte offset n*8+4   (read/write)
//
// Register field reference — RP1-TRM Rev 1.0, Section 9 (GPIO):
//
//	STATUS bit 17: INFROMPAD — current logic level driven by the pad.
//	  Always readable regardless of pin direction or function.
//
//	CTRL bits [4:0]:   FUNCSEL — peripheral function (5 = SIO/GPIO).
//	CTRL bits [9:8]:   OUTOVER — output override:
//	  0 = normal (follow FUNCSEL output), 2 = force low, 3 = force high.
//	CTRL bits [11:10]: OEOVER  — output-enable override:
//	  0 = normal (follow FUNCSEL OE), 2 = force OE=0 (input), 3 = force OE=1 (output).
//
// These positions match the RP2040 GPIO_CTRL layout (RP1 is register-compatible).
// If a pin reads or drives incorrectly compare the raw CTRL value against the RP1-TRM.

import (
	"fmt"
	"log"
	"os"
	"syscall"
	"unsafe"
)

const (
	rp1GPIOMapSize = 0x1000 // 4 KB — covers all 28 GPIO pairs (224 bytes used)

	rp1OutoverShift = 8
	rp1OutoverMask  = uint32(0x3) << rp1OutoverShift
	rp1OutoverLow   = uint32(2) << rp1OutoverShift // force pad low
	rp1OutoverHigh  = uint32(3) << rp1OutoverShift // force pad high

	rp1OeoverShift  = 10
	rp1OeoverMask   = uint32(0x3) << rp1OeoverShift
	rp1OeoverInput  = uint32(2) << rp1OeoverShift // force OE=0 (input)
	rp1OeoverOutput = uint32(3) << rp1OeoverShift // force OE=1 (output)

	rp1InFromPadBit = 17 // STATUS bit: current pad level (valid regardless of direction)

	rp1OverrideMask = rp1OutoverMask | rp1OeoverMask
)

// rp1MmapGPIO provides direct register access to RP1 IO_BANK0 GPIO.
// When readOnly is true (PROT_READ-only mapping) write methods are no-ops;
// the caller must use chardev for any output operations.
type rp1MmapGPIO struct {
	mem      []byte
	readOnly bool

	pinCtrlInput [28]uint32
	pinCtrlLow   [28]uint32
	pinCtrlHigh  [28]uint32
}

// newRp1MmapGPIO maps /dev/gpiomem0. It tries read-write first (needed for fast
// output); if the kernel rejects the writable mapping it falls back to read-only,
// which still gives fast PHI2 polling and input sampling.
func newRp1MmapGPIO() (*rp1MmapGPIO, error) {
	// Attempt read-write.
	if m, err := tryMmap(os.O_RDWR, syscall.PROT_READ|syscall.PROT_WRITE); err == nil {
		return &rp1MmapGPIO{mem: m, readOnly: false}, nil
	}

	// Fall back to read-only — still useful for PHI2 and input sampling.
	m, err := tryMmap(os.O_RDONLY, syscall.PROT_READ)
	if err != nil {
		return nil, fmt.Errorf("mmap /dev/gpiomem0: %w", err)
	}
	log.Print("gpio: /dev/gpiomem0 mapped read-only; PHI2 and inputs are fast, outputs use chardev")
	return &rp1MmapGPIO{mem: m, readOnly: true}, nil
}

func tryMmap(openFlag int, prot int) ([]byte, error) {
	f, err := os.OpenFile("/dev/gpiomem0", openFlag, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return syscall.Mmap(int(f.Fd()), 0, rp1GPIOMapSize, prot, syscall.MAP_SHARED)
}

func (m *rp1MmapGPIO) close() error {
	return syscall.Munmap(m.mem)
}

// Writable reports whether the mapping supports register writes.
func (m *rp1MmapGPIO) Writable() bool { return !m.readOnly }

func (m *rp1MmapGPIO) statusReg(pin int) *uint32 {
	return (*uint32)(unsafe.Pointer(&m.mem[pin*8]))
}

func (m *rp1MmapGPIO) ctrlReg(pin int) *uint32 {
	return (*uint32)(unsafe.Pointer(&m.mem[pin*8+4]))
}

// InitPin precomputes the three CTRL values for this pin from the FUNCSEL already
// programmed by gpiocdev. Call after gpiocdev RequestLine(s) has completed.
func (m *rp1MmapGPIO) InitPin(pin int, isOutput bool, initialValue int) {
	base := *m.ctrlReg(pin) & ^rp1OverrideMask
	m.pinCtrlInput[pin] = base | rp1OeoverInput
	m.pinCtrlLow[pin] = base | rp1OeoverOutput | rp1OutoverLow
	m.pinCtrlHigh[pin] = base | rp1OeoverOutput | rp1OutoverHigh

	if !m.readOnly {
		if isOutput {
			m.SetOutput(pin, initialValue)
		} else {
			m.SetInput(pin)
		}
	}
}

// ReadPin returns the current pad level (0 or 1) via STATUS.INFROMPAD.
func (m *rp1MmapGPIO) ReadPin(pin int) int {
	return int((*m.statusReg(pin) >> rp1InFromPadBit) & 1)
}

// SetOutput drives an output pin. No-op when the mapping is read-only.
func (m *rp1MmapGPIO) SetOutput(pin, value int) {
	if m.readOnly {
		return
	}
	if value != 0 {
		*m.ctrlReg(pin) = m.pinCtrlHigh[pin]
	} else {
		*m.ctrlReg(pin) = m.pinCtrlLow[pin]
	}
}

// SetInput switches a pin to input mode. No-op when the mapping is read-only.
func (m *rp1MmapGPIO) SetInput(pin int) {
	if m.readOnly {
		return
	}
	*m.ctrlReg(pin) = m.pinCtrlInput[pin]
}
