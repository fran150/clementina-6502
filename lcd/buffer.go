package lcd

type LcdBuffer struct {
	is8BitMode bool
	value      uint8
	index      uint8
}

func createLcdBuffer() *LcdBuffer {
	return &LcdBuffer{
		is8BitMode: true,
		value:      0,
		index:      0,
	}
}

func (buf *LcdBuffer) push(value uint8) {
	if !buf.isFull() {
		if !buf.is8BitMode {
			if buf.index == 0 {
				value &= 0xF0

				buf.value &= 0x0F
				buf.value |= value
			} else {
				value &= 0xF0
				value = value >> 4

				buf.value &= 0xF0
				buf.value |= value
			}

			buf.index++
		} else {
			buf.index = 2
			buf.value = value
		}
	}
}

func (buf *LcdBuffer) pull() uint8 {
	if !buf.isEmpty() {
		if !buf.is8BitMode {
			buf.index--

			if buf.index == 0 {
				value := buf.value & 0x0F
				return value << 4
			} else {
				return buf.value & 0xF0
			}
		} else {
			buf.index = 0
			return buf.value
		}
	}

	return 0
}

func (buf *LcdBuffer) load(value uint8) {
	buf.value = value
	buf.index = 2
}

func (buf *LcdBuffer) flush() {
	buf.value = 0
	buf.index = 0
}

func (buf *LcdBuffer) isFull() bool {
	return buf.index == 2
}

func (buf *LcdBuffer) isEmpty() bool {
	return buf.index == 0
}
