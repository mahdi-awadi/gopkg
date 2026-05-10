package twilio

import (
	"context"
	"testing"
	"time"

	"github.com/mahdi-awadi/gopkg/voice/pipeline"
)

func TestFiller_ImplementsHoldFiller(t *testing.T) {
	var _ pipeline.HoldFiller = New()
}

func TestFiller_EmitsFramesUntilContextDone(t *testing.T) {
	f := New()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()

	ch := f.Frames(ctx)
	got := 0
	for range ch {
		got++
	}
	if got < 1 {
		t.Errorf("got %d frames, want >= 1", got)
	}
}

func TestFiller_FrameIsMulaw160Bytes(t *testing.T) {
	f := New()
	ctx, cancel := context.WithCancel(context.Background())
	ch := f.Frames(ctx)

	select {
	case fr := <-ch:
		cancel()
		if len(fr.Data) != 160 {
			t.Errorf("frame Data length=%d, want 160 (20ms mulaw@8k)", len(fr.Data))
		}
	case <-time.After(100 * time.Millisecond):
		cancel()
		t.Fatal("no frame received within 100ms")
	}
}
