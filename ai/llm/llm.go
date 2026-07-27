// Package llm defines the Provider interface for text-mode LLM adapters.
package llm

import (
	"context"
	"errors"
)

// ChatRequest is one inbound user message plus optional history and tools.
type ChatRequest struct {
	SystemPrompt string
	History      []ChatTurn
	UserText     string
	Tools        []ToolDecl
	LocaleHint   string
}

// ChatTurn is one prior turn replayed into the model.
type ChatTurn struct {
	Role       string
	Text       string
	ToolName   string
	ToolArgs   map[string]any
	ToolResult any
}

// ChatResponse is the model's reply.
type ChatResponse struct {
	Text         string
	ToolCalls    []ToolCall
	FinishReason string
	Usage        TokenUsage
}

// ToolCall is a single function-call request from the model.
type ToolCall struct {
	ID   string
	Name string
	Args map[string]any
}

// ToolDecl declares a tool the model may call.
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
	// Items describes the element schema when Type == "array". Required by
	// providers such as Gemini, which reject an array parameter that omits it.
	Items *ToolProperty
}

// TokenUsage is best-effort token accounting.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Provider is implemented by every text LLM adapter.
type Provider interface {
	Name() string
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}

// ErrNoProviders is returned when a Registry is empty.
var ErrNoProviders = errors.New("llm: no providers registered")
