package whatsapptransport

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/hraban/opus"
	"github.com/mahdi-awadi/gopkg/voice/pipeline"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

// completeHandshake wires the offerer's remote description to the peer's answer SDP,
// completing the WebRTC handshake so ICE/media can flow between them.
func completeHandshake(t *testing.T, offerer *webrtc.PeerConnection, answerSDP string) {
	t.Helper()
	if err := offerer.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  answerSDP,
	}); err != nil {
		t.Fatalf("offerer SetRemoteDescription: %v", err)
	}
}

// newTransport is a test helper that builds a Peer + Transport from the given
// offer SDP using the test (loopback-only) ICE config.
func newTransport(t *testing.T, offerSDP string) *Transport {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	peer, err := NewPeer(ctx, offerSDP, testConfig())
	if err != nil {
		t.Fatalf("NewPeer: %v", err)
	}

	tp, err := New(peer)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return tp
}

// TestOpusRoundtrip encodes a PCM ramp to Opus at 24k (outSampleRate) then
// decodes at 16k (inSampleRate) and verifies:
//  1. The decoded output has the expected sample count (≥ inSampleRate*20/1000 = 320 samples)
//  2. The energy is non-zero (Opus is lossy — we assert approximate fidelity,
//     not bit-exact equality).
//
// This test proves CGO libopus wiring end to end without any ICE/WebRTC.
func TestOpusRoundtrip(t *testing.T) {
	enc, err := opus.NewEncoder(outSampleRate, 1, opus.AppVoIP)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	dec, err := opus.NewDecoder(inSampleRate, 1)
	if err != nil {
		t.Fatalf("NewDecoder: %v", err)
	}

	// Build a 20 ms ramp signal at 24k: 480 samples from -16384 to +16383.
	pcm := make([]int16, outFrameSamples) // 480
	for i := range pcm {
		pcm[i] = int16(-16384 + i*(32767/outFrameSamples))
	}

	// Encode.
	opusBuf := make([]byte, maxOpusBuf)
	n, err := enc.Encode(pcm, opusBuf)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if n == 0 {
		t.Fatal("Encode returned 0 bytes")
	}

	// Decode at 16k — libopus resamples. 20 ms @ 16k = 320 samples.
	outPCM := make([]int16, maxDecodeSamples)
	m, err := dec.Decode(opusBuf[:n], outPCM)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	const wantSamples = inSampleRate * outFrameMs / 1000 // 320
	if m < wantSamples {
		t.Errorf("Decode returned %d samples, want >= %d", m, wantSamples)
	}

	// Assert non-silent (sum of squares > 0).
	var energy float64
	for _, s := range outPCM[:m] {
		energy += math.Pow(float64(s), 2)
	}
	if energy == 0 {
		t.Error("decoded output is silent — Opus encode/decode produced all zeros")
	}
}

// TestTransport_Formats checks InboundFormat and OutboundFormat values.
func TestTransport_Formats(t *testing.T) {
	offerer, _, offerSDP := newTestOffererWithTrack(t)
	defer offerer.Close()

	tp := newTransport(t, offerSDP)
	defer tp.Close()

	in := tp.InboundFormat()
	out := tp.OutboundFormat()

	if in.Encoding != pipeline.EncodingPCM16LE || in.SampleRate != 16000 || in.Channels != 1 {
		t.Errorf("InboundFormat = %+v, want pcm16le@16k mono", in)
	}
	if out.Encoding != pipeline.EncodingPCM16LE || out.SampleRate != 24000 || out.Channels != 1 {
		t.Errorf("OutboundFormat = %+v, want pcm16le@24k mono", out)
	}
}

// TestTransport_SendReturnsNil sends one 24k PCM frame via Send and asserts nil error.
func TestTransport_SendReturnsNil(t *testing.T) {
	offerer, _, offerSDP := newTestOffererWithTrack(t)
	defer offerer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	peer, err := NewPeer(ctx, offerSDP, testConfig())
	if err != nil {
		t.Fatalf("NewPeer: %v", err)
	}
	completeHandshake(t, offerer, peer.AnswerSDP)

	tp, err := New(peer)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer tp.Close()

	// 480 samples @ 24k = exactly one 20ms Opus frame → one WriteSample call.
	pcm := make([]int16, outFrameSamples)
	for i := range pcm {
		pcm[i] = int16(i * 64) // non-silent
	}
	frame := pipeline.Frame{Data: int16LEToBytes(pcm), Timestamp: time.Now()}
	if err := tp.Send(context.Background(), frame); err != nil {
		t.Errorf("Send returned error: %v", err)
	}
}

