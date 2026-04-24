package pipeline

import "context"

// Observer receives business-event callbacks synchronously from the
// pipeline's read loops. Slow callbacks block the pipeline; push work
// into a goroutine from inside a callback if you need async behavior.
type Observer interface {
	OnSessionStart(ctx context.Context, s Session)
	OnHistoryInjected(ctx context.Context, s Session, count int)
	OnCallerTranscript(ctx context.Context, s Session, text string)
	OnAssistantText(ctx context.Context, s Session, text string, final bool)
	OnToolCall(ctx context.Context, s Session, call ToolCall)
	OnToolResponse(ctx context.Context, s Session, call ToolCall, result any, err error)
	OnInterrupted(ctx context.Context, s Session)
	OnTurnComplete(ctx context.Context, s Session, turn int)
	OnError(ctx context.Context, s Session, err error)
	OnSessionEnd(ctx context.Context, s Session, reason EndReason)
}

// NoopObserver is an Observer with no-op methods. Embed it in your
// observer type and override only the methods you care about.
type NoopObserver struct{}

func (NoopObserver) OnSessionStart(context.Context, Session)                       {}
func (NoopObserver) OnHistoryInjected(context.Context, Session, int)               {}
func (NoopObserver) OnCallerTranscript(context.Context, Session, string)           {}
func (NoopObserver) OnAssistantText(context.Context, Session, string, bool)        {}
func (NoopObserver) OnToolCall(context.Context, Session, ToolCall)                 {}
func (NoopObserver) OnToolResponse(context.Context, Session, ToolCall, any, error) {}
func (NoopObserver) OnInterrupted(context.Context, Session)                        {}
func (NoopObserver) OnTurnComplete(context.Context, Session, int)                  {}
func (NoopObserver) OnError(context.Context, Session, error)                       {}
func (NoopObserver) OnSessionEnd(context.Context, Session, EndReason)              {}

// Multi returns an Observer that forwards every callback to each
// argument in registration order. Panics if any arg is nil.
func Multi(obs ...Observer) Observer {
	for i, o := range obs {
		if o == nil {
			panic("pipeline.Multi: nil observer at index " + itoa(i))
		}
	}
	cp := make([]Observer, len(obs))
	copy(cp, obs)
	return multiObserver(cp)
}

type multiObserver []Observer

func (m multiObserver) OnSessionStart(ctx context.Context, s Session) {
	for _, o := range m {
		o.OnSessionStart(ctx, s)
	}
}
func (m multiObserver) OnHistoryInjected(ctx context.Context, s Session, c int) {
	for _, o := range m {
		o.OnHistoryInjected(ctx, s, c)
	}
}
func (m multiObserver) OnCallerTranscript(ctx context.Context, s Session, t string) {
	for _, o := range m {
		o.OnCallerTranscript(ctx, s, t)
	}
}
func (m multiObserver) OnAssistantText(ctx context.Context, s Session, t string, f bool) {
	for _, o := range m {
		o.OnAssistantText(ctx, s, t, f)
	}
}
func (m multiObserver) OnToolCall(ctx context.Context, s Session, c ToolCall) {
	for _, o := range m {
		o.OnToolCall(ctx, s, c)
	}
}
func (m multiObserver) OnToolResponse(ctx context.Context, s Session, c ToolCall, r any, e error) {
	for _, o := range m {
		o.OnToolResponse(ctx, s, c, r, e)
	}
}
func (m multiObserver) OnInterrupted(ctx context.Context, s Session) {
	for _, o := range m {
		o.OnInterrupted(ctx, s)
	}
}
func (m multiObserver) OnTurnComplete(ctx context.Context, s Session, t int) {
	for _, o := range m {
		o.OnTurnComplete(ctx, s, t)
	}
}
func (m multiObserver) OnError(ctx context.Context, s Session, e error) {
	for _, o := range m {
		o.OnError(ctx, s, e)
	}
}
func (m multiObserver) OnSessionEnd(ctx context.Context, s Session, r EndReason) {
	for _, o := range m {
		o.OnSessionEnd(ctx, s, r)
	}
}

// itoa is a tiny stdlib-free int-to-string for the panic message.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
