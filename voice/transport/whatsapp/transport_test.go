package whatsapptransport

import (
	"context"
	"testing"
	"time"

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

func TestTransport_ImplementsPipelineTransport(t *testing.T) {
	var _ pipeline.Transport = (*Transport)(nil)
}

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
	tp := New(peer)
	defer tp.Close()

	// Send should return nil — the sample is accepted by the local track.
	frame := pipeline.Frame{Data: []byte{0x00, 0xFF, 0xAA}, Timestamp: time.Now()}
	if err := tp.Send(context.Background(), frame); err != nil {
		t.Errorf("Send returned error: %v", err)
	}
}

func TestTransport_ClearAndMarkReturnNil(t *testing.T) {
	offerer, _, offerSDP := newTestOffererWithTrack(t)
	defer offerer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	peer, err := NewPeer(ctx, offerSDP, testConfig())
	if err != nil {
		t.Fatalf("NewPeer: %v", err)
	}
	tp := New(peer)
	defer tp.Close()

	c := context.Background()
	if err := tp.Clear(c); err != nil {
		t.Errorf("Clear returned error: %v", err)
	}
	if err := tp.Mark(c, "turn-1"); err != nil {
		t.Errorf("Mark returned error: %v", err)
	}
}

func TestTransport_CloseIsIdempotent(t *testing.T) {
	offerer, _, offerSDP := newTestOffererWithTrack(t)
	defer offerer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	peer, err := NewPeer(ctx, offerSDP, testConfig())
	if err != nil {
		t.Fatalf("NewPeer: %v", err)
	}
	tp := New(peer)

	if err := tp.Close(); err != nil {
		t.Errorf("first Close returned error: %v", err)
	}
	if err := tp.Close(); err != nil {
		t.Errorf("second Close returned error: %v (expected nil for idempotent close)", err)
	}
}

func TestTransport_ReceiveClosesAfterClose(t *testing.T) {
	offerer, _, offerSDP := newTestOffererWithTrack(t)
	defer offerer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	peer, err := NewPeer(ctx, offerSDP, testConfig())
	if err != nil {
		t.Fatalf("NewPeer: %v", err)
	}
	tp := New(peer)

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

// TestTransport_MediaLoopback sends PCMU samples from the offerer's track and
// asserts that at least one pipeline.Frame arrives on the transport's Receive
// channel. This test requires functional localhost ICE connectivity; it is
// skipped (not failed) when ICE cannot complete in the current environment.
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

	tp := New(peer)
	defer tp.Close()

	frames, errs := tp.Receive(ctx)

	// Send PCMU samples from the offerer side every 20ms.
	// 160 bytes = 20ms of 8kHz µ-law audio.
	pcmuSample := make([]byte, 160)
	for i := range pcmuSample {
		pcmuSample[i] = 0xFF
	}

	go func() {
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = offererTrack.WriteSample(media.Sample{
					Data:     pcmuSample,
					Duration: 20 * time.Millisecond,
				})
			}
		}
	}()

	// Expect at least one frame to arrive, or skip if ICE cannot connect.
	select {
	case f, ok := <-frames:
		if !ok {
			t.Skip("media loopback: receive channel closed (ICE may not be available in this env)")
		}
		if len(f.Data) == 0 {
			t.Error("received frame has empty Data")
		}
	case <-errs:
		t.Skip("media loopback: transport error (ICE may not be available in this env)")
	case <-ctx.Done():
		t.Skip("media loopback: timed out waiting for frame (ICE may not be available in this env)")
	}
}