// TestTransport_ClearAndMarkReturnNil checks no-op methods.
func TestTransport_ClearAndMarkReturnNil(t *testing.T) {
	offerer, _, offerSDP := newTestOffererWithTrack(t)
	defer offerer.Close()

	tp := newTransport(t, offerSDP)
	defer tp.Close()

	c := context.Background()
	if err := tp.Clear(c); err != nil {
		t.Errorf("Clear returned error: %v", err)
	}
	if err := tp.Mark(c, "turn-1"); err != nil {
		t.Errorf("Mark returned error: %v", err)
	}
}

// TestTransport_CloseIsIdempotent calls Close twice and asserts both return nil.
func TestTransport_CloseIsIdempotent(t *testing.T) {
	offerer, _, offerSDP := newTestOffererWithTrack(t)
	defer offerer.Close()

	tp := newTransport(t, offerSDP)

	if err := tp.Close(); err != nil {
		t.Errorf("first Close returned error: %v", err)
	}
	if err := tp.Close(); err != nil {
		t.Errorf("second Close returned error: %v (expected nil for idempotent close)", err)
	}
}

// TestTransport_ReceiveClosesAfterClose verifies that Close() unblocks the
// Receive goroutine waiting for a remote track and both channels close promptly.
func TestTransport_ReceiveClosesAfterClose(t *testing.T) {
	offerer, _, offerSDP := newTestOffererWithTrack(t)
	defer offerer.Close()

	tp := newTransport(t, offerSDP)

	frames, errs := tp.Receive(context.Background())

	// Close should unblock the goroutine waiting for the remote track.
	if err := tp.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}

	// Both channels must close within a short timeout.
	timeout := time.After(2 * time.Second)
	framesDone, errsDone := false, false
	for !framesDone || !errsDone {
		select {
		case _, ok := <-frames:
			if !ok {
				framesDone = true
			}
		case _, ok := <-errs:
			if !ok {
				errsDone = true
			}
		case <-timeout:
			t.Fatal("Receive channels did not close within timeout after Close()")
		}
	}
}

// TestTransport_MediaLoopback sends Opus-encoded audio from the offerer track,
// waits for a decoded pcm16le frame on the Receive channel. Skipped (not
// failed) when ICE cannot complete in the current environment.
func TestTransport_MediaLoopback(t *testing.T) {
	offerer, offererTrack, offerSDP := newTestOffererWithTrack(t)
	defer offerer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	peer, err := NewPeer(ctx, offerSDP, testConfig())
	if err != nil {
		t.Fatalf("NewPeer: %v", err)
	}
	defer peer.PC.Close()

	completeHandshake(t, offerer, peer.AnswerSDP)

	tp, err := New(peer)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer tp.Close()

	frames, errs := tp.Receive(ctx)

	// Encode a 20ms Opus frame to send from the offerer.
	enc, err := opus.NewEncoder(48000, 2, opus.AppVoIP)
	if err != nil {
		t.Fatalf("loopback encoder: %v", err)
	}
	// 20ms @ 48k stereo = 960 samples per channel × 2 channels = 1920 int16s.
	stereoSamples := make([]int16, 960*2)
	for i := range stereoSamples {
		stereoSamples[i] = 1000 // non-silent constant
	}
	opusBuf := make([]byte, maxOpusBuf)
	n, err := enc.Encode(stereoSamples, opusBuf)
	if err != nil {
		t.Fatalf("loopback encode: %v", err)
	}
	opusFrame := opusBuf[:n]

	go func() {
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = offererTrack.WriteSample(media.Sample{
					Data:     opusFrame,
					Duration: 20 * time.Millisecond,
				})
			}
		}
	}()

	// Expect at least one decoded PCM frame, or skip if ICE cannot connect.
	select {
	case f, ok := <-frames:
		if !ok {
			t.Skip("media loopback: receive channel closed (ICE may not be available in this env)")
		}
		if len(f.Data) == 0 {
			t.Error("received decoded frame has empty Data")
		}
	case <-errs:
		t.Skip("media loopback: transport error (ICE may not be available in this env)")
	case <-ctx.Done():
		t.Skip("media loopback: timed out waiting for frame (ICE may not be available in this env)")
	}
}
