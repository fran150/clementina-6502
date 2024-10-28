package lcd

type LcdBuffer struct {
	value [2]uint8
	index uint8
}

func (buf *LcdBuffer) push(value uint8) {
	if !buf.isFull() {
		buf.index++
	}

	buf.value[buf.index-1] = value
}

func (buf *LcdBuffer) pull() uint8 {
	if !buf.isEmpty() {
		buf.index--
	}

	return buf.value[buf.index-1]
}

func (buf *LcdBuffer) isFull() bool {
	return buf.index == 2
}

func (buf *LcdBuffer) isEmpty() bool {
	return buf.index == 0
}
