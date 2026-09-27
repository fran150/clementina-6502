package mia

// MIA_CTX ($FFF5) values, mirroring regs.h. PUSH saves the state an interrupt
// handler would clobber - both window selections, the CFG selection, the three
// command parameters and the two descriptors bound to the windows - on a stack
// miaCtxDepth deep; POP restores it and reloads the ports, as select writes do.
// Other values do nothing. Reads return the last value written.
const (
	miaCtxPush  uint8 = 0x01
	miaCtxPop   uint8 = 0x02
	miaCtxDepth       = 4
)

// miaContext is one MIA_CTX stack entry, mirroring mia_ctx_t.
type miaContext struct {
	idxASelector uint8
	idxBSelector uint8
	cfgSelector  uint8
	cmdParams    [3]uint8
	idxA         miaIndex
	idxB         miaIndex
}

// contextWrite mirrors the firmware's MIA_CTX write handler on core 1.
func (c *emulated_mia) contextWrite(value uint8) {
	switch value {
	case miaCtxPush:
		c.contextPush()
	case miaCtxPop:
		c.contextPop()
	}
}

// contextPush mirrors mia_core1_ctx_push. The firmware defers the overflow
// error to core 0; the emulator queues it at once.
func (c *emulated_mia) contextPush() {
	if c.ctxDepth == miaCtxDepth {
		c.errors.Push(c, miaErrorCtxOverflow)
		return
	}

	ctx := &c.ctxStack[c.ctxDepth]
	c.ctxDepth++
	ctx.idxASelector = c.readRegister(miaRegIdxASelector)
	ctx.idxBSelector = c.readRegister(miaRegIdxBSelector)
	ctx.cfgSelector = c.readRegister(miaRegCfgSelector)
	ctx.cmdParams = [3]uint8{
		c.readRegister(miaRegCmdParam1),
		c.readRegister(miaRegCmdParam2),
		c.readRegister(miaRegCmdParam3),
	}
	ctx.idxA = c.indexes[ctx.idxASelector]
	ctx.idxB = c.indexes[ctx.idxBSelector]
}

// contextPop mirrors mia_core1_ctx_pop.
func (c *emulated_mia) contextPop() {
	if c.ctxDepth == 0 {
		c.errors.Push(c, miaErrorCtxUnderflow)
		return
	}

	c.ctxDepth--
	ctx := &c.ctxStack[c.ctxDepth]
	c.indexes[ctx.idxASelector] = ctx.idxA
	c.indexes[ctx.idxBSelector] = ctx.idxB
	c.writeRegister(miaRegIdxASelector, ctx.idxASelector)
	c.writeRegister(miaRegIdxBSelector, ctx.idxBSelector)
	c.writeRegister(miaRegCfgSelector, ctx.cfgSelector)
	c.writeRegister(miaRegCmdParam1, ctx.cmdParams[0])
	c.writeRegister(miaRegCmdParam2, ctx.cmdParams[1])
	c.writeRegister(miaRegCmdParam3, ctx.cmdParams[2])

	// Reload the ports as select writes do. CFG fields $00-$0F act on the
	// descriptor in window A, so window A is restored first.
	c.writeRegister(miaRegIdxAPort, c.indexRead(ctx.idxASelector))
	c.writeRegister(miaRegIdxBPort, c.indexRead(ctx.idxBSelector))
	c.writeRegister(miaRegCfgPort, c.getCfg(ctx.cfgSelector))
}
