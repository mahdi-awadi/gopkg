package fake

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/mahdi-awadi/gopkg/voice/pipeline"
)

func TestFakeTransport_Script_DeliversInOrder(t *testing.T) {
	ft := NewTransport(
		pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1},
		pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1},
	)
	ft.Script(
		pipeline.Frame{Data: []byte{1}},
		pipeline.Frame{Data: []byte{2}},
		pipeline.Frame{Data: []byte{3}},
	)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	in, _ := ft.Receive(ctx)

	var got []byte
	for f := range in {
		got = append(got, f.Data...)
		if len(got) == 3 {
			break
		}
	}
	if string(got) != string([]byte{1, 2, 3}) {
		t.Errorf("got %v, want [1 2 3]", got)
	}
}

func TestFakeTransport_Send_RecordsFrames(t *testing.T) {
	ft := NewTransport(
		pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 16000, Channels: 1},
		pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1},
	)
	_ = ft.Send(context.Background(), pipeline.Frame{Data: []byte{0xA}})
	_ = ft.Send(context.Background(), pipeline.Frame{Data: []byte{0xB}})
	out := ft.Outbound()
	if len(out) != 2 || out[0].Data[0] != 0xA || out[1].Data[0] != 0xB {
		t.Errorf("Outbound()=%v", out)
	}
}

func TestFakeTransport_ClearAndMark_RecordCounts(t *testing.T) {
	ft := NewTransport(
		pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1},
		pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1},
	)
	_ = ft.Clear(context.Background())
	_ = ft.Clear(context.Background())
	_ = ft.Mark(context.Background(), "turn-1")
	if ft.Clears() != 2 {
		t.Errorf("Clears()=%d, want 2", ft.Clears())
	}
	if m := ft.Marks(); len(m) != 1 || m[0] != "turn-1" {
		t.Errorf("Marks()=%v", m)
	}
}

func TestFakeTransport_EndOfScriptClosesReceive(t *testing.T) {
	ft := NewTransport(
		pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1},
		pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1},
	)
	ft.Script(pipeline.Frame{Data: []byte{1}})
	ft.CloseInbound()

	ctx := context.Background()
	in, errCh := ft.Receive(ctx)
	var wg sync.WaitGroup
	wg.Add(1)
	var count int
	go func() {
		defer wg.Done()
		for range in {
			count++
		}
	}()
	wg.Wait()
	if count != 1 {
		t.Errorf("got %d frames, want 1", count)
	}
	// Error channel closed with no error (clean close).
	select {
	case err, ok := <-errCh:
		if ok && err != nil {
			t.Errorf("expected clean close, got err=%v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("error channel never closed")
	}
}

func TestFakeLLM_ScriptEventsDelivered(t *testing.T) {
	fl := NewLLM(
		pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 24000, Channels: 1},
		pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 16000, Channels: 1},
	)
	fl.Script(
		pipeline.EventAssistantText{Text: "hi", Final: true},
		pipeline.EventTurnComplete{},
	)
	fl.CloseEvents()
	ctx := context.Background()
	_ = fl.Open(ctx, pipeline.SetupRequest{})

	ch, _ := fl.Events(ctx)
	var got []string
	for ev := range ch {
		switch e := ev.(type) {
		case pipeline.EventAssistantText:
			got = append(got, "text:"+e.Text)
		case pipeline.EventTurnComplete:
			got = append(got, "turn")
		}
	}
	want := []string{"text:hi", "turn"}
	if len(got) != len(want) {
		t.Errorf("got %v, want %v", got, want)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d]=%q, want %q", i, got[i], want[i])
		}
	}
}

func TestFakeLLM_SendAudioRecorded(t *testing.T) {
	fl := NewLLM(
		pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 24000, Channels: 1},
		pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 16000, Channels: 1},
	)
	_ = fl.SendAudio(context.Background(), pipeline.Frame{Data: []byte{0x1}})
	if got := fl.AudioIn(); len(got) != 1 || got[0].Data[0] != 0x1 {
		t.Errorf("AudioIn=%v", got)
	}
}

func TestFakeLLM_OpenErrorPropagates(t *testing.T) {
	fl := NewLLM(
		pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 24000, Channels: 1},
		pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 16000, Channels: 1},
	)
	fl.SetOpenErr(errors.New("bad setup"))
	err := fl.Open(context.Background(), pipeline.SetupRequest{})
	if err == nil || err.Error() != "bad setup" {
		t.Errorf("want 'bad setup', got %v", err)
	}
}

