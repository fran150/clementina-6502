package mia

import (
	"fmt"
	"os"
	"sync"

	"github.com/fran150/clementina-6502/pkg/common"
	"github.com/fran150/clementina-6502/pkg/components"
	"github.com/fran150/clementina-6502/pkg/components/buses"
)

var kernelSync sync.Once

type emulated_mia struct {
	mu sync.Mutex

	addressBus *buses.BusConnector[uint8]
	dataBus    *buses.BusConnector[uint8]

	miaCS        buses.LineConnector
	reset        buses.LineConnector
	resetRequest buses.LineConnector
	writeEnable  buses.LineConnector
	irq          buses.LineConnector

	registers [miaRegisterCount]uint8
	memory    []uint8
	indexes   [miaIndexCount]miaIndex
	errors    miaErrorQueue

	state                  miaState
	kernelIndex            uint32
	kernelTargetAddress    uint16
	canUpdateKernelPointer bool
	irqAsserted            bool
	cpuResetCycles         uint8
	resetRequestAsserted   bool
	resetReleasePending    bool
	stagedPhi2Hz           uint32
	requestedPhi2Hz        uint32
	appliedPhi2Hz          uint32
	speedChangeRequested   bool

	video miaVideoState
}

// NewEmulatedMia creates a software implementation of the Clementina MIA chip.
func NewEmulatedMia() components.MiaChip {
	kernelSync.Do(func() {
		if len(miaKernelData) == 0 {
			data, err := os.ReadFile("assets/computer/mia/kernel.bin")
			if err != nil {
				panic(fmt.Sprintf("mia: failed to load kernel.bin: %v. Ensure the file exists at the correct path.", err))
			}
			miaKernelData = data
		}
	})

	chip := &emulated_mia{
		addressBus:   buses.NewBusConnector[uint8](),
		dataBus:      buses.NewBusConnector[uint8](),
		miaCS:        buses.NewConnectorEnabledHigh(),
		reset:        buses.NewConnectorEnabledLow(),
		resetRequest: buses.NewConnectorEnabledLow(),
		writeEnable:  buses.NewConnectorEnabledLow(),
		irq:          buses.NewConnectorEnabledLow(),
		memory:       make([]uint8, miaRAMSize),

		state:               miaStateLoader,
		kernelTargetAddress: miaKernelTargetAddress,
		appliedPhi2Hz:       miaDefaultPhi2Hz,
	}

	chip.init()

	return chip
}

// NewEmulatedMiaWithVideoUDP creates an emulated MIA and starts its UDP video service.
func NewEmulatedMiaWithVideoUDP(bindAddress string) (components.MiaChip, error) {
	chip := NewEmulatedMia().(*emulated_mia)
	if err := chip.StartVideoUDP(bindAddress); err != nil {
		return nil, err
	}

	return chip, nil
}

// AddressBus returns the 5-bit MIA register address bus connector.
func (c *emulated_mia) AddressBus() *buses.BusConnector[uint8] {
	return c.addressBus
}

// DataBus returns the MIA data bus connector.
func (c *emulated_mia) DataBus() *buses.BusConnector[uint8] {
	return c.dataBus
}

// MiaCS returns the active-high MIA chip select connector.
func (c *emulated_mia) MiaCS() buses.LineConnector {
	return c.miaCS
}

// Reset returns the active-low CPU reset line driven by the MIA.
func (c *emulated_mia) Reset() buses.LineConnector {
	return c.reset
}

// ResetRequest returns the active-low input that asks MIA to reset the computer.
func (c *emulated_mia) ResetRequest() buses.LineConnector {
	return c.resetRequest
}

// WriteEnable returns the active-low write enable connector.
func (c *emulated_mia) WriteEnable() buses.LineConnector {
	return c.writeEnable
}

// Irq returns the active-low IRQ connector driven by the MIA.
func (c *emulated_mia) Irq() buses.LineConnector {
	return c.irq
}

// Peek returns a side-effect-free byte from the MIA register window.
func (c *emulated_mia) Peek(address uint16) uint8 {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.readRegister(uint8(address))
}

