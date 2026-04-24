// Package fake ships scriptable test fakes for every pipeline
// interface. The fakes are deterministic and inspectable so unit
// tests can stage scenarios and assert on recorded activity.
package fake

import (
	"context"
	"errors"
	"sync"

	"github.com/mahdi-awadi/gopkg/voice/pipeline"
)

// Transport is a scriptable pipeline.Transport for tests.
type Transport struct {
	inFormat  pipeline.AudioFormat
	outFormat pipeline.AudioFormat

	mu       sync.Mutex
	script   []pipeline.Frame
	outbound []pipeline.Frame
	marks    []string
	clears   int
	closed   bool

	inboundClosed bool
	sendErr       error // optional error to return from Send
}

// NewTransport builds a Transport fake that delivers frames from
// Script via Receive and records every Send / Clear / Mark.
func NewTransport(inbound, outbound pipeline.AudioFormat) *Transport {
	return &Transport{inFormat: inbound, outFormat: outbound}
}

// Script queues frames to be delivered through Receive. Can be called
// before Receive or after (frames appended to pending queue).
func (t *Transport) Script(frames ...pipeline.Frame) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.script = append(t.script, frames...)
}

// CloseInbound signals "caller hung up" — Receive's channels close
// cleanly once the scripted frames are drained.
func (t *Transport) CloseInbound() {
	t.mu.Lock()
	t.inboundClosed = true
	t.mu.Unlock()
}

// SetSendErr makes the next Send call return the given error.
// Useful for fatal-error tests.
func (t *Transport) SetSendErr(err error) {
	t.mu.Lock()
	t.sendErr = err
	t.mu.Unlock()
}

// Outbound returns a snapshot of frames passed to Send.
func (t *Transport) Outbound() []pipeline.Frame {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]pipeline.Frame, len(t.outbound))
	copy(out, t.outbound)
	return out
}

// Marks returns a snapshot of mark names passed to Mark.
func (t *Transport) Marks() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]string, len(t.marks))
	copy(out, t.marks)
	return out
}

// Clears returns the number of Clear() calls received.
func (t *Transport) Clears() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.clears
}

// InboundFormat implements pipeline.Transport.
func (t *Transport) InboundFormat() pipeline.AudioFormat { return t.inFormat }

// OutboundFormat implements pipeline.Transport.
func (t *Transport) OutboundFormat() pipeline.AudioFormat { return t.outFormat }

// Receive delivers scripted frames then closes. If CloseInbound has
// not been called before Script runs out, Receive keeps the channel
// open until CloseInbound or ctx cancellation.
func (t *Transport) Receive(ctx context.Context) (<-chan pipeline.Frame, <-chan error) {
	frames := make(chan pipeline.Frame)
	errs := make(chan error, 1)
	go func() {
		defer close(frames)
		defer close(errs)
		for {
			t.mu.Lock()
			if len(t.script) > 0 {
				f := t.script[0]
				t.script = t.script[1:]
				t.mu.Unlock()
				select {
				case frames <- f:
				case <-ctx.Done():
					return
				}
				continue
			}
			closed := t.inboundClosed
			t.mu.Unlock()
			if closed {
				return
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
			// busy-wait tiny sleep — tests typically close inbound
			// synchronously, so we rarely loop.
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()
	return frames, errs
}

// Send records the frame.
func (t *Transport) Send(_ context.Context, f pipeline.Frame) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.sendErr != nil {
		err := t.sendErr
		t.sendErr = nil
		return err
	}
	t.outbound = append(t.outbound, f)
	return nil
}

// Clear increments the clear counter.
func (t *Transport) Clear(context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.clears++
	return nil
}

// Mark records the mark name.
func (t *Transport) Mark(_ context.Context, name string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.marks = append(t.marks, name)
	return nil
}

// Close flags the fake closed. Idempotent.
func (t *Transport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
	return nil
}

var _ pipeline.Transport = (*Transport)(nil)

// ErrTransportClosed is exposed so tests can assert on it.
var ErrTransportClosed = errors.New("fake transport: closed")
