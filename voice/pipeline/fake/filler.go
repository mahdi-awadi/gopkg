package fake

import (
	"context"
	"sync"

	"github.com/mahdi-awadi/gopkg/voice/pipeline"
)

// Filler is a scriptable pipeline.HoldFiller. By default it emits
// the given frames once and closes; call Loop() to make it emit
// them forever until ctx is cancelled.
type Filler struct {
	mu     sync.Mutex
	frames []pipeline.Frame
	loop   bool
}

// NewFiller returns a Filler pre-loaded with frames.
func NewFiller(frames ...pipeline.Frame) *Filler {
	cp := make([]pipeline.Frame, len(frames))
	copy(cp, frames)
	return &Filler{frames: cp}
}

// Loop switches the Filler into looping mode.
func (f *Filler) Loop() {
	f.mu.Lock()
	f.loop = true
	f.mu.Unlock()
}

// Frames implements pipeline.HoldFiller.
func (f *Filler) Frames(ctx context.Context) <-chan pipeline.Frame {
	ch := make(chan pipeline.Frame)
	go func() {
		defer close(ch)
		for {
			f.mu.Lock()
			frames := make([]pipeline.Frame, len(f.frames))
			copy(frames, f.frames)
			loop := f.loop
			f.mu.Unlock()
			for _, fr := range frames {
				select {
				case ch <- fr:
				case <-ctx.Done():
					return
				}
			}
			if !loop {
				return
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()
	return ch
}

var _ pipeline.HoldFiller = (*Filler)(nil)