// Tick processes one bus cycle against the MIA register window.
func (c *emulated_mia) Tick(context *common.StepContext) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.handleResetRequest() {
		c.driveIRQLine()
		return
	}

	c.speedService()

	if !c.miaCS.Enabled() {
		c.driveIRQLine()
		return
	}

	address := c.addressBus.Read() & miaRegisterMask

	if c.writeEnable.Enabled() {
		data := c.dataBus.Read()
		c.writeRegister(address, data)
		c.afterWrite(address, data)
	} else {
		data := c.readRegister(address)
		c.dataBus.Write(data)
		c.afterRead(address)
	}

	c.driveIRQLine()
}

// PostTick completes the MIA cycle. The emulated MIA resolves its cycle during Tick.
func (c *emulated_mia) PostTick(context *common.StepContext) {}

// handleResetRequest applies the MIA-controlled system reset behavior.
func (c *emulated_mia) handleResetRequest() bool {
	resetRequested := c.resetRequest.Enabled()

	if resetRequested && !c.resetRequestAsserted {
		c.resetRequestAsserted = true
		c.resetReleasePending = false
		c.cpuResetCycles = 0
		c.init()
	}

	if resetRequested {
		c.driveResetLine(true)
		return true
	}

	if c.resetRequestAsserted {
		c.resetRequestAsserted = false
		c.scheduleCPUResetRelease()
	}

	if c.resetReleasePending && c.cpuResetCycles > 0 {
		c.cpuResetCycles--
	}

	if c.resetReleasePending && c.cpuResetCycles == 0 {
		c.resetReleasePending = false
	}

	cpuResetAsserted := c.resetReleasePending
	c.driveResetLine(cpuResetAsserted)

	return cpuResetAsserted
}

// scheduleCPUResetRelease keeps RESB asserted for the configured reset pulse length.
func (c *emulated_mia) scheduleCPUResetRelease() {
	c.resetReleasePending = true
	c.cpuResetCycles = miaCPUResetPulseCycles
}

// driveResetLine writes MIA's active-low CPU reset output.
func (c *emulated_mia) driveResetLine(asserted bool) {
	if c.reset.GetLine() == nil {
		return
	}

	c.reset.SetEnable(asserted)
}

// init initializes MIA internal state to match the Pico firmware startup path.
func (c *emulated_mia) init() {
	c.registers = [miaRegisterCount]uint8{}
	c.indexes = [miaIndexCount]miaIndex{}
	c.errors = miaErrorQueue{}
	c.state = miaStateLoader
	c.canUpdateKernelPointer = false
	c.irqAsserted = false
	c.resetReleasePending = false
	c.cpuResetCycles = 0

	c.irqInit()
	c.speedResetRuntimeState()
	c.videoResetRuntimeState()
	c.fastLoaderInit()
}

// readRegister returns a byte from the 32-byte MIA register window.
func (c *emulated_mia) readRegister(address uint8) uint8 {
	return c.registers[address&miaRegisterMask]
}

// writeRegister stores a byte in the 32-byte MIA register window.
func (c *emulated_mia) writeRegister(address uint8, value uint8) {
	c.registers[address&miaRegisterMask] = value
}

// readRegisterWord returns a little-endian 16-bit value from adjacent registers.
func (c *emulated_mia) readRegisterWord(address uint8) uint16 {
	lsb := uint16(c.readRegister(address))
	msb := uint16(c.readRegister(address + 1))

	return lsb | (msb << 8)
}

// writeRegisterWord stores a little-endian 16-bit value in adjacent registers.
func (c *emulated_mia) writeRegisterWord(address uint8, value uint16) {
	c.writeRegister(address, uint8(value))
	c.writeRegister(address+1, uint8(value>>8))
}

// afterRead applies firmware side effects that happen after a CPU register read.
func (c *emulated_mia) afterRead(address uint8) {
	switch c.state {
	case miaStateLoader:
		if address == miaRegIdxASelector {
			c.canUpdateKernelPointer = true
		}
	case miaStateNormal:
		switch address {
		case miaRegIdxAPort:
			selector := c.readRegister(miaRegIdxASelector)
			c.writeRegister(miaRegIdxAPort, c.indexStepAndRead(selector, miaIndexWindowA))
		case miaRegIdxBPort:
			selector := c.readRegister(miaRegIdxBSelector)
			c.writeRegister(miaRegIdxBPort, c.indexStepAndRead(selector, miaIndexWindowB))
		case miaRegErrorLSB:
			c.writeRegisterWord(miaRegErrorLSB, uint16(c.errors.Pull(c)))
		}
	}
}

