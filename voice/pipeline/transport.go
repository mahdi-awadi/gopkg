package pipeline

import "context"

// Transport is the caller-side endpoint (Twilio Media Streams,
// WhatsApp Calling, SIP/RTP, …). It delivers caller audio to the
// pipeline via Receive and accepts outbound audio via Send.
//
// Transport implementations MUST serialize their own Send / Clear /
// Mark calls internally — the pipeline invokes these concurrently
// from the LLM-events goroutine and the hold-filler pump goroutine.
type Transport interface {
	// InboundFormat is the format of frames Receive emits.
	InboundFormat() AudioFormat
	// OutboundFormat is the format Send expects.
	OutboundFormat() AudioFormat

	// Receive returns two channels: inbound audio frames and a single
	// terminal error (nil on clean close). Both channels close when
	// the caller hangs up or the underlying connection errors.
	Receive(ctx context.Context) (<-chan Frame, <-chan error)

	// Send pushes one frame toward the caller. Synchronous; returns
	// after the frame is accepted by the underlying wire.
	Send(ctx context.Context, f Frame) error

	// Clear aborts any queued outbound audio immediately.
	Clear(ctx context.Context) error

	// Mark sends a named sync point. Adapters without a native mark
	// concept return nil.
	Mark(ctx context.Context, name string) error

	// Close releases transport resources. Idempotent.
	Close() error
}
