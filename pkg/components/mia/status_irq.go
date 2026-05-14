package mia

// status returns the 16-bit MIA status register value.
func (c *emulated_mia) status() uint16 {
	return c.readRegisterWord(miaRegStatusLSB)
}

// setStatus updates the 16-bit MIA status register value.
func (c *emulated_mia) setStatus(value uint16) {
	c.writeRegisterWord(miaRegStatusLSB, value)
}

// statusSet sets one or more bits in the MIA status register.
func (c *emulated_mia) statusSet(flag uint16) {
	c.setStatus(c.status() | flag)
}

// statusClear clears one or more bits in the MIA status register.
func (c *emulated_mia) statusClear(flag uint16) {
	c.setStatus(c.status() &^ flag)
}

// irqMask returns the 16-bit IRQ mask register value.
func (c *emulated_mia) irqMask() uint16 {
	return c.readRegisterWord(miaRegIRQMaskLSB)
}

// irqStatus returns the 16-bit IRQ status register value.
func (c *emulated_mia) irqStatus() uint16 {
	return c.readRegisterWord(miaRegIRQStatusLSB)
}

// setIRQStatus updates the 16-bit IRQ status register value.
func (c *emulated_mia) setIRQStatus(value uint16) {
	c.writeRegisterWord(miaRegIRQStatusLSB, value)
}

// irqInit initializes IRQ registers and the emulated IRQ output latch.
func (c *emulated_mia) irqInit() {
	c.writeRegisterWord(miaRegIRQMaskLSB, 0x0000)
	c.setIRQStatus(miaIRQTriggered)
	c.irqAsserted = false
}

// irqEval updates IRQ_TRIGGERED and the emulated IRQ output from status and mask bits.
func (c *emulated_mia) irqEval() {
	if c.irqStatus()&c.irqMask() != 0 {
		c.setIRQStatus(c.irqStatus() | miaIRQTriggered)
		c.irqAsserted = true
	} else {
		c.setIRQStatus(c.irqStatus() &^ miaIRQTriggered)
		c.irqAsserted = false
	}
}

// irqSetFlag sets an IRQ status bit and re-evaluates the IRQ output.
func (c *emulated_mia) irqSetFlag(flag uint16) {
	c.setIRQStatus(c.irqStatus() | flag)
	c.irqEval()
}

// driveIRQLine writes the current emulated IRQ output to the connected line.
func (c *emulated_mia) driveIRQLine() {
	if c.irq.GetLine() == nil {
		return
	}

	c.irq.SetEnable(c.irqAsserted)
}

type miaErrorQueue struct {
	first uint8
	last  uint8
	buf   [16]uint8
}

// Push appends an error code to the MIA error queue if space is available.
func (q *miaErrorQueue) Push(chip *emulated_mia, value uint8) {
	next := (q.last + 1) & 0x0F
	if next == q.first {
		return
	}

	chip.statusSet(miaStatusErrors)
	chip.irqSetFlag(miaIRQError)
	q.buf[q.last] = value
	q.last = next
}

// Pull removes and returns the oldest MIA error code, or zero if the queue is empty.
func (q *miaErrorQueue) Pull(chip *emulated_mia) uint8 {
	head := q.first
	if head == q.last {
		chip.statusClear(miaStatusErrors)
		return 0
	}

	value := q.buf[head]
	q.first = (head + 1) & 0x0F

	return value
}
