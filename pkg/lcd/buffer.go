package lcd

const UPPER_NIBBLE_MASK uint8 = 0xF0 // Mask to get the upper four bits of a byte
const LOWER_NIBBLE_MASK uint8 = 0x0F // Mask to get the lower four bits of a byte
const BUFFER_FULL_INDEX uint8 = 2    // Index when buffer is full
const BUFFER_EMPTY_INDEX uint8 = 0   // Index when buffer is empty

/*For 4-bit interface data, only four bus lines (DB4 to DB7) are used for transfer. Bus lines DB0 to DB3
are disabled. The data transfer between the HD44780U and the MPU is completed after the 4-bit data
has been transferred twice. As for the order of data transfer, the four high order bits (for 8-bit operation,
DB4 to DB7) are transferred before the four low order bits (for 8-bit operation, DB0 to DB3).
The busy flag must be checked (one instruction) after the 4-bit data has been transferred twice. Two
more 4-bit operations then transfer the busy flag and address counter data.*/

type LcdBuffer struct {
	is8BitMode bool
	value      uint8
	index      uint8
}

func createLcdBuffer() *LcdBuffer {
	return &LcdBuffer{
		is8BitMode: true,
		value:      0x00,
		index:      BUFFER_EMPTY_INDEX,
	}
}

// Writes the specified value on the most significant nibble of the buffer
func (buf *LcdBuffer) writeMSNibble(value uint8) {
	// Clear the new value's lower nibble
	value &= UPPER_NIBBLE_MASK

	// Clear the current value's upper nibble and assign
	buf.value &= LOWER_NIBBLE_MASK
	buf.value |= value

}

// Reads the current value's upper nibble
func (buf *LcdBuffer) readMSNibble() uint8 {
	return buf.value & UPPER_NIBBLE_MASK
}

// Writes the specified value on the least significant nibble of the buffer
func (buf *LcdBuffer) writeLSNibble(value uint8) {
	// Clear the new value's lower nibble and move it to the upper
	value &= UPPER_NIBBLE_MASK
	value = value >> 4

	// Clear the current value's lower nibble and assign
	buf.value &= UPPER_NIBBLE_MASK
	buf.value |= value
}

// Reads the current value lower nibble
func (buf *LcdBuffer) readLSNibble() uint8 {
	// Clear current value's upper nibble and move the lower nibble up
	value := buf.value & LOWER_NIBBLE_MASK
	return value << 4
}

// Pushes the specified value to the buffer.
// In 8 bit mode, this only require one operation
// In 4 bit mode, this requires 2 operations. The value is specified in the upper 4 bits.
// First the MSN and then the LSN. For example, to push 0xA4, first a 0xA0 must be sent
// and then 0x40
func (buf *LcdBuffer) push(value uint8) {
	if !buf.isFull() {
		if !buf.is8BitMode {
			if buf.index == 0 {
				buf.writeMSNibble(value)
			} else {
				buf.writeLSNibble(value)
			}

			buf.index++
		} else {
			buf.index = BUFFER_FULL_INDEX
			buf.value = value
		}
	}
}

// Pulls the current value from the buffer.
// In 8 bit mode this requires only one operation and the value return will be 8 bits.
// In 4 bit mode this require two operations and the value will be returned in the upper nibble.
// For example, if value is 0xA4, it will be read as 0xA0 and 0x40
func (buf *LcdBuffer) pull() uint8 {
	if !buf.isEmpty() {
		if !buf.is8BitMode {
			buf.index--

			if buf.index == 0 {
				return buf.readLSNibble()
			} else {
				return buf.readMSNibble()
			}
		} else {
			buf.index = BUFFER_EMPTY_INDEX
			return buf.value
		}
	}

	return 0x00
}

// Sets the specified value in the buffer, used when copying value from
// instruction or data registers
func (buf *LcdBuffer) load(value uint8) {
	buf.value = value
	buf.index = BUFFER_FULL_INDEX
}

// Clears the buffer
func (buf *LcdBuffer) flush() {
	buf.value = 0x00
	buf.index = BUFFER_EMPTY_INDEX
}

// Returns if buffer is full
func (buf *LcdBuffer) isFull() bool {
	return buf.index == BUFFER_FULL_INDEX
}

// Returns if buffer is empty
func (buf *LcdBuffer) isEmpty() bool {
	return buf.index == BUFFER_EMPTY_INDEX
}
