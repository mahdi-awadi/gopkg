package geminitext

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mahdi-awadi/gopkg/ai/llm"
)

func TestChatParsesTextAndToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/test-model:generateContent" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("X-goog-api-key"); got != "key" {
			t.Fatalf("api key header = %q", got)
		}
		_ = json.NewEncoder(w).Encode(geminiResponse{
			Candidates: []geminiCandidate{{
				FinishReason: "STOP",
				Content: geminiContent{Parts: []geminiPart{
					{Text: "Let me check."},
					{FunctionCall: &geminiFunctionCall{Name: "search", Args: map[string]any{"q": "wrench"}}},
				}},
			}},
			UsageMetadata: usageMetadata{PromptTokenCount: 1, CandidatesTokenCount: 2, TotalTokenCount: 3},
		})
	}))
	defer server.Close()

	c := New("gemini", "key", "test-model", WithAPIURL(server.URL))
	resp, err := c.Chat(context.Background(), llm.ChatRequest{
		SystemPrompt: "system",
		UserText:     "hello",
		Tools: []llm.ToolDecl{{
			Name:        "search",
			Description: "search catalog",
			Parameters: llm.ToolSchema{
				Type:       "object",
				Properties: map[string]llm.ToolProperty{"q": {Type: "string"}},
				Required:   []string{"q"},
			},
		}},
	})
	if err != nil {
		t.Fatalf("Chat() err = %v", err)
	}
	if resp.Text != "Let me check." || len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "search" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Usage.TotalTokens != 3 {
		t.Fatalf("usage = %+v", resp.Usage)
	}
}

func TestChatRequiresAPIKey(t *testing.T) {
	c := New("", "", "")
	if _, err := c.Chat(context.Background(), llm.ChatRequest{}); err == nil {
		t.Fatal("expected api key error")
	}
}

func TestChatContents_ToolTurnEmitsCallThenResponse(t *testing.T) {
	req := llm.ChatRequest{
		History: []llm.ChatTurn{
			{Role: "user", Text: "find a mug"},
			{Role: "tool", ToolName: "search_products", ToolArgs: map[string]any{"query": "mug"}, ToolResult: map[string]any{"items": 1}},
		},
		UserText: "", // continuation hop: no new user text
	}
	c := chatContents(req)
	// expect: [user:"find a mug"], [model: functionCall search_products], [user: functionResponse]
	if len(c) != 3 {
		t.Fatalf("want 3 contents, got %d: %+v", len(c), c)
	}
	if c[1].Role != "model" || c[1].Parts[0].FunctionCall == nil || c[1].Parts[0].FunctionCall.Name != "search_products" {
		t.Fatalf("content[1] must be a model functionCall for search_products, got %+v", c[1])
	}
	if c[2].Role != "user" || c[2].Parts[0].FunctionResponse == nil || c[2].Parts[0].FunctionResponse.Name != "search_products" {
		t.Fatalf("content[2] must be a user functionResponse for search_products, got %+v", c[2])
	}
}

func TestChatContents_ParallelToolCalls_Grouped(t *testing.T) {
	req := llm.ChatRequest{
		History: []llm.ChatTurn{
			{Role: "user", Text: "x"},
			{Role: "tool", ToolName: "toolA", ToolArgs: map[string]any{"a": 1}, ToolResult: map[string]any{"ra": 1}},
			{Role: "tool", ToolName: "toolB", ToolArgs: map[string]any{"b": 2}, ToolResult: map[string]any{"rb": 2}},
		},
		UserText: "",
	}
	c := chatContents(req)
	// Expect: [0] user "x"; [1] model with 2 FunctionCall parts; [2] user with 2 FunctionResponse parts.
	if len(c) != 3 {
		t.Fatalf("want 3 contents, got %d: %+v", len(c), c)
	}
	// [0] user text
	if c[0].Role != "user" || c[0].Parts[0].Text != "x" {
		t.Fatalf("content[0] must be user 'x', got %+v", c[0])
	}
	// [1] model with TWO FunctionCall parts
	if c[1].Role != "model" {
		t.Fatalf("content[1] must be role model, got %q", c[1].Role)
	}
	if len(c[1].Parts) != 2 {
		t.Fatalf("content[1] must have 2 parts (one per tool call), got %d: %+v", len(c[1].Parts), c[1].Parts)
	}
	if c[1].Parts[0].FunctionCall == nil || c[1].Parts[0].FunctionCall.Name != "toolA" {
		t.Fatalf("content[1].Parts[0] must be FunctionCall toolA, got %+v", c[1].Parts[0])
	}
	if c[1].Parts[1].FunctionCall == nil || c[1].Parts[1].FunctionCall.Name != "toolB" {
		t.Fatalf("content[1].Parts[1] must be FunctionCall toolB, got %+v", c[1].Parts[1])
	}
	// [2] user with TWO FunctionResponse parts
	if c[2].Role != "user" {
		t.Fatalf("content[2] must be role user, got %q", c[2].Role)
	}
	if len(c[2].Parts) != 2 {
		t.Fatalf("content[2] must have 2 parts (one per tool response), got %d: %+v", len(c[2].Parts), c[2].Parts)
	}
	if c[2].Parts[0].FunctionResponse == nil || c[2].Parts[0].FunctionResponse.Name != "toolA" {
		t.Fatalf("content[2].Parts[0] must be FunctionResponse toolA, got %+v", c[2].Parts[0])
	}
	if c[2].Parts[1].FunctionResponse == nil || c[2].Parts[1].FunctionResponse.Name != "toolB" {
		t.Fatalf("content[2].Parts[1] must be FunctionResponse toolB, got %+v", c[2].Parts[1])
	}
}

// TestChatContents_ReplaysThoughtSignature guards the Gemini 3.x requirement:
// a replayed functionCall MUST carry the thoughtSignature the model returned, or
// the API 400s ("Function call is missing a thought_signature").
func TestChatContents_ReplaysThoughtSignature(t *testing.T) {
	req := llm.ChatRequest{
		History: []llm.ChatTurn{
			{Role: "user", Text: "mug?"},
			{Role: "tool", ToolName: "search_products", ToolArgs: map[string]any{"query": "mug"},
				ToolResult: map[string]any{"n": 1}, ToolSig: "SIG123"},
		},
	}
	c := chatContents(req)
	if len(c) < 2 || len(c[1].Parts) == 0 || c[1].Parts[0].FunctionCall == nil {
		t.Fatalf("expected a model functionCall content at index 1, got %+v", c)
	}
	if got := c[1].Parts[0].ThoughtSignature; got != "SIG123" {
		t.Fatalf("thoughtSignature not replayed on functionCall: got %q want SIG123", got)
	}
}
