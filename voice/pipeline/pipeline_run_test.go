package pipeline_test

import (
	"context"
	"errors"
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

func TestRun_EmitsCallerAndAssistantTranscripts(t *testing.T) {
	mulaw8k := pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1}
	pcm16k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 16000, Channels: 1}
	pcm24k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 24000, Channels: 1}

	tr := fake.NewTransport(mulaw8k, mulaw8k)
	ll := fake.NewLLM(pcm24k, pcm16k)
	ll.Script(
		pipeline.EventCallerTranscript{Text: "hello"},
		pipeline.EventAssistantText{Text: "hi there", Final: true},
	)
	ll.CloseEvents()
	tr.CloseInbound()

	rec := fake.NewRecorder()
	p, _ := pipeline.New(pipeline.Options{Observer: rec})
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = p.Run(ctx, tr, ll, fake.NewExecutor(), pipeline.SetupRequest{}, nil)

	var caller, assistant bool
	for _, e := range rec.Events() {
		if c, ok := e.(fake.RecCallerTranscript); ok && c.Text == "hello" {
			caller = true
		}
		if a, ok := e.(fake.RecAssistantText); ok && a.Text == "hi there" && a.Final {
			assistant = true
		}
	}
	if !caller || !assistant {
		t.Errorf("caller=%v assistant=%v", caller, assistant)
	}
}

func TestRun_TurnCompleteEmitsMarkAndObserver(t *testing.T) {
	mulaw8k := pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1}
	pcm16k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 16000, Channels: 1}
	pcm24k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 24000, Channels: 1}

	tr := fake.NewTransport(mulaw8k, mulaw8k)
	ll := fake.NewLLM(pcm24k, pcm16k)
	ll.Script(pipeline.EventTurnComplete{}, pipeline.EventTurnComplete{})
	ll.CloseEvents()
	tr.CloseInbound()

	rec := fake.NewRecorder()
	p, _ := pipeline.New(pipeline.Options{Observer: rec})
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = p.Run(ctx, tr, ll, fake.NewExecutor(), pipeline.SetupRequest{}, nil)

	marks := tr.Marks()
	if len(marks) != 2 || marks[0] != "turn-1" || marks[1] != "turn-2" {
		t.Errorf("marks=%v, want [turn-1 turn-2]", marks)
	}
	turns := 0
	for _, ev := range rec.Events() {
		if tc, ok := ev.(fake.RecTurnComplete); ok {
			turns++
			_ = tc
		}
	}
	if turns != 2 {
		t.Errorf("turn callbacks=%d, want 2", turns)
	}
}

func TestRun_InterruptedClearsAndFires(t *testing.T) {
	mulaw8k := pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1}
	pcm16k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 16000, Channels: 1}
	pcm24k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 24000, Channels: 1}

	tr := fake.NewTransport(mulaw8k, mulaw8k)
	ll := fake.NewLLM(pcm24k, pcm16k)
	ll.Script(pipeline.EventInterrupted{})
	ll.CloseEvents()
	tr.CloseInbound()

	rec := fake.NewRecorder()
	p, _ := pipeline.New(pipeline.Options{Observer: rec})
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = p.Run(ctx, tr, ll, fake.NewExecutor(), pipeline.SetupRequest{}, nil)

	if tr.Clears() != 1 {
		t.Errorf("Clears()=%d, want 1", tr.Clears())
	}
	var gotInterrupt bool
	for _, ev := range rec.Events() {
		if _, ok := ev.(fake.RecInterrupted); ok {
			gotInterrupt = true
		}
	}
	if !gotInterrupt {
		t.Error("OnInterrupted never fired")
	}
}

func TestRun_ReasonTransportClosed(t *testing.T) {
	mulaw8k := pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1}
	pcm16k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 16000, Channels: 1}
	pcm24k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 24000, Channels: 1}

	tr := fake.NewTransport(mulaw8k, mulaw8k)
	ll := fake.NewLLM(pcm24k, pcm16k)
	tr.CloseInbound() // caller hangs up
	// LLM events stay open but blocked — we want transport-closed to win.

	rec := fake.NewRecorder()
	p, _ := pipeline.New(pipeline.Options{Observer: rec})
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	err := p.Run(ctx, tr, ll, fake.NewExecutor(), pipeline.SetupRequest{}, nil)
	if err != nil {
		t.Errorf("Run returned err=%v on clean transport close", err)
	}
	ev := rec.Events()
	last, ok := ev[len(ev)-1].(fake.RecSessionEnd)
	if !ok || last.Reason != pipeline.EndReasonTransportClosed {
		t.Errorf("last=%+v, want RecSessionEnd{Reason=transport_closed}", ev[len(ev)-1])
	}
}

func TestRun_ReasonLLMClosed(t *testing.T) {
	mulaw8k := pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1}
	pcm16k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 16000, Channels: 1}
	pcm24k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 24000, Channels: 1}

	tr := fake.NewTransport(mulaw8k, mulaw8k)
	ll := fake.NewLLM(pcm24k, pcm16k)
	ll.CloseEvents() // LLM ends first
	// Transport stays open — LLM-closed should win.

	rec := fake.NewRecorder()
	p, _ := pipeline.New(pipeline.Options{Observer: rec})
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	err := p.Run(ctx, tr, ll, fake.NewExecutor(), pipeline.SetupRequest{}, nil)
	if err != nil {
		t.Errorf("Run returned err=%v on clean LLM close", err)
	}
	ev := rec.Events()
	last, ok := ev[len(ev)-1].(fake.RecSessionEnd)
	if !ok || last.Reason != pipeline.EndReasonLLMClosed {
		t.Errorf("last=%+v, want RecSessionEnd{Reason=llm_closed}", ev[len(ev)-1])
	}
}

