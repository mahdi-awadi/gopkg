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
