//go:build linux && !cgo

package mia

import (
	"errors"
	"io"
)

// miaAudioNewPlayer returns an error: on Linux oto reaches the sound device
// through cgo, so a binary built without cgo (any cross build) has no host
// audio. The emulated audio engine still runs, and the MIA console's audio
// status shows this error as the output state.
func miaAudioNewPlayer(r io.Reader) (miaAudioPlayer, error) {
	return nil, errors.New("this Linux binary was built without cgo, which host audio needs")
}
