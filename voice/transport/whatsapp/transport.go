package whatsapptransport

import (
	"context"
	"encoding/binary"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/hraban/opus"
	"github.com/mahdi-awadi/gopkg/voice/pipeline"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

const (
	// inSampleRate is the PCM rate we ask libopus to decode to (matching Gemini
	// Live's expected input: pcm16le @ 16 kHz mono).
	inSampleRate = 16000

	// outSampleRate is the PCM rate Gemini Live emits (pcm16le @ 24 kHz mono).
	outSampleRate = 24000

	// outFrameMs is the Opus encode frame size in ms. Opus requires exact frame
	// sizes (2.5/5/10/20/40/60 ms); 20 ms is the standard voice frame.
	outFrameMs = 20

	// outFrameSamples is the number of 24k PCM samples per 20 ms Opus frame.
	outFrameSamples = outSampleRate * outFrameMs / 1000 // 480

	// maxOpusBuf is the maximum encoded Opus frame size in bytes.
	maxOpusBuf = 4096

	// maxDecodeSamples is the max PCM output per Decode call.
	// 60 ms @ 16 kHz = 960 samples.
	maxDecodeSamples = inSampleRate * 60 / 1000
)

// Transport is a pipeline.Transport over a pion WebRTC PeerConnection.
// Inbound Opus RTP is decoded to pcm16le@16k mono for Gemini Live; outbound
// pcm16le@24k mono from Gemini is encoded to Opus before writing to the local
// WebRTC track. Outbound writes are serialised by writeMu; only Send and Close
// acquire writeMu (Clear and Mark are lock-free no-ops).
type Transport struct {
	peer *Peer

	// dec decodes inbound Opus frames to PCM @16k mono.
	dec *opus.Decoder

	// enc encodes outbound PCM @24k mono to Opus frames.
	enc *opus.Encoder

	// writeMu serialises Send + Close and protects sendBuf/enc.
	writeMu sync.Mutex

	// sendBuf accumulates PCM int16 samples from Send until we have a full
	// 20 ms frame (480 samples @24k) to encode.
	sendBuf []int16

	// outQueue holds encoded Opus frames awaiting paced transmission. The
	// paceSender goroutine drains it at real time (one 20 ms frame every 20 ms)
	// so Gemini's faster-than-real-time audio bursts don't overflow the caller's
	// jitter buffer (which caused "first few words then silence").
	outQueue chan []byte

	// silenceFrame is a pre-encoded 20 ms Opus silence packet. paceSender writes
	// it whenever the queue is momentarily empty so the outbound RTP stream is
	// CONTINUOUS (monotonic timestamps, one frame every 20 ms). Gaps in the RTP
	// timeline break the caller's playout clock — that, not the burst, is the
	// real cause of choppy/cut-out audio.
	silenceFrame []byte

	closed bool
	done   chan struct{} // closed by Close() to unblock Receive + paceSender
}

// New wraps the negotiated Peer in a pipeline.Transport. Close() closes both
// the Transport and the underlying PeerConnection. Returns an error if the
// CGO libopus decoder/encoder cannot be created (should not happen in practice
// since libopus is statically linked in the container).
func New(p *Peer) (*Transport, error) {
	dec, err := opus.NewDecoder(inSampleRate, 1)
	if err != nil {
		return nil, err
	}
	enc, err := opus.NewEncoder(outSampleRate, 1, opus.AppVoIP)
	if err != nil {
		return nil, err
	}
	// Pre-encode one 20 ms Opus silence frame (480 zero samples @24k) for gap fill.
	silenceBuf := make([]byte, maxOpusBuf)
	sn, err := enc.Encode(make([]int16, outFrameSamples), silenceBuf)
	if err != nil {
		return nil, err
	}

	t := &Transport{
		peer:         p,
		dec:          dec,
		enc:          enc,
		sendBuf:      make([]int16, 0, outFrameSamples*2),
		outQueue:     make(chan []byte, 1000), // ~20 s of 20 ms frames
		silenceFrame: append([]byte(nil), silenceBuf[:sn]...),
		done:         make(chan struct{}),
	}
	go t.paceSender()
	return t, nil
}

// paceSender writes exactly one 20 ms Opus frame every 20 ms on a drift-free
// deadline clock — a queued frame when available, otherwise the pre-encoded
// silence frame. The CONTINUOUS stream (no skipped ticks) keeps RTP timestamps
// monotonic and the caller's playout clock locked, which is what actually fixes
// the choppy/cut-out audio. Stops when Close() closes done.
func (t *Transport) paceSender() {
	frameDur := outFrameMs * time.Millisecond
	next := time.Now()
	for {
		select {
		case <-t.done:
			return
		default:
		}

		next = next.Add(frameDur)
		if wait := time.Until(next); wait > 0 {
			select {
			case <-t.done:
				return
			case <-time.After(wait):
			}
		} else if wait < -time.Second {
			// Fell far behind (e.g. scheduler stall) — resync instead of bursting.
			next = time.Now()
		}

		frame := t.silenceFrame
		select {
		case q := <-t.outQueue:
			frame = q
		default:
		}
		_ = t.peer.localTrack.WriteSample(media.Sample{Data: frame, Duration: frameDur})
	}
}

// InboundFormat reports the format of frames emitted by Receive: pcm16le@16k mono.
// This matches Gemini Live's expected audio input format.
func (t *Transport) InboundFormat() pipeline.AudioFormat {
	return pipeline.AudioFormat{
		Encoding:   pipeline.EncodingPCM16LE,
		SampleRate: inSampleRate,
		Channels:   1,
	}
}

// OutboundFormat reports the format Send expects: pcm16le@24k mono.
// This matches Gemini Live's audio output format.
func (t *Transport) OutboundFormat() pipeline.AudioFormat {
	return pipeline.AudioFormat{
		Encoding:   pipeline.EncodingPCM16LE,
		SampleRate: outSampleRate,
		Channels:   1,
	}
}

// Receive starts a goroutine that reads RTP packets from the remote caller's
// Opus track, decodes each frame to pcm16le@16k mono, and emits pipeline.Frames.
// Both returned channels close when the caller hangs up, the PeerConnection
// errors, ctx is cancelled, or Close() is called. errs carries a single
// terminal error (nil for clean close).
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

		// PCM decode buffer: max 60 ms @ 16k mono = 960 samples.
		pcmBuf := make([]int16, maxDecodeSamples)

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

			f, decErr := rtpToFrame(t.dec, pkt, pcmBuf)
			if decErr != nil {
				// Decode error on a single packet: skip and continue.
				continue
			}

			select {
			case frames <- f:
			case <-t.done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return frames, errs
}

// Send accepts one pcm16le@24k mono frame from Gemini, buffers it, and encodes
// complete 20 ms (480-sample) Opus frames toward the caller. Leftover samples
// (<480) are held for the next call. No-ops if the transport is closed.
func (t *Transport) Send(_ context.Context, f pipeline.Frame) error {
	t.writeMu.Lock()
	defer t.writeMu.Unlock()
	if t.closed {
		return nil
	}

	// Convert pcm16le bytes to int16 samples.
	samples := bytesToInt16LE(f.Data)
	t.sendBuf = append(t.sendBuf, samples...)

	// Encode in fixed 480-sample (20 ms @24k) chunks and ENQUEUE them; the
	// paceSender goroutine writes them to the wire at real time. A fresh buffer
	// per frame is required because the encoded slice lives in the queue until
	// the pacer consumes it (no reuse).
	for len(t.sendBuf) >= outFrameSamples {
		frame := append([]int16(nil), t.sendBuf[:outFrameSamples]...)
		t.sendBuf = t.sendBuf[outFrameSamples:]

		opusBuf := make([]byte, maxOpusBuf)
		n, encErr := t.enc.Encode(frame, opusBuf)
		if encErr != nil {
			return encErr
		}

		select {
		case t.outQueue <- opusBuf[:n]:
		default:
			// Queue full (>~20 s backlog) — drop to bound latency rather than
			// block the pipeline's read loop.
		}
	}
	return nil
}

// Clear drops all queued outbound audio immediately. The pipeline calls this on
// caller interruption (barge-in) so the AI stops mid-reply at once instead of
// finishing a now-stale sentence from the paced queue.
func (t *Transport) Clear(_ context.Context) error {
	for {
		select {
		case <-t.outQueue:
		default:
			return nil
		}
	}
}

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

// rtpToFrame decodes one Opus RTP packet into a pipeline.Frame containing
// pcm16le@16k mono bytes. pcmBuf is a reusable scratch buffer (max 960 int16s).
func rtpToFrame(dec *opus.Decoder, pkt *rtp.Packet, pcmBuf []int16) (pipeline.Frame, error) {
	n, err := dec.Decode(pkt.Payload, pcmBuf)
	if err != nil {
		return pipeline.Frame{}, err
	}
	leBytes := int16LEToBytes(pcmBuf[:n])
	return pipeline.Frame{Data: leBytes, Timestamp: time.Now()}, nil
}

// bytesToInt16LE converts a pcm16le byte slice to []int16 (little-endian).
func bytesToInt16LE(b []byte) []int16 {
	n := len(b) / 2
	out := make([]int16, n)
	for i := range out {
		out[i] = int16(binary.LittleEndian.Uint16(b[i*2:]))
	}
	return out
}

// int16LEToBytes converts []int16 to a pcm16le byte slice (little-endian).
func int16LEToBytes(s []int16) []byte {
	out := make([]byte, len(s)*2)
	for i, v := range s {
		binary.LittleEndian.PutUint16(out[i*2:], uint16(v))
	}
	return out
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
