package pipeline

import "context"

// LLM is the realtime-LLM session abstraction (Gemini Live,
// OpenAI Realtime, …). The pipeline drives audio in via SendAudio
// and reads events (audio out, text, tool calls, interruptions,
// turn completion) from Events.
type LLM interface {
	InboundFormat() AudioFormat
	OutboundFormat() AudioFormat

	// Open establishes the session. Implementations SHOULD send the
	// setup payload AND any SetupRequest.History turns before returning.
	Open(ctx context.Context, setup SetupRequest) error

	// SendAudio pushes one frame of caller audio to the LLM.
	SendAudio(ctx context.Context, f Frame) error

	// SendToolResults delivers all results for a batch of tool calls
	// in one wire message (adapters batch if their protocol allows).
	SendToolResults(ctx context.Context, results []ToolResult) error

	// InjectTurn pushes a text turn into the live conversation
	// history mid-session. Rare; most consumers use SetupRequest.History.
	InjectTurn(ctx context.Context, turn HistoryTurn) error

	// Events returns the LLM's event stream and a terminal error.
	// Both channels close when the session ends.
	Events(ctx context.Context) (<-chan LLMEvent, <-chan error)

	// Close releases LLM resources. Idempotent.
	Close() error
}

// LLMEvent is a sealed sum type. Variants below each implement it
// with a no-op method so switch statements cover the full set.
type LLMEvent interface{ isLLMEvent() }

// EventAudioOut carries one frame of LLM-generated audio.
type EventAudioOut struct{ Frame Frame }

// EventAssistantText carries an assistant-text snippet.
// Final=true on turn completion; Final=false for streaming deltas
// (adapters without deltas always set Final=true).
type EventAssistantText struct {
	Text  string
	Final bool
}

// EventCallerTranscript carries recognized caller speech.
type EventCallerTranscript struct{ Text string }

// EventTurnComplete fires when the assistant finishes a turn.
type EventTurnComplete struct{}

// EventInterrupted fires when the LLM detects caller interruption.
type EventInterrupted struct{}

// EventToolCalls carries one or more LLM-requested tool invocations.
type EventToolCalls struct{ Calls []ToolCall }

func (EventAudioOut) isLLMEvent()         {}
func (EventAssistantText) isLLMEvent()    {}
func (EventCallerTranscript) isLLMEvent() {}
func (EventTurnComplete) isLLMEvent()     {}
func (EventInterrupted) isLLMEvent()      {}
func (EventToolCalls) isLLMEvent()        {}

// ToolExecutor runs a single tool call. The pipeline invokes this in
// a goroutine; implementations must honor ctx cancellation.
type ToolExecutor interface {
	Execute(ctx context.Context, call ToolCall, s Session) (any, error)
}