// afterWrite applies firmware side effects that happen after a CPU register write.
func (c *emulated_mia) afterWrite(address uint8, data uint8) {
	switch c.state {
	case miaStateLoader:
		if address == miaRegIRQStatusMSB {
			c.advanceKernelLoader()
		}
	case miaStateNormal:
		c.afterNormalWrite(address, data)
	}
}

// afterNormalWrite dispatches normal-mode register write side effects.
func (c *emulated_mia) afterNormalWrite(address uint8, data uint8) {
	switch address {
	case miaRegIdxAPort:
		selector := c.readRegister(miaRegIdxASelector)
		c.indexWriteAndStep(selector, data, miaIndexWindowA)
		c.writeRegister(miaRegIdxAPort, c.indexRead(selector))
	case miaRegIdxASelector:
		c.writeRegister(miaRegIdxAPort, c.indexRead(data))
	case miaRegCfgSelector:
		c.writeRegister(miaRegCfgPort, c.getCfg(data))
	case miaRegCfgPort:
		c.setCfg(c.readRegister(miaRegCfgSelector), data)
	case miaRegIdxBPort:
		selector := c.readRegister(miaRegIdxBSelector)
		c.indexWriteAndStep(selector, data, miaIndexWindowB)
		c.writeRegister(miaRegIdxBPort, c.indexRead(selector))
	case miaRegIdxBSelector:
		c.writeRegister(miaRegIdxBPort, c.indexRead(data))
	case miaRegCmdTrigger:
		c.executeCommand(c.readRegister(miaRegCmdTrigger), [3]uint8{
			c.readRegister(miaRegCmdParam1),
			c.readRegister(miaRegCmdParam2),
			c.readRegister(miaRegCmdParam3),
		})
	case miaRegIRQMaskLSB, miaRegIRQMaskMSB, miaRegIRQStatusLSB, miaRegIRQStatusMSB:
		c.irqEval()
	}
}

// advanceKernelLoader moves the boot-loader data register to the next kernel byte.
func (c *emulated_mia) advanceKernelLoader() {
	if !c.canUpdateKernelPointer {
		return
	}

	if c.kernelIndex < uint32(len(miaKernelData)) {
		c.writeRegister(miaRegIdxASelector, miaKernelData[c.kernelIndex])
		c.kernelIndex++
		c.writeRegisterWord(0x03, c.readRegisterWord(0x03)+1)
	} else {
		c.writeRegister(miaRegCmdTrigger, 0x00)
		c.enterNormalMode()
	}

	c.canUpdateKernelPointer = false
}

// enterNormalMode replaces the loader register window with normal MIA registers.
func (c *emulated_mia) enterNormalMode() {
	c.driveResetLine(true)

	c.registers = [miaRegisterCount]uint8{}
	c.errors = miaErrorQueue{}

	c.irqInit()
	c.speedResetRuntimeState()

	c.writeRegisterWord(miaRegResetVectorLSB, c.kernelTargetAddress)

	c.state = miaStateNormal
	c.statusSet(miaStatusMasterMode)
	c.scheduleCPUResetRelease()
}

// fastLoaderInit seeds the MIA register window with the Pico fast-loader program.
func (c *emulated_mia) fastLoaderInit() {
	c.kernelIndex = 0

	c.writeRegister(miaRegIdxAPort, 0xA9)
	if len(miaKernelData) > 0 {
		c.writeRegister(miaRegIdxASelector, miaKernelData[c.kernelIndex])
		c.kernelIndex++
	}

	c.writeRegister(0x02, 0x8D)
	c.writeRegister(0x03, uint8(c.kernelTargetAddress))
	c.writeRegister(miaRegIdxBPort, uint8(c.kernelTargetAddress>>8))

	c.writeRegister(miaRegIdxBSelector, 0x8D)
	c.writeRegister(miaRegCmdParam1, 0xF1)
	c.writeRegister(miaRegCmdParam2, 0xFF)

	c.writeRegister(miaRegCmdParam3, 0x80)
	c.writeRegister(miaRegCmdTrigger, 0xF6)

	c.writeRegister(miaRegStatusLSB, 0x4C)
	c.writeRegister(miaRegStatusMSB, 0xEA)
	c.writeRegister(miaRegErrorLSB, 0xFF)

	c.writeRegister(miaRegResetVectorLSB, 0xE0)
	c.writeRegister(miaRegResetVectorMSB, 0xFF)
}