func TestFakeLLM_SendToolResultsRecorded(t *testing.T) {
	fl := NewLLM(
		pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 24000, Channels: 1},
		pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 16000, Channels: 1},
	)
	results := []pipeline.ToolResult{{CallID: "a", Data: "x"}, {CallID: "b", Data: "y"}}
	_ = fl.SendToolResults(context.Background(), results)
	got := fl.ToolResultsIn()
	if len(got) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(got))
	}
	if len(got[0]) != 2 || got[0][0].CallID != "a" || got[0][1].CallID != "b" {
		t.Errorf("batch=%v", got[0])
	}
}

func TestFakeLLM_InjectTurnRecorded(t *testing.T) {
	fl := NewLLM(
		pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 24000, Channels: 1},
		pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 16000, Channels: 1},
	)
	_ = fl.InjectTurn(context.Background(), pipeline.HistoryTurn{Role: pipeline.RoleUser, Content: "prior"})
	got := fl.InjectedTurns()
	if len(got) != 1 || got[0].Content != "prior" {
		t.Errorf("InjectedTurns=%v", got)
	}
}

func TestFakeExecutor_ProgrammedResult(t *testing.T) {
	fe := NewExecutor()
	fe.Register("greet", func(_ context.Context, call pipeline.ToolCall, _ pipeline.Session) (any, error) {
		return map[string]string{"hello": call.Args["name"].(string)}, nil
	})
	res, err := fe.Execute(context.Background(),
		pipeline.ToolCall{ID: "1", Name: "greet", Args: map[string]any{"name": "world"}},
		pipeline.Session{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	m := res.(map[string]string)
	if m["hello"] != "world" {
		t.Errorf("got %v", m)
	}
}

func TestFakeExecutor_UnknownReturnsError(t *testing.T) {
	fe := NewExecutor()
	_, err := fe.Execute(context.Background(), pipeline.ToolCall{Name: "missing"}, pipeline.Session{})
	if err == nil {
		t.Error("expected error for unregistered tool")
	}
}

func TestFakeFiller_EmitsScriptedFramesThenCloses(t *testing.T) {
	ff := NewFiller(pipeline.Frame{Data: []byte{1}}, pipeline.Frame{Data: []byte{2}})
	ch := ff.Frames(context.Background())
	var got [][]byte
	for f := range ch {
		got = append(got, f.Data)
	}
	if len(got) != 2 || got[0][0] != 1 || got[1][0] != 2 {
		t.Errorf("got %v, want [[1] [2]]", got)
	}
}

func TestFakeFiller_LoopMode(t *testing.T) {
	ff := NewFiller(pipeline.Frame{Data: []byte{42}})
	ff.Loop() // emit forever
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := ff.Frames(ctx)
	count := 0
	for f := range ch {
		count++
		_ = f
		if count == 10 {
			cancel()
		}
	}
	if count < 10 {
		t.Errorf("loop stopped early at %d", count)
	}
}

func TestRecorderObserver_CapturesCallbacksInOrder(t *testing.T) {
	r := NewRecorder()
	ctx := context.Background()
	s := pipeline.Session{ID: "abc"}
	r.OnSessionStart(ctx, s)
	r.OnAssistantText(ctx, s, "hello", true)
	r.OnToolCall(ctx, s, pipeline.ToolCall{ID: "1", Name: "ping"})
	r.OnSessionEnd(ctx, s, pipeline.EndReasonTransportClosed)

	ev := r.Events()
	if len(ev) != 4 {
		t.Fatalf("got %d events, want 4", len(ev))
	}
	if _, ok := ev[0].(RecSessionStart); !ok {
		t.Errorf("ev[0]=%T, want RecSessionStart", ev[0])
	}
	if a, ok := ev[1].(RecAssistantText); !ok || a.Text != "hello" {
		t.Errorf("ev[1]=%+v", ev[1])
	}
	if e, ok := ev[3].(RecSessionEnd); !ok || e.Reason != pipeline.EndReasonTransportClosed {
		t.Errorf("ev[3]=%+v", ev[3])
	}
}

func TestRecorderObserver_ImplementsObserver(t *testing.T) {
	var _ pipeline.Observer = NewRecorder()
}
