package fake

import (
	"context"
	"errors"
	"sync"

	"github.com/mahdi-awadi/gopkg/voice/pipeline"
)

// Executor is a programmable pipeline.ToolExecutor.
// Register handlers per tool name; unregistered calls return an error.
type Executor struct {
	mu       sync.Mutex
	handlers map[string]func(context.Context, pipeline.ToolCall, pipeline.Session) (any, error)
	calls    []pipeline.ToolCall
}

// NewExecutor returns an empty Executor.
func NewExecutor() *Executor {
	return &Executor{handlers: make(map[string]func(context.Context, pipeline.ToolCall, pipeline.Session) (any, error))}
}

// Register installs a handler for the tool of the given name.
func (e *Executor) Register(name string, fn func(context.Context, pipeline.ToolCall, pipeline.Session) (any, error)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers[name] = fn
}

// Calls returns a snapshot of every ToolCall received.
func (e *Executor) Calls() []pipeline.ToolCall {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]pipeline.ToolCall, len(e.calls))
	copy(out, e.calls)
	return out
}

// Execute implements pipeline.ToolExecutor.
func (e *Executor) Execute(ctx context.Context, call pipeline.ToolCall, s pipeline.Session) (any, error) {
	e.mu.Lock()
	e.calls = append(e.calls, call)
	fn, ok := e.handlers[call.Name]
	e.mu.Unlock()
	if !ok {
		return nil, errors.New("fake executor: no handler for " + call.Name)
	}
	return fn(ctx, call, s)
}

var _ pipeline.ToolExecutor = (*Executor)(nil)
