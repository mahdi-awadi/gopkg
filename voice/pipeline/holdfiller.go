package pipeline

import "context"

// HoldFiller produces audio frames the pipeline plays during slow
// tool calls (past Options.HoldFillerDelay). Frames must be in the
// Transport.OutboundFormat() — the pipeline will not codec-bridge
// filler frames.
type HoldFiller interface {
	Frames(ctx context.Context) <-chan Frame
}

// SilentFiller is a HoldFiller that produces zero frames. Use when
// the caller should hear silence during slow tool calls.
type SilentFiller struct{}

// Frames returns a pre-closed channel — callers pull nothing and the
// pipeline plays nothing.
func (SilentFiller) Frames(context.Context) <-chan Frame {
	c := make(chan Frame)
	close(c)
	return c
}