func TestRun_HistoryInjectionFiresObserver(t *testing.T) {
	mulaw8k := pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1}
	pcm16k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 16000, Channels: 1}
	pcm24k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 24000, Channels: 1}

	tr := fake.NewTransport(mulaw8k, mulaw8k)
	ll := fake.NewLLM(pcm24k, pcm16k)
	ll.CloseEvents()
	tr.CloseInbound()

	rec := fake.NewRecorder()
	p, _ := pipeline.New(pipeline.Options{Observer: rec})
	setup := pipeline.SetupRequest{History: []pipeline.HistoryTurn{
		{Role: pipeline.RoleUser, Content: "prior 1"},
		{Role: pipeline.RoleAssistant, Content: "prior 2"},
	}}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = p.Run(ctx, tr, ll, fake.NewExecutor(), setup, nil)

	var injected int
	for _, ev := range rec.Events() {
		if h, ok := ev.(fake.RecHistoryInjected); ok {
			injected = h.Count
		}
	}
	if injected != 2 {
		t.Errorf("OnHistoryInjected count=%d, want 2", injected)
	}

	// Fake LLM also recorded the flushed history inside Open.
	turns := ll.InjectedTurns()
	if len(turns) != 2 || turns[0].Content != "prior 1" || turns[1].Content != "prior 2" {
		t.Errorf("InjectedTurns=%v", turns)
	}
}

func TestRun_FatalOpenError(t *testing.T) {
	mulaw8k := pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1}
	pcm16k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 16000, Channels: 1}
	pcm24k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 24000, Channels: 1}

	tr := fake.NewTransport(mulaw8k, mulaw8k)
	ll := fake.NewLLM(pcm24k, pcm16k)
	ll.SetOpenErr(errors.New("bad setup"))

	rec := fake.NewRecorder()
	p, _ := pipeline.New(pipeline.Options{Observer: rec})
	err := p.Run(context.Background(), tr, ll, fake.NewExecutor(), pipeline.SetupRequest{}, nil)
	if err == nil || err.Error() != "bad setup" {
		t.Errorf("Run err=%v, want 'bad setup'", err)
	}
	ev := rec.Events()
	// No SessionStart expected, but OnError + OnSessionEnd(FatalError) must fire.
	var sawError, sawEnd bool
	for _, e := range ev {
		switch x := e.(type) {
		case fake.RecError:
			if x.Err.Error() == "bad setup" {
				sawError = true
			}
		case fake.RecSessionEnd:
			if x.Reason == pipeline.EndReasonFatalError {
				sawEnd = true
			}
		case fake.RecSessionStart:
			t.Errorf("SessionStart should not fire when Open fails")
		}
	}
	if !sawError || !sawEnd {
		t.Errorf("sawError=%v sawEnd=%v", sawError, sawEnd)
	}
}

func TestRun_FatalSendAudioError(t *testing.T) {
	mulaw8k := pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1}
	pcm16k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 16000, Channels: 1}
	pcm24k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 24000, Channels: 1}

	tr := fake.NewTransport(mulaw8k, mulaw8k)
	ll := fake.NewLLM(pcm24k, pcm16k)
	tr.Script(pipeline.Frame{Data: make([]byte, 160)})
	ll.SetSendAudioErr(errors.New("wire closed"))

	rec := fake.NewRecorder()
	p, _ := pipeline.New(pipeline.Options{Observer: rec})
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	err := p.Run(ctx, tr, ll, fake.NewExecutor(), pipeline.SetupRequest{}, nil)
	if err == nil || err.Error() != "wire closed" {
		t.Errorf("Run err=%v, want 'wire closed'", err)
	}
	ev := rec.Events()
	last, ok := ev[len(ev)-1].(fake.RecSessionEnd)
	if !ok || last.Reason != pipeline.EndReasonFatalError {
		t.Errorf("last=%+v", ev[len(ev)-1])
	}
}

func TestRun_FatalTransportSendError(t *testing.T) {
	mulaw8k := pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1}
	pcm16k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 16000, Channels: 1}
	pcm24k := pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 24000, Channels: 1}

	tr := fake.NewTransport(mulaw8k, mulaw8k)
	ll := fake.NewLLM(pcm24k, pcm16k)
	ll.Script(pipeline.EventAudioOut{Frame: pipeline.Frame{Data: make([]byte, 960)}})
	tr.SetSendErr(errors.New("dropped"))

	rec := fake.NewRecorder()
	p, _ := pipeline.New(pipeline.Options{Observer: rec})
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	err := p.Run(ctx, tr, ll, fake.NewExecutor(), pipeline.SetupRequest{}, nil)
	if err == nil || err.Error() != "dropped" {
		t.Errorf("Run err=%v, want 'dropped'", err)
	}
}
