// Package geminitext implements the llm.Provider contract using the
// Gemini generateContent text API.
package geminitext

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mahdi-awadi/gopkg/ai/llm"
)

const defaultAPIURL = "https://generativelanguage.googleapis.com/v1beta/models"

// Client is a Gemini text LLM provider.
type Client struct {
	name       string
	apiKey     string
	apiURL     string
	model      string
	httpClient *http.Client
}

var _ llm.Provider = (*Client)(nil)

// Option configures a Client.
type Option func(*Client)

// WithAPIURL overrides the Gemini base API URL. Mostly useful for tests.
func WithAPIURL(apiURL string) Option {
	return func(c *Client) {
		if apiURL != "" {
			c.apiURL = apiURL
		}
	}
}

// WithHTTPClient overrides the HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// New creates a Gemini text provider.
func New(name, apiKey, model string, opts ...Option) *Client {
	if name == "" {
		name = "gemini"
	}
	if model == "" {
		model = "gemini-2.5-flash-lite"
	}
	c := &Client{
		name:       name,
		apiKey:     apiKey,
		apiURL:     defaultAPIURL,
		model:      model,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Name identifies this provider.
func (c *Client) Name() string {
	if c == nil {
		return "gemini:<nil>"
	}
	return c.name + ":" + c.model
}

// IsAvailable reports whether the provider has credentials.
func (c *Client) IsAvailable() bool {
	return c != nil && c.apiKey != ""
}

// Chat sends one text-mode request to Gemini.
func (c *Client) Chat(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	return c.ChatWithConfig(ctx, req, 0.7, 2048, "")
}

// GenerateText sends a plain text prompt without tools.
func (c *Client) GenerateText(ctx context.Context, prompt string) (string, error) {
	resp, err := c.ChatWithConfig(ctx, llm.ChatRequest{UserText: prompt}, 0.7, 500, "")
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

// GenerateTextWithConfig sends a plain text prompt with caller-supplied
// generation settings.
func (c *Client) GenerateTextWithConfig(ctx context.Context, prompt string, temperature float64, maxOutputTokens int, model string) (string, error) {
	resp, err := c.ChatWithConfig(ctx, llm.ChatRequest{UserText: prompt}, temperature, maxOutputTokens, model)
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

// ChatWithConfig sends one text-mode request with caller-supplied generation
// settings. Zero values fall back to the normal Chat defaults.
func (c *Client) ChatWithConfig(ctx context.Context, req llm.ChatRequest, temperature float64, maxOutputTokens int, model string) (llm.ChatResponse, error) {
	if c == nil || c.apiKey == "" {
		return llm.ChatResponse{}, fmt.Errorf("geminitext: api key required")
	}
	if temperature == 0 {
		temperature = 0.7
	}
	if maxOutputTokens == 0 {
		maxOutputTokens = 2048
	}
	useModel := c.model
	if model != "" {
		useModel = model
	}

	greq := geminiRequest{
		Contents:         chatContents(req),
		GenerationConfig: &generationConfig{Temperature: temperature, TopK: 40, TopP: 0.95, MaxOutputTokens: maxOutputTokens},
	}
	if req.SystemPrompt != "" {
		greq.SystemInstruction = &geminiContent{Parts: []geminiPart{{Text: req.SystemPrompt}}}
	}
	if len(req.Tools) > 0 {
		greq.Tools = []geminiTool{{FunctionDeclarations: toolDecls(req.Tools)}}
		greq.ToolConfig = &toolConfig{FunctionCallingConfig: &functionCallingConfig{Mode: "AUTO"}}
	}

	body, err := json.Marshal(greq)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf("geminitext: marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/%s:generateContent", c.apiURL, useModel), bytes.NewReader(body))
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf("geminitext: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-goog-api-key", c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf("geminitext: call api: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf("geminitext: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return llm.ChatResponse{}, fmt.Errorf("geminitext: api status %d: %s", resp.StatusCode, string(raw))
	}

	var gres geminiResponse
	if err := json.Unmarshal(raw, &gres); err != nil {
		return llm.ChatResponse{}, fmt.Errorf("geminitext: unmarshal response: %w", err)
	}
	return chatResponse(gres), nil
}

func chatContents(req llm.ChatRequest) []geminiContent {
	out := make([]geminiContent, 0, len(req.History)+1)
	for _, turn := range req.History {
		out = append(out, turnContent(turn))
	}
	out = append(out, geminiContent{Role: "user", Parts: []geminiPart{{Text: req.UserText}}})
	return out
}

func turnContent(turn llm.ChatTurn) geminiContent {
	role := "user"
	if turn.Role == "assistant" {
		role = "model"
	}
	if turn.Role == "tool" {
		return geminiContent{
			Role: "function",
			Parts: []geminiPart{{
				FunctionResponse: &geminiFunctionResponse{
					Name:     turn.ToolName,
					Response: map[string]any{"result": turn.ToolResult},
				},
			}},
		}
	}
	return geminiContent{Role: role, Parts: []geminiPart{{Text: turn.Text}}}
}

func toolDecls(tools []llm.ToolDecl) []geminiFunction {
	out := make([]geminiFunction, 0, len(tools))
	for _, t := range tools {
		props := make(map[string]geminiFunctionProperty, len(t.Parameters.Properties))
		for name, p := range t.Parameters.Properties {
			props[name] = geminiFunctionProperty{
				Type:        p.Type,
				Description: p.Description,
				Enum:        p.Enum,
				Format:      p.Format,
			}
		}
		out = append(out, geminiFunction{
			Name:        t.Name,
			Description: t.Description,
			Parameters: geminiFunctionParams{
				Type:       t.Parameters.Type,
				Properties: props,
				Required:   t.Parameters.Required,
			},
		})
	}
	return out
}

func chatResponse(resp geminiResponse) llm.ChatResponse {
	if len(resp.Candidates) == 0 {
		return llm.ChatResponse{FinishReason: "empty"}
	}
	cand := resp.Candidates[0]
	out := llm.ChatResponse{
		FinishReason: cand.FinishReason,
		Usage: llm.TokenUsage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		},
	}
	for _, part := range cand.Content.Parts {
		if part.Text != "" {
			out.Text += part.Text
		}
		if part.FunctionCall != nil {
			out.ToolCalls = append(out.ToolCalls, llm.ToolCall{
				ID:   part.FunctionCall.Name,
				Name: part.FunctionCall.Name,
				Args: part.FunctionCall.Args,
			})
		}
	}
	return out
}
