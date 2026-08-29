package orchestrator

import (
	"context"
	"strings"
	"testing"

	"github.com/mahdi-awadi/gopkg/ai/llm"
)

type stubProvider struct {
	responses []llm.ChatResponse
	requests  []llm.ChatRequest
	idx       int
}

func (s *stubProvider) Name() string { return "stub" }
func (s *stubProvider) Chat(_ context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	s.requests = append(s.requests, req)
	r := s.responses[s.idx]
	s.idx++
	return r, nil
}

func TestProcessSimpleTextReply(t *testing.T) {
	reg := llm.NewRegistry()
	reg.Register(&stubProvider{responses: []llm.ChatResponse{{Text: "hello back", FinishReason: "stop"}}})

	res, err := Process(context.Background(), reg, Config{SystemPrompt: "helper"}, "hi")
	if err != nil {
		t.Fatalf("Process() err = %v", err)
	}
	if res.AssistantText != "hello back" {
		t.Fatalf("AssistantText = %q", res.AssistantText)
	}
}

func TestProcessToolCallThenText(t *testing.T) {
	reg := llm.NewRegistry()
	provider := &stubProvider{responses: []llm.ChatResponse{
		{ToolCalls: []llm.ToolCall{{ID: "1", Name: "search", Args: map[string]any{"q": "wrench"}}}, FinishReason: "tool_call"},
		{Text: "found 3 wrenches", FinishReason: "stop"},
	}}
	reg.Register(provider)

	dispatched := false
	res, err := Process(context.Background(), reg, Config{
		Dispatcher: func(_ context.Context, call llm.ToolCall) (any, error) {
			dispatched = true
			if call.Name != "search" {
				t.Fatalf("call.Name = %q", call.Name)
			}
			return map[string]any{"hits": 3}, nil
		},
	}, "find wrench")
	if err != nil {
		t.Fatalf("Process() err = %v", err)
	}
	if !dispatched || res.AssistantText != "found 3 wrenches" || len(res.ToolHistory) != 1 {
		t.Fatalf("unexpected result: dispatched=%v res=%+v", dispatched, res)
	}
	if len(provider.requests) != 2 {
		t.Fatalf("provider request count = %d, want 2", len(provider.requests))
	}
	// Continuation hop passes full History with empty UserText — no synthetic turn.
	if provider.requests[1].UserText != "" {
		t.Fatalf("second tool-hop request UserText must be empty (continuation hop), got %q", provider.requests[1].UserText)
	}
	if got := provider.requests[1].History[len(provider.requests[1].History)-1]; got.Role != "tool" {
		t.Fatalf("last history turn after tool call = %+v, want tool", got)
	}
}

func TestProcessMultiHopNoSyntheticTurn(t *testing.T) {
	reg := llm.NewRegistry()
	provider := &stubProvider{responses: []llm.ChatResponse{
		{ToolCalls: []llm.ToolCall{{ID: "1", Name: "search_products", Args: map[string]any{"query": "mug"}}}, FinishReason: "tool_call"},
		{Text: "I found a mug for you", FinishReason: "stop"},
	}}
	reg.Register(provider)

	dispatched := false
	res, err := Process(context.Background(), reg, Config{
		Dispatcher: func(_ context.Context, call llm.ToolCall) (any, error) {
			dispatched = true
			return map[string]any{"items": 1}, nil
		},
	}, "find a mug")
	if err != nil {
		t.Fatalf("Process() err = %v", err)
	}
	if !dispatched || res.AssistantText != "I found a mug for you" {
		t.Fatalf("unexpected result: dispatched=%v res=%+v", dispatched, res)
	}
	if len(provider.requests) != 2 {
		t.Fatalf("provider request count = %d, want 2", len(provider.requests))
	}

	// Hop 1 must pass full History with empty UserText — no synthetic "continue" turn.
	hop1 := provider.requests[1]
	if hop1.UserText != "" {
		t.Fatalf("hop 1 UserText must be empty (continuation hop), got %q", hop1.UserText)
	}

	// No synthetic turn in History.
	syntheticText := "Use the tool result above and continue with the final assistant reply."
	for _, turn := range hop1.History {
		if turn.Text == syntheticText {
			t.Fatalf("synthetic 'continue' turn found in hop 1 History: %+v", turn)
		}
	}

	// Tool turn must be present in History.
	foundTool := false
	for _, turn := range hop1.History {
		if turn.Role == "tool" && turn.ToolName == "search_products" {
			foundTool = true
		}
	}
	if !foundTool {
		t.Fatalf("tool turn not found in hop 1 History: %+v", hop1.History)
	}
}

func TestProcessHopLimit(t *testing.T) {
	reg := llm.NewRegistry()
	reg.Register(&stubProvider{responses: []llm.ChatResponse{
		{ToolCalls: []llm.ToolCall{{ID: "1", Name: "x"}}},
		{ToolCalls: []llm.ToolCall{{ID: "2", Name: "x"}}},
	}})
	_, err := Process(context.Background(), reg, Config{
		MaxToolHops: 2,
		Dispatcher:  func(context.Context, llm.ToolCall) (any, error) { return "ok", nil },
	}, "go")
	if err == nil || !strings.Contains(err.Error(), "tool hop limit") {
		t.Fatalf("err = %v, want hop limit", err)
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	got := BuildSystemPrompt("persona", ChannelPrompt{Channel: "telegram", Hint: "use markdown"})
	if got != "[Channel: telegram - use markdown]\n\npersona" {
		t.Fatalf("BuildSystemPrompt() = %q", got)
	}
	if BuildSystemPrompt("persona", ChannelPrompt{}) != "persona" {
		t.Fatal("empty hint should return persona unchanged")
	}
}

func TestDefaultChannelHintsHasExpectedChannels(t *testing.T) {
	hints := DefaultChannelHints()
	for _, channel := range []string{"twilio:voice", "twilio:whatsapp", "meta:whatsapp", "twilio:sms", "telegram", "web", "email:sendgrid", "email:ses"} {
		if _, ok := hints[channel]; !ok {
			t.Fatalf("missing channel hint %q", channel)
		}
	}
}
