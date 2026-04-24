package fake

import (
	"context"
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
