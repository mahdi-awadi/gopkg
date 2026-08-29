package gemini

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/mahdi-awadi/gopkg/voice/pipeline"
)

// LLM is a pipeline.LLM over a Gemini Live WebSocket.
//
// Concurrency: all writes acquire writeMu (SendAudio + SendToolResults +
// InjectTurn run concurrently from pipeline goroutines — gorilla
// websocket forbids concurrent writes).
type LLM struct {
	conn *websocket.Conn

	writeMu sync.Mutex
	closed  bool
}

// NewLLM constructs an LLM over an already-dialed Gemini Live WS. The
// dial (with API key + URL) is done by the caller — this keeps the
// adapter independent of how credentials are sourced. Per-session
// configuration (system prompt, tool list, history, optional wake
// signal) is supplied via pipeline.SetupRequest at Open time.
func NewLLM(conn *websocket.Conn) *LLM {
	return &LLM{conn: conn}
}

// InboundFormat — format of audio the LLM EMITS to us via EventAudioOut.
// Gemini Live emits pcm16le @ 24 kHz. Matches the gopkg naming convention
// from Transport: "Inbound" = what enters our pipeline from this endpoint.
func (l *LLM) InboundFormat() pipeline.AudioFormat {
	return pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 24000, Channels: 1}
}

// OutboundFormat — format of audio we SEND to the LLM via SendAudio.
// Gemini Live expects pcm16le @ 16 kHz (see also SendAudio's mimeType
// "audio/pcm;rate=16000"). Matches gopkg's "Outbound" = what leaves our
// pipeline toward this endpoint.
func (l *LLM) OutboundFormat() pipeline.AudioFormat {
	return pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 16000, Channels: 1}
}

// Open sends the setup envelope, waits for setupComplete, replays
// user SetupRequest.History as clientContent messages, then — if
// SetupRequest.Extra["wake_signal"] is a non-empty string — sends it
// as a clientContent user turn to wake the model into producing its
// first response. The wake payload should be a non-natural-language
// tag (e.g. "[CALL_CONNECTED]") so the model treats it as a system
// signal and not as user speech. Without a wake signal the model
// stays silent until the caller speaks.
func (l *LLM) Open(_ context.Context, setup pipeline.SetupRequest) error {
	if err := l.writeJSON(BuildSetup(setup.SystemPrompt, setup.Tools)); err != nil {
		return fmt.Errorf("send setup: %w", err)
	}

	var resp GeminiServerMessage
	if err := l.conn.ReadJSON(&resp); err != nil {
		return fmt.Errorf("read setupComplete: %w", err)
	}
	if resp.SetupComplete == nil {
		return fmt.Errorf("expected setupComplete, got: %+v", resp)
	}

	// Replay only user history. Gemini Live rejects clientContent turns
	// with role="model"; assistant output history arrives from the server
	// stream, not as client input.
	for _, turn := range setup.History {
		if turn.Content == "" || turn.Role != pipeline.RoleUser {
			continue
		}
		err := l.writeJSON(GeminiClientContent{
			ClientContent: GeminiClientContentData{
				TurnComplete: true,
				Turns: []GeminiTurn{{
					Role:  "user",
					Parts: []GeminiPart{{Text: turn.Content}},
				}},
			},
		})
		if err != nil {
			return fmt.Errorf("send history turn: %w", err)
		}
	}

	// Optional non-content wake signal — caller-supplied via Extra.
	// Sent as a clientContent user turn so the model produces its first
	// response based on the system prompt. Use a bracketed tag like
	// "[CALL_CONNECTED]" — anything that's clearly not natural language —
	// to avoid the role-inversion bug where the model treats the wake
	// payload as something the user said.
	if wake, _ := setup.Extra["wake_signal"].(string); wake != "" {
		if err := l.writeJSON(GeminiClientContent{
			ClientContent: GeminiClientContentData{
				TurnComplete: true,
				Turns: []GeminiTurn{{
					Role:  "user",
					Parts: []GeminiPart{{Text: wake}},
				}},
			},
		}); err != nil {
			return fmt.Errorf("send wake signal: %w", err)
		}
	}
	return nil
}

// writeJSON acquires writeMu and writes a JSON message to the WS.
func (l *LLM) writeJSON(v any) error {
	l.writeMu.Lock()
	defer l.writeMu.Unlock()
	return l.conn.WriteJSON(v)
}

