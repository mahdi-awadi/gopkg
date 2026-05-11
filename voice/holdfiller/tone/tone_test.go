package tone

import (
	"context"
	"testing"
	"time"

	"github.com/mahdi-awadi/gopkg/voice/pipeline"
)

func TestFiller_ImplementsPipelineHoldFiller(t *testing.T) {
	var _ pipeline.HoldFiller = (*Filler)(nil)
}

func TestFiller_EmitsMulaw20msFramesUntilCancelled(t *testing.T) {
	f := New()
	ctx, cancel := context.WithCancel(context.Background())
	frames := f.Frames(ctx)

	for i := 0; i < 3; i++ {
		select {
		case frame := <-frames:
			if len(frame.Data) != 160 {
				t.Fatalf("frame len=%d, want 160 bytes", len(frame.Data))
			}
			if frame.Timestamp.IsZero() {
				t.Fatalf("frame timestamp is zero")
			}
		case <-time.After(500 * time.Millisecond):
			t.Fatal("timed out waiting for frame")
		}
	}

	cancel()
	select {
	case _, ok := <-frames:
		if ok {
			t.Fatal("frames channel should close after cancellation")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("frames channel did not close after cancellation")
	}
}

func TestNewWithOptionsControlsPattern(t *testing.T) {
	f := NewWithOptions(Options{
		SampleRate:      8000,
		FrameDuration:   20 * time.Millisecond,
		PatternDuration: 100 * time.Millisecond,
		ToneDuration:    20 * time.Millisecond,
		FrequencyHz:     440,
		Amplitude:       1200,
	})
	if len(f.chunks) != 5 {
		t.Fatalf("chunks=%d, want 5", len(f.chunks))
	}
	for _, chunk := range f.chunks {
		if len(chunk) != 160 {
			t.Fatalf("chunk len=%d, want 160", len(chunk))
		}
	}
}
