// Package orchestrator wires text LLM providers, conversation history,
// and caller-owned tool dispatch into a small domain-neutral loop.
package orchestrator

import (
	"context"
	"fmt"

	"github.com/mahdi-awadi/gopkg/ai/conversation"
	"github.com/mahdi-awadi/gopkg/ai/llm"
)

// ToolDispatcher executes a model-requested tool call.
type ToolDispatcher func(ctx context.Context, call llm.ToolCall) (any, error)

// Config configures one inbound message processing pass.
type Config struct {
	SystemPrompt string
	Tools        []llm.ToolDecl
	LocaleHint   string
	History      []conversation.Message
	MaxToolHops  int
	Dispatcher   ToolDispatcher
}

// Result is the final assistant reply plus any tool executions.
type Result struct {
	AssistantText string
	ToolHistory   []ToolExecution
	FinishReason  string
}

// ToolExecution is one tool call and its result.
type ToolExecution struct {
	Call   llm.ToolCall
	Result any
	Error  error
}

// Process runs one user-message to assistant-reply cycle.
func Process(ctx context.Context, registry *llm.Registry, cfg Config, userText string) (*Result, error) {
	if cfg.MaxToolHops <= 0 {
		cfg.MaxToolHops = 5
	}

	turns := messagesToTurns(cfg.History)
	turns = append(turns, llm.ChatTurn{Role: "user", Text: userText})

	var toolHistory []ToolExecution
	for hop := 0; hop < cfg.MaxToolHops; hop++ {
		req := llm.ChatRequest{
			SystemPrompt: cfg.SystemPrompt,
			History:      turns, // full list; tool turns render as call+response pairs
			UserText:     "",    // no new user text on continuation hops
			Tools:        cfg.Tools,
			LocaleHint:   cfg.LocaleHint,
		}
		resp, err := registry.Chat(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("orchestrator: llm chat: %w", err)
		}
		if len(resp.ToolCalls) == 0 {
			return &Result{
				AssistantText: resp.Text,
				ToolHistory:   toolHistory,
				FinishReason:  resp.FinishReason,
			}, nil
		}

		for _, call := range resp.ToolCalls {
			if cfg.Dispatcher == nil {
				return nil, fmt.Errorf("orchestrator: model called %q but no dispatcher provided", call.Name)
			}
			result, toolErr := cfg.Dispatcher(ctx, call)
			toolHistory = append(toolHistory, ToolExecution{Call: call, Result: result, Error: toolErr})
			turns = append(turns, llm.ChatTurn{Role: "tool", ToolName: call.Name, ToolArgs: call.Args, ToolResult: result})
		}
		// NO synthetic user turn — the tool turns above already render as
		// functionCall + functionResponse pairs, which is a complete continuation.
	}

	return nil, fmt.Errorf("orchestrator: tool hop limit (%d) reached without a final text reply", cfg.MaxToolHops)
}

func messagesToTurns(msgs []conversation.Message) []llm.ChatTurn {
	out := make([]llm.ChatTurn, 0, len(msgs))
	for _, m := range msgs {
		role := "user"
		if m.Role == "assistant" || m.Role == "model" {
			role = "assistant"
		}
		out = append(out, llm.ChatTurn{Role: role, Text: m.Content})
	}
	return out
}
