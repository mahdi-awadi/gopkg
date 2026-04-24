package pipeline_test

import (
	"context"
	"testing"
	"time"

	"github.com/mahdi-awadi/gopkg/voice/pipeline"
	"github.com/mahdi-awadi/gopkg/voice/pipeline/fake"
)

func TestRun_HappyPath_CtxCancel(t *testing.T) {
	mulaw8k := pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1}
	pcm16k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 16000, Channels: 1}
	pcm24k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 24000, Channels: 1}

	tr := fake.NewTransport(mulaw8k, mulaw8k)
	ll := fake.NewLLM(pcm24k, pcm16k)
	rec := fake.NewRecorder()
	p, err := pipeline.New(pipeline.Options{Observer: rec})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()

	runErr := p.Run(ctx, tr, ll, fake.NewExecutor(), pipeline.SetupRequest{}, nil)
	if runErr != nil && runErr != context.Canceled {
		t.Errorf("Run err=%v", runErr)
	}

	ev := rec.Events()
	if len(ev) < 3 {
		t.Fatalf("expected at least HistoryInjected + SessionStart + SessionEnd, got %v", ev)
	}
	if _, ok := ev[1].(fake.RecSessionStart); !ok {
		t.Errorf("second event=%T, want RecSessionStart (after HistoryInjected)", ev[1])
	}
	last := ev[len(ev)-1]
	if e, ok := last.(fake.RecSessionEnd); !ok || e.Reason != pipeline.EndReasonContextDone {
		t.Errorf("last event=%+v, want RecSessionEnd{Reason=context_done}", last)
	}
}

func TestRun_CallerAudioFlowsToLLM_WithCodecBridge(t *testing.T) {
	mulaw8k := pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1}
	pcm16k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 16000, Channels: 1}
	pcm24k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 24000, Channels: 1}

	tr := fake.NewTransport(mulaw8k, mulaw8k)
	ll := fake.NewLLM(pcm24k, pcm16k)

	// Script 160 bytes of mulaw silence (20ms @ 8kHz).
	frame := make([]byte, 160)
	for i := range frame {
		frame[i] = 0xFF
	}
	tr.Script(pipeline.Frame{Data: frame})
	tr.CloseInbound()
	ll.CloseEvents()

	p, _ := pipeline.New(pipeline.Options{})
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = p.Run(ctx, tr, ll, fake.NewExecutor(), pipeline.SetupRequest{}, nil)

	got := ll.AudioIn()
	if len(got) != 1 {
		t.Fatalf("LLM received %d frames, want 1", len(got))
	}
	// 160 mulaw bytes → 640 pcm16le@16k bytes after bridging.
	if len(got[0].Data) != 640 {
		t.Errorf("bridged frame size=%d, want 640", len(got[0].Data))
	}
}

func TestRun_LLMAudioFlowsToTransport_WithCodecBridge(t *testing.T) {
	mulaw8k := pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1}
	pcm16k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 16000, Channels: 1}
	pcm24k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 24000, Channels: 1}

	tr := fake.NewTransport(mulaw8k, mulaw8k)
	ll := fake.NewLLM(pcm24k, pcm16k)

	// Script one 960-byte pcm16le@24k frame (20ms) from the LLM.
	ll.Script(pipeline.EventAudioOut{Frame: pipeline.Frame{Data: make([]byte, 960)}})
	ll.CloseEvents()
	tr.CloseInbound()

	p, _ := pipeline.New(pipeline.Options{})
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = p.Run(ctx, tr, ll, fake.NewExecutor(), pipeline.SetupRequest{}, nil)

	out := tr.Outbound()
	if len(out) != 1 {
		t.Fatalf("transport received %d frames, want 1", len(out))
	}
	// 960 pcm16le@24k bytes → 160 mulaw@8k bytes after bridging.
	if len(out[0].Data) != 160 {
		t.Errorf("bridged frame size=%d, want 160", len(out[0].Data))
	}
}
