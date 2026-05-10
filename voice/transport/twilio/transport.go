package twiliotransport

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mahdi-awadi/gopkg/voice/pipeline"
)

// Transport is a pipeline.Transport over a Twilio Media Streams
// WebSocket. Owns the *websocket.Conn — callers transfer ownership at
// NewTransport time (Transport.Close() closes the underlying conn).
//
// Concurrency: Send / Clear / Mark / Close acquire writeMu so concurrent
// callers (LLM-events goroutine + hold-filler pump goroutine in the
// pipeline) don't interleave WS writes. gorilla websocket.Conn is NOT
// safe for concurrent writes — writeMu is load-bearing.
type Transport struct {
	conn      *websocket.Conn
	streamSid string

	writeMu sync.Mutex
	closed  bool
}

// NewTransport builds a Transport from an already-upgraded Twilio
// Media Streams WS connection. The caller (typically HandleMediaStream)
// has already consumed the pre-stream "connected" + "start" handshake
// events and extracted the StreamSid for use on outbound messages.
func NewTransport(conn *websocket.Conn, streamSid string) *Transport {
	return &Transport{conn: conn, streamSid: streamSid}
}

// InboundFormat reports Twilio's fixed delivery format: mulaw@8k mono.
func (t *Transport) InboundFormat() pipeline.AudioFormat {
	return pipeline.AudioFormat{
		Encoding:   pipeline.EncodingMulaw,
		SampleRate: 8000,
		Channels:   1,
	}
}

// OutboundFormat reports the format Twilio expects on outbound media:
// mulaw@8k mono, same as InboundFormat.
func (t *Transport) OutboundFormat() pipeline.AudioFormat {
	return pipeline.AudioFormat{
		Encoding:   pipeline.EncodingMulaw,
		SampleRate: 8000,
		Channels:   1,
	}
}

// Receive reads Twilio WS messages, decodes "media" events into
// Frames, and exits cleanly on "stop" or ctx cancellation. Both returned
// channels close on exit; errs carries the final read error (nil for
// clean close, the WS error for abnormal close).
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
			var msg TwilioMessage
			if err := json.Unmarshal(raw, &msg); err != nil {
				continue // skip malformed JSON, keep reading
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
			// "mark" ack events are informational; swallowed here. The
			// pipeline's Mark() side already advanced before Twilio
			// echoed it back.
		}
	}()
	return frames, errs
}


// Send writes one outbound media frame as a TwilioOutMessage.
func (t *Transport) Send(ctx context.Context, f pipeline.Frame) error {
	_ = ctx // Twilio WS writes honor the underlying conn deadline, not ctx.
	t.writeMu.Lock()
	defer t.writeMu.Unlock()
	return t.conn.WriteJSON(TwilioOutMessage{
		Event:     "media",
		StreamSid: t.streamSid,
		Media: &TwilioOutMedia{
			Payload: base64.StdEncoding.EncodeToString(f.Data),
		},
	})
}

// Clear flushes any buffered outbound audio on Twilio's end (used after
// an interruption to prevent the AI's cut-off sentence from leaking
// through to the caller).
func (t *Transport) Clear(ctx context.Context) error {
	_ = ctx
	t.writeMu.Lock()
	defer t.writeMu.Unlock()
	return t.conn.WriteJSON(TwilioOutMessage{
		Event:     "clear",
		StreamSid: t.streamSid,
	})
}

// Mark sends a named checkpoint. Twilio echoes the mark back when the
// audio prior to it has finished playing through the caller's device.
// The pipeline uses this for turn-complete tracking ("turn-1", "turn-2"
// ...); see gopkg/voice/pipeline's dispatchLLMEvent for EventTurnComplete.
func (t *Transport) Mark(ctx context.Context, name string) error {
	_ = ctx
	t.writeMu.Lock()
	defer t.writeMu.Unlock()
	return t.conn.WriteJSON(TwilioOutMessage{
		Event:     "mark",
		StreamSid: t.streamSid,
		Mark:      &TwilioMark{Name: name},
	})
}

// Close is idempotent. Subsequent calls return nil even if the
// underlying WS has already been closed.
func (t *Transport) Close() error {
	t.writeMu.Lock()
	if t.closed {
		t.writeMu.Unlock()
		return nil
	}
	t.closed = true
	t.writeMu.Unlock()
	return t.conn.Close()
}

// Compile-time interface check.
var _ pipeline.Transport = (*Transport)(nil)

