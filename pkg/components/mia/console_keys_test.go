package mia

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEmulatedMiaConsoleKeysDecode(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []byte
	}{
		{"up", "\x1b[A", []byte{0x91}},
		{"down", "\x1b[B", []byte{0x11}},
		{"right", "\x1b[C", []byte{0x1D}},
		{"left", "\x1b[D", []byte{0x9D}},
		{"home H", "\x1b[H", []byte{0x13}},
		{"home 1~", "\x1b[1~", []byte{0x13}},
		{"home 7~", "\x1b[7~", []byte{0x13}},
		{"insert", "\x1b[2~", []byte{0x94}},
		{"delete", "\x1b[3~", []byte{0x7F}},
		{"application down", "\x1bOB", []byte{0x11}},
		{"application home", "\x1bOH", []byte{0x13}},
		{"ctrl right", "\x1b[1;5C", []byte{0x1D}},
		{"shift up", "\x1b[1;2A", []byte{0x91}},
		{"end dropped", "\x1b[F\x1b[4~x", []byte{'x'}},
		{"page up dropped", "\x1b[5~y", []byte{'y'}},
		{"function key dropped", "\x1b[15~z", []byte{'z'}},
		{"lone esc", "\x1b", []byte{0x1B}},
		{"esc esc", "\x1b\x1b", []byte{0x1B, 0x1B}},
		{"alt x", "\x1bx", []byte{0x1B, 'x'}},
		{"esc enter", "\x1b\r", []byte{0x1B, 0x0D}},
		{"text", "Hi\n", []byte{'H', 'i', 0x0D}},
		{"backspace", "a\x7f", []byte{'a', 0x08}},
		{"control keys", "\t\x01", []byte{0x09, 0x01}},
		{"cut by enter", "\x1b[1\r", []byte{0x0D}},
		{"cut by esc", "\x1b[\x1b[A", []byte{0x91}},
		{"mixed", "A\x1b[DB", []byte{'A', 0x9D, 'B'}},
		{"unfinished dropped", "\x1b[1", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var keys miaConsoleKeys
			now := time.Unix(0, 0)
			var got []byte
			for _, value := range []byte(tc.in) {
				got = append(got, keys.feed(value, now)...)
				now = now.Add(time.Millisecond)
			}
			// Not yet: the bytes of a sequence arrive closer together than this.
			got = append(got, keys.expire(now.Add(miaConsoleEscTimeout/2))...)
			got = append(got, keys.expire(now.Add(miaConsoleEscTimeout))...)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestEmulatedMiaConsoleKeysDroppedSequenceLeavesIdle(t *testing.T) {
	var keys miaConsoleKeys
	now := time.Unix(0, 0)
	keys.feed(0x1B, now)
	keys.feed('[', now)
	assert.Nil(t, keys.expire(now.Add(miaConsoleEscTimeout)))
	assert.Equal(t, []byte{'A'}, keys.feed('A', now.Add(miaConsoleEscTimeout)))
}
