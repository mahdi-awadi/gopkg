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
