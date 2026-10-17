package mia

import "time"

// miaConsoleEscTimeout is how long a lone Esc waits for the rest of a sequence
// before it goes to the text FIFO as $1B. It matches the firmware's
// CON_KEYS_ESC_TIMEOUT_US.
const miaConsoleEscTimeout = 20 * time.Millisecond

const miaConsoleEsc = 0x1B

type miaConsoleKeysState uint8

const (
	miaConsoleKeysIdle miaConsoleKeysState = iota
	miaConsoleKeysEsc
	miaConsoleKeysCSI
	miaConsoleKeysSS3
)

// miaConsoleKeys is console input mode's key decoder, a port of the firmware's
// con_keys.c. A terminal sends its cursor, Home, Insert and Delete keys as
// escape sequences (ESC [ A, ESC O A, ESC [ 3 ~, ...); the decoder turns them
// into the key codes MIA's keyboard decode gives those keys and drops other
// sequences.
type miaConsoleKeys struct {
	state     miaConsoleKeysState
	param     uint8 // first number of an ESC [ sequence, capped at 99
	paramDone bool  // a ';' ended the first number
	escAt     time.Time
}

func (k *miaConsoleKeys) reset() {
	*k = miaConsoleKeys{}
}

// feed decodes one terminal byte received at now and returns the text FIFO
// bytes it produces (none, one or two).
func (k *miaConsoleKeys) feed(value byte, now time.Time) []byte {
	switch k.state {
	case miaConsoleKeysEsc:
		switch value {
		case '[':
			k.state = miaConsoleKeysCSI
			k.param = 0
			k.paramDone = false
			return nil
		case 'O':
			k.state = miaConsoleKeysSS3
			return nil
		}
		// Esc followed by an ordinary key (or Alt+key): both go out.
		k.state = miaConsoleKeysIdle
		return append([]byte{miaConsoleEsc}, k.idle(value, now)...)

	case miaConsoleKeysCSI:
		switch {
		case value >= '0' && value <= '9':
			if !k.paramDone && k.param < 99 {
				k.param = k.param*10 + (value - '0')
			}
			return nil
		case value >= 0x20 && value <= 0x3F:
			k.paramDone = true // ';' and other parameter bytes
			return nil
		}
		k.state = miaConsoleKeysIdle
		if value >= 0x40 && value <= 0x7E {
			return miaConsoleKeyCode(value, k.param)
		}
		// A control byte cut the sequence short: drop it, keep the byte.
		return k.idle(value, now)

	case miaConsoleKeysSS3:
		k.state = miaConsoleKeysIdle
		return miaConsoleKeyCode(value, 0)

	default:
		return k.idle(value, now)
	}
}

// expire is called when no byte arrived. Once the timeout passes, a lone Esc
// goes out as $1B and an unfinished sequence is dropped.
func (k *miaConsoleKeys) expire(now time.Time) []byte {
	if k.state == miaConsoleKeysIdle || now.Sub(k.escAt) < miaConsoleEscTimeout {
		return nil
	}
	lone := k.state == miaConsoleKeysEsc
	k.state = miaConsoleKeysIdle
	if lone {
		return []byte{miaConsoleEsc}
	}
	return nil
}

// idle handles a byte outside any sequence: Enter and Backspace as the BASIC
// editor wants them, an Esc starts a sequence, everything else passes through.
func (k *miaConsoleKeys) idle(value byte, now time.Time) []byte {
	switch value {
	case miaConsoleEsc:
		k.state = miaConsoleKeysEsc
		k.escAt = now
		return nil
	case '\n':
		value = '\r'
	case 0x7F:
		value = '\b' // terminals send Backspace as DEL
	}
	return []byte{value}
}

// miaConsoleKeyCode returns the key code for a sequence's final byte, or nil
// to drop it. The codes match inputDecodeKeyUsage: PETSCII cursor and Home,
// $94 Insert, $7F Delete.
func miaConsoleKeyCode(final byte, param uint8) []byte {
	switch final {
	case 'A':
		return []byte{0x91} // Up
	case 'B':
		return []byte{0x11} // Down
	case 'C':
		return []byte{0x1D} // Right
	case 'D':
		return []byte{0x9D} // Left
	case 'H':
		return []byte{0x13} // Home
	case '~':
		switch param {
		case 1, 7:
			return []byte{0x13} // Home
		case 2:
			return []byte{0x94} // Insert
		case 3:
			return []byte{0x7F} // Delete
		}
	}
	return nil // End, Page Up, Page Down, function keys
}
