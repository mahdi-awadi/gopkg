package twilio

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
	return tp, func() { _ = client.Close(); srv.Close() }
}

func TestTransport_ImplementsPipelineTransport(t *testing.T) {
	var _ pipeline.Transport = (*Transport)(nil)
}

func TestTransport_FormatsAreTwilioMulaw8kMono(t *testing.T) {
	tp := &Transport{}
	for name, got := range map[string]pipeline.AudioFormat{
		"inbound":  tp.InboundFormat(),
		"outbound": tp.OutboundFormat(),
	} {
		if got.Encoding != pipeline.EncodingMulaw || got.SampleRate != 8000 || got.Channels != 1 {
			t.Fatalf("%s format=%+v, want mulaw@8000 mono", name, got)
		}
	}
}

func TestTransport_ReceiveDecodesMediaFramesAndStopsCleanly(t *testing.T) {
	tp, cleanup := dialTestServer(t, func(conn *websocket.Conn) {
		for i := 0; i < 3; i++ {
			payload := base64.StdEncoding.EncodeToString([]byte{byte(i), 0xFF})
			_ = conn.WriteJSON(Message{Event: "media", Media: &Media{Payload: payload}})
		}
		_ = conn.WriteJSON(Message{Event: "stop"})
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	frames, errs := tp.Receive(ctx)

	var got [][]byte
	for f := range frames {
		got = append(got, f.Data)
	}
	if len(got) != 3 {
		t.Fatalf("got %d frames, want 3", len(got))
	}
	for i, f := range got {
		if len(f) != 2 || f[0] != byte(i) || f[1] != 0xFF {
			t.Fatalf("frame[%d]=%v", i, f)
		}
	}
	if err, ok := <-errs; ok && err != nil {
		t.Fatalf("clean stop returned err=%v", err)
	}
}

func TestTransport_SendClearAndMarkWriteTwilioEvents(t *testing.T) {
	received := make(chan OutMessage, 3)
	tp, cleanup := dialTestServer(t, func(conn *websocket.Conn) {
		for i := 0; i < 3; i++ {
			var m OutMessage
			if err := conn.ReadJSON(&m); err != nil {
				return
			}
			received <- m
		}
	})
	defer cleanup()

	if err := tp.Send(context.Background(), pipeline.Frame{Data: []byte{0xAA, 0xBB}}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if err := tp.Clear(context.Background()); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if err := tp.Mark(context.Background(), "turn-1"); err != nil {
		t.Fatalf("Mark: %v", err)
	}

	media := <-received
	if media.Event != "media" || media.StreamSid != "test-stream-sid" || media.Media == nil {
		t.Fatalf("media message=%+v", media)
	}
	decoded, err := base64.StdEncoding.DecodeString(media.Media.Payload)
	if err != nil || string(decoded) != string([]byte{0xAA, 0xBB}) {
		t.Fatalf("media payload decoded=%v err=%v", decoded, err)
	}
	clear := <-received
	if clear.Event != "clear" || clear.StreamSid != "test-stream-sid" {
		t.Fatalf("clear message=%+v", clear)
	}
	mark := <-received
	if mark.Event != "mark" || mark.Mark == nil || mark.Mark.Name != "turn-1" {
		t.Fatalf("mark message=%+v", mark)
	}
}

func TestTransport_CloseIsIdempotent(t *testing.T) {
	tp, cleanup := dialTestServer(t, func(conn *websocket.Conn) {
		_, _, _ = conn.ReadMessage()
	})
	defer cleanup()
	if err := tp.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := tp.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestReadStartConsumesConnectedAndStart(t *testing.T) {
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		_ = conn.WriteJSON(Message{Event: "connected"})
		_ = conn.WriteJSON(Message{Event: "start", Start: &Start{
			StreamSid: "MZ123",
			CallSid:   "CA123",
			CustomParameters: map[string]string{
				"from_number": "+12025550199",
			},
		}})
	}))
	defer srv.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	start, err := ReadStart(context.Background(), conn)
	if err != nil {
		t.Fatalf("ReadStart: %v", err)
	}
	if start.StreamSid != "MZ123" || start.CallSid != "CA123" || start.CustomParameters["from_number"] == "" {
		t.Fatalf("start=%+v", start)
	}
}
