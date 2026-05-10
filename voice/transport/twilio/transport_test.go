package twiliotransport

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mahdi-awadi/gopkg/voice/pipeline"
)

// dialTestServer spins up an httptest WS server with the provided
// server-side handler, dials it, and returns a Transport wrapped
// around the client conn.
func dialTestServer(t *testing.T, handler func(*websocket.Conn)) (*Transport, func()) {
	t.Helper()
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		handler(conn)
	}))
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		srv.Close()
		t.Fatalf("dial: %v", err)
	}
	tp := NewTransport(client, "test-stream-sid")
	return tp, func() { client.Close(); srv.Close() }
}

func TestTransport_Receive_DecodesMediaFrames(t *testing.T) {
	tp, cleanup := dialTestServer(t, func(conn *websocket.Conn) {
		for i := 0; i < 3; i++ {
			payload := base64.StdEncoding.EncodeToString([]byte{byte(i), 0xFF})
			_ = conn.WriteJSON(TwilioMessage{
				Event: "media",
				Media: &TwilioMedia{Payload: payload},
			})
		}
		_ = conn.WriteJSON(TwilioMessage{Event: "stop"})
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	frames, _ := tp.Receive(ctx)

	var got [][]byte
	for f := range frames {
		got = append(got, f.Data)
	}
	if len(got) != 3 {
		t.Fatalf("got %d frames, want 3", len(got))
	}
	for i, f := range got {
		if len(f) != 2 || f[0] != byte(i) || f[1] != 0xFF {
			t.Errorf("frame[%d]=%v", i, f)
		}
	}
}

func TestTransport_StopClosesBothChannels(t *testing.T) {
	tp, cleanup := dialTestServer(t, func(conn *websocket.Conn) {
		_ = conn.WriteJSON(TwilioMessage{Event: "stop"})
	})
	defer cleanup()

	ctx := context.Background()
	frames, errs := tp.Receive(ctx)

	select {
	case _, ok := <-frames:
		if ok {
			t.Error("expected frames channel closed after stop")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("frames channel never closed")
	}
	select {
	case err, ok := <-errs:
		if ok && err != nil {
			t.Errorf("expected clean close, got err=%v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("errs channel never closed")
	}
}

func TestTransport_InboundAndOutboundFormats(t *testing.T) {
	tp := &Transport{streamSid: "x"}
	in := tp.InboundFormat()
	if in.Encoding != pipeline.EncodingMulaw || in.SampleRate != 8000 || in.Channels != 1 {
		t.Errorf("inbound=%+v", in)
	}
	out := tp.OutboundFormat()
	if out.Encoding != pipeline.EncodingMulaw || out.SampleRate != 8000 || out.Channels != 1 {
		t.Errorf("outbound=%+v", out)
	}
}


func TestTransport_ImplementsPipelineTransport(t *testing.T) {
	var _ pipeline.Transport = (*Transport)(nil)
}

func TestTransport_Send_EmitsMediaEvent(t *testing.T) {
	received := make(chan TwilioOutMessage, 1)
	tp, cleanup := dialTestServer(t, func(conn *websocket.Conn) {
		var m TwilioOutMessage
		_ = conn.ReadJSON(&m)
		received <- m
		<-make(chan struct{}) // block forever
	})
	defer cleanup()

	frame := pipeline.Frame{Data: []byte{0xAA, 0xBB, 0xCC}}
	if err := tp.Send(context.Background(), frame); err != nil {
		t.Fatalf("Send err: %v", err)
	}

	select {
	case got := <-received:
		if got.Event != "media" || got.StreamSid != "test-stream-sid" {
			t.Errorf("got=%+v", got)
		}
		decoded, _ := base64.StdEncoding.DecodeString(got.Media.Payload)
		if len(decoded) != 3 || decoded[0] != 0xAA || decoded[2] != 0xCC {
			t.Errorf("decoded payload=%v", decoded)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("no message received")
	}
}

func TestTransport_Clear_EmitsClearEvent(t *testing.T) {
	received := make(chan TwilioOutMessage, 1)
	tp, cleanup := dialTestServer(t, func(conn *websocket.Conn) {
		var m TwilioOutMessage
		_ = conn.ReadJSON(&m)
		received <- m
		<-make(chan struct{})
	})
	defer cleanup()

	if err := tp.Clear(context.Background()); err != nil {
		t.Fatalf("Clear err: %v", err)
	}
	got := <-received
	if got.Event != "clear" || got.StreamSid != "test-stream-sid" {
		t.Errorf("got=%+v", got)
	}
}

func TestTransport_Mark_EmitsMarkEventWithName(t *testing.T) {
	received := make(chan TwilioOutMessage, 1)
	tp, cleanup := dialTestServer(t, func(conn *websocket.Conn) {
		var m TwilioOutMessage
		_ = conn.ReadJSON(&m)
		received <- m
		<-make(chan struct{})
	})
	defer cleanup()

	if err := tp.Mark(context.Background(), "turn-3"); err != nil {
		t.Fatalf("Mark err: %v", err)
	}
	got := <-received
	if got.Event != "mark" || got.Mark == nil || got.Mark.Name != "turn-3" {
		t.Errorf("got=%+v", got)
	}
}

func TestTransport_Close_IsIdempotent(t *testing.T) {
	tp, cleanup := dialTestServer(t, func(conn *websocket.Conn) { <-make(chan struct{}) })
	defer cleanup()
	if err := tp.Close(); err != nil {
		t.Errorf("first Close err: %v", err)
	}
	if err := tp.Close(); err != nil {
		t.Errorf("second Close err: %v (expected nil for idempotent close)", err)
	}
}