// SendAudio writes one pcm16@16k frame as a Gemini realtimeInput.audio
// message. Base64-encodes the raw pcm16 bytes; Gemini's mimeType is
// the canonical "audio/pcm;rate=16000".
func (l *LLM) SendAudio(_ context.Context, f pipeline.Frame) error {
	return l.writeJSON(GeminiRealtimeInput{
		RealtimeInput: GeminiRealtimeInputData{
			Audio: &GeminiAudioData{
				Data:     base64.StdEncoding.EncodeToString(f.Data),
				MimeType: "audio/pcm;rate=16000",
			},
		},
	})
}

// SendToolResults writes a single toolResponse envelope containing all
// results in the batch. Gemini Live expects the whole batch in one
// message per call; the pipeline already collects results per batch
// before invoking this method.
func (l *LLM) SendToolResults(_ context.Context, results []pipeline.ToolResult) error {
	fr := make([]GeminiFunctionResponse, 0, len(results))
	for _, r := range results {
		response := any(r.Data)
		if r.Err != nil {
			response = map[string]any{"error": r.Err.Error()}
		}
		fr = append(fr, GeminiFunctionResponse{ID: r.CallID, Response: response})
	}
	return l.writeJSON(GeminiToolResponse{
		ToolResponse: GeminiToolResponseData{FunctionResponses: fr},
	})
}

// InjectTurn pushes a single clientContent turn mid-session.
func (l *LLM) InjectTurn(_ context.Context, turn pipeline.HistoryTurn) error {
	role := "user"
	if turn.Role == pipeline.RoleAssistant {
		role = "model"
	}
	return l.writeJSON(GeminiClientContent{
		ClientContent: GeminiClientContentData{
			TurnComplete: true,
			Turns: []GeminiTurn{{
				Role:  role,
				Parts: []GeminiPart{{Text: turn.Content}},
			}},
		},
	})
}

// Events reads Gemini server messages and translates each into one or
// more pipeline.LLMEvent values, in wire order. Exits when the WS
// closes, ctx is cancelled, or a read error occurs.
//
// Per-message translation rules:
//   - serverContent.inputTranscription.text → EventCallerTranscript
//   - serverContent.outputTranscription.text → EventAssistantText{Final:false}
//   - serverContent.modelTurn.parts[i].inlineData.data → EventAudioOut
//     (one event per part; Data is raw base64-decoded pcm16@24k bytes)
//   - serverContent.modelTurn.parts[i].text → EventAssistantText{Final:false}
//     (streaming); accumulated per turn and re-emitted with Final:true
//     on turnComplete
//   - serverContent.turnComplete → (flush accumulated text as Final:true)
//   - EventTurnComplete
//   - serverContent.interrupted → EventInterrupted
//   - toolCall.functionCalls → EventToolCalls
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
			var msg GeminiServerMessage
			if err := json.Unmarshal(raw, &msg); err != nil {
				continue
			}

			if msg.ServerContent != nil {
				if tx := msg.ServerContent.InputTranscription; tx != nil && tx.Text != "" {
					if !emit(pipeline.EventCallerTranscript{Text: tx.Text}) {
						return
					}
				}
				if tx := msg.ServerContent.OutputTranscription; tx != nil && tx.Text != "" {
					turnText.WriteString(tx.Text)
					if !emit(pipeline.EventAssistantText{Text: tx.Text, Final: false}) {
						return
					}
				}
				if mt := msg.ServerContent.ModelTurn; mt != nil {
					for _, part := range mt.Parts {
						if part.InlineData != nil && part.InlineData.Data != "" {
							data, derr := base64.StdEncoding.DecodeString(part.InlineData.Data)
							if derr != nil {
								continue
							}
							if !emit(pipeline.EventAudioOut{Frame: pipeline.Frame{Data: data}}) {
								return
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

// Close is idempotent.
func (l *LLM) Close() error {
	l.writeMu.Lock()
	if l.closed {
		l.writeMu.Unlock()
		return nil
	}
	l.closed = true
	l.writeMu.Unlock()
	return l.conn.Close()
}

// Compile-time interface check.
var _ pipeline.LLM = (*LLM)(nil)
