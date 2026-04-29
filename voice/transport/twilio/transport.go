package twilio

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mahdi-awadi/gopkg/voice/pipeline"
)

// Transport is a pipeline.Transport over a Twilio Media Streams WebSocket.
// It owns the supplied connection. Send, Clear, Mark, and Close serialize
// writes because gorilla/websocket does not permit concurrent writers.
type Transport struct {
	conn      *websocket.Conn
	streamSid string

	writeMu sync.Mutex
	closed  bool
}

// NewTransport builds a Transport from an already-upgraded Twilio Media
// Streams connection. The caller should pass the stream SID from the start
// event; ReadStart is provided as a convenience helper.
func NewTransport(conn *websocket.Conn, streamSid string) *Transport {
	return &Transport{conn: conn, streamSid: streamSid}
}

// ReadStart consumes Twilio's initial connected and start messages and returns
// the parsed start payload.
func ReadStart(ctx context.Context, conn *websocket.Conn) (*Start, error) {
	if conn == nil {
		return nil, errors.New("twilio transport: nil websocket connection")
	}
	if err := expectEvent(ctx, conn, "connected", nil); err != nil {
		return nil, err
	}
	var start *Start
	if err := expectEvent(ctx, conn, "start", func(m Message) {
		start = m.Start
	}); err != nil {
		return nil, err
	}
	if start == nil {
		return nil, errors.New("twilio transport: start event missing payload")
	}
	return start, nil
}

func expectEvent(ctx context.Context, conn *websocket.Conn, event string, capture func(Message)) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	_, raw, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("twilio transport: read %s: %w", event, err)
	}
	var msg Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		return fmt.Errorf("twilio transport: decode %s: %w", event, err)
	}
	if msg.Event != event {
		return fmt.Errorf("twilio transport: expected %s event, got %q", event, msg.Event)
	}
	if capture != nil {
		capture(msg)
	}
	return nil
}

// InboundFormat reports Twilio's fixed caller-audio format.
func (t *Transport) InboundFormat() pipeline.AudioFormat {
	return pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1}
}

// OutboundFormat reports Twilio's fixed outbound-audio format.
func (t *Transport) OutboundFormat() pipeline.AudioFormat {
	return pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1}
}

// Receive reads Twilio media messages and emits raw mu-law frames. Malformed
// media messages are skipped; stop closes the stream cleanly.
func (t *Transport) Receive(ctx context.Context) (<-chan pipeline.Frame, <-chan error) {
	frames := make(chan pipeline.Frame)
	errs := make(chan error, 1)
	go func() {
		defer close(frames)
		defer close(errs)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			_, raw, err := t.conn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					errs <- err
				}
				return
			}
			var msg Message
			if err := json.Unmarshal(raw, &msg); err != nil {
				continue
			}
			switch msg.Event {
			case "media":
				if msg.Media == nil || msg.Media.Payload == "" {
					continue
				}
				data, err := base64.StdEncoding.DecodeString(msg.Media.Payload)
				if err != nil {
					continue
				}
				select {
				case frames <- pipeline.Frame{Data: data, Timestamp: time.Now()}:
				case <-ctx.Done():
					return
				}
			case "stop":
				return
			}
		}
	}()
	return frames, errs
}

// Send writes one outbound media frame to Twilio.
func (t *Transport) Send(_ context.Context, f pipeline.Frame) error {
	t.writeMu.Lock()
	defer t.writeMu.Unlock()
	return t.conn.WriteJSON(OutMessage{
		Event:     "media",
		StreamSid: t.streamSid,
		Media:     &OutMedia{Payload: base64.StdEncoding.EncodeToString(f.Data)},
	})
}

// Clear flushes buffered outbound audio on Twilio's side.
func (t *Transport) Clear(_ context.Context) error {
	t.writeMu.Lock()
	defer t.writeMu.Unlock()
	return t.conn.WriteJSON(OutMessage{Event: "clear", StreamSid: t.streamSid})
}

// Mark sends a playback checkpoint. Twilio echoes it when prior audio has
// played through.
func (t *Transport) Mark(_ context.Context, name string) error {
	t.writeMu.Lock()
	defer t.writeMu.Unlock()
	return t.conn.WriteJSON(OutMessage{
		Event:     "mark",
		StreamSid: t.streamSid,
		Mark:      &Mark{Name: name},
	})
}

// Close closes the underlying WebSocket once.
func (t *Transport) Close() error {
	t.writeMu.Lock()
	if t.closed {
		t.writeMu.Unlock()
		return nil
	}
	t.closed = true
	t.writeMu.Unlock()
	if t.conn == nil {
		return nil
	}
	return t.conn.Close()
}

var _ pipeline.Transport = (*Transport)(nil)
