package pipeline

import "time"

// AudioFormat describes the wire format a Transport or LLM speaks.
type AudioFormat struct {
	Encoding   Encoding
	SampleRate int
	Channels   int
}

// Encoding is the codec identifier used in AudioFormat.
type Encoding string

const (
	EncodingMulaw   Encoding = "mulaw"
	EncodingPCM16LE Encoding = "pcm16le"
)

// Frame is one chunk of audio flowing through the pipeline.
// Format is a property of the stream (declared by the adapter),
// not of the individual frame.
type Frame struct {
	Data      []byte
	Timestamp time.Time
}

// Role tags a HistoryTurn as user- or assistant-originated.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// HistoryTurn is a past utterance replayed into a new session.
type HistoryTurn struct {
	Role    Role
	Content string
}

// ToolCall is an LLM-initiated function invocation request.
type ToolCall struct {
	ID   string
	Name string
	Args map[string]any
}

// ToolResult is an executor's reply to a ToolCall.
type ToolResult struct {
	CallID string
	Data   any
	Err    error
}

// ToolDecl declares a tool available to the LLM.
type ToolDecl struct {
	Name        string
	Description string
	Parameters  ToolSchema
}

// ToolSchema is a minimal JSON-Schema subset for tool parameters.
type ToolSchema struct {
	Type       string
	Properties map[string]ToolProperty
	Required   []string
}

// ToolProperty describes a single parameter of a tool.
type ToolProperty struct {
	Type        string
	Description string
	Enum        []string
	Format      string
}

// Session carries metadata passed through every Observer callback.
type Session struct {
	ID        string
	StartedAt time.Time
	Attrs     map[string]string
}

// EndReason enumerates why a session terminated.
type EndReason string

const (
	EndReasonTransportClosed EndReason = "transport_closed"
	EndReasonLLMClosed       EndReason = "llm_closed"
	EndReasonContextDone     EndReason = "context_done"
	EndReasonFatalError      EndReason = "fatal_error"
)

// SetupRequest is the payload handed to LLM.Open.
type SetupRequest struct {
	SystemPrompt string
	Tools        []ToolDecl
	History      []HistoryTurn
	LocaleHint   string
	Voice        string
	Extra        map[string]any
}
