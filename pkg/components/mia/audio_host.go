//go:build !linux || cgo

package mia

import (
	"io"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
)

// Player source-buffer size in bytes (4 bytes/frame). Matched to the lead so
// oto's mux goroutine fills it and then blocks on its condition variable
// (rather than spinning on a reader that hands out only lead-sized chunks),
// and so the player's initial fill completes in one Read. oto's own device
// buffer (NewContextOptions.BufferSize) is the real underrun cushion.
const miaAudioPlayerBufBytes = miaAudioLeadSamples * 4

// The oto context is process-wide; oto refuses to create more than one.
var (
	miaAudioCtxOnce sync.Once
	miaAudioCtx     *oto.Context
	miaAudioCtxErr  error
)

func miaAudioContext() (*oto.Context, error) {
	miaAudioCtxOnce.Do(func() {
		ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
			SampleRate:   miaAudioSampleRate,
			ChannelCount: 2,
			Format:       oto.FormatSignedInt16LE,
			BufferSize:   50 * time.Millisecond,
		})
		if err != nil {
			miaAudioCtxErr = err
			return
		}
		<-ready
		miaAudioCtx = ctx
	})

	return miaAudioCtx, miaAudioCtxErr
}

// miaAudioNewPlayer opens the shared oto context and returns a paused player
// that pulls PCM from r.
func miaAudioNewPlayer(r io.Reader) (miaAudioPlayer, error) {
	ctx, err := miaAudioContext()
	if err != nil {
		return nil, err
	}

	player := ctx.NewPlayer(r)
	// Shrink the player's source buffer from oto's 500 ms default: the reader
	// paces itself to wall-clock, so a big buffer just makes the mux goroutine
	// spin asking for frames the reader will not hand out yet.
	player.SetBufferSize(miaAudioPlayerBufBytes)

	return player, nil
}
