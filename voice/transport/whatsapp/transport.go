package whatsapptransport

import (
	"context"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/mahdi-awadi/gopkg/voice/pipeline"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

// Transport is a pipeline.Transport over a pion WebRTC PeerConnection.
// It delivers PCMU RTP payloads from the remote caller (inbound) and sends
// PCMU media.Samples back (outbound). Outbound writes are serialised by
// writeMu; only Send and Close acquire writeMu (Clear and Mark are lock-free
// no-ops).
type Transport struct {
	peer *Peer

	writeMu sync.Mutex
	closed  bool
	done    chan struct{} // closed by Close() to unblock Receive goroutine
}

// New wraps the negotiated Peer in a pipeline.Transport. Close() closes both
// the Transport and the underlying PeerConnection.
func New(p *Peer) *Transport {
	return &Transport{
		peer: p,
		done: make(chan struct{}),
	}
}

// InboundFormat reports the format of frames emitted by Receive: mulaw@8k mono.
func (t *Transport) InboundFormat() pipeline.AudioFormat {
	return pipeline.AudioFormat{
		Encoding:   pipeline.EncodingMulaw,
		SampleRate: 8000,
		Channels:   1,
	}
}

// OutboundFormat reports the format Send expects: mulaw@8k mono.
func (t *Transport) OutboundFormat() pipeline.AudioFormat {
	return pipeline.AudioFormat{
		Encoding:   pipeline.EncodingMulaw,
		SampleRate: 8000,
		Channels:   1,
	}
}

// Receive starts a goroutine that reads RTP packets from the remote caller's
// track and emits them as pipeline.Frames. Both returned channels close when
// the caller hangs up, the PeerConnection errors, ctx is cancelled, or
// Close() is called. errs carries a single terminal error (nil for clean close).
func (t *Transport) Receive(ctx context.Context) (<-chan pipeline.Frame, <-chan error) {
	frames := make(chan pipeline.Frame)
	errs := make(chan error, 1)

	go func() {
		defer close(frames)
		defer close(errs)

		// Wait for the remote track to arrive (delivered by OnTrack), or for
		// ctx cancellation / Close().
		var tr *webrtc.TrackRemote
		select {
		case r, ok := <-t.peer.remoteTrack:
			if !ok {
				return
			}
			tr = r
		case <-t.done:
			return
		case <-ctx.Done():
			return
		}

		for {
			pkt, _, err := tr.ReadRTP()
			if err != nil {
				if err == io.EOF || isCloseErr(err) {
					errs <- nil
				} else {
					errs <- err
				}
				return
			}

			select {
			case frames <- rtpToFrame(pkt):
			case <-t.done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return frames, errs
}

// Send pushes one PCMU frame toward the caller. No-ops if the transport is closed.
func (t *Transport) Send(_ context.Context, f pipeline.Frame) error {
	t.writeMu.Lock()
	defer t.writeMu.Unlock()
	if t.closed {
		return nil
	}
	return t.peer.localTrack.WriteSample(media.Sample{
		Data:     f.Data,
		Duration: 20 * time.Millisecond,
	})
}

// Clear is a no-op for pion — TrackLocalStaticSample does not buffer queued audio.
// Interruptions are handled by the pipeline ceasing to call Send.
func (t *Transport) Clear(_ context.Context) error { return nil }

// Mark is a no-op for pion — WebRTC has no native named sync-point mechanism.
func (t *Transport) Mark(_ context.Context, _ string) error { return nil }

// Close is idempotent. It closes the underlying PeerConnection and unblocks
// any goroutine blocked in Receive.
func (t *Transport) Close() error {
	t.writeMu.Lock()
	if t.closed {
		t.writeMu.Unlock()
		return nil
	}
	t.closed = true
	close(t.done)
	t.writeMu.Unlock()
	return t.peer.PC.Close()
}

// rtpToFrame converts a received RTP packet into a pipeline.Frame by copying
// the raw payload bytes. This is the sole mapping point between the WebRTC
// layer and the pipeline layer, exposed as a named function so it can be
// unit-tested independently of ICE/DTLS connectivity.
func rtpToFrame(pkt *rtp.Packet) pipeline.Frame {
	return pipeline.Frame{Data: pkt.Payload, Timestamp: time.Now()}
}

// isCloseErr returns true for errors that indicate a normal connection teardown.
func isCloseErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "closed") ||
		strings.Contains(msg, "EOF") ||
		strings.Contains(msg, "use of closed")
}

// Compile-time interface check.
var _ pipeline.Transport = (*Transport)(nil)
