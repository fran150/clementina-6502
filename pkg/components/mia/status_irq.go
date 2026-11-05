package mia

// status returns the 16-bit MIA status word. Like the firmware, the word lives
// outside the register window: MIA_STATUS ($FFEA-$FFEB) is only a copy, written
// in normal mode, because in loader mode those bytes hold the loader's parking
// loop. Console output reads the word, never the register.
func (c *emulated_mia) status() uint16 {
	return c.statusWord
}

// setStatus stores the status word and copies it into MIA_STATUS when MIA is in
// normal mode (mirrors mia_status_store, which tests the MASTER_MODE bit that
// is set together with the normal state).
func (c *emulated_mia) setStatus(value uint16) {
	c.statusWord = value
	if c.state == miaStateNormal {
		c.writeRegisterWord(miaRegStatusLSB, value)
	}
}

// currentError returns the error value the 6502 reads at MIA_ERROR: the queue
// head, or zero (mirrors error_current).
func (c *emulated_mia) currentError() uint16 {
	return c.errorWord
}

// setCurrentError stores the error value and copies it into MIA_ERROR when MIA
// is in normal mode; in loader mode $FFEC belongs to the loader program
// (mirrors error_set_current).
func (c *emulated_mia) setCurrentError(value uint16) {
	c.errorWord = value
	if c.state == miaStateNormal {
		c.writeRegisterWord(miaRegErrorLSB, value)
	}
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
	c.setIRQStatus(0x0000)
	c.irqAsserted = false
}

// irqEval updates IRQ_TRIGGERED and the emulated IRQ output from status and mask bits.
func (c *emulated_mia) irqEval() {
	status := c.irqStatus()
	if status&c.irqMask()&^miaIRQTriggered != 0 {
		c.setIRQStatus(status | miaIRQTriggered)
		c.irqAsserted = true
	} else {
		c.setIRQStatus(status &^ miaIRQTriggered)
		c.irqAsserted = false
	}
}

// irqSetFlag sets an IRQ status bit and re-evaluates the IRQ output.
func (c *emulated_mia) irqSetFlag(flag uint16) {
	c.setIRQStatus(c.irqStatus() | flag)
	c.irqEval()
}

// irqClearStatus acknowledges all latched IRQ sources and deasserts the output.
func (c *emulated_mia) irqClearStatus() {
	c.setIRQStatus(0x0000)
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

// Push appends an error code to the MIA error queue. If the queue is full, the
// oldest unread entry is discarded and the queued value becomes
// ERROR_QUEUE_OVERFLOW. This keeps producers non-blocking while making loss
// visible to the 6502, matching the firmware error_push semantics.
func (q *miaErrorQueue) Push(chip *emulated_mia, value uint8) {
	next := (q.last + 1) & 0x0F
	wasEmpty := q.first == q.last
	overwroteHead := false

	if next == q.first {
		q.first = (q.first + 1) & 0x0F
		value = miaErrorQueueOverflow
		overwroteHead = true
	}

	q.buf[q.last] = value
	q.last = next
	chip.statusSet(miaStatusErrors)
	chip.irqSetFlag(miaIRQError)

	if wasEmpty {
		chip.setCurrentError(uint16(value))
	} else if overwroteHead {
		chip.setCurrentError(uint16(q.buf[q.first]))
	}
}

// reset clears the error queue, the CPU-visible ERROR register, and the error
// status bit. It mirrors the firmware error_reset and backs 'errors clear'.
func (q *miaErrorQueue) reset(chip *emulated_mia) {
	*q = miaErrorQueue{}
	chip.setCurrentError(0)
	chip.statusClear(miaStatusErrors)
}

// Consume advances after the CPU reads the visible error register and preloads
// the next queued error, or zero when the queue drains.
func (q *miaErrorQueue) Consume(chip *emulated_mia) uint8 {
	if q.first != q.last {
		q.first = (q.first + 1) & 0x0F
	}

	if q.first == q.last {
		chip.statusClear(miaStatusErrors)
		chip.setCurrentError(0)
		return 0
	}

	next := q.buf[q.first]
	chip.setCurrentError(uint16(next))

	return next
}
