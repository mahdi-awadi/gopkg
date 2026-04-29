package gemini

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/mahdi-awadi/gopkg/voice/pipeline"
)

// Config controls Gemini Live WebSocket dialing.
type Config struct {
	Endpoint string
	APIKey   string
}

// BuildURL returns the Gemini Live WebSocket URL with the API key query
// parameter attached. The returned value is sensitive; use RedactURL for logs.
func BuildURL(cfg Config) (string, error) {
	if cfg.APIKey == "" {
		return "", errors.New("gemini live: APIKey is required")
	}
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("gemini live: parse endpoint: %w", err)
	}
	q := u.Query()
	q.Set("key", cfg.APIKey)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// RedactURL removes the Gemini API key from a URL for safe logging.
func RedactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "<redacted>"
	}
	q := u.Query()
	if q.Has("key") {
		q.Set("key", "REDACTED")
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// Dial opens a Gemini Live WebSocket connection using websocket.DefaultDialer.
func Dial(ctx context.Context, cfg Config) (*websocket.Conn, error) {
	rawURL, err := BuildURL(cfg)
	if err != nil {
		return nil, err
	}
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("gemini live: dial %s: %w", RedactURL(rawURL), err)
	}
	return conn, nil
}

// Options configures the LLM adapter. Secrets do not live here; pass them to
// Dial via Config or construct the WebSocket connection yourself.
type Options struct {
	Model                    string
	VoiceName                string
	LanguageCode             string
	SystemPrompt             string
	GreetingText             string
	EnableInputTranscription bool
}

// LLM implements pipeline.LLM over a Gemini Live WebSocket. It owns the
// supplied connection.
type LLM struct {
	conn *websocket.Conn
	opts Options

	writeMu sync.Mutex
	closed  bool
}

// NewLLM constructs an adapter around an already-open Gemini Live WebSocket.
func NewLLM(conn *websocket.Conn, opts Options) *LLM {
	return &LLM{conn: conn, opts: opts}
}

// InboundFormat is audio emitted by Gemini Live.
func (l *LLM) InboundFormat() pipeline.AudioFormat {
	return pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 24000, Channels: 1}
}

// OutboundFormat is audio sent to Gemini Live.
func (l *LLM) OutboundFormat() pipeline.AudioFormat {
	return pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 16000, Channels: 1}
}

// Open sends setup, waits for setupComplete, replays history, then sends a
// text greeting trigger.
func (l *LLM) Open(_ context.Context, setup pipeline.SetupRequest) error {
	if err := l.writeJSON(buildSetup(setup, l.opts)); err != nil {
		return fmt.Errorf("gemini live: send setup: %w", err)
	}
	var resp ServerMessage
	if err := l.conn.ReadJSON(&resp); err != nil {
		return fmt.Errorf("gemini live: read setupComplete: %w", err)
	}
	if resp.SetupComplete == nil {
		return missingSetupComplete(resp)
	}
	for _, turn := range setup.History {
		if strings.TrimSpace(turn.Content) == "" {
			continue
		}
		if err := l.writeJSON(historyMessage(turn)); err != nil {
			return fmt.Errorf("gemini live: send history turn: %w", err)
		}
	}
	if err := l.writeJSON(RealtimeInput{RealtimeInput: RealtimeInputData{Text: greetingText(setup, l.opts)}}); err != nil {
		return fmt.Errorf("gemini live: send greeting trigger: %w", err)
	}
	return nil
}

func (l *LLM) writeJSON(v any) error {
	l.writeMu.Lock()
	defer l.writeMu.Unlock()
	return l.conn.WriteJSON(v)
}

// SendAudio sends one pcm16le@16k audio frame.
func (l *LLM) SendAudio(_ context.Context, f pipeline.Frame) error {
	return l.writeJSON(RealtimeInput{RealtimeInput: RealtimeInputData{
		Audio: &AudioData{
			Data:     base64.StdEncoding.EncodeToString(f.Data),
			MimeType: "audio/pcm;rate=16000",
		},
	}})
}

// SendToolResults sends a Gemini toolResponse with all results in the batch.
func (l *LLM) SendToolResults(_ context.Context, results []pipeline.ToolResult) error {
	responses := make([]FunctionResponse, 0, len(results))
	for _, r := range results {
		response := r.Data
		if r.Err != nil {
			response = map[string]any{"error": r.Err.Error()}
		}
		responses = append(responses, FunctionResponse{ID: r.CallID, Response: response})
	}
	return l.writeJSON(ToolResponse{ToolResponse: ToolResponseData{FunctionResponses: responses}})
}

// InjectTurn appends a text turn to the live conversation.
func (l *LLM) InjectTurn(_ context.Context, turn pipeline.HistoryTurn) error {
	return l.writeJSON(historyMessage(turn))
}

// Events translates Gemini Live server messages into pipeline events.
func (l *LLM) Events(ctx context.Context) (<-chan pipeline.LLMEvent, <-chan error) {
	events := make(chan pipeline.LLMEvent)
	errs := make(chan error, 1)
	go func() {
		defer close(events)
		defer close(errs)

		var turnText strings.Builder
		emit := func(ev pipeline.LLMEvent) bool {
			select {
			case events <- ev:
				return true
			case <-ctx.Done():
				return false
			}
		}

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			_, raw, err := l.conn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					errs <- err
				}
				return
			}
			var msg ServerMessage
			if err := json.Unmarshal(raw, &msg); err != nil {
				continue
			}
			if msg.ServerContent != nil {
				if tx := msg.ServerContent.InputTranscription; tx != nil && tx.Text != "" {
					if !emit(pipeline.EventCallerTranscript{Text: tx.Text}) {
						return
					}
				}
				if mt := msg.ServerContent.ModelTurn; mt != nil {
					for _, part := range mt.Parts {
						if part.InlineData != nil && part.InlineData.Data != "" {
							data, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
							if err == nil {
								if !emit(pipeline.EventAudioOut{Frame: pipeline.Frame{Data: data}}) {
									return
								}
							}
						}
						if part.Text != "" {
							turnText.WriteString(part.Text)
							if !emit(pipeline.EventAssistantText{Text: part.Text, Final: false}) {
								return
							}
						}
					}
				}
				if msg.ServerContent.Interrupted {
					if !emit(pipeline.EventInterrupted{}) {
						return
					}
				}
				if msg.ServerContent.TurnComplete {
					if turnText.Len() > 0 {
						text := turnText.String()
						turnText.Reset()
						if !emit(pipeline.EventAssistantText{Text: text, Final: true}) {
							return
						}
					}
					if !emit(pipeline.EventTurnComplete{}) {
						return
					}
				}
			}
			if msg.ToolCall != nil && len(msg.ToolCall.FunctionCalls) > 0 {
				calls := make([]pipeline.ToolCall, 0, len(msg.ToolCall.FunctionCalls))
				for _, fc := range msg.ToolCall.FunctionCalls {
					calls = append(calls, pipeline.ToolCall{ID: fc.ID, Name: fc.Name, Args: fc.Args})
				}
				if !emit(pipeline.EventToolCalls{Calls: calls}) {
					return
				}
			}
		}
	}()
	return events, errs
}

// Close closes the underlying WebSocket once.
func (l *LLM) Close() error {
	l.writeMu.Lock()
	if l.closed {
		l.writeMu.Unlock()
		return nil
	}
	l.closed = true
	l.writeMu.Unlock()
	if l.conn == nil {
		return nil
	}
	return l.conn.Close()
}

var _ pipeline.LLM = (*LLM)(nil)
