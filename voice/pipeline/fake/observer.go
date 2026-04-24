package fake

import (
	"context"
	"sync"

	"github.com/mahdi-awadi/gopkg/voice/pipeline"
)

// RecorderObserver captures every callback into a typed slice.
// Safe for concurrent callbacks. Zero-alloc for unused methods.
type RecorderObserver struct {
	pipeline.NoopObserver
	mu     sync.Mutex
	events []any
}

// NewRecorder returns an empty recorder.
func NewRecorder() *RecorderObserver { return &RecorderObserver{} }

// Events returns a snapshot of recorded events in firing order.
func (r *RecorderObserver) Events() []any {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]any, len(r.events))
	copy(out, r.events)
	return out
}

// record appends an event under the lock.
func (r *RecorderObserver) record(ev any) {
	r.mu.Lock()
	r.events = append(r.events, ev)
	r.mu.Unlock()
}

// Typed records surfaced on Events().
type (
	RecSessionStart    struct{ S pipeline.Session }
	RecHistoryInjected struct {
		S     pipeline.Session
		Count int
	}
	RecCallerTranscript struct {
		S    pipeline.Session
		Text string
	}
	RecAssistantText struct {
		S     pipeline.Session
		Text  string
		Final bool
	}
	RecToolCall struct {
		S    pipeline.Session
		Call pipeline.ToolCall
	}
	RecToolResponse struct {
		S      pipeline.Session
		Call   pipeline.ToolCall
		Result any
		Err    error
	}
	RecInterrupted  struct{ S pipeline.Session }
	RecTurnComplete struct {
		S    pipeline.Session
		Turn int
	}
	RecError struct {
		S   pipeline.Session
		Err error
	}
	RecSessionEnd struct {
		S      pipeline.Session
		Reason pipeline.EndReason
	}
)

// Callback implementations:

func (r *RecorderObserver) OnSessionStart(_ context.Context, s pipeline.Session) {
	r.record(RecSessionStart{S: s})
}
func (r *RecorderObserver) OnHistoryInjected(_ context.Context, s pipeline.Session, c int) {
	r.record(RecHistoryInjected{S: s, Count: c})
}
func (r *RecorderObserver) OnCallerTranscript(_ context.Context, s pipeline.Session, t string) {
	r.record(RecCallerTranscript{S: s, Text: t})
}
func (r *RecorderObserver) OnAssistantText(_ context.Context, s pipeline.Session, t string, f bool) {
	r.record(RecAssistantText{S: s, Text: t, Final: f})
}
func (r *RecorderObserver) OnToolCall(_ context.Context, s pipeline.Session, c pipeline.ToolCall) {
	r.record(RecToolCall{S: s, Call: c})
}
func (r *RecorderObserver) OnToolResponse(_ context.Context, s pipeline.Session, c pipeline.ToolCall, res any, e error) {
	r.record(RecToolResponse{S: s, Call: c, Result: res, Err: e})
}
func (r *RecorderObserver) OnInterrupted(_ context.Context, s pipeline.Session) {
	r.record(RecInterrupted{S: s})
}
func (r *RecorderObserver) OnTurnComplete(_ context.Context, s pipeline.Session, t int) {
	r.record(RecTurnComplete{S: s, Turn: t})
}
func (r *RecorderObserver) OnError(_ context.Context, s pipeline.Session, e error) {
	r.record(RecError{S: s, Err: e})
}
func (r *RecorderObserver) OnSessionEnd(_ context.Context, s pipeline.Session, reason pipeline.EndReason) {
	r.record(RecSessionEnd{S: s, Reason: reason})
}

var _ pipeline.Observer = (*RecorderObserver)(nil)
