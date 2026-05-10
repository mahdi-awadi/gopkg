// Package toolexec is a thin pipeline.ToolExecutor wrapper around a
// caller-supplied dispatch function. It exists so consumers can keep
// per-call context (auth bits, tenant IDs, registry pointers) in their
// own closure rather than threading it through this package as named
// arguments.
//
// Used by:
//   - eticket-v3 communication-service voice handler — closure over
//     ServiceClients.Invoke that pulls user_token / user_id / partner_id
//     / conv_id from pipeline.Session.Attrs.
//   - saffar apps/voice — closure over the per-call tool Registry.
package toolexec

import (
	"context"
	"errors"

	"github.com/mahdi-awadi/gopkg/voice/pipeline"
)

// Func is a tool-dispatch function. Implementations route by call.Name
// and return either an arbitrary serialisable value (success) or an
// error the pipeline will surface as ToolResult.Err to the LLM.
type Func func(ctx context.Context, call pipeline.ToolCall, sess pipeline.Session) (any, error)

// Executor wraps a Func as a pipeline.ToolExecutor.
type Executor struct {
	fn Func
}

// New returns an Executor backed by the given dispatch function.
// Passing nil is allowed but Execute will then return an error for every
// call — useful only for tests that exercise the no-tools path.
func New(fn Func) *Executor {
	return &Executor{fn: fn}
}

// Execute implements pipeline.ToolExecutor.
func (e *Executor) Execute(ctx context.Context, call pipeline.ToolCall, sess pipeline.Session) (any, error) {
	if e.fn == nil {
		return nil, errNilFunc
	}
	return e.fn(ctx, call, sess)
}

var errNilFunc = errors.New("toolexec: Executor has no dispatch func")

// Compile-time check.
var _ pipeline.ToolExecutor = (*Executor)(nil)
